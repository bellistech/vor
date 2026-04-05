# Cisco Nexus -- NX-OS Architecture and Data Center Switching

> NX-OS is Cisco's purpose-built data center network operating system, running on a Linux kernel with a modular, process-oriented architecture that delivers non-stop forwarding, in-service software upgrades, and programmable infrastructure across the Nexus switching family. From spine-leaf fabrics to multi-site DCI, the Nexus platform provides the switching foundation for modern data center networks.

## 1. NX-OS Architecture

### 1.1 Kernel and System Design

NX-OS is built on a Wind River Linux kernel. Unlike monolithic IOS, NX-OS runs every protocol and service as an independent, protected user-space process. This design provides three critical properties:

1. **Fault isolation** -- a crash in BGP does not affect OSPF, STP, or forwarding
2. **Independent restartability** -- any process can be restarted without a full reload
3. **Memory protection** -- each process has its own address space

```
                         NX-OS Architecture
+================================================================+
|                        User Space                              |
|  +--------+ +--------+ +------+ +------+ +------+ +--------+  |
|  |  BGP   | |  OSPF  | | STP  | |  vPC | | LACP | | LLDP   |  |
|  |process | |process | |proc  | |proc  | |proc  | |process |  |
|  +---+----+ +---+----+ +--+---+ +--+---+ +--+---+ +---+----+  |
|      |          |          |        |        |         |       |
|  +---+----------+----------+--------+--------+---------+---+  |
|  |              MTS  (Message & Transaction Service)        |  |
|  |              Inter-Process Communication Bus             |  |
|  +---+----------+----------+--------+--------+---------+---+  |
|      |          |          |        |        |         |       |
|  +---+----+ +---+----+ +--+---+ +--+---+                      |
|  |  URIB  | |  UFDM  | |pixm  | |ethpm |   ... managers      |
|  |(routes)| |(fwd mgr)| |(port)| |(intf)|                      |
|  +--------+ +--------+ +------+ +------+                      |
|                                                                |
|  +----------------------------------------------------------+  |
|  |               sysmgr (System Manager)                    |  |
|  |  - Process lifecycle (start, stop, restart, monitor)     |  |
|  |  - Heartbeat monitoring                                  |  |
|  |  - Dependency graph management                           |  |
|  +----------------------------------------------------------+  |
+================================================================+
|                     Linux Kernel (Wind River)                  |
|  - Scheduler, memory management, device drivers               |
|  - Netlink interface to user-space                             |
+================================================================+
|                Hardware Abstraction Layer (HAL)                |
|  - Uniform API across different ASIC families                  |
|  - Programs forwarding tables, ACLs, QoS into hardware        |
+================================================================+
|                   Forwarding ASICs                             |
|  - Memory (TCAM, tables)                                       |
|  - Memory Architecture: Memory for forwarding entries          |
|  Memory used for: routes, MAC, ACL, QoS, adjacency            |
+================================================================+
```

### 1.2 Message and Transaction Service (MTS)

MTS is the IPC backbone of NX-OS. All inter-process communication flows through MTS using a publish-subscribe model. This provides:

- **Reliable delivery** -- messages are queued and delivered even if a destination process restarts
- **Transaction semantics** -- configuration changes are atomic across multiple processes
- **Abstraction** -- processes communicate via service access points (SAPs) without knowing process IDs

When you configure a VLAN, the CLI process sends an MTS message to the VLAN manager, which programs the hardware via HAL and notifies STP, ethpm, and other consumers via MTS.

### 1.3 System Manager (sysmgr)

The system manager is the init process of NX-OS. It maintains a directed acyclic graph of process dependencies and manages the lifecycle of every service:

- **Startup ordering** -- ensures processes start in correct dependency order
- **Health monitoring** -- each process sends periodic heartbeats; missed heartbeats trigger restart
- **Restart policies** -- configurable per service: restart process, restart service group, or reload
- **Service states** -- SRV_STATE_HANDSHAKED (fully operational), SRV_STATE_STARTED, etc.

```
show system internal sysmgr service name bgp
   Service "bgp" ("bgp", 1):
       UUID = 0x22E, PID = 8412, SAP = 320
       State: SRV_STATE_HANDSHAKED (entered at ...)
       Restart count: 2
       Internal restart count: 0
       Last restart reason: User initiated restart
```

### 1.4 Process Restart and Non-Stop Forwarding

NX-OS supports three levels of high availability:

| Level                       | Impact                           | Mechanism                        |
|-----------------------------|----------------------------------|----------------------------------|
| Process restart             | No traffic loss                  | Individual process restart       |
| Stateful switchover (SSO)   | Sub-second loss on dual-sup      | Standby supervisor takes over    |
| In-Service Software Upgrade | Minimal loss during upgrade      | ISSU with dual supervisors       |

When a process restarts, the hardware forwarding tables remain programmed. The restarted process recovers its state from persistent storage or re-learns it from neighbors (graceful restart).

```
# Force a process restart (no traffic impact)
restart bgp 65001

# ISSU pre-check
show install all impact nxos bootflash:nxos.9.3.10.bin

# Perform ISSU
install all nxos bootflash:nxos.9.3.10.bin
```

## 2. Virtual Device Contexts (VDC)

### 2.1 Concept and Use Cases

A VDC partitions a single physical Nexus 7000 into multiple independent logical switches. Each VDC has its own:

- Configuration (running-config and startup-config)
- Management plane (CLI sessions, SNMP, syslog)
- Control plane (routing protocols, STP instances)
- Fault domain (a crash in VDC 2 does not affect VDC 1)
- Administrative domain (separate admin credentials)

VDCs do NOT share:

- Physical interfaces (each interface is allocated to exactly one VDC)
- Routing tables
- MAC address tables

```
                    Physical Nexus 7000 Chassis
+===============================================================+
|  +------------------+  +------------------+  +---------------+ |
|  |     VDC 1        |  |     VDC 2        |  |    VDC 3      | |
|  | (default/admin)  |  | (production)     |  | (dmz)         | |
|  |                  |  |                  |  |               | |
|  | Admin access     |  | OSPF, BGP        |  | Static routes | |
|  | VDC management   |  | HSRP, vPC        |  | ACLs          | |
|  | Sup interfaces   |  | e2/1-48, e3/1-48 |  | e4/1-12       | |
|  | mgmt0, e1/1-4    |  |                  |  |               | |
|  +------------------+  +------------------+  +---------------+ |
|                                                                 |
|  Shared: Supervisors, power supplies, fans, fabric modules      |
+===============================================================+
```

### 2.2 VDC Types

| VDC Type    | Description                                           |
|-------------|-------------------------------------------------------|
| Admin VDC   | VDC 1 (default). Manages other VDCs, allocates resources |
| Non-admin   | User-created VDCs with allocated interfaces            |
| Storage VDC | Dedicated for FCoE (Fibre Channel over Ethernet)       |

### 2.3 Creating and Configuring VDCs

VDC creation and interface allocation must be performed from the admin VDC (VDC 1):

```
! --- From VDC 1 (admin) ---

! Create a new VDC
vdc production
  limit-resource vlan minimum 16 maximum 4094
  limit-resource vrf minimum 2 maximum 4096
  limit-resource u4route-mem minimum 8 maximum 8
  limit-resource u6route-mem minimum 4 maximum 4
  limit-resource port-channel minimum 0 maximum 768
  limit-resource m4route-mem minimum 1 maximum 2
  allocate interface ethernet 2/1-48
  allocate interface ethernet 3/1-48

! Switch into the new VDC
switchto vdc production

! --- Now operating inside VDC "production" ---
configure terminal
hostname PROD-VDC
feature ospf
feature bgp
feature vpc
interface ethernet 2/1
  no shutdown
  ip address 10.0.0.1/30
end

! Return to admin VDC
switchback
```

### 2.4 Resource Management

Resources are finite and must be budgeted across VDCs. The admin VDC controls allocation:

```
! View current resource allocation
show vdc resource

! View resource usage within a VDC
show vdc production resource

! Apply a resource template
vdc production
  template default-f2e-template
```

The total resources allocated across all VDCs cannot exceed the physical hardware limits. Over-provisioning is not supported.

### 2.5 VDC Platform Limitation

VDCs are only supported on the Nexus 7000 platform. The Nexus 9000, 5000, and 3000 do not support VDCs. For multi-tenancy on these platforms, use VRFs and VXLAN EVPN instead.

## 3. vPC (Virtual Port Channel)

### 3.1 Problem Statement

Traditional spanning tree blocks redundant links, wasting 50% of available bandwidth. Multichassis EtherChannel (MLAG) solutions like vPC allow a downstream device to form a single port-channel across two upstream switches, utilizing all links while maintaining a loop-free topology.

### 3.2 vPC Components

```
                       vPC Domain 100
        +==============+  Peer-Keepalive  +==============+
        |   Switch A   |  (mgmt VRF,      |   Switch B   |
        |   Role:      |   UDP/3200)      |   Role:      |
        |   PRIMARY    |<================>|   SECONDARY  |
        |  Priority:   |                  |  Priority:   |
        |    1000      |                  |    2000      |
        +==+====+======+                  +======+====+==+
           |    |    Peer-Link (Po1)           |    |
           |    +==============================+    |
           |    |  (Trunk carrying all VLANs)  |    |
           |    +==============================+    |
           |                                        |
      Po10 |  vPC 10                         vPC 10 | Po10
           |                                        |
        +--+----------------------------------------+--+
        |              Downstream Switch               |
        |              (or Server with LACP)           |
        +----------------------------------------------+

    CFS (Cisco Fabric Services) syncs MAC/ARP/IGMP tables
    over the peer-link between vPC peers.
```

**Component breakdown:**

| Component           | Purpose                                                    |
|---------------------|------------------------------------------------------------|
| vPC Domain          | Logical grouping; domain ID must match on both peers       |
| Peer-Link           | Port-channel carrying control + data between peers         |
| Peer-Keepalive      | Heartbeat link to detect peer failure (split-brain prevention) |
| vPC Member Port     | Port-channel toward downstream with `vpc <id>` assigned    |
| CFS                 | Cisco Fabric Services -- syncs state tables over peer-link |
| Orphan Port         | Single-attached port on a vPC switch in a vPC VLAN         |

### 3.3 vPC Election and Role

The primary/secondary role is elected based on:

1. Lower role priority wins (default 32667)
2. If tied, lower system MAC wins
3. Role is sticky -- a recovered peer returns as secondary unless `role preempt` is configured

### 3.4 vPC Consistency Checks

vPC performs two categories of consistency checks to ensure both peers have compatible configurations:

**Type-1 (Mandatory -- will suspend vPC):**
- STP mode (RPVST+ vs MST)
- STP VLAN-to-instance mapping
- VLAN membership on vPC member ports
- MTU per interface
- STP global settings (Bridge Assurance, Loop Guard)
- Allowed VLAN list on peer-link

**Type-2 (Advisory -- generates syslog warning):**
- STP port type
- BPDU Filter/Guard settings
- Storm control settings
- DHCP relay configuration

```
! Verify consistency -- RUN THIS AFTER EVERY CHANGE
show vpc consistency-parameters global

! Per-interface consistency
show vpc consistency-parameters interface port-channel 10

! Expected output should show "SUCCESS" for all parameters
```

### 3.5 Failure Scenarios

```
Scenario 1: vPC member link failure (one side)
+--------+          +--------+
|  Sw-A  |----X     |  Sw-B  |
+---+----+          +----+---+
    |  peer-link ok      |
    +===================+
Traffic shifts to surviving member link via LACP.

Scenario 2: Peer-link failure
+--------+  PL DOWN  +--------+
|  Sw-A  |----X------| Sw-B  |
+---+----+            +---+---+
If keepalive is up: secondary suspends all vPC ports.
Primary continues forwarding.

Scenario 3: Peer-keepalive failure only
+--------+  KA DOWN  +--------+
|  Sw-A  |-----X-----| Sw-B  |
+---+----+            +---+---+
    |    peer-link ok      |
    +======================+
No immediate impact. Syslog warning generated.
Both switches continue forwarding.

Scenario 4: Complete peer isolation (both PL and KA down)
+--------+  BOTH DOWN +--------+
|  Sw-A  |----X--X----| Sw-B  |
+---+----+             +---+---+
DUAL-ACTIVE / SPLIT-BRAIN
Both peers think the other is dead. Both become primary.
auto-recovery timer determines behavior after stabilization.
```

### 3.6 vPC Best Practices

1. **Peer-keepalive**: Use the management VRF or a dedicated routed link. Never use the peer-link.
2. **Peer-link sizing**: Minimum two 10G links in a port-channel. Carry all VLANs.
3. **peer-gateway**: Always enable. Allows each peer to route packets destined to the other peer's HSRP MAC.
4. **ip arp synchronize**: Sync ARP tables across peers to avoid drops during failover.
5. **delay restore**: Set 30-60 seconds to allow routing protocols to converge before enabling vPC ports after reload.
6. **Orphan ports**: Use `vpc orphan-port suspend` on single-attached ports in vPC VLANs.
7. **Layer 3 over vPC**: Use SVIs with `peer-gateway` and `ip arp synchronize`.

## 4. FabricPath

### 4.1 The Problem with Large Layer 2 Domains

Traditional Layer 2 networks rely on Spanning Tree Protocol to prevent loops. STP has fundamental limitations:

- Blocks redundant paths, wasting bandwidth
- Slow convergence (seconds with RPVST+, longer with 802.1D)
- Flood-and-learn MAC address behavior does not scale
- Difficult to troubleshoot in large deployments

FabricPath solves these problems by introducing a Layer 2 routing protocol (IS-IS) to build a loop-free, multipath Layer 2 fabric.

### 4.2 FabricPath Operation

FabricPath works by encapsulating Ethernet frames with a FabricPath header that includes source and destination switch IDs. The fabric uses IS-IS to compute equal-cost multipaths between switches.

```
Classical Ethernet Frame:
+--------+--------+------+---------+-----+
| Dst MAC| Src MAC| Type | Payload | FCS |
+--------+--------+------+---------+-----+

FabricPath-Encapsulated Frame:
+----------+----------+-------+--------+--------+------+---------+-----+
| FP Outer | FP Outer | FTag  | Dst MAC| Src MAC| Type | Payload | FCS |
| Dst SID  | Src SID  | (ECMP)|        |        |      |         |     |
+----------+----------+-------+--------+--------+------+---------+-----+
            ^-- IS-IS routed --^        ^-- Original frame preserved --^
```

### 4.3 Conversational MAC Learning

Unlike traditional flood-and-learn, FabricPath uses conversational MAC learning:

- A switch only learns remote MAC addresses when there is active conversation (traffic in both directions)
- This dramatically reduces MAC table size on intermediate spine switches
- Spine switches may never learn end-host MACs if they only transit traffic

### 4.4 FabricPath IS-IS Control Plane

FabricPath uses a modified IS-IS to:

- Distribute switch-ID reachability (like router IDs)
- Compute shortest-path trees for unicast
- Compute multi-destination trees for broadcast/multicast/unknown unicast
- Support multiple topologies for traffic engineering

```
IS-IS Adjacency Formation:
  Leaf-1 (SID 1) ---- IS-IS Hello ----> Spine-1 (SID 100)
  Leaf-1 (SID 1) <--- IS-IS Hello ----- Spine-1 (SID 100)
  Adjacency: UP

  LSP flooding: Each switch floods its switch-ID and links
  SPF computation: Each switch computes shortest paths to all other switches
```

### 4.5 FabricPath Topologies and FTags

FabricPath supports multiple forwarding topologies identified by FTags (Forwarding Tags):

| FTag | Purpose                                    |
|------|--------------------------------------------|
| 1    | Default topology for all VLANs             |
| 2+   | Additional topologies for traffic engineering|

BUM (Broadcast, Unknown unicast, Multicast) traffic uses multi-destination trees rooted at configurable root switches.

### 4.6 FabricPath vs VXLAN EVPN

FabricPath has largely been superseded by VXLAN EVPN for new deployments:

| Aspect             | FabricPath          | VXLAN EVPN            |
|--------------------|---------------------|-----------------------|
| Encapsulation      | FabricPath header   | VXLAN (UDP)           |
| Control plane      | IS-IS               | BGP EVPN              |
| Transport          | Layer 2 only        | Layer 3 (IP routed)   |
| Scale              | Single fabric       | Multi-site, multi-DC  |
| Platforms          | 7000, 5000          | 9000, 7000            |
| Status             | Maintenance mode    | Active development    |

## 5. OTV (Overlay Transport Virtualization)

### 5.1 Data Center Interconnect Challenge

Extending Layer 2 domains between data center sites is necessary for VM mobility, cluster heartbeats, and disaster recovery. Traditional approaches (dark fiber, VPLS, EoMPLS) have drawbacks:

- STP must span both sites (fragile)
- BUM traffic floods across the WAN
- Failure domains extend across sites
- No optimal traffic engineering

OTV solves these by creating a MAC-in-IP overlay that extends Layer 2 only where needed, without extending STP.

### 5.2 OTV Operation

```
          Site A                     IP Network                    Site B
+-------------------+        +-----------------+        +-------------------+
|                   |        |                 |        |                   |
| VLAN 100: Hosts   |        |  Routed L3 Core |        | VLAN 100: Hosts   |
|                   |        |  (MPLS, Internet,|       |                   |
| +---------------+ |        |   dark fiber)   |        | +---------------+ |
| | OTV Edge Dev  | |        |                 |        | | OTV Edge Dev  | |
| | Join: Lo0     +----------+                 +----------+ Join: Lo0     | |
| | 10.1.1.1      | |  OTV   |                 |  OTV   | | 10.2.2.1      | |
| +-------+-------+ |  Encap |                 |  Encap | +-------+-------+ |
|         |         |        |                 |        |         |         |
| Site VLAN 999     |        |                 |        | Site VLAN 998     |
| (never extended)  |        |                 |        | (never extended)  |
+-------------------+        +-----------------+        +-------------------+

OTV frame format:
+------+------+------+------+--------+--------+------+---------+
| Outer| Outer| Outer| OTV  | Inner  | Inner  | Type | Payload |
| L2   | IP   | UDP  | Shim | Dst MAC| Src MAC|      |         |
| Hdr  | Hdr  | Hdr  | Hdr  |        |        |      |         |
+------+------+------+------+--------+--------+------+---------+
         ^-- Routed across      ^-- Original Ethernet frame --^
             transport
```

### 5.3 OTV Key Mechanisms

**Authoritative Edge Device (AED) Election:**
When multiple OTV edge devices exist at a site (multi-homing), they elect an AED per VLAN. Only the AED for a given VLAN forwards BUM traffic into/out of the overlay for that VLAN. This prevents duplicate frames.

**Site VLAN:**
The site VLAN is a locally significant VLAN (never extended across OTV) used for AED election. All OTV edge devices at a site must share the same site VLAN. This is how OTV detects multi-homing -- if two edge devices see each other on the site VLAN, they know they are at the same site.

**MAC Routing:**
OTV advertises MAC reachability using an IS-IS control plane over the overlay. When Host A at Site A sends traffic to Host B at Site B, the OTV edge device at Site A looks up Host B's MAC in the OTV routing table, encapsulates the frame in IP, and sends it to Site B's OTV edge device.

### 5.4 OTV vs VXLAN for DCI

| Aspect              | OTV                        | VXLAN EVPN Multi-Site     |
|---------------------|----------------------------|---------------------------|
| Encapsulation       | MAC-in-IP (OTV shim)       | MAC-in-UDP (VXLAN)        |
| Control plane       | IS-IS                      | BGP EVPN                  |
| STP extension       | Blocked at overlay         | Blocked at overlay        |
| ARP optimization    | Limited                    | ARP suppression           |
| Multi-site scale    | Good                       | Better (BGP scalability)  |
| Platform support    | 7000 only                  | 9000, 7000                |
| Status              | Maintenance mode           | Active development        |

## 6. Fabric Extender (FEX) -- Nexus 2000

### 6.1 Architecture Concept

The Nexus 2000 FEX is not an independent switch. It is a remote line card managed entirely by a parent switch (Nexus 5000, 7000, or 9000). The FEX has no local control plane, no CLI, and no independent management. All configuration, forwarding decisions, and policy enforcement happen on the parent switch.

```
        Parent Switch (Nexus 5000/7000/9000)
        +-----------------------------------------+
        |  Control Plane     |    Data Plane       |
        |  (runs all         |    (programs FEX    |
        |   protocols)       |     forwarding)     |
        +--------+-----------+----------+----------+
                 |  Fabric Interfaces   |
                 |  (10G/40G/100G)      |
                 |  FEX Control Traffic |
                 |  + User Data Traffic |
                 |                      |
        +--------+----------------------+----------+
        |           Nexus 2000 FEX                 |
        |  +----+ +----+ +----+     +----+ +----+  |
        |  |1/1 | |1/2 | |1/3 | ... |1/47| |1/48|  |
        |  +----+ +----+ +----+     +----+ +----+  |
        |     Host Interfaces (1G/10G servers)      |
        +-------------------------------------------+

    Interface naming: ethernet <FEX-ID>/<slot>/<port>
    Example: ethernet 101/1/1 = FEX 101, slot 1, port 1
```

### 6.2 FEX Connectivity Models

**Straight-through (static pinning):**
Each host interface is statically pinned to a single fabric interface. Simple but no redundancy per host port.

**Active-Active (port-channel to parent):**
Fabric interfaces are bundled into a port-channel. Host traffic is distributed across fabric links. If one fabric link fails, traffic redistributes. This is the recommended model.

**Dual-homed FEX (vPC):**
The FEX connects to two parent switches via a vPC. Provides both FEX and parent-switch redundancy.

```
Dual-Homed FEX with vPC:

  +----------+     vPC     +----------+
  | Parent A |<===========>| Parent B |
  +----+-----+  peer-link  +-----+----+
       |                         |
       | Fabric Po               | Fabric Po
       |                         |
  +----+-------------------------+----+
  |          Nexus 2000 FEX          |
  |     (connected to both parents)  |
  +--+--+--+--+--+--+--+--+--+--+--++
     |  |  |  |  |  |  |  |  |  |  |
   Host Interfaces (servers)
```

### 6.3 FEX Pinning

Pinning determines how host interfaces map to fabric (uplink) interfaces:

| Pinning Mode    | Behavior                                          |
|-----------------|---------------------------------------------------|
| max-links 1     | Each host port pinned to exactly one fabric link   |
| max-links N     | Host traffic distributed across N fabric links     |
| (default)       | All available fabric links used for distribution   |

When a fabric link fails, host ports pinned to that link are re-pinned to surviving links. During re-pinning, there is a brief traffic interruption.

## 7. NX-OS CLI Differences from IOS

### 7.1 Feature Enablement Model

The most fundamental difference between NX-OS and IOS is the feature enablement model. In IOS, all features are available by default. In NX-OS, features must be explicitly enabled before any related configuration is accepted.

This design reduces memory consumption (unused features are not loaded), shrinks the attack surface, and simplifies the running configuration by eliminating default denials.

```
! IOS -- just start configuring
router ospf 1
  network 0.0.0.0 255.255.255.255 area 0

! NX-OS -- enable first, then configure
feature ospf
router ospf 1
  router-id 10.0.0.1
```

### 7.2 Configuration Modes and Syntax

```
+---------------------------+-----------------------------+-----------------------------+
| Operation                 | IOS                         | NX-OS                       |
+---------------------------+-----------------------------+-----------------------------+
| Enter config mode         | configure terminal          | configure terminal          |
| Exit to exec              | end                         | end                         |
| Save config               | copy run start              | copy run start              |
| Interface range            | interface range Gi0/1 - 4   | interface e1/1-4            |
| Show section               | show run | section router  | show run | section router   |
| Filter include             | show run | include vlan    | show run | include vlan     |
| No paging                  | terminal length 0           | terminal length 0           |
| VRF routing                | show ip route vrf X         | show ip route vrf X         |
| Default interface state    | no shutdown (varies)        | shutdown (admin down)       |
| Checkpoint/Rollback        | archive (limited)           | checkpoint + rollback       |
| JSON output                | N/A                         | show ... | json             |
| XML output                 | N/A (some via NETCONF)      | show ... | xml              |
+---------------------------+-----------------------------+-----------------------------+
```

### 7.3 Checkpoint and Rollback

NX-OS has a built-in rollback mechanism far more capable than IOS archive:

```
! Create named checkpoint
checkpoint BEFORE-CHANGE description "Pre-maintenance window"

! List all checkpoints
show checkpoint summary

! View diff between running and checkpoint
show diff rollback-patch checkpoint BEFORE-CHANGE

! Preview what rollback would do (dry run)
show diff rollback-patch checkpoint BEFORE-CHANGE

! Execute rollback
rollback running-config checkpoint BEFORE-CHANGE

! Atomic rollback with automatic revert
rollback running-config checkpoint BEFORE-CHANGE atomic timeout 300
! If not confirmed within 300 seconds, auto-reverts
```

## 8. Platform Deep Dive

### 8.1 Nexus 9000

The Nexus 9000 is the current flagship platform supporting two operating modes:

- **Standalone NX-OS mode**: Traditional CLI-driven switching with VXLAN EVPN
- **ACI mode**: Application Centric Infrastructure with APIC controller

Key capabilities in standalone NX-OS mode:
- VXLAN EVPN fabric (spine-leaf)
- 400G interfaces (9000-EX, FX3, GX, GX2)
- CloudSec (MACsec across VXLAN)
- Tetration/ThousandEyes integration
- NX-API, gRPC telemetry, OpenConfig

### 8.2 Nexus 7000

The Nexus 7000 is the modular chassis platform for large DC core deployments:

- Supports VDCs (unique to this platform)
- OTV for DC interconnect
- FabricPath
- MPLS (limited)
- Up to 768 10G or 192 100G ports per chassis (N7718)
- Dual supervisors with SSO/ISSU

### 8.3 Nexus 5000/6000

Unified fabric switches combining Ethernet and Fibre Channel:

- FCoE (Fibre Channel over Ethernet) native support
- FEX parent switch capability
- vPC support
- Compact 1-2 RU form factor

### 8.4 Nexus 3000

Ultra-low-latency ToR switches optimized for:

- Algorithmic/high-frequency trading (cut-through, sub-microsecond)
- Leaf switches in spine-leaf fabrics
- Compact 1 RU form factor
- Wirespeed L2/L3 forwarding

### 8.5 Nexus 2000 (FEX)

Not a standalone switch -- operates as a remote line card:

- No local management or control plane
- 1G/10G host interfaces
- Managed entirely by parent Nexus 5000/7000/9000
- Reduces per-port management overhead dramatically

## 9. NX-API Deep Dive

### 9.1 NX-API Architecture

```
+-------------------+     HTTPS/HTTP      +--------------------+
|                   |  (JSON/XML/JSON-RPC)|                    |
|  Automation Host  +-------------------->+   Nexus Switch     |
|  (Python, Ansible,|                     |                    |
|   Terraform, etc.)|<--------------------+   nginx -> nxapi   |
|                   |     JSON/XML        |   backend process  |
+-------------------+                     +--------------------+
```

### 9.2 NX-API Message Types

| Type       | Description                                    | Use Case                        |
|------------|------------------------------------------------|---------------------------------|
| cli_show   | Execute show commands, return structured output | Monitoring, inventory           |
| cli_show_array | Show command returning array results       | Multi-row outputs               |
| cli_conf   | Execute configuration commands                 | Provisioning, changes           |
| bash       | Execute bash commands on the NX-OS shell       | Advanced troubleshooting        |

### 9.3 NX-API Practical Examples

**Python automation with requests:**

```python
import requests
import json
import urllib3
urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)

class NXAPI:
    def __init__(self, host, username, password):
        self.url = f"https://{host}/ins"
        self.auth = (username, password)
        self.headers = {"Content-Type": "application/json"}

    def show(self, command):
        payload = {
            "ins_api": {
                "version": "1.0",
                "type": "cli_show",
                "chunk": "0",
                "sid": "1",
                "input": command,
                "output_format": "json"
            }
        }
        resp = requests.post(
            self.url, json=payload, headers=self.headers,
            auth=self.auth, verify=False
        )
        return resp.json()

    def configure(self, commands):
        """commands: semicolon-separated config commands"""
        payload = {
            "ins_api": {
                "version": "1.0",
                "type": "cli_conf",
                "chunk": "0",
                "sid": "1",
                "input": commands,
                "output_format": "json"
            }
        }
        resp = requests.post(
            self.url, json=payload, headers=self.headers,
            auth=self.auth, verify=False
        )
        return resp.json()

# Usage
switch = NXAPI("10.1.1.1", "admin", "password")

# Get VLAN info
vlans = switch.show("show vlan brief")
print(json.dumps(vlans, indent=2))

# Create a VLAN
result = switch.configure(
    "vlan 100 ; name PRODUCTION ; exit ; "
    "interface vlan 100 ; ip address 10.100.0.1/24 ; no shutdown"
)
print(json.dumps(result, indent=2))
```

**Ansible NX-OS modules:**

```yaml
- name: Configure Nexus switches
  hosts: nexus_switches
  gather_facts: no
  connection: httpapi
  vars:
    ansible_httpapi_use_ssl: yes
    ansible_httpapi_validate_certs: no
    ansible_network_os: cisco.nxos.nxos

  tasks:
    - name: Enable features
      cisco.nxos.nxos_feature:
        feature: "{{ item }}"
        state: enabled
      loop:
        - ospf
        - bgp
        - vpc
        - lacp
        - interface-vlan

    - name: Configure VLANs
      cisco.nxos.nxos_vlans:
        config:
          - vlan_id: 100
            name: PRODUCTION
          - vlan_id: 200
            name: MANAGEMENT

    - name: Configure interfaces
      cisco.nxos.nxos_interfaces:
        config:
          - name: Ethernet1/1
            description: UPLINK-TO-SPINE
            enabled: true
            mtu: 9216
```

### 9.4 NX-API Sandbox

The NX-API sandbox is a built-in web GUI that allows you to test API calls interactively:

```
feature nxapi
nxapi sandbox

! Access via browser: https://<switch-ip>/ins
! The sandbox provides:
!   - Command input with format selection (JSON, XML, JSON-RPC)
!   - Request/response preview
!   - Python code generation
!   - cURL command generation
```

## 10. Comprehensive Show Command Reference

### 10.1 System Health and Inventory

```bash
show version                            # NX-OS version, uptime, model, serial
show module                             # Line cards, supervisors, status
show module 1                           # Detailed info for slot 1
show inventory                          # All physical components with PID/SN
show environment                        # PSU, fans, temperature sensors
show environment power                  # Power supply status and wattage
show environment fan                    # Fan tray status and RPM
show system resources                   # CPU %, memory used/free
show processes cpu sort                 # Top CPU consumers
show processes cpu history              # CPU utilization graph over time
show processes memory                   # Per-process memory usage
show cores                              # Core dump files from crashes
show feature                            # Feature enable/disable status
show license usage                      # License consumption
show accounting log                     # Command accounting
```

### 10.2 Interface and L2 Diagnostics

```bash
show interface brief                    # Summary: state, speed, VLAN
show interface status                   # Port status with description
show interface ethernet 1/1             # Detailed counters, errors, CRC
show interface ethernet 1/1 transceiver # SFP/QSFP details, DOM readings
show interface counters errors          # Error counters per interface
show interface trunk                    # Trunk ports, native/allowed VLANs
show vlan brief                         # VLAN table
show mac address-table                  # MAC address table
show mac address-table count            # MAC count per VLAN
show port-channel summary               # LAG status and members
show lacp counters                      # LACP PDU statistics
show lacp neighbor                      # LACP partner info
show spanning-tree                      # Full STP state
show spanning-tree summary              # STP overview
show cdp neighbors detail               # CDP neighbor details
show lldp neighbors detail              # LLDP neighbor details
```

### 10.3 Routing Protocol Status

```bash
show ip route                           # IPv4 routing table
show ip route summary                   # Route count by protocol
show ip route vrf all                   # Routes across all VRFs
show ip bgp summary                     # BGP neighbor table
show ip bgp                             # BGP RIB
show ip bgp neighbors <IP> routes       # Routes from specific peer
show ip ospf neighbors                  # OSPF adjacency table
show ip ospf interface brief            # OSPF-enabled interfaces
show ip ospf database                   # OSPF LSDB
show ip eigrp neighbors                 # EIGRP neighbors
show hsrp brief                         # HSRP state per interface
show vrf                                # VRF instances
show ip arp vrf all                     # ARP table all VRFs
```

### 10.4 Hardware Forwarding Verification

```bash
show forwarding route                   # Hardware FIB entries
show forwarding adjacency               # Adjacency table in hardware
show hardware capacity                  # TCAM utilization
show hardware internal memory           # ASIC memory usage
show system internal forwarding route   # Internal forwarding state
```

## Prerequisites

- Familiarity with Ethernet switching concepts (VLANs, trunking, STP, LACP)
- Understanding of IP routing fundamentals (OSPF, BGP)
- Basic Linux command-line knowledge (helpful for NX-OS shell access)
- Understanding of data center network topologies (spine-leaf, core-aggregation-access)
- Familiarity with at least one programming language for NX-API automation (Python recommended)

## References

- [Cisco NX-OS Verified Scalability Guide](https://www.cisco.com/c/en/us/td/docs/switches/datacenter/nexus9000/sw/93x/scalability/guide-9336c-fx2/cisco-nexus-9000-series-nx-os-verified-scalability-guide-93x.html)
- [Cisco Nexus 9000 NX-OS Configuration Guides](https://www.cisco.com/c/en/us/support/switches/nexus-9000-series-switches/products-installation-and-configuration-guides-list.html)
- [NX-OS NX-API REST SDK Developer Guide](https://developer.cisco.com/docs/nx-os/)
- [Cisco vPC Design and Configuration Guide](https://www.cisco.com/c/en/us/products/collateral/switches/nexus-5000-series-switches/design_guide_c07-625857.html)
- [OTV Design Guide](https://www.cisco.com/c/en/us/products/collateral/switches/nexus-7000-series-switches/white_paper_c11-729383.html)
- [FabricPath Design and Deployment Guide](https://www.cisco.com/c/en/us/products/collateral/switches/nexus-7000-series-switches/guide_c07-690079.html)
- [Nexus 2000 FEX Architecture White Paper](https://www.cisco.com/c/en/us/products/collateral/switches/nexus-2000-series-fabric-extenders/white_paper_c11-516426.html)
- [Data Center Networking: NX-OS and NX-API (Cisco Press)](https://www.ciscopress.com/)
- [RFC 7348 - VXLAN](https://datatracker.ietf.org/doc/html/rfc7348)
- [RFC 7432 - BGP MPLS-Based Ethernet VPN](https://datatracker.ietf.org/doc/html/rfc7432)
