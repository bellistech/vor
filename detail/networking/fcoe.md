# FCoE -- Fibre Channel over Ethernet Architecture

> FCoE (Fibre Channel over Ethernet) encapsulates native Fibre Channel frames within
> IEEE 802.3 Ethernet, enabling storage area network (SAN) and local area network (LAN)
> traffic to share a single physical infrastructure. It operates at Layer 2, requires
> a lossless Ethernet fabric built on Data Center Bridging (DCB) standards, and preserves
> full compatibility with existing FC SAN constructs including zoning, VSANs, and WWN-based
> access control. FCoE is defined in the T11 FC-BB-5 and FC-BB-6 specifications.

## 1. Historical Context and Motivation

### 1.1 The Cabling Problem

Traditional data centers maintained two entirely separate physical networks:

```
Server Rack (circa 2008)
+---------------------------------------------------+
|  Server 1                                          |
|    NIC port 1 -----> Ethernet Switch (LAN)         |
|    NIC port 2 -----> Ethernet Switch (LAN)         |
|    HBA port 1 -----> FC Switch (SAN Fabric A)      |
|    HBA port 2 -----> FC Switch (SAN Fabric B)      |
|                                                    |
|  Per server: 4 cables, 4 adapters, 4 switch ports  |
|  Per rack (40 servers): 160 cables                  |
+---------------------------------------------------+
```

This meant:
- **Double the cabling** -- separate copper/fiber for LAN and SAN
- **Double the adapters** -- NICs and HBAs are distinct hardware
- **Double the switches** -- Ethernet switches and FC switches coexist
- **Double the management** -- separate tools, firmware, monitoring
- **Double the optics** -- SFP modules for both fabrics

### 1.2 The I/O Consolidation Vision

FCoE was conceived to collapse these parallel networks into one:

```
Server Rack (with FCoE)
+---------------------------------------------------+
|  Server 1                                          |
|    CNA port 1 -----> DCB Ethernet Switch (Nexus)   |
|    CNA port 2 -----> DCB Ethernet Switch (Nexus)   |
|                                                    |
|  Per server: 2 cables, 1 dual-port CNA             |
|  Per rack (40 servers): 80 cables                   |
|  Savings: 50% fewer cables, adapters, switch ports  |
+---------------------------------------------------+
```

The key insight: **do not replace FC -- encapsulate it**. This preserved:
- Existing FC SAN investments (arrays, zoning databases, management tools)
- FC operational model (FLOGI, PLOGI, PRLI, zoning, LUN masking)
- FC performance characteristics (low latency, deterministic behavior)

### 1.3 Standards Bodies

| Standard       | Body  | Scope                                    |
|----------------|-------|------------------------------------------|
| FC-BB-5/BB-6   | T11   | FCoE encapsulation, FIP protocol         |
| 802.1Qbb       | IEEE  | Priority Flow Control (PFC)              |
| 802.1Qaz       | IEEE  | Enhanced Transmission Selection (ETS)    |
| 802.1Qaz       | IEEE  | DCBX (same document as ETS)             |
| 802.1Qau       | IEEE  | Congestion Notification (QCN)            |
| CEE            | Cisco | Pre-standard DCB (Converged Enhanced Ethernet) |

## 2. Data Center Bridging (DCB) In Depth

FCoE cannot function over standard Ethernet because Ethernet is inherently lossy.
Switches drop frames when output buffers overflow. For IP traffic this is acceptable
because TCP retransmits. For FC/SCSI traffic, frame drops cause SCSI abort/reset
cascades that are catastrophic to storage performance.

DCB transforms Ethernet into a lossless transport for selected traffic classes.

### 2.1 Priority Flow Control (PFC) -- IEEE 802.1Qbb

#### The Problem with 802.3x PAUSE

Standard Ethernet PAUSE (802.3x) stops **all** traffic on a link when buffers fill.
This creates head-of-line blocking: low-priority bulk transfers can pause
latency-sensitive storage or voice traffic.

#### Per-Priority PAUSE

PFC extends PAUSE to operate independently on each of the 8 CoS (Class of Service)
values defined by 802.1p (the priority bits in the 802.1Q VLAN tag).

```
802.1Q VLAN Tag (4 bytes):
+--------+-------+------------+
| TPID   | PRI   | VLAN ID    |
| 0x8100 | 3-bit | 12-bit     |
+--------+-------+------------+
           |
           +-> CoS 0-7 (8 possible priorities)

PFC Configuration per CoS:
  CoS 0: no-drop = false  (standard IP, best-effort)
  CoS 1: no-drop = false
  CoS 2: no-drop = false
  CoS 3: no-drop = true   <-- FCoE traffic, PFC-protected
  CoS 4: no-drop = false  (or true for iSCSI)
  CoS 5: no-drop = false
  CoS 6: no-drop = false
  CoS 7: no-drop = false
```

#### PFC Frame Format

PFC uses a MAC control frame (EtherType 0x8808) with opcode 0x0101:

```
+-----------+-----------+--------+--------+--...--+
| Dest MAC  | Src MAC   | 0x8808 | 0x0101 | Per-  |
| (01:80:C2 |           | (MAC   | (PFC   | prio  |
| :00:00:01)|           |  ctrl) | opcode)| timers|
+-----------+-----------+--------+--------+--...--+
                                           |
                    8 x 16-bit quanta timers (one per CoS)
                    0x0000 = unpause, 0xFFFF = max pause
```

When a switch detects its ingress buffer for CoS 3 (FCoE) reaching a threshold,
it sends a PFC frame to its upstream neighbor with a non-zero quanta value for
priority 3. The upstream device stops transmitting CoS 3 frames for the specified
duration but continues sending all other CoS classes.

#### PFC Deadlock

A critical risk with PFC is **deadlock** -- circular dependencies where two or
more switches mutually pause each other, permanently halting traffic. Mitigations:

- **Careful topology design** -- avoid loops in the FCoE VLAN
- **PFC watchdog** -- Nexus feature that detects and breaks deadlocks
- **Limit PFC scope** -- only enable on FCoE-carrying links

```
! Nexus PFC watchdog configuration
priority-flow-control watch-dog-interval on
priority-flow-control watch-dog shutdown-multiplier 1

show priority-flow-control watch-dog interface
```

### 2.2 Enhanced Transmission Selection (ETS) -- IEEE 802.1Qaz

ETS provides **bandwidth management** across traffic classes (TCs). Without ETS,
a burst of IP traffic could consume all available bandwidth, starving FCoE even
if PFC prevents drops.

#### Traffic Class Mapping

802.1p defines 8 CoS values; ETS groups these into Traffic Classes (max 8 TCs,
but most implementations support fewer).

```
Traffic Class Architecture:

  CoS 0 --|
  CoS 1 --|-- TC0 (Best-Effort IP) --> 40% minimum BW
  CoS 2 --|
  CoS 5 --|
  CoS 6 --|
  CoS 7 --|

  CoS 3 ----  TC1 (FCoE/Storage)   --> 50% minimum BW, no-drop

  CoS 4 ----  TC2 (iSCSI/optional) --> 10% minimum BW

Total: 100%
```

#### Scheduling Algorithms

ETS supports two scheduling modes per TC:

| Mode                | Behavior                                      |
|---------------------|-----------------------------------------------|
| Strict Priority     | TC always served first (risk of starvation)   |
| Weighted (ETS)      | Minimum BW guaranteed, excess shared fairly   |

Typical FCoE deployments use weighted scheduling so that neither IP nor FCoE
can completely starve the other.

#### Bandwidth Borrowing

If TC1 (FCoE) is idle, its 50% allocation is redistributed to active TCs.
Bandwidth is guaranteed only under contention -- the full link is available
to any TC when others are idle.

### 2.3 DCBX -- Data Center Bridging Capability Exchange

DCBX is a discovery and negotiation protocol that runs over LLDP (Link Layer
Discovery Protocol). It enables two connected devices to automatically agree
on PFC, ETS, and application priority settings.

#### DCBX Versions

| Version     | Origin          | TLV Format                  |
|-------------|-----------------|-----------------------------|
| CEE (CIN)   | Cisco/Intel     | Pre-standard, proprietary   |
| CEE (DCBX)  | Industry draft  | Pre-standard, interoperable |
| IEEE        | 802.1Qaz        | Final standard              |

**Critical**: Both ends of a link must use the same DCBX version. A mismatch
causes negotiation failure and FCoE will not come up. Common interoperability
issue when mixing vendor equipment.

#### DCBX TLV Structure

```
LLDP Frame:
+---------------------------------------------------+
| Chassis ID TLV | Port ID TLV | TTL TLV |          |
+---------------------------------------------------+
| Org-Specific TLVs (DCBX):                         |
|   +-- PFC Configuration TLV                       |
|   |     - PFC enable bitmap (per CoS)              |
|   |     - PFC willing bit                          |
|   +-- ETS Configuration TLV                        |
|   |     - TC bandwidth allocations                 |
|   |     - Priority-to-TC mapping                   |
|   |     - TSA (scheduling algorithm per TC)         |
|   +-- ETS Recommendation TLV                       |
|   +-- Application Priority TLV                     |
|         - FCoE: EtherType 0x8906, Priority 3       |
|         - FIP:  EtherType 0x8914, Priority 3       |
|         - iSCSI: TCP port 3260, Priority 4         |
+---------------------------------------------------+
```

#### Willing vs Unwilling

Each DCBX peer can be **willing** (accepts the other side's config) or
**unwilling** (insists on its own config):

- **Switch unwilling, CNA willing** -- most common; switch dictates policy
- **Both unwilling** -- must have matching configs or negotiation fails
- **Both willing** -- lower-numbered peer wins (vendor-specific tiebreak)

### 2.4 Congestion Notification (QCN) -- IEEE 802.1Qau

QCN provides end-to-end congestion management. When a congestion point (CP)
detects sustained queue buildup, it generates Congestion Notification Messages
(CNMs) back to the source (Reaction Point / RP), which throttles its
transmission rate.

```
Source (RP) -----> Switch (CP, detecting congestion) -----> Destination
                         |
                         |  CNM (congestion notification message)
                         |  with feedback: severity, rate adjustment
                         v
Source throttles rate <---+
```

QCN is complementary to PFC:
- **PFC** is hop-by-hop and reactive (pause after buffer fills)
- **QCN** is end-to-end and proactive (slow down before drops)

In practice, QCN is rarely deployed. PFC alone handles most FCoE congestion
scenarios adequately.

## 3. FCoE Initialization Protocol (FIP)

Native FC relies on physical-layer signaling (LIP, loop initialization, link
reset) to discover the fabric and perform login. Since FCoE runs over Ethernet,
these physical-layer mechanisms do not exist. FIP replaces them.

### 3.1 FIP EtherType

FIP uses EtherType **0x8914** (distinct from FCoE's 0x8906). FIP frames are
regular Ethernet frames and do not require the FCoE VLAN to be established
yet -- this is how VLAN discovery can happen before the FCoE VLAN is known.

### 3.2 FIP Multicast Addresses

| Address              | Name         | Purpose                           |
|----------------------|--------------|-----------------------------------|
| 01:10:18:01:00:01    | ALL-FCF-MACs | ENode -> FCF solicitations        |
| 01:10:18:01:00:02    | ALL-ENode-MACs| FCF -> ENode advertisements      |

### 3.3 VLAN Discovery

```
Step 1: VLAN Discovery

ENode (CNA)                                    Switch (FCF)
    |                                               |
    |  FIP VLAN Request                              |
    |  Dest: ALL-FCF-MACs                            |
    |  EtherType: 0x8914                             |
    |  Sent on default/native VLAN                   |
    |---------------------------------------------->|
    |                                               |
    |  FIP VLAN Notification                         |
    |  Contains: FCoE VLAN ID(s)                     |
    |  (e.g., VLAN 100)                              |
    |<----------------------------------------------|
    |                                               |
    |  ENode now tags all FCoE traffic               |
    |  with VLAN 100                                 |
```

The ENode sends the VLAN discovery request on whatever VLAN is available
(typically native VLAN). The FCF responds with the FCoE VLAN IDs. From this
point forward, all FCoE and FIP traffic uses the FCoE VLAN.

### 3.4 FCF Discovery

```
Step 2: FCF Discovery

ENode                                          FCF
    |                                               |
    |  FIP Discovery Solicitation                    |
    |  Dest: ALL-FCF-MACs                            |
    |  On FCoE VLAN                                  |
    |---------------------------------------------->|
    |                                               |
    |  FIP FCF Advertisement (unicast reply)         |
    |  Contains:                                     |
    |    - FC-MAP (3 bytes, e.g., 0E:FC:00)         |
    |    - Switch Name (WWN)                         |
    |    - Fabric Name (WWN)                         |
    |    - FKA_ADV_PERIOD (keep-alive interval)      |
    |    - FCF Priority (lower = preferred)          |
    |<----------------------------------------------|
```

If multiple FCFs respond, the ENode selects the one with the **lowest priority
value** (highest priority). This enables load balancing and failover across
multiple FCF switches.

#### FC-MAP (FC MAC Address Prefix)

The FC-MAP is a 3-byte value (default 0E:FC:00) that forms the upper 24 bits
of the FCoE MAC address. The lower 24 bits come from the assigned FCID.

```
FCoE VN-Port MAC Address Construction:
+--------------------+--------------------+
| FC-MAP (3 bytes)   |  FCID (3 bytes)    |
| e.g., 0E:FC:00     |  e.g., 01:02:03   |
+--------------------+--------------------+
= 0E:FC:00:01:02:03

This MAC is used as the source MAC for all FCoE frames from this VN-Port.
```

### 3.5 Fabric Login (FLOGI)

```
Step 3: FLOGI

ENode                                          FCF
    |                                               |
    |  FIP FLOGI Request                             |
    |  Contains:                                     |
    |    - Node Name WWN                             |
    |    - Port Name WWN                             |
    |    - Max frame size                            |
    |    - Requested features                        |
    |---------------------------------------------->|
    |                                               |
    |  FIP FLOGI Accept (LS_ACC)                     |
    |  Contains:                                     |
    |    - Assigned FCID (e.g., 0x010203)            |
    |    - Fabric parameters                         |
    |    - Confirmed features                        |
    |<----------------------------------------------|
    |                                               |
    |  ENode now has:                                |
    |    - FCID for addressing                       |
    |    - VN-Port MAC (FC-MAP + FCID)               |
    |    - Full FC fabric membership                 |
```

After FLOGI, the ENode proceeds with standard FC operations:
- **PLOGI** (Port Login) to target ports
- **PRLI** (Process Login) for SCSI FCP
- Normal FC zoning and LUN masking apply

### 3.6 FIP Keep-Alive and Clear Virtual Link (CVL)

```
Maintenance Phase:

ENode                                          FCF
    |                                               |
    |  FIP Keep-Alive (periodic)                     |
    |  Interval: FKA_ADV_PERIOD (default 8 sec)      |
    |---------------------------------------------->|
    |                                               |
    |  If FCF does not receive keep-alive            |
    |  within 2.5x FKA_ADV_PERIOD:                   |
    |    -> FCF purges VN-Port, releases FCID        |
    |                                               |
    |  If ENode needs to be logged out:              |
    |  FIP CVL (Clear Virtual Link)                  |
    |  Contains: VN-Port MAC, VF-Port MAC            |
    |<----------------------------------------------|
    |                                               |
    |  ENode tears down FC session                   |
```

CVL is used when:
- The switch is rebooting or a port is going down
- VSAN configuration changes
- Administrative clearing of sessions

## 4. FCoE Port Types In Detail

### 4.1 VN-Port (Virtual N-Port)

The FCoE equivalent of an FC N-Port (node port). Instantiated on the CNA.

| Property       | Value                                          |
|----------------|------------------------------------------------|
| Location       | CNA / ENode (server-side)                      |
| MAC address    | FC-MAP + FCID (assigned during FLOGI)          |
| FC equivalent  | N-Port                                         |
| Connects to    | VF-Port on FCF                                 |
| Identified by  | PWWN (Port World-Wide Name)                    |

### 4.2 VF-Port (Virtual F-Port)

The switch-side counterpart to VN-Port. Instantiated on the FCF (FCoE Forwarder).

| Property       | Value                                          |
|----------------|------------------------------------------------|
| Location       | FCF switch (e.g., Nexus 5000/5500/7000)        |
| FC equivalent  | F-Port                                         |
| Connects to    | VN-Port on ENode                               |
| Created by     | VFC interface configuration                    |

### 4.3 VE-Port (Virtual E-Port)

Used for inter-switch FCoE links (FCoE ISLs). Enables multi-hop FCoE.

| Property       | Value                                          |
|----------------|------------------------------------------------|
| Location       | FCF switch                                     |
| FC equivalent  | E-Port (ISL)                                   |
| Connects to    | VE-Port on adjacent FCF                        |
| Purpose        | Extend FCoE across multiple switches           |
| Requirements   | End-to-end lossless Ethernet between switches   |

### 4.4 Port Relationships

```
                  Single-hop FCoE:

  Server                     Nexus (FCF)              FC SAN
+--------+                +------------+           +----------+
|        |                |            |           |          |
| VN-Port|----FCoE------->| VF-Port    |           |          |
|  (CNA) |   Ethernet     |            |           |          |
|        |                |     FC     |--native-->| F-Port   |
|        |                |    Port    |    FC     |          |
+--------+                +------------+           +----------+


                  Multi-hop FCoE:

  Server                Nexus A (FCF)         Nexus B (FCF)        FC SAN
+--------+            +------------+        +------------+      +----------+
|        |            |            |        |            |      |          |
| VN-Port|---FCoE---->| VF-Port    |        |            |      |          |
|  (CNA) |            |            |        |            |      |          |
|        |            | VE-Port    |--FCoE--| VE-Port    |      |          |
|        |            |            |  ISL   |            |      |          |
|        |            |            |        |     FC     |--FC--| F-Port   |
+--------+            +------------+        +------------+      +----------+
```

## 5. Converged Network Adapter (CNA) Architecture

### 5.1 Hardware Design

A CNA is a single PCIe adapter that integrates both an FC HBA and an Ethernet
NIC into one silicon package.

```
CNA Internal Architecture:

+------------------------------------------------------------------+
|                        CNA (PCIe Card)                            |
|                                                                   |
|  +-----------------------------+  +----------------------------+  |
|  |      FC Engine              |  |     Ethernet Engine        |  |
|  |  +----------+ +---------+  |  |  +---------+ +----------+  |  |
|  |  | FCoE     | | SCSI    |  |  |  | TCP/UDP | | RSS/LRO  |  |  |
|  |  | Encap/   | | Offload |  |  |  | Offload | | Engine   |  |  |
|  |  | Decap    | | Engine  |  |  |  | Engine  | |          |  |  |
|  |  +----------+ +---------+  |  |  +---------+ +----------+  |  |
|  |  | FIP      | | FC      |  |  |  | VXLAN   | | SR-IOV   |  |  |
|  |  | Protocol | | Login   |  |  |  | Offload | | Engine   |  |  |
|  |  | Engine   | | State   |  |  |  | (opt)   | |          |  |  |
|  |  +----------+ +---------+  |  |  +---------+ +----------+  |  |
|  +-----------------------------+  +----------------------------+  |
|                                                                   |
|  +-----------------------------+  +----------------------------+  |
|  |  Shared Components:         |  |  Physical Layer:           |  |
|  |  - DMA engines              |  |  - 10/25/40/100 GbE MAC   |  |
|  |  - PCIe Gen3/Gen4 interface |  |  - SFP+/QSFP28 optics     |  |
|  |  - Memory controller        |  |  - SerDes / PHY            |  |
|  +-----------------------------+  +----------------------------+  |
+------------------------------------------------------------------+
```

### 5.2 OS Presentation

The CNA presents **two distinct devices** to the operating system:

```
Linux Host:

  $ lspci | grep -i converge
  04:00.0 Fibre Channel: Emulex OneConnect (FCoE)
  04:00.1 Ethernet controller: Emulex OneConnect

  $ ls /sys/class/fc_host/
  host3    <-- FC HBA (VN-Port)

  $ ip link show
  ens3f0   <-- Ethernet NIC

  $ lsscsi
  [3:0:0:0]  disk  NETAPP  LUN  ...  /dev/sdb   <-- FC LUN via FCoE
```

The FC driver (e.g., `lpfc` for Emulex, `qla2xxx` for QLogic) manages the
VN-Port. The network driver (e.g., `be2net`, `qede`) manages the Ethernet
interface. From the OS perspective, these are independent devices sharing
one physical port.

### 5.3 CNA Vendors

| Vendor            | Product Line        | Speeds          | Driver (Linux)     |
|-------------------|---------------------|-----------------|--------------------|
| Broadcom (Emulex) | OneConnect OCe14000 | 10/25/40 GbE    | lpfc, be2net       |
| Marvell (QLogic)  | QLE8362             | 10/16 Gb        | qla2xxx, qede      |
| Intel             | X710/XL710 (limited)| 10/40 GbE       | i40e (limited FCoE)|
| Cisco             | VIC 1340/1380       | 10/40 GbE       | fnic, enic         |

## 6. FCoE Frame Format In Detail

### 6.1 Complete Frame Layout

```
FCoE Frame (maximum 2180 bytes on the wire):

Byte Offset   Field                    Size      Description
-----------   -----                    ----      -----------
 0            Destination MAC          6 bytes   VN-Port or FCF MAC
 6            Source MAC               6 bytes   Sender's FCoE MAC
12            802.1Q Tag (TPID)        2 bytes   0x8100
14            TCI (PRI + VLAN)         2 bytes   CoS 3 + FCoE VLAN ID
16            EtherType                2 bytes   0x8906 (FCoE)
18            FCoE Header:
                Version                4 bits    0 (current version)
                Reserved               100 bits  Must be zero
                SOF                    8 bits    Start of Frame delimiter
                                                 (SOFi3=0x2E, SOFn3=0x36)
32            FC Frame:
                R_CTL                  1 byte    Routing control
                D_ID                   3 bytes   Destination FCID
                CS_CTL / Priority      1 byte    Class-specific control
                S_ID                   3 bytes   Source FCID
                TYPE                   1 byte    Protocol type (0x08=FCP)
                F_CTL                  3 bytes   Frame control bits
                SEQ_ID                 1 byte    Sequence identifier
                DF_CTL                 1 byte    Data field control
                SEQ_CNT               2 bytes   Sequence count
                OX_ID                  2 bytes   Originator exchange ID
                RX_ID                  2 bytes   Responder exchange ID
                Parameter              4 bytes   Relative offset / other
                Payload                0-2112    FC payload data
                                       bytes
              FC CRC                   4 bytes   FC frame check sequence

              EOF                      1 byte    End of Frame delimiter
                                                 (EOFt=0x42, EOFn=0x41)
              Padding (to 4-byte)      0-3 bytes
              Ethernet FCS             4 bytes   Ethernet CRC-32
```

### 6.2 Frame Size Implications

```
Size breakdown:
  Ethernet overhead:   14 bytes (MAC + EtherType)
  802.1Q tag:           4 bytes
  FCoE header:         14 bytes (ver + reserved + SOF)
  FC header:           24 bytes
  FC payload:        2112 bytes (maximum)
  FC CRC:              4 bytes
  EOF:                 1 byte
  Padding:             3 bytes (worst case)
  Ethernet FCS:        4 bytes
  -----------------------------------
  Total:             2180 bytes

Standard Ethernet MTU: 1500 bytes  --> INSUFFICIENT
Required MTU:          >= 2500 bytes (absolute minimum)
Recommended MTU:       9216 bytes (baby jumbo / jumbo)
```

This is why **jumbo frames are mandatory** for FCoE. Every switch, router
interface, and host NIC in the FCoE path must support and be configured
for jumbo frames.

### 6.3 FIP Frame Format

```
FIP Frame:

Byte Offset   Field                    Size      Description
-----------   -----                    ----      -----------
 0            Destination MAC          6 bytes   Unicast or multicast
 6            Source MAC               6 bytes   Sender MAC
12            EtherType                2 bytes   0x8914 (FIP)
14            FIP Header:
                Version                4 bits    1
                Reserved               12 bits
                Protocol Code          2 bytes   Operation category
                Sub-code               1 byte    Specific operation
                Descriptor List Length  2 bytes   In 32-bit words
                Flags                  2 bytes   FP, SP, etc.
22            FIP Descriptors:
                (variable-length TLVs)
                - Priority Descriptor
                - MAC Address Descriptor
                - FC-MAP Descriptor
                - Name Identifier Descriptor
                - Fabric Descriptor
                - Max FCoE Size Descriptor
                - FLOGI Descriptor
                - VLAN Descriptor
                ...

FIP Protocol Codes:
  0x0001  Discovery (solicitation / advertisement)
  0x0002  Login / Logout (FLOGI, FDISC, LOGO)
  0x0003  ELP (for VE-Port setup)
  0x0004  VLAN (VLAN discovery)
  0x0005  VN2VN (direct VN-Port to VN-Port, no FCF)
```

## 7. Single-Hop vs Multi-Hop FCoE

### 7.1 Single-Hop FCoE (FCF at First Hop)

This is the most widely deployed FCoE model. The CNA connects via FCoE to
an adjacent switch that acts as the FCF. The FCF terminates FCoE and connects
to the existing FC SAN fabric via native FC uplinks.

```
Single-Hop Architecture:

                    FCoE Domain            FC Domain
                 (Ethernet/DCB)         (Native FC)

+--------+      +----------------+     +-----------+     +----------+
| Server |      |   Nexus 5596   |     |  MDS 9148 |     | Storage  |
|  CNA   |======|  (FCF)         |=====|  (FC SW)  |=====| Array    |
|        | FCoE |  VF  |  FC     | FC  |           |     | (Target) |
| VN-Port|      | Port | Ports   |     |           |     |          |
+--------+      +----------------+     +-----------+     +----------+

  Legend: ====  physical connection
          FCoE  = Ethernet + FCoE encapsulation
          FC    = native Fibre Channel
```

**Advantages:**
- Simple -- only one FCoE hop to manage
- Well-tested, widely supported by all CNA and switch vendors
- FC SAN fabric remains unchanged
- Easy troubleshooting -- FCoE issues isolated to access layer

**Disadvantages:**
- Every FCoE switch must have native FC ports (cost)
- FC and Ethernet domains remain somewhat separate

### 7.2 Multi-Hop FCoE (FCoE Between Switches)

FCoE traffic traverses multiple Ethernet switches before reaching an FCF
with native FC connectivity (or the target itself supports FCoE).

```
Multi-Hop Architecture:

+--------+     +-----------+     +-----------+     +-----------+     +----------+
| Server |     | Nexus A   |     | Nexus B   |     | Nexus C   |     | Storage  |
|  CNA   |=====| (FCF)     |=====| (FCF)     |=====| (FCF)     |=====| Array    |
| VN-Port| FCoE| VF  | VE  | FCoE| VE  | VE  | FCoE| VE  | FC  | FC |          |
+--------+     +-----------+     +-----------+     +-----------+     +----------+
                    ^                  ^                  ^
                 hop 1              hop 2              hop 3
                                  VE-Port             VE-Port
                                    ISL                 ISL
```

**Advantages:**
- Not every switch needs native FC ports
- Can build an all-Ethernet data center fabric

**Disadvantages:**
- Every hop must be lossless (PFC end-to-end)
- PFC deadlock risk increases with each additional hop
- More complex VSAN/VLAN planning
- Harder to troubleshoot
- Less vendor support and real-world deployment experience

### 7.3 FEX (Fabric Extender) Model

Fabric Extenders (e.g., Nexus 2000) are not full switches -- they are remote
line cards managed by a parent switch. From FCoE's perspective, the FEX is
transparent and the parent switch acts as the FCF.

```
FEX Model:

+--------+     +----------+                   +-----------+     +----------+
| Server |     | Nexus    |     Fabric Link   | Nexus     |     | FC SAN   |
|  CNA   |=====| 2348     |===================| 5596      |=====| Fabric   |
| VN-Port| FCoE| (FEX)    |    (FCoE/DCB)     | (Parent/  | FC  |          |
+--------+     | (no FCF) |                   |  FCF)     |     +----------+
               +----------+                   +-----------+
                    ^                               ^
             Not a switch;                   All FC processing
             just extends                    happens here
             parent ports
```

This is effectively single-hop from an FC perspective because the FEX does
not participate in FC fabric services.

## 8. FCoE Topologies

### 8.1 Directly Connected CNA to FCF

The simplest topology. Each server's CNA connects directly to a port on
the FCF switch.

```
+--------+
| Srv 1  |===\
+--------+    \    +-----------+     +---------+
               +===| Nexus     |=====| FC SAN  |
+--------+    /    | 5596      |     | Fabric  |
| Srv 2  |===/    | (FCF)     |     |         |
+--------+    \    +-----------+     +---------+
               \
+--------+      \
| Srv 3  |=======+
+--------+
```

### 8.2 vPC + FCoE (Dual-Fabric Design)

This is the recommended production topology for FCoE. It provides redundancy
through dual-attached servers and separate SAN fabrics.

```
                          Fabric A                    Fabric B

+--------+            +-----------+               +-----------+
| Server |            | Nexus A   |               | Nexus B   |
|        |            | (FCF)     |               | (FCF)     |
|  CNA   |----FCoE--->| VFC 1     |               | VFC 1     |
|  Port 1|            | VSAN 100  |               |           |
|        |            | VLAN 100  |               |           |
|        |            |     |     |               |     |     |
|  CNA   |----FCoE----|-----|-----|----FCoE------->| VFC 2     |
|  Port 2|            |     |     |               | VSAN 200  |
+--------+            |     |     |               | VLAN 200  |
                      |     |     |               |     |     |
                      |   FC     |               |   FC     |
                      |   Ports   |               |   Ports   |
                      +----|------+               +----|------+
                           |                           |
                      +----|------+               +----|------+
                      | FC Switch |               | FC Switch |
                      | Fabric A  |               | Fabric B  |
                      +----|------+               +----|------+
                           |                           |
                      +----|------+               +----|------+
                      | Storage   |               | Storage   |
                      | Port A    |               | Port B    |
                      +-----------+               +-----------+

Key Design Rules:
  1. FCoE NEVER crosses the vPC peer-link
  2. Each Nexus is an INDEPENDENT FCF
  3. Each path uses a DIFFERENT VSAN (100 vs 200)
  4. Each path uses a DIFFERENT VLAN (100 vs 200)
  5. Server uses MPIO (DM-Multipath) for path failover
```

**Why FCoE cannot cross the vPC peer-link:**
- vPC peer-link is a standard Ethernet port-channel
- FCoE requires dedicated VFC interface bindings
- FC fabric services (FLOGI, zoning) are per-switch, not shared across vPC
- Crossing the peer-link would create split-brain FC fabric issues

### 8.3 UCS with FCoE

Cisco UCS (Unified Computing System) uses FCoE internally between the
blade chassis IOM (I/O Module) and the Fabric Interconnect:

```
+----------------------------------------------------------+
|  UCS Chassis (5108)                                       |
|  +----------+  +----------+  +----------+  +----------+  |
|  | Blade 1  |  | Blade 2  |  | Blade 3  |  | Blade 4  |  |
|  | (VIC)    |  | (VIC)    |  | (VIC)    |  | (VIC)    |  |
|  +----+-----+  +----+-----+  +----+-----+  +----+-----+  |
|       |              |              |              |       |
|  +----|--------------|--------------|--------------|----+  |
|  |         IOM 2208 (Fabric Extender)                  |  |
|  +----------------------------+----------------------------+
|                               |                            |
+-------------------------------|----------------------------+
                                |  FCoE over 10/40G Ethernet
                                |
                     +----------+-----------+
                     |  UCS Fabric           |
                     |  Interconnect 6332    |
                     |  (FCF)                |
                     +----------+-----------+
                                |
                           Native FC uplinks
                                |
                     +----------+-----------+
                     |    FC SAN Fabric      |
                     +----------------------+
```

## 9. Nexus Configuration Deep Dive

### 9.1 Complete Single-Hop FCoE Configuration

```
! ====== System-Level Configuration ======

! Step 1: Enable required features
feature fcoe
feature npiv
feature lldp

! Step 2: Configure system QoS for FCoE
! Define network-qos policy with no-drop for FCoE class
policy-map type network-qos FCoE-NQ-Policy
  class type network-qos class-fcoe
    pause no-drop
    mtu 9216
  class type network-qos class-default
    mtu 9216

system qos
  service-policy type network-qos FCoE-NQ-Policy

! Step 3: Configure queuing policy (output scheduling)
policy-map type queuing FCoE-Queuing-Policy
  class type queuing class-fcoe
    bandwidth percent 50
  class type queuing class-default
    bandwidth percent 50

! ====== VSAN Configuration ======

! Step 4: Create VSAN
vsan database
  vsan 100
  vsan 100 name FCoE-SAN-A

! ====== VLAN Configuration ======

! Step 5: Create FCoE VLAN and bind to VSAN
vlan 100
  fcoe vsan 100
  name FCoE-VLAN-100

! ====== Interface Configuration ======

! Step 6: Configure physical Ethernet interface
interface ethernet 1/1
  description FCoE-to-Server-1
  switchport mode trunk
  switchport trunk allowed vlan 1,100
  spanning-tree port type edge trunk
  priority-flow-control mode on
  service-policy type queuing output FCoE-Queuing-Policy
  mtu 9216
  no shutdown

! Step 7: Create and bind VFC interface
interface vfc 1
  description VFC-to-Server-1
  bind interface ethernet 1/1
  switchport trunk allowed vsan 100
  no shutdown

! ====== FC Uplink Configuration ======

! Step 8: Configure native FC uplinks to SAN fabric
interface fc 2/1
  switchport mode auto
  switchport trunk allowed vsan 100
  no shutdown

! ====== Zoning (same as native FC) ======

! Step 9: Create zone and zoneset
zone name Server1-to-Storage vsan 100
  member pwwn 20:00:00:25:b5:01:00:01   ! Server CNA PWWN
  member pwwn 50:00:09:72:08:00:00:01   ! Storage target PWWN

zoneset name Production vsan 100
  member Server1-to-Storage

zoneset activate name Production vsan 100
```

### 9.2 VFC Bound to Port-Channel (for vPC)

```
! Physical port-channel members
interface ethernet 1/1
  channel-group 10 mode active
  no shutdown

interface ethernet 1/2
  channel-group 10 mode active
  no shutdown

! Port-channel configuration
interface port-channel 10
  switchport mode trunk
  switchport trunk allowed vlan 100
  spanning-tree port type edge trunk
  priority-flow-control mode on
  mtu 9216
  no shutdown

! VFC bound to port-channel
interface vfc 10
  bind interface port-channel 10
  switchport trunk allowed vsan 100
  no shutdown
```

### 9.3 Multi-Hop FCoE (VE-Port Configuration)

```
! ====== Switch A (upstream FCF) ======

! Physical interface for FCoE ISL
interface ethernet 1/48
  switchport mode trunk
  switchport trunk allowed vlan 100
  priority-flow-control mode on
  mtu 9216
  no shutdown

! VFC interface in VE-Port mode
interface vfc 48
  bind interface ethernet 1/48
  switchport mode E
  switchport trunk allowed vsan 100
  no shutdown

! ====== Switch B (downstream FCF) ======

! Matching configuration on the other end
interface ethernet 1/48
  switchport mode trunk
  switchport trunk allowed vlan 100
  priority-flow-control mode on
  mtu 9216
  no shutdown

interface vfc 48
  bind interface ethernet 1/48
  switchport mode E
  switchport trunk allowed vsan 100
  no shutdown
```

### 9.4 Verification Commands

```
! FCoE operational status
show fcoe
show vfc
show vfc database

! FIP/FLOGI verification
show flogi database
show flogi database vsan 100
show fcns database vsan 100

! VLAN-VSAN binding
show vlan fcoe

! DCB/PFC verification
show interface ethernet 1/1 priority-flow-control
show lldp neighbors detail
show dcbx interface ethernet 1/1

! QoS and queuing
show queuing interface ethernet 1/1
show policy-map type queuing
show policy-map type network-qos

! Counters and errors
show interface ethernet 1/1 counters
show interface ethernet 1/1 counters errors
show interface vfc 1 counters

! FCoE-specific debugging
show platform fwm info pif ethernet 1/1
show system internal dcbx info interface ethernet 1/1
```

## 10. Lossless Ethernet Requirements

### 10.1 End-to-End Lossless Checklist

Every component in the FCoE data path must meet these requirements:

```
Component Checklist:

+---+-------------------------------------------+-------------------------------+
| # | Requirement                               | Verification                   |
+---+-------------------------------------------+-------------------------------+
| 1 | Jumbo frames (MTU >= 2500, rec. 9216)     | show interface eth X/Y         |
| 2 | PFC enabled on FCoE CoS (typically 3)     | show int eth X/Y pfc           |
| 3 | DCBX enabled and converged                | show dcbx int eth X/Y          |
| 4 | DCBX version matched (CEE or IEEE)        | show lldp neighbors detail     |
| 5 | No-drop queue mapped to FCoE CoS          | show policy-map type network-qos|
| 6 | ETS bandwidth reserved for FCoE TC        | show policy-map type queuing   |
| 7 | STP edge port (portfast trunk)            | show spanning-tree int eth X/Y |
| 8 | Dedicated FCoE VLAN (not VLAN 1)          | show vlan fcoe                 |
| 9 | VLAN pruned on all trunks (only FCoE)     | show int trunk                 |
|10 | CNA firmware supports FCoE/DCB            | CNA vendor tools               |
+---+-------------------------------------------+-------------------------------+
```

### 10.2 What Happens When Lossless Fails

```
Scenario: PFC disabled or misconfigured on one hop

Server ---FCoE--> Switch A (PFC ON) ---FCoE--> Switch B (PFC OFF) --> Storage
                                                    ^
                                                    |
                                              Buffer overflow here
                                              drops FCoE frames
                                                    |
                                                    v
                                           SCSI exchange timeout
                                           SCSI abort issued
                                           I/O retry (slow)
                                           Application sees latency spike
                                           Repeated: link reset, path failover
```

Even a single misconfigured hop in the FCoE path can cause cascading failures.
This is why single-hop FCoE is strongly preferred -- fewer hops means fewer
places for configuration errors.

## 11. FCoE vs iSCSI -- Detailed Comparison

```
+------------------------+---------------------------+-----------------------------+
| Attribute              | FCoE                      | iSCSI                       |
+------------------------+---------------------------+-----------------------------+
| Standards body         | T11 (FC-BB-5/6)           | IETF (RFC 7143)             |
| Transport              | Ethernet (Layer 2)        | TCP/IP (Layer 3/4)          |
| Encapsulation          | FC frame in Ethernet      | SCSI CDB in TCP stream      |
| Routing                | NOT routable (L2)         | Fully routable (L3)         |
| Lossless requirement   | Mandatory (DCB/PFC)       | Not required (TCP retrans)  |
| MTU                    | Jumbo required (>=2500)   | 1500 works (jumbo helps)    |
| Adapter                | CNA (specialized HW)      | Standard NIC or iSCSI HBA  |
| Software initiator     | No (requires CNA)         | Yes (built into all OSes)   |
| Latency (typical)      | 50-200 us                 | 200-1000 us                 |
| Throughput             | Line rate (HW offload)    | Near line rate (with TOE)   |
| CPU overhead           | Very low                  | Low-medium (with offload)   |
| Switch requirements    | DCB-capable (expensive)   | Any Ethernet switch         |
| Max distance           | L2 domain (~few km)       | Unlimited (IP routed)       |
| Multi-tenancy          | VSAN (FC virtual fabrics)  | VLAN + IP subnets           |
| Authentication         | FC zoning + LUN masking   | CHAP + IP ACLs              |
| Boot from SAN          | Yes                       | Yes                         |
| Existing FC SAN compat | Yes (native FC mapping)   | No (different protocol)     |
| NVMe support           | Yes (NVMe/FC, NVMe/FCoE)  | Yes (NVMe/TCP)              |
| Market trend (2024+)   | Declining / legacy         | Growing / preferred for new |
| Cloud/hyperscaler use  | Rare                      | Common (or NVMe/TCP)        |
+------------------------+---------------------------+-----------------------------+

Decision Matrix:

  Choose FCoE when:
    - You have an existing FC SAN investment to protect
    - You want I/O consolidation without replacing FC infrastructure
    - Ultra-low latency is critical (trading, HPC)
    - You are already running Cisco Nexus + MDS

  Choose iSCSI when:
    - Building a new SAN from scratch
    - Budget is a concern (no DCB switches needed)
    - You need storage traffic to cross L3 boundaries
    - You want software-only initiators (no special HW)
    - Cloud or multi-site deployments

  Choose NVMe/TCP or NVMe/FC when:
    - Greenfield deployment with NVMe storage arrays
    - Maximum performance is the priority
    - Modern infrastructure (2022+ hardware)
```

## Prerequisites

- Solid understanding of Fibre Channel concepts (N-Port, F-Port, FLOGI, zoning, VSAN)
- Ethernet switching fundamentals (VLANs, trunking, 802.1Q)
- QoS concepts (CoS, traffic classes, scheduling)
- Familiarity with Cisco NX-OS CLI (for configuration examples)
- Understanding of SCSI protocol and LUN concepts
- Basic knowledge of LLDP (Link Layer Discovery Protocol)
- Awareness of STP (Spanning Tree Protocol) and its impact on convergence

## References

- [T11 FC-BB-5: FCoE specification](https://www.t11.org/ftp/t11/pub/fc/bb-5/)
- [T11 FC-BB-6: FCoE enhancements](https://www.t11.org/ftp/t11/pub/fc/bb-6/)
- [IEEE 802.1Qbb: Priority-based Flow Control](https://standards.ieee.org/standard/802_1Qbb-2011.html)
- [IEEE 802.1Qaz: Enhanced Transmission Selection and DCBX](https://standards.ieee.org/standard/802_1Qaz-2011.html)
- [IEEE 802.1Qau: Congestion Notification](https://standards.ieee.org/standard/802_1Qau-2010.html)
- [Cisco Nexus 5000 FCoE Configuration Guide](https://www.cisco.com/c/en/us/td/docs/switches/datacenter/nexus5000/sw/fcoe/b_Cisco_Nexus_5000_Series_NX-OS_FCoE_Configuration_Guide.html)
- [Cisco FCoE Design and Deployment Guide](https://www.cisco.com/c/en/us/products/collateral/switches/nexus-5000-series-switches/white_paper_c11-495061.html)
- [Cisco UCS FCoE Design Guide](https://www.cisco.com/c/en/us/td/docs/unified_computing/ucs/sw/configuration/guide/6-1/b_UCSM_GUI_Configuration_Guide_6_1/b_UCSM_GUI_Configuration_Guide_6_1_chapter_01000.html)
- [RFC 7143: iSCSI Protocol (for comparison)](https://www.rfc-editor.org/rfc/rfc7143)
- [Data Center Bridging overview -- IEEE 802.1](https://1.ieee802.org/dcb/)
- [EMC/Dell FCoE Best Practices](https://www.dell.com/support/kbdoc/en-us/000176233/)
