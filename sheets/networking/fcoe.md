# FCoE (Fibre Channel over Ethernet)

Encapsulates native Fibre Channel frames inside Ethernet, enabling I/O consolidation over a single converged data-center fabric.

## Architecture Overview

FCoE sits at Layer 2 — it is **not routable**. FC frames are wrapped in Ethernet
headers and carried over lossless DCB-capable switches. No TCP/IP stack is involved.

```
+------------------------------------------------------------+
|                    Server / Blade                           |
|  +------------------+  +------------------+                 |
|  |   FC HBA         |  |   NIC            |                 |
|  +--------+---------+  +--------+---------+                 |
|           |                      |            BEFORE        |
|     FC SAN fabric          IP/Ethernet                      |
+------------------------------------------------------------+

+------------------------------------------------------------+
|                    Server / Blade                           |
|  +----------------------------------------------+          |
|  |         CNA (Converged Network Adapter)       |          |
|  +----------------------+-----------------------+           |
|                         |                 AFTER             |
|              Lossless DCB Ethernet                          |
|              (carries both FC + IP)                         |
+------------------------------------------------------------+
```

### Motivation: I/O Consolidation

- Collapse separate FC SAN and Ethernet LAN cables into one link
- Fewer adapters, cables, switch ports, optics
- Single management plane (DCBX auto-negotiation)
- Preserve existing FC SAN infrastructure, zoning, LUN masking

## Data Center Bridging (DCB)

DCB is the set of IEEE standards that make Ethernet lossless enough for storage traffic.

### Priority Flow Control (PFC) -- IEEE 802.1Qbb

Extends 802.3x PAUSE to operate **per-priority** (per CoS / 802.1p value).

```
+-----+-----+-----+-----+-----+-----+-----+-----+
| CoS |  0  |  1  |  2  |  3  |  4  |  5  |  6  |  7  |
+-----+-----+-----+-----+-----+-----+-----+-----+
| PFC | off | off | off | ON  | off | off | off | off |
+-----+-----+-----+-----+-----+-----+-----+-----+
         ^                  ^
      best-effort       FCoE no-drop
```

- Only the FCoE CoS class (typically 3) is paused when buffers fill
- Other traffic (IP, VoIP) continues unaffected
- Prevents FC frame drops that would cause SCSI aborts

### Enhanced Transmission Selection (ETS) -- IEEE 802.1Qaz

Allocates **bandwidth guarantees** per traffic class (TC).

```
TC   Description     Min BW    Priority (CoS)
---  --------------  --------  --------------
TC0  Best-effort IP   30%       0,1,2,4,5,6,7
TC1  FCoE storage     50%       3
TC2  iSCSI (optional) 20%       4
```

- Strict priority or weighted scheduling
- Guarantees FCoE gets enough bandwidth under contention

### DCBX -- Data Center Bridging Capability Exchange

- Link-layer protocol (LLDP TLV extensions)
- Auto-negotiates PFC, ETS, and application priority between peers
- Two versions: CEE (Cisco/Intel pre-standard) and IEEE (802.1Qaz)
- **Both ends must agree** or FCoE will not initialize

### Congestion Notification (CN) -- IEEE 802.1Qau

- End-to-end congestion signaling (quantized congestion notification)
- Rarely deployed in practice; PFC handles most congestion scenarios

## FCoE Initialization Protocol (FIP)

FIP replaces FC's physical-layer login with Ethernet-layer discovery.

### FIP Stages

```
 CNA (ENode)                    FCF (FCoE Forwarder / Switch)
     |                                    |
     |--- FIP VLAN Discovery Request ---->|   (1) Find FCoE VLAN
     |<-- FIP VLAN Notification ----------|
     |                                    |
     |--- FIP FCF Discovery (Solicit) --->|   (2) Find FCF
     |<-- FIP FCF Advertisement ----------|
     |                                    |
     |--- FIP FLOGI Request ------------->|   (3) Fabric Login
     |<-- FIP FLOGI Accept/Reject --------|
     |                                    |
     |--- FIP Keep-Alive (periodic) ----->|   (4) Maintain session
     |<-- FIP Clear Virtual Link (CVL) ---|   (tear down if needed)
```

1. **VLAN Discovery** -- ENode sends to ALL-FCF-MACs multicast; FCF replies with FCoE VLAN ID
2. **FCF Discovery** -- ENode solicits on FCoE VLAN; FCF advertises FC-MAP, fabric name, priority
3. **FLOGI** -- Fabric login; FCF assigns FCID, ENode gets VN-Port MAC (FC-MAP + FCID)
4. **Keep-Alive / CVL** -- Periodic health checks; CVL forcibly logs out stale sessions

## Port Types

```
+-------------+-------------------------------------------+
| Port Type   | Description                               |
+-------------+-------------------------------------------+
| VN-Port     | Virtual N-Port on CNA (end-device)        |
| VF-Port     | Virtual F-Port on FCF (switch-side)       |
| VE-Port     | Virtual E-Port for inter-switch links     |
|             | (multi-hop FCoE, ISL equivalent)           |
+-------------+-------------------------------------------+
```

- VN-Port <-> VF-Port: host-to-switch (analogous to N-Port <-> F-Port in FC)
- VE-Port <-> VE-Port: switch-to-switch (analogous to E-Port ISL in FC)

## CNA (Converged Network Adapter)

### How It Works

```
+------------------------------------------------------+
|                     CNA Hardware                      |
|  +-------------------+  +-------------------------+  |
|  |  FC Offload Engine |  |  Ethernet NIC Engine    |  |
|  |  (FCoE, SCSI)      |  |  (TCP/IP, RSS, RDMA)   |  |
|  +--------+----------+  +--------+----------------+  |
|           |                      |                    |
|     virtual HBA            virtual NIC                |
|     (seen by OS as          (seen by OS as            |
|      FC HBA: /dev/sdX)      eth: ens1f0)             |
+------------------------------------------------------+
```

- Single physical port presents as **two logical devices** to the OS
- FC stack sees a standard HBA; IP stack sees a standard NIC
- Hardware handles FCoE encapsulation/decapsulation at line rate
- Vendors: Emulex (Broadcom), QLogic (Marvell), Intel

## FCoE Frame Format

```
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|         Dest MAC (6 bytes)          |     Src MAC (6 bytes)   |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
| 802.1Q Tag (opt) | VLAN |PRI| EtherType = 0x8906 (FCoE)     |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
| Ver | Reserved          | SOF (Start of Frame)               |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                                                               |
|              FC Frame (up to 2112 bytes payload)              |
|              [ R_CTL | D_ID | S_ID | TYPE | ... ]             |
|                                                               |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
| EOF (End of Frame)  | FCS (Ethernet CRC)                     |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+

EtherType: 0x8906 (FCoE)
EtherType: 0x8914 (FIP)
Max frame size: 2180 bytes (2112 FC payload + headers)
  -> Requires jumbo frames (MTU >= 2500, typically 9216)
```

### Key Fields

| Field         | Size     | Purpose                                |
|---------------|----------|----------------------------------------|
| Dest MAC      | 6 bytes  | VN-Port MAC or FCF MAC                 |
| EtherType     | 2 bytes  | 0x8906 (FCoE) or 0x8914 (FIP)         |
| Ver           | 4 bits   | FCoE version (currently 0)             |
| SOF           | 1 byte   | FC Start-of-Frame delimiter            |
| FC Frame      | variable | Native FC frame (R_CTL, D_ID, S_ID...) |
| EOF           | 1 byte   | FC End-of-Frame delimiter              |

## Topologies

### Single-Hop FCoE (Most Common)

```
+--------+          +-----------+          +---------+
|  CNA   |---FCoE---|  Nexus    |----FC----|  FC     |
| Server |          |  5000/    |          |  SAN    |
|        |          |  5500     |          | Switch  |
+--------+          +-----------+          +---------+
                    (FCF)                  (native FC)
```

- CNA connects via FCoE to a Nexus switch acting as FCF
- Nexus has native FC uplinks to the existing SAN fabric
- **Most deployed model** -- simple, well-supported

### Multi-Hop FCoE

```
+--------+        +----------+  VE-Port  +----------+        +---------+
|  CNA   |--FCoE--| Nexus    |---FCoE----| Nexus    |--FC----| Storage |
| Server |        | 5K (FCF) |  (ISL)    | 5K (FCF) |        | Array   |
+--------+        +----------+           +----------+        +---------+
```

- FCoE traffic traverses multiple Layer 2 hops via VE-Port links
- Each hop must be lossless (PFC end-to-end)
- Adds complexity; **less commonly deployed** than single-hop
- Requires careful VSAN/VLAN planning across all hops

### FEX (Fabric Extender) Topology

```
+--------+        +----------+           +----------+
|  CNA   |--FCoE--| Nexus    |---FCoE----| Nexus    |
| Server |        | 2000     |  (FEX)    | 5K/7K    |
+--------+        | (FEX)    |           | (FCF)    |
                  +----------+           +----------+
```

- FEX is a remote line card, not a full switch
- All FCoE processing happens on the parent Nexus (FCF)
- Simplifies top-of-rack wiring

### vPC with FCoE

```
+--------+       +----------+        +---------+
|  CNA   |--+----|  Nexus A |---FC---| SAN     |
| Server |  |    |  (FCF)   |        | Fabric  |
|        |  |    +----------+        |   A     |
|        |  |                        +---------+
|        |  |    +----------+        +---------+
|        |  +----|  Nexus B |---FC---| SAN     |
+--------+       |  (FCF)   |        | Fabric  |
                 +----------+        |   B     |
                                     +---------+
```

- CNA has two FCoE paths (active/active with multipath)
- Each Nexus is an independent FCF
- FCoE does **NOT** traverse the vPC peer-link
- Each path maps to a separate VSAN/SAN fabric (A/B)

## Nexus Configuration Examples

### Enable FCoE Features

```
! Enable required features
feature fcoe
feature npiv
feature lldp

! Set MTU for FCoE (system-wide or per-interface)
policy-map type network-qos jumbo
  class type network-qos class-fcoe
    pause no-drop
    mtu 9216
  class type network-qos class-default
    mtu 9216
system qos
  service-policy type network-qos jumbo
```

### VSAN and VLAN Binding

```
! Create VSAN
vsan database
  vsan 100

! Create dedicated FCoE VLAN (must NOT be VLAN 1 or native VLAN)
vlan 100
  fcoe vsan 100
  name FCoE_VLAN_100

! Verify binding
show vlan fcoe
```

### VFC (Virtual Fibre Channel) Interface

```
! Create VFC interface bound to physical Ethernet port
interface vfc 1
  bind interface ethernet 1/1
  switchport trunk allowed vsan 100
  no shutdown

! Or bind to a port-channel (for vPC)
interface vfc 10
  bind interface port-channel 10
  switchport trunk allowed vsan 100
  no shutdown

! Verify
show interface vfc 1
show fcoe
```

### Physical Interface for FCoE

```
interface ethernet 1/1
  switchport mode trunk
  switchport trunk allowed vlan 100
  spanning-tree port type edge trunk
  no shutdown

! PFC must be enabled on the interface
interface ethernet 1/1
  priority-flow-control mode on
```

### DCBX and QoS

```
! Verify DCBX negotiation
show lldp neighbors detail
show dcbx interface ethernet 1/1

! Check PFC status
show interface ethernet 1/1 priority-flow-control
show queuing interface ethernet 1/1

! Verify FCoE is operational
show fcoe
show vfc
show flogi database
show fcns database vsan 100
```

## Lossless Ethernet Requirements

| Requirement       | Setting                      | Why                                          |
|-------------------|------------------------------|----------------------------------------------|
| Jumbo Frames      | MTU >= 2500 (rec. 9216)      | FC frames up to 2180 bytes + overhead        |
| PFC               | Enabled on FCoE CoS (3)     | Prevents frame drops during congestion       |
| ETS               | Bandwidth allocated to FCoE  | Guarantees minimum throughput for storage     |
| DCBX              | Enabled, matching versions   | Auto-negotiates PFC/ETS between endpoints    |
| No-Drop Queue     | Map FCoE CoS to no-drop Q   | Switch buffers must absorb bursts             |
| Spanning Tree     | Edge port (portfast trunk)   | Avoid STP transitions disrupting FCoE        |
| VLAN Dedicated    | Separate VLAN for FCoE       | Isolate storage from general traffic          |

## FCoE vs iSCSI Comparison

```
+-------------------+---------------------------+----------------------------+
| Feature           | FCoE                      | iSCSI                      |
+-------------------+---------------------------+----------------------------+
| Layer             | Layer 2 (Ethernet)         | Layer 3 (TCP/IP)           |
| Routing           | Not routable               | Fully routable             |
| Protocol          | FC frames over Ethernet    | SCSI over TCP              |
| Lossless fabric   | Required (DCB/PFC)         | Not required (TCP retrans) |
| MTU               | Jumbo required (>=2500)    | Standard 1500 works        |
| Latency           | Lower (~50-200 us)         | Higher (~200-1000 us)      |
| CPU overhead      | Low (CNA offload)          | Medium (TOE helps)         |
| Multi-hop         | Limited (L2 only)          | Unlimited (L3 routed)      |
| Switch cost       | High (DCB-capable)         | Low (any Ethernet switch)  |
| Distance          | Data center only (L2)      | WAN/campus (L3/IP)         |
| FC SAN compat     | Yes (native FC mapping)    | No (separate protocol)     |
| Zoning            | FC zoning (VSAN/WWN)       | iSCSI targets/IQN/CHAP     |
| Maturity          | Declining adoption         | Growing adoption           |
| Typical use case  | Existing FC SAN migration  | New deployments, SMB       |
+-------------------+---------------------------+----------------------------+
```

## Troubleshooting Quick Reference

| Symptom                          | Check                                        |
|----------------------------------|----------------------------------------------|
| FIP VLAN discovery fails         | VLAN configured, DCBX agreed, PFC on         |
| FLOGI not completing             | VFC bound, VSAN-VLAN mapping, FCF reachable  |
| CRC errors on FCoE frames        | MTU mismatch (need jumbo end-to-end)         |
| Intermittent SCSI timeouts       | PFC not enabled, drops on no-drop queue      |
| VFC interface stays down         | Physical port down, binding mismatch          |
| DCBX not converging              | Version mismatch (CEE vs IEEE)               |
| FCoE works single-hop, not multi | VE-Port config, VSAN allowed on trunk         |

## Tips

- Always use a **dedicated VLAN** for FCoE -- never VLAN 1 or the native VLAN
- Enable PFC **before** bringing up VFC interfaces to avoid transient drops
- Match DCBX versions (CEE vs IEEE) on both ends or negotiation will fail
- FCoE is Layer 2 only -- design your topology to avoid needing to route it
- Use `show flogi database` and `show fcns database` to verify end-to-end FC login
- Monitor `show interface ethernet X/Y counters` for PFC pause frame counts
- Single-hop FCoE with native FC uplinks is the simplest and most reliable design
- Test failover by shutting individual VFC interfaces and verifying multipath switchover
- Keep FCoE and LAN traffic on separate queuing classes to prevent head-of-line blocking
- In vPC scenarios, each Nexus must be an independent FCF -- FCoE never crosses the peer-link

## See Also

- fibre-channel -- native FC SAN concepts, zoning, VSAN
- iscsi -- iSCSI protocol, CHAP, multipath
- dcb -- Data Center Bridging deep dive (PFC, ETS, DCBX)
- nexus -- Cisco Nexus platform configuration
- qos -- Quality of Service, queuing, scheduling
- vpc -- Virtual Port Channel design and configuration

## References

- [T11 FC-BB-5 (FCoE standard)](https://www.t11.org/ftp/t11/pub/fc/bb-5/)
- [IEEE 802.1Qbb (PFC)](https://standards.ieee.org/standard/802_1Qbb-2011.html)
- [IEEE 802.1Qaz (ETS/DCBX)](https://standards.ieee.org/standard/802_1Qaz-2011.html)
- [Cisco FCoE Configuration Guide (Nexus 5000)](https://www.cisco.com/c/en/us/td/docs/switches/datacenter/nexus5000/sw/fcoe/b_Cisco_Nexus_5000_Series_NX-OS_FCoE_Configuration_Guide.html)
- [Cisco FCoE Design Guide](https://www.cisco.com/c/en/us/products/collateral/switches/nexus-5000-series-switches/white_paper_c11-495061.html)
- [Data Center Bridging (DCB) overview -- IEEE](https://1.ieee802.org/dcb/)
