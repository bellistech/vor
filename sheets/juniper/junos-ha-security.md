# JunOS Security HA (Chassis Cluster)

SRX chassis cluster provides stateful failover with session synchronization. Redundancy groups control failover units, fabric links carry control and data plane synchronization, and reth interfaces provide a single logical interface across nodes.

## Chassis Cluster Fundamentals

### Architecture
```
                    ┌──── Fabric Links ────┐
                    │  Control (fxp1)      │
                    │  Data    (fab0/fab1) │
   ┌────────────┐   │                      │   ┌────────────┐
   │  Node 0    │◄──┘                      └──►│  Node 1    │
   │  (primary) │                              │  (secondary)│
   │            │                              │            │
   │  RG0: pri  │                              │  RG0: sec  │
   │  RG1: pri  │                              │  RG1: sec  │
   └────────────┘                              └────────────┘
        │                                            │
    reth0 (active)                             reth0 (standby)
        │                                            │
   ─────┴────────────────────────────────────────────┴─────
                        Network
```

### Modes
```
# Active/Passive: one node handles all traffic, other is standby
#   - RG1 primary on one node, secondary on the other
#   - Simple, most common for branch/edge deployments

# Active/Active: both nodes forward traffic simultaneously
#   - Multiple RGs (RG1, RG2, etc.) with different primaries
#   - Each RG owns a set of reth interfaces
#   - Both nodes actively forward traffic for their respective RGs
#   - More complex, used for load distribution in large deployments
```

## Cluster Configuration

### Initial cluster setup (both nodes)
```
# On Node 0:
set groups node0 system host-name SRX-NODE0
set groups node0 interfaces fxp0 unit 0 family inet address 10.1.0.1/24
set apply-groups node0

# On Node 1:
set groups node1 system host-name SRX-NODE1
set groups node1 interfaces fxp0 unit 0 family inet address 10.1.0.2/24
set apply-groups node1

# Cluster configuration (same on both nodes)
set chassis cluster cluster-id 1 node 0 reboot
set chassis cluster cluster-id 1 node 1 reboot
# WARNING: this command reboots the device immediately
```

### Cluster ID and node ID
```
# Set cluster-id and node-id (requires reboot)
set chassis cluster cluster-id 1 node 0 reboot    # run on node 0
set chassis cluster cluster-id 1 node 1 reboot    # run on node 1

# Cluster ID: identifies the cluster (1-15 for SRX)
# Node ID: 0 or 1 within a cluster
# Cluster ID 0 disables clustering
```

## Redundancy Groups

### RG0 — Control plane
```
# RG0 manages the Routing Engine — always exists in a cluster
# The RG0 primary node is the "master" RE — handles all config and control plane

set chassis cluster redundancy-group 0 node 0 priority 200
set chassis cluster redundancy-group 0 node 1 priority 100
# Higher priority = preferred primary
# Node 0 will be RG0 primary (handles control plane)
```

### RG1+ — Data plane
```
# RG1 and above manage reth interfaces and data plane forwarding
set chassis cluster redundancy-group 1 node 0 priority 200
set chassis cluster redundancy-group 1 node 1 priority 100

# Active/active: create RG2 with opposite priority
set chassis cluster redundancy-group 2 node 0 priority 100
set chassis cluster redundancy-group 2 node 1 priority 200
# RG1 primary on node 0, RG2 primary on node 1
```

### Failover thresholds
```
# Threshold: a weight value that triggers failover when exceeded
# Each monitored item has a weight; when sum of failed weights >= threshold, RG fails over
set chassis cluster redundancy-group 1 node 0 priority 200
set chassis cluster redundancy-group 1 node 1 priority 100

# Interface monitoring
set chassis cluster redundancy-group 1 interface-monitor ge-0/0/1 weight 255
set chassis cluster redundancy-group 1 interface-monitor ge-0/0/2 weight 128
set chassis cluster redundancy-group 1 interface-monitor ge-5/0/1 weight 255
set chassis cluster redundancy-group 1 interface-monitor ge-5/0/2 weight 128
# Weight 255 = critical (single failure triggers failover)
# Weight 128 = important (two failures trigger failover)

# Default threshold is 255
```

### IP monitoring
```
# Monitor reachability of upstream/downstream devices
set chassis cluster redundancy-group 1 ip-monitoring global-weight 255
set chassis cluster redundancy-group 1 ip-monitoring global-threshold 200

set chassis cluster redundancy-group 1 ip-monitoring family inet 10.0.0.1 weight 100
set chassis cluster redundancy-group 1 ip-monitoring family inet 10.0.0.1 interface ge-0/0/1
set chassis cluster redundancy-group 1 ip-monitoring family inet 10.0.0.1 secondary-ip-address 10.0.1.1

set chassis cluster redundancy-group 1 ip-monitoring family inet 10.0.0.2 weight 100
set chassis cluster redundancy-group 1 ip-monitoring family inet 10.0.0.2 interface ge-0/0/2

# When sum of failed IP monitors >= global-threshold → RG failover
```

### Preemption
```
# Preemption: allows higher-priority node to reclaim RG after recovery
set chassis cluster redundancy-group 1 preempt
set chassis cluster redundancy-group 1 preempt delay 300     # wait 300 sec before preempting
set chassis cluster redundancy-group 1 preempt period 300    # re-evaluate every 300 sec
set chassis cluster redundancy-group 1 preempt limit 3       # max preemption attempts

# Without preemption: after failover, traffic stays on secondary even when primary recovers
# With preemption: primary reclaims after delay (allows for route convergence)
# RG0 does NOT support preemption — manual switchback only
```

## Fabric Links

### Control link (fxp1)
```
# Control link: heartbeat, config sync, RG negotiation
# MUST be a dedicated point-to-point link between nodes
set interfaces fxp1 unit 0 family inet address 100.64.0.0/31    # node 0
set interfaces fxp1 unit 0 family inet address 100.64.0.1/31    # node 1

# Heartbeat interval: 1 second (not configurable)
# Heartbeat miss threshold: configurable via heartbeat-threshold
set chassis cluster heartbeat-threshold 8     # 8 missed = declare peer dead (default: 3)
```

### Data link (fab0/fab1)
```
# Data fabric: session sync (RTOs), transit traffic for reth backup path
set interfaces fab0 fabric-options member-interfaces ge-0/0/3    # node 0 fabric
set interfaces fab1 fabric-options member-interfaces ge-5/0/3    # node 1 fabric

# Redundant fabric: use multiple physical interfaces
set interfaces fab0 fabric-options member-interfaces ge-0/0/3
set interfaces fab0 fabric-options member-interfaces ge-0/0/4
set interfaces fab1 fabric-options member-interfaces ge-5/0/3
set interfaces fab1 fabric-options member-interfaces ge-5/0/4

# Data link carries:
#   - RTO (real-time objects): session table sync, NAT state, IDP state
#   - Transit traffic: packets arriving on standby node's physical interface
```

## Reth Interfaces

### Configuration
```
# Reth (redundant ethernet): logical interface spanning both nodes
# Child interfaces from each node are bound to the reth
set chassis cluster reth-count 4    # max reth interfaces (must set before configuring)

set interfaces reth0 redundant-ether-options redundancy-group 1
set interfaces reth0 unit 0 family inet address 10.0.0.1/24

# Bind physical interfaces to reth
set interfaces ge-0/0/0 gigether-options redundant-parent reth0    # node 0 member
set interfaces ge-5/0/0 gigether-options redundant-parent reth0    # node 1 member

# Active reth member: the physical interface on the RG primary node
# Standby reth member: physical interface on secondary (does not forward)
```

### Reth with LACP
```
# Reth can use LACP aggregation on each node
set chassis cluster reth-count 2
set interfaces reth0 redundant-ether-options redundancy-group 1
set interfaces reth0 redundant-ether-options lacp active
set interfaces reth0 redundant-ether-options lacp periodic fast

set interfaces ge-0/0/0 gigether-options 802.3ad reth0
set interfaces ge-0/0/1 gigether-options 802.3ad reth0
set interfaces ge-5/0/0 gigether-options 802.3ad reth0
set interfaces ge-5/0/1 gigether-options 802.3ad reth0
```

## Session Synchronization

### RTO (Real-Time Objects)
```
# RTOs synchronize stateful data over the fabric data link:
#   - Security flow sessions (full 5-tuple + state + timers)
#   - NAT translations (source, destination, persistent bindings)
#   - IPsec SA state (for VPN session continuity)
#   - IDP/IPS session state
#   - ALG state (pinholes, protocol state machines)
#   - Screen session counters

# Sync is continuous — every session creation, modification, and deletion is synced
# Small async window: ~100ms of sessions may be lost on failover
```

### Verify session sync
```
show chassis cluster status
show chassis cluster data-plane interfaces
show chassis cluster control-plane statistics
show chassis cluster statistics

# Session count comparison
show security flow session summary         # on active node
# Compare with standby via:
show chassis cluster information detail
```

## HA with VPN

### IPsec HA configuration
```
# IPsec sessions are synced via RTOs
# On failover, IKE SA and IPsec SA are maintained
# Peer sees no tunnel renegotiation (graceful)

set security ike gateway VPN-GW address 198.51.100.1
set security ike gateway VPN-GW external-interface reth0      # use reth, not physical
set security ike gateway VPN-GW ike-policy IKE-POL

set security ipsec vpn SITE-VPN bind-interface st0.0
set security ipsec vpn SITE-VPN ike gateway VPN-GW
set security ipsec vpn SITE-VPN ike ipsec-policy IPSEC-POL

# st0 interface assigned to redundancy group
set interfaces st0 unit 0 family inet address 169.254.0.1/30
set chassis cluster redundancy-group 1 interface-monitor st0 weight 200
```

### VPN HA considerations
```
# - Always use reth as external-interface for IKE gateway
# - Tunnel interface (st0) must be in the same RG as the reth
# - DPD (Dead Peer Detection) should account for failover time
#   set security ike gateway VPN-GW dead-peer-detection interval 20 threshold 5
# - Peer device should have long enough DPD timeout to survive failover
# - IKE SA rekey during failover may require renegotiation
```

## HA with NAT

### NAT state synchronization
```
# NAT sessions and persistent NAT table are synced via RTOs
# On failover:
#   - Active sessions with SNAT/DNAT continue without renegotiation
#   - Persistent NAT bindings survive — external hosts can still reach mapped addresses
#   - NAT pool allocation state is synced — no port conflicts

# Requirements:
#   - Use reth interfaces (not physical) for NAT rule from/to zones
#   - NAT pool addresses should be reachable via reth (ARP handled by active node)
#   - Avoid interface-based SNAT on physical interfaces — use reth-based
```

## Manual Failover

### Perform manual failover
```
# Failover RG1 to the secondary node
request chassis cluster failover redundancy-group 1 node 1

# Failover RG0 (control plane) to the secondary node
request chassis cluster failover redundancy-group 0 node 1

# Reset failover (restore to configured priorities)
request chassis cluster failover reset redundancy-group 1
```

### Disable/enable a node
```
# Disable a node (for maintenance)
request chassis cluster disable node 1

# Re-enable
request chassis cluster enable node 1

# Disable a redundancy group on a node
request chassis cluster disable redundancy-group 1 node 1
```

## Graceful Switchover (GRES)

### Configuration
```
# GRES: Routing Engine switchover preserving kernel state
set chassis redundancy graceful-switchover

# Nonstop active routing (NSR): protocols maintain adjacency during switchover
set routing-options nonstop-routing

# Combined GRES + NSR ensures:
#   - Routing protocol adjacencies maintained
#   - Forwarding table preserved
#   - Security sessions preserved (via RTOs)
#   - Minimal traffic loss during RE switchover
```

## Multi-Node HA (SRX4600)

### Concept
```
# Multi-node HA: 2+ SRX4600 devices in a resilient cluster
# Uses VXLAN-based data plane interconnect
# Each node independently processes traffic
# Session sync across all nodes via VXLAN fabric
# Supports N+M redundancy (N active, M standby)

# Configuration differs from traditional chassis cluster
set multi-node high-availability peer-id 1 ip-address 10.0.0.1
set multi-node high-availability peer-id 2 ip-address 10.0.0.2
```

## Verification Commands

### Cluster status
```
show chassis cluster status                              # overall cluster health + RG status
show chassis cluster status redundancy-group 0           # specific RG status
show chassis cluster status redundancy-group 1
show chassis cluster information                         # node info, cluster ID
show chassis cluster information detail                  # verbose cluster state
```

### Fabric and heartbeat
```
show chassis cluster control-plane statistics            # heartbeat stats, missed beats
show chassis cluster data-plane interfaces               # fabric link status
show chassis cluster statistics                          # RTO sync counters
```

### Interface status
```
show chassis cluster interfaces                          # reth status + member interfaces
show interfaces reth0 terse                              # reth operational state
show interfaces reth0 detail                             # reth member details
show chassis cluster interface-monitor                   # monitored interfaces + weights
```

### Session synchronization
```
show chassis cluster ip-monitoring status                # IP monitor status
show security flow session summary                       # session counts on active node
show security flow session count                         # fast session count
show chassis cluster data-plane statistics               # RTO sync statistics
```

### Failover history
```
show chassis cluster failover-count                      # number of failovers per RG
show chassis cluster switch-events                       # failover event log
show log messages | match "chassis-cluster"              # syslog events
show log jsrpd                                           # JSRP daemon log (cluster events)
```

## Tips

- Always configure heartbeat-threshold conservatively — false failovers are worse than delayed failovers
- Use dedicated physical links for fxp1 (control) and fab (data) — never share with transit traffic
- Set preemption with a delay (300+ seconds) to allow routing convergence before failover back
- Interface monitoring weight 255 means a single interface failure triggers failover — use lower weights for non-critical interfaces
- RG0 failover changes the master RE — this disrupts the CLI session and management plane
- Always use reth interfaces in security zones, NAT, and VPN configs — never reference physical interfaces for traffic forwarding
- Test failover regularly with `request chassis cluster failover` — verify session continuity
- Monitor `show chassis cluster switch-events` for unexpected failover patterns
- In active/active deployments, ensure return traffic follows the same path (asymmetric routing breaks stateful inspection)
- Keep firmware versions identical on both nodes — mismatched versions cause unpredictable behavior

## See Also

- junos-nat-security, junos-security-policies, junos-high-availability, ipsec

## References

- [Juniper TechLibrary — Chassis Cluster Overview](https://www.juniper.net/documentation/us/en/software/junos/chassis-cluster-security/topics/concept/chassis-cluster-overview.html)
- [Juniper TechLibrary — Redundancy Groups](https://www.juniper.net/documentation/us/en/software/junos/chassis-cluster-security/topics/concept/chassis-cluster-redundancy-groups.html)
- [Juniper TechLibrary — Reth Interfaces](https://www.juniper.net/documentation/us/en/software/junos/chassis-cluster-security/topics/concept/chassis-cluster-reth-interfaces.html)
- [Juniper TechLibrary — Fabric Links](https://www.juniper.net/documentation/us/en/software/junos/chassis-cluster-security/topics/concept/chassis-cluster-fabric-links.html)
- [Juniper TechLibrary — Session Synchronization](https://www.juniper.net/documentation/us/en/software/junos/chassis-cluster-security/topics/concept/chassis-cluster-session-synchronization.html)
- [Juniper TechLibrary — Multi-Node HA](https://www.juniper.net/documentation/us/en/software/junos/multi-node-ha/topics/concept/multi-node-ha-overview.html)
- [Juniper Day One — SRX Chassis Cluster](https://www.juniper.net/documentation/en_US/day-one-books/DO_SRXChassisCluster.pdf)
