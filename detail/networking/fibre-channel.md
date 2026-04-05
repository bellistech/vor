# Fibre Channel -- SAN Protocol Architecture and Operations

> Fibre Channel is a high-speed, lossless network technology designed for Storage Area Networks. It provides reliable, in-order delivery of raw SCSI block data at speeds from 1 to 64 Gbps, using a layered protocol stack (FC-0 through FC-4), credit-based flow control, and a dedicated fabric infrastructure that separates storage traffic from general-purpose networking. This document covers the full protocol architecture, frame internals, fabric services, zoning, VSANs, and operational procedures.

## 1. Protocol Layer Architecture

Fibre Channel uses a five-layer model that maps loosely to the OSI stack but is purpose-built for storage transport. Each layer has a distinct responsibility, and understanding the boundaries is essential for troubleshooting.

### 1.1 FC-0: Physical Layer

FC-0 defines the physical media, connectors, transceivers, and electrical/optical signaling characteristics.

**Transceiver types:**

| Form Factor | Description                                  |
|-------------|----------------------------------------------|
| SFP         | Small Form-factor Pluggable (up to 8G)       |
| SFP+        | Enhanced SFP (8G and 16G)                    |
| SFP28       | 28 Gbaud SFP for 32GFC                       |
| SFP56       | 56 Gbaud SFP for 64GFC                       |
| QSFP        | Quad SFP, 4 lanes (used for port-channels)   |

**Cable types:**

| Media                | Connector | Typical Use                  |
|----------------------|-----------|------------------------------|
| Multimode Fiber (MMF)| LC duplex | Short-range (up to 500m)     |
| Single-mode Fiber (SMF)| LC duplex| Long-range (up to 10km+)   |
| OM3 MMF (50/125)     | LC duplex | Standard datacenter (100m@16G)|
| OM4 MMF (50/125)     | LC duplex | Extended datacenter (150m@16G)|
| Active Optical Cable | Built-in  | Short runs, lower cost       |

**Speed evolution and encoding overhead:**

```
Generation   Line Rate    Data Rate    Encoding    Overhead
-----------  -----------  ----------   ----------  --------
1GFC         1.0625 Gbd   1.0625 Gbps  8b/10b      25%
2GFC         2.125 Gbd    2.125 Gbps   8b/10b      25%
4GFC         4.25 Gbd     4.25 Gbps    8b/10b      25%
8GFC         8.5 Gbd      8.5 Gbps     8b/10b      25%
16GFC        14.025 Gbd   14.025 Gbps  64b/66b     3.125%
32GFC        28.05 Gbd    28.05 Gbps   64b/66b     3.125%
64GFC        57.8 Gbd     57.8 Gbps    256b/257b   ~0.4%
```

Note the encoding efficiency jump at 16GFC. The 8b/10b encoding used by older speeds wastes 20% of the line rate (each 8 data bits require 10 transmitted bits). The 64b/66b encoding at 16G+ brings overhead below 4%, which is why 16GFC delivers nearly a full 16 Gbps of usable throughput.

### 1.2 FC-1: Encode/Decode Layer

FC-1 handles transmission encoding, word alignment, and ordered set recognition.

**Ordered sets** are special 4-byte transmission words used for link-level signaling:

| Ordered Set | Abbreviation | Purpose                            |
|-------------|--------------|-------------------------------------|
| Idle        | IDLE         | Fill between frames                 |
| R_RDY       | R_RDY        | Receiver Ready (BB credit return)   |
| ARB(x)      | ARB          | Arbitration request (FC-AL only)    |
| OPN(y)      | OPN          | Open connection (FC-AL only)        |
| CLS         | CLS          | Close connection (FC-AL only)       |
| LIP         | LIP          | Loop Initialization Primitive       |
| NOS         | NOS          | Not Operational Sequence            |
| OLS         | OLS          | Offline Sequence                    |
| LR          | LR           | Link Reset                          |
| LRR         | LRR          | Link Reset Response                 |

**Word synchronization:** The receiver must lock onto the 40-bit (8b/10b) or 66-bit (64b/66b) word boundaries in the incoming bit stream. Comma characters (K28.5 in 8b/10b) provide unique bit patterns that cannot appear in normal data, serving as synchronization anchors.

### 1.3 FC-2: Framing and Flow Control

FC-2 is the core of the protocol. It defines frame structure, sequences, exchanges, classes of service, and flow control mechanisms.

**Hierarchy of data organization:**

```
Exchange
  +-- Sequence 1
  |     +-- Frame 1
  |     +-- Frame 2
  |     +-- Frame N
  +-- Sequence 2
  |     +-- Frame 1
  |     +-- Frame N
  +-- ...
```

- **Frame:** The basic unit of transport (up to 2148 bytes total)
- **Sequence:** A group of related frames flowing in one direction (e.g., a SCSI data phase)
- **Exchange:** A bidirectional conversation (e.g., a complete SCSI command-response cycle)

**Classes of Service:**

| Class   | Delivery    | Flow Control   | Usage                          |
|---------|-------------|----------------|--------------------------------|
| Class 1 | Dedicated   | EE credits     | Obsolete; guaranteed bandwidth |
| Class 2 | Multiplexed | EE + BB credits| Acknowledged, rarely used      |
| Class 3 | Multiplexed | BB credits only| Standard for storage (no ACK)  |
| Class F | Fabric      | BB credits     | Inter-switch control traffic   |

Class 3 is overwhelmingly dominant in modern SANs. Reliability is achieved through BB credit flow control (preventing buffer overrun) and upper-layer protocol retransmission (SCSI error recovery).

### 1.4 FC-3: Common Services

FC-3 provides services common across multiple N_Ports on a node:

- **Striping:** Distributing data across multiple N_Ports for bandwidth aggregation
- **Hunt Groups:** Load balancing incoming requests across multiple N_Ports
- **Multicast/Broadcast:** Delivering frames to multiple destinations

In practice, FC-3 is a thin layer. Most of its theoretical functions are implemented at FC-4 or within the fabric itself.

### 1.5 FC-4: Protocol Mapping

FC-4 maps upper-layer protocols onto the FC transport:

| FC-4 Protocol | Description                                      |
|---------------|--------------------------------------------------|
| FCP (SCSI)    | Fibre Channel Protocol for SCSI; dominant use case|
| IPFC          | IP over Fibre Channel (RFC 2625 / 4338)          |
| FC-NVMe       | NVMe over Fabrics using FC transport              |
| FICON         | IBM mainframe channel protocol                    |
| FC-SB-5       | Single Byte command (mainframe peripheral)        |

**FCP (SCSI over FC) exchange flow:**

```
Initiator                              Target
    |                                      |
    |---FCP_CMND (SCSI CDB)------------->|
    |                                      |
    |<--FCP_XFER_RDY (for writes)---------|  (target ready for data)
    |---FCP_DATA (write payload)--------->|
    |         or                           |
    |<--FCP_DATA (read payload)-----------|
    |                                      |
    |<--FCP_RSP (SCSI status)-------------|
    |                                      |
```

## 2. FC Frame Anatomy

Understanding the frame structure is critical for protocol analysis and troubleshooting with FC analyzers.

### 2.1 Complete Frame Layout

```
+--------+-------------------+-------------------+--------+--------+
|  SOF   |   Frame Header    |     Payload       |  CRC   |  EOF   |
| 4 bytes|   24 bytes        |   0-2112 bytes    | 4 bytes| 4 bytes|
+--------+-------------------+-------------------+--------+--------+
         |<-------- CRC coverage ---------------->|
```

Total frame size: 36 to 2148 bytes (excluding SOF/EOF which are encoded as ordered sets, not data bytes on the wire).

### 2.2 Frame Header Fields

```
Byte:  0       1       2       3
      +-------+-------+-------+-------+
  0   | R_CTL |        D_ID           |  Routing Control + Destination ID
      +-------+-------+-------+-------+
  4   | CS_CTL|        S_ID           |  Class Control + Source ID
      +-------+-------+-------+-------+
  8   | TYPE  |        F_CTL          |  Data type + Frame Control
      +-------+-------+-------+-------+
 12   |SEQ_ID | DF_CTL|   SEQ_CNT     |  Sequence tracking
      +-------+-------+-------+-------+
 16   |     OX_ID     |     RX_ID     |  Exchange originator/responder IDs
      +-------+-------+-------+-------+
 20   |          Parameter            |  Relative offset or other param
      +-------+-------+-------+-------+
```

**Key header field details:**

| Field    | Size  | Description                                          |
|----------|-------|------------------------------------------------------|
| R_CTL    | 1B    | Routing control: data vs. link vs. extended link      |
| D_ID     | 3B    | Destination FCID (24-bit)                            |
| CS_CTL   | 1B    | Class-specific control / priority                    |
| S_ID     | 3B    | Source FCID (24-bit)                                 |
| TYPE     | 1B    | Upper protocol type (0x08 = FCP-SCSI, 0x05 = IP)    |
| F_CTL    | 3B    | Frame control bits (first/last/sequence initiative)  |
| SEQ_ID   | 1B    | Sequence identifier within the exchange              |
| DF_CTL   | 1B    | Data field control (optional headers present)        |
| SEQ_CNT  | 2B    | Frame sequence count (ordering within sequence)      |
| OX_ID    | 2B    | Originator exchange ID (initiator assigned)          |
| RX_ID    | 2B    | Responder exchange ID (target assigned)              |
| Parameter| 4B    | Relative offset for data frames                     |

**F_CTL bit meanings (selected):**

| Bit Position | Name              | Meaning when set                    |
|-------------|-------------------|--------------------------------------|
| 23          | Exchange Context  | 0 = originator, 1 = responder        |
| 22          | Sequence Context  | 0 = initiator of sequence            |
| 21          | First Sequence    | First sequence of exchange           |
| 20          | Last Sequence     | Last sequence of exchange            |
| 19          | End Sequence      | Last frame in sequence               |
| 16          | Sequence Initiative| Transfer initiative to other party  |

### 2.3 SOF and EOF Delimiters

SOF and EOF are transmitted as ordered sets (not payload data) and define frame boundaries and class of service.

| SOF Delimiter | Class | Usage                                |
|---------------|-------|--------------------------------------|
| SOFi2         | 2     | Initiate Class 2 exchange            |
| SOFn2         | 2     | Normal Class 2 (continuing exchange) |
| SOFi3         | 3     | Initiate Class 3 exchange            |
| SOFn3         | 3     | Normal Class 3 (continuing exchange) |
| SOFf          | F     | Fabric (inter-switch)                |

| EOF Delimiter | Meaning                                     |
|---------------|---------------------------------------------|
| EOFn          | Normal termination (frame valid)             |
| EOFt          | Terminate (last frame of sequence)           |
| EOFa          | Abort (frame should be discarded)            |
| EOFni         | Normal Invalid (CRC bad, for error handling) |

## 3. Addressing Deep Dive

### 3.1 FCID Architecture

The 24-bit FCID is a hierarchical address that encodes the topology path:

```
         FCID: 0x0A0301
         +----+----+----+
         | 0A | 03 | 01 |
         +----+----+----+
           |     |    |
           |     |    +-- Port 01 on this linecard/ASIC
           |     +------- Area 03 (linecard / port group)
           +------------- Domain 0A (Switch 10)
```

**Domain assignment process:**

1. Switches elect a Principal Switch using Principal Switch Selection (PSS) protocol
2. Principal Switch assigns itself Domain 1
3. Other switches request domain IDs via Build Fabric (BF) frames
4. Principal Switch assigns unique domain IDs (1-239)
5. Domain ID persists across reboots if configured statically

**Well-known FCIDs (fabric services):**

| FCID       | Service                                     |
|------------|---------------------------------------------|
| 0xFFFFFE   | Fabric Login Server (FLOGI target)          |
| 0xFFFFFC   | Directory Server / Name Server              |
| 0xFFFFFD   | Fabric Controller (RSCNs, BF)               |
| 0xFFFFF8   | Management Server (FDMI)                    |
| 0xFFFFF6   | Security Key Distribution                   |
| 0xFFFFFB   | Multicast Server                            |
| 0xFFFFFA   | Quality of Service Facilitator              |

### 3.2 WWN Format Details

The first nibble (4 bits) of a WWN indicates its NAA (Network Address Authority) type:

| NAA | Format                                              |
|-----|-----------------------------------------------------|
| 1   | IEEE 802.1a standard                                |
| 2   | IEEE extended                                       |
| 5   | IEEE registered (most common)                       |
| 6   | IEEE registered extended                            |

```
NAA 5 format (most common):

  5x:xx:xx:xx:xx:xx:xx:xx
  |  |           |
  |  +-OUI (24 bits, IEEE vendor ID)
  |              |
  |              +-- Vendor-specific (36 bits)
  +-- NAA type (4 bits)

Example breakdown:
  WWPN: 50:06:01:60:c7:e0:00:1a
  NAA:  5 (IEEE Registered)
  OUI:  006016 -> EMC/Dell
  VSID: 0c7e0001a -> vendor serial
```

**WWNN vs WWPN relationships:**

```
Physical HBA Card (WWNN: 20:00:00:e0:8b:05:05:00)
  |
  +-- Port 0 (WWPN: 21:00:00:e0:8b:05:05:04)
  +-- Port 1 (WWPN: 21:01:00:e0:8b:05:05:04)

With NPIV (virtual ports):
  Port 0 Physical WWPN: 21:00:00:e0:8b:05:05:04
    +-- Virtual WWPN 1: c0:03:ff:e0:8b:05:05:04 (VM1)
    +-- Virtual WWPN 2: c0:03:ff:e0:8b:05:05:05 (VM2)
```

## 4. Login Process Deep Dive

The FC login process establishes communication parameters at three levels: fabric, port, and process.

### 4.1 FLOGI (Fabric Login)

```
Step 1: N_Port sends FLOGI to well-known address 0xFFFFFE

FLOGI Payload (116 bytes):
+------------------+
| Common Svc Params|  BB_Credit offered, supported classes,
|   (16 bytes)     |  max frame size, E_D_TOV, R_A_TOV
+------------------+
| Port Name (WWPN) |  Requesting port's WWPN
|   (8 bytes)      |
+------------------+
| Node Name (WWNN) |  Requesting node's WWNN
|   (8 bytes)      |
+------------------+
| Class 1/2/3 Svc  |  Per-class parameters
| Params (64 bytes)|
+------------------+
| Vendor Version   |  Optional vendor extensions
|   (16 bytes)     |
+------------------+

Step 2: Switch responds with FLOGI ACC
  - Assigns FCID to the N_Port
  - Returns fabric's BB_Credit, max frame size
  - BB_Credit negotiation: minimum of offered values wins
```

**Important FLOGI parameters:**

| Parameter     | Typical Value | Description                          |
|---------------|---------------|--------------------------------------|
| BB_Credit     | 16-500        | Buffer credits offered to peer       |
| Max Frame Size| 2112          | Maximum payload (almost always 2112) |
| E_D_TOV       | 2000 ms       | Error Detect Timeout Value           |
| R_A_TOV       | 10000 ms      | Resource Allocation Timeout Value    |
| BB_SC_N       | 0 (or 1-15)   | BB State Change Number (perf feature)|

### 4.2 PLOGI (Port Login)

After FLOGI establishes fabric presence, the initiator performs PLOGI to each target port:

```
Initiator (0x0A0201)                   Target (0x0B0301)
    |                                       |
    |---PLOGI (same format as FLOGI)------>|
    |   S_ID=0x0A0201, D_ID=0x0B0301      |
    |                                       |
    |<--PLOGI ACC-----------------------------|
    |   Negotiated: EE_Credit, max frame,   |
    |   supported classes                    |
    |                                       |
```

PLOGI establishes EE (end-to-end) credits and the operating parameters between the two ports.

### 4.3 PRLI (Process Login)

PRLI negotiates the upper-layer protocol parameters:

```
PRLI for FCP (SCSI):
+----------------------------+
| TYPE = 0x08 (SCSI-FCP)    |
| Originator Process Assoc.  |
| Responder Process Assoc.   |
| Service Parameters:        |
|   - Read XFER_RDY disabled |
|   - Write XFER_RDY disabled|
|   - Target function        |
|   - Initiator function     |
|   - Confirmed completion   |
+----------------------------+
```

The PRLI response identifies whether the remote port is an initiator, a target, or both. This information is critical for smart zoning and name server queries.

### 4.4 FDISC (Fabric Discover)

Used by NPIV to log in additional virtual N_Ports on the same physical port:

```
Physical HBA (after FLOGI, FCID 0x0A0201)
    |
    |---FDISC (virtual WWPN 1)---> Fabric
    |<--FDISC ACC (FCID 0x0A0202)----|
    |                                 |
    |---FDISC (virtual WWPN 2)---> Fabric
    |<--FDISC ACC (FCID 0x0A0203)----|
```

Each FDISC gets a unique FCID. The virtual WWPNs can be independently zoned, providing per-VM or per-application isolation.

## 5. Zoning Architecture

Zoning is the primary access-control mechanism in FC SANs. It restricts which N_Ports can communicate, similar to ACLs or firewall rules but enforced at the fabric level.

### 5.1 Zone Database Hierarchy

```
+--Fabric (VSAN 100)-----------------------------------+
|                                                       |
|  +--Active Zoneset (only one per VSAN)-------------+ |
|  |                                                  | |
|  |  +--Zone: ESX01_VNXA_SPA0---+                   | |
|  |  |  member: pwwn 21:00:...04|                   | |
|  |  |  member: pwwn 50:06:...1a|                   | |
|  |  +--------------------------+                   | |
|  |                                                  | |
|  |  +--Zone: ESX01_VNXA_SPB0---+                   | |
|  |  |  member: pwwn 21:00:...04|                   | |
|  |  |  member: pwwn 50:06:...2a|                   | |
|  |  +--------------------------+                   | |
|  |                                                  | |
|  +--------------------------------------------------+ |
|                                                       |
|  (Inactive zonesets also stored in zone database)     |
+-------------------------------------------------------+
```

### 5.2 Zoning Enforcement Mechanisms

**Soft zoning** restricts Name Server query responses. A host in Zone A will only see name server entries for other Zone A members. However, if a host already knows the FCID of a device outside its zone, frames will be delivered -- the hardware does not filter.

**Hard zoning** programs the switch ASICs to drop frames between ports not in the same zone. This is true hardware-level enforcement and cannot be bypassed by address manipulation.

**Comparison matrix:**

```
Feature          | Soft Zoning  | Hard Zoning  | Smart Zoning
-----------------+--------------+--------------+---------------
Enforcement      | Name Server  | ASIC/HW      | Both
Security         | Moderate     | High         | High
Granularity      | WWPN-based   | Port-based   | Role-aware
Port Move        | Transparent  | Requires rezone| Transparent
Spoofing Risk    | Yes          | No           | Reduced
Vendor Support   | Universal    | Universal    | Newer switches
```

### 5.3 Zoning Best Practices

**Single-initiator/single-target zoning** is the gold standard:

```
GOOD (one initiator, one target port per zone):

  Zone: ESX01_HBA0_VNX_SPA0
    init: 21:00:00:e0:8b:05:05:04
    target: 50:06:01:60:c7:e0:00:1a

  Zone: ESX01_HBA0_VNX_SPB0
    init: 21:00:00:e0:8b:05:05:04
    target: 50:06:01:60:c7:e0:00:2a

BAD (multiple initiators in same zone -- RSCN blast radius):

  Zone: ALL_HOSTS_TO_ARRAY
    init: 21:00:00:e0:8b:05:05:04
    init: 21:00:00:e0:8b:06:06:04
    init: 21:00:00:e0:8b:07:07:04
    target: 50:06:01:60:c7:e0:00:1a
```

The reason single-initiator zoning matters: when a target port goes offline, an RSCN is sent to all zone members. With large zones, every host gets an RSCN and re-scans the fabric, causing I/O pauses across the entire environment.

### 5.4 Smart Zoning Configuration

Smart zoning (Cisco MDS feature) adds role awareness to zone members:

```
! Define device aliases for readability
device-alias database
  device-alias name ESX01_HBA0 pwwn 21:00:00:e0:8b:05:05:04
  device-alias name ESX02_HBA0 pwwn 21:00:00:e0:8b:06:06:04
  device-alias name VNX_SPA0   pwwn 50:06:01:60:c7:e0:00:1a
  device-alias name VNX_SPA1   pwwn 50:06:01:60:c7:e0:00:1b

device-alias commit

! Smart zone with roles
zone name ESX_CLUSTER_VNX_SPA vsan 100
  member device-alias ESX01_HBA0 init
  member device-alias ESX02_HBA0 init
  member device-alias VNX_SPA0 target
  member device-alias VNX_SPA1 target

! The fabric suppresses init-to-init pairings automatically
! Only init->target and target->init paths are active
```

### 5.5 Zone Merge and ISL Activation

When two switches form an ISL, their zone databases for each VSAN must merge. Merge rules:

1. If one side has no active zoneset, it accepts the other's
2. If both have active zonesets with the same name and identical content, merge succeeds
3. If zonesets differ, the merge fails and the ISL is isolated for that VSAN

```
! Pre-check merge compatibility:
show zone merge-check vsan 100

! Force zone distribution after changes:
zoneset distribute full vsan 100

! Verify consistency:
show zone status vsan 100
! Check "Merge Status" and "Checksum" fields
```

## 6. VSANs (Virtual SANs)

VSANs provide logical fabric isolation on shared physical infrastructure, analogous to VLANs for Ethernet.

### 6.1 VSAN Architecture

```
Physical Switch Chassis
+----------------------------------------------------------+
|                                                          |
|  VSAN 100 (Production)          VSAN 200 (Development)   |
|  +----------------------+      +----------------------+  |
|  | Domain: 1            |      | Domain: 1            |  |
|  | Principal Switch: Yes|      | Principal Switch: Yes|  |
|  | Zones: PROD_ZONESET  |      | Zones: DEV_ZONESET   |  |
|  | Name Server: separate|      | Name Server: separate|  |
|  | RSCN: separate       |      | RSCN: separate       |  |
|  +----------------------+      +----------------------+  |
|                                                          |
|  Ports fc1/1-8 -> VSAN 100                               |
|  Ports fc1/9-16 -> VSAN 200                              |
|  Port fc1/17 -> Trunk (VSAN 100,200) ISL                 |
+----------------------------------------------------------+
```

Each VSAN maintains completely independent:
- Fabric services (Name Server, RSCN, Zone Server)
- Domain ID space (Domain 1 can exist in both VSAN 100 and 200)
- FSPF routing topology
- Login sessions

### 6.2 VSAN Configuration

```
! Create VSANs
vsan database
  vsan 100 name Production_SAN
  vsan 200 name Development_SAN
  vsan 100 interface fc1/1-8
  vsan 200 interface fc1/9-16

! Verify
show vsan
show vsan membership

! Suspend a VSAN (all ports go down)
vsan database
  vsan 200 suspend
```

### 6.3 VSAN Trunking

VSAN trunking carries multiple VSANs over a single ISL using VSAN tagging (similar to 802.1Q VLAN tagging):

```
Switch A                              Switch B
  fc1/17 (TE_Port) ----ISL---- fc1/17 (TE_Port)

  Trunk carries:
    VSAN 100 frames with VSAN tag
    VSAN 200 frames with VSAN tag
```

```
! Configure trunk on ISL ports
interface fc1/17
  switchport mode E
  switchport trunk mode on
  switchport trunk allowed vsan 100,200

! Verify
show interface fc1/17 trunk

! Output shows:
!   fc1/17 is trunking
!   Trunk vsans (allowed):  100,200
!   Trunk vsans (active):   100,200
!   Trunk vsans (up):       100,200
```

### 6.4 IVR (Inter-VSAN Routing)

IVR enables controlled traffic flow between VSANs without merging their fabrics:

```
VSAN 100                    IVR Switch                    VSAN 200
+---------+     +-----------------------------------+     +---------+
| Host A  |-----|  IVR creates virtual FCIDs for    |-----| Array B |
| 0x0A0201|     |  devices in the "other" VSAN.     |     | 0x010301|
+---------+     |                                   |     +---------+
                |  Host A appears in VSAN 200 as    |
                |  phantom FCID: 0x7F0001           |
                |  Array B appears in VSAN 100 as   |
                |  phantom FCID: 0x7F0002           |
                +-----------------------------------+
```

```
! Enable IVR
feature ivr

! Define IVR topology
ivr vsan-topology
  member vsan 100
  member vsan 200

! Create IVR zones
ivr zone name HOST_A_TO_ARRAY_B
  member pwwn 21:00:00:e0:8b:05:05:04 vsan 100
  member pwwn 50:06:01:60:c7:e0:00:1a vsan 200

ivr zoneset name IVR_PROD_SET
  member HOST_A_TO_ARRAY_B

ivr zoneset activate name IVR_PROD_SET

! Verify
show ivr zone
show ivr virtual-fcs-id
```

## 7. Flow Control Mechanisms

FC achieves lossless delivery through credit-based flow control at two levels. This is fundamentally different from TCP/IP, where packet drops are expected and retransmission handles loss.

### 7.1 Buffer-to-Buffer Credits (BB Credits)

BB credits regulate frame flow between two directly connected ports. Each port advertises a BB_Credit count during FLOGI, representing the number of frames it can buffer.

```
Port A (BB_Credit_offered = 4)     Port B (BB_Credit_offered = 8)
  |                                     |
  | A can send 8 frames before stalling |
  | B can send 4 frames before stalling |
  |                                     |
  |---Frame 1---> (A's tx_credits: 7)   |
  |---Frame 2---> (A's tx_credits: 6)   |
  |---Frame 3---> (A's tx_credits: 5)   |
  |<--R_RDY------ (A's tx_credits: 6)   |
  |---Frame 4---> (A's tx_credits: 5)   |
  |---Frame 5---> (A's tx_credits: 4)   |
  |<--R_RDY------ (A's tx_credits: 5)   |
  |<--R_RDY------ (A's tx_credits: 6)   |
  |                                     |
```

**Credit starvation** occurs when `tx_bb_credit = 0`. The port cannot send any more frames until it receives an R_RDY. This is the most common cause of FC performance problems.

**Monitoring BB credits:**

```
show interface fc1/1
  ! Look for:
  !   BB credit:  16
  !   Transmit BB_credit:  16
  !   Receive BB_credit:  16

show interface fc1/1 bbcredit
  ! Shows detailed credit state

show interface fc1/1 counters
  ! Look for:
  !   TxBBcredit_zero_count: 0    <-- should be 0!
  !   TxBBcredit_zero_dur:   0    <-- duration in ms, should be 0
```

### 7.2 BB Credit Calculation for Distance

For long-distance ISLs, the propagation delay means more frames are in flight, requiring more credits:

```
Formula:
  Credits_needed = (Link_Speed_bytes_per_sec * RTT_seconds) / Frame_Size_bytes

Propagation delay through fiber:
  Speed of light in fiber: ~200,000 km/s (2/3 of vacuum speed)
  RTT for distance D: (2 * D) / 200,000

Example 1: 16G link, 10 km distance
  RTT = (2 * 10) / 200,000 = 0.0001 sec (100 us)
  Speed = 16 Gbps = 2,000,000,000 bytes/sec
  Frame = 2148 bytes (max)
  Credits = (2,000,000,000 * 0.0001) / 2148 = 93.1 -> 94 credits

Example 2: 32G link, 100 km (metro dark fiber)
  RTT = (2 * 100) / 200,000 = 0.001 sec (1 ms)
  Speed = 32 Gbps = 4,000,000,000 bytes/sec
  Credits = (4,000,000,000 * 0.001) / 2148 = 1862 credits

! Configure extended BB credits (MDS):
interface fc1/1
  switchport fcrxbbcredit 94
```

### 7.3 End-to-End Credits (EE Credits)

EE credits flow between the original source and final destination N_Ports:

```
Initiator          Switch A       Switch B       Target
    |                 |              |              |
    |---Frame 1------>|--forward---->|--forward---->|
    |                 |              |              |
    |<---------ACK frame (EE credit return)--------|
    |                 |              |              |
```

EE credits use ACK frames (in Class 2) or are managed by the N_Ports' login parameters (in Class 3, where EE credits are typically not used because Class 3 is unacknowledged).

## 8. Name Server and RSCN

### 8.1 Fabric Name Server (dNS / FCNS)

The Name Server at 0xFFFFFC maintains a database of all logged-in ports. Each switch maintains its local Name Server entries and distributes them across the fabric.

**Name Server entry contents:**

| Field               | Description                              |
|---------------------|------------------------------------------|
| Port ID (FCID)      | 24-bit fabric address                    |
| Port Name (WWPN)    | 64-bit port world-wide name              |
| Node Name (WWNN)    | 64-bit node world-wide name              |
| Class of Service    | Supported classes (1, 2, 3, F)           |
| FC-4 Types          | Supported protocols (FCP, IPFC, etc.)    |
| Port Type           | N_Port, NL_Port, etc.                    |
| Fabric Port Name    | WWPN of the F_Port it connected to       |
| Symbolic Port Name  | Human-readable string (e.g., "vmhba2")   |
| Symbolic Node Name  | Human-readable string (e.g., "esx01.dc1")|

```
! Query name server:
show fcns database vsan 100

! Detailed output with symbolic names:
show fcns database detail vsan 100

! Sample output:
! VSAN 100:
! FCID        TYPE  PWWN                    (VENDOR)
! 0x0a0201    N     21:00:00:e0:8b:05:05:04 (QLogic)
!               SWWN: 20:01:54:7f:ee:1a:3b:80
!               port-symbolic-name: vmhba2
!               node-symbolic-name: esx01.prod.dc1
! 0x0b0301    N     50:06:01:60:c7:e0:00:1a (EMC)
!               SWWN: 20:01:54:7f:ee:2b:4c:90
```

### 8.2 RSCN (Registered State Change Notification)

RSCNs alert registered N_Ports about fabric topology changes. The Fabric Controller at 0xFFFFFD generates and distributes RSCNs.

**RSCN trigger events:**

| Event                        | RSCN Payload                        |
|------------------------------|-------------------------------------|
| New device FLOGI             | Affected FCID, port address format  |
| Device logout (LOGO)         | Affected FCID                       |
| Device link failure          | Affected FCID                       |
| Zone configuration change    | Domain address format               |
| Domain change (switch add/rm)| Domain address format               |

**RSCN payload format:**

```
+----------+--------------------+
| Page Len | Affected FCID      |
| (1 byte) | (3 bytes)          |
+----------+--------------------+
| Address Format:               |
|   00 = Port Address           |
|   01 = Area Address (wildcard)|
|   02 = Domain Address (wild)  |
|   03 = Fabric Address (wild)  |
+-------------------------------+
```

**RSCN optimization techniques:**

```
! Suppress RSCNs for domain changes (switch reboots):
rscn suppress domain-swchange vsan 100

! Enable multi-pid RSCN (bundle multiple changes in one frame):
rscn multi-pid vsan 100

! Monitor RSCN activity:
show rscn statistics vsan 100

! Clear statistics:
clear rscn statistics vsan 100
```

**RSCN storm scenario and mitigation:**

```
Problem: 50-host zone, one target bounces
  -> 50 RSCNs generated
  -> 50 hosts re-query name server
  -> 50 hosts re-scan SCSI bus
  -> I/O latency spike across entire fabric

Mitigation:
  1. Use single-initiator zoning (reduces RSCN blast radius)
  2. Enable RSCN suppression for non-critical events
  3. Tune host-side RSCN handling (timeout, queue depth)
  4. Use device-alias for stable naming
```

## 9. FSPF (Fabric Shortest Path First)

FSPF is the FC routing protocol, analogous to OSPF in IP networking. It computes shortest paths through the switched fabric.

```
Switch A (Domain 1)          Switch B (Domain 2)
    |                            |
    E_Port ---ISL (cost 100)--- E_Port
    |                            |
    E_Port ---ISL (cost 100)--- E_Port (ECMP path)
    |                            |
    E_Port                    E_Port
      \                        /
       \---ISL (cost 200)----/   Switch C (Domain 3)
```

- Link cost is inversely proportional to speed (higher speed = lower cost)
- Equal-cost multipath (ECMP) distributes flows across parallel ISLs
- Path selection uses exchange-based load balancing (all frames of one exchange take the same path)

```
! Show FSPF topology:
show fspf database vsan 100

! Show routes:
show fspf internal route vsan 100

! Modify link cost:
interface fc1/17
  fspf cost 500 vsan 100
```

## 10. Troubleshooting Workflow

### 10.1 Systematic Approach

```
1. Physical Layer Check
   show interface fc1/1           # Link state, speed, errors
   show interface fc1/1 counters  # CRC, encoding errors, link failures

2. Login Verification
   show flogi database vsan 100   # Is the device logged in?
   show fcns database vsan 100    # Is it in the name server?

3. Zoning Verification
   show zone active vsan 100      # Is the device in an active zone?
   show zone member pwwn 21:00:... vsan 100  # What zones contain this WWPN?

4. Path Verification
   show fspf database vsan 100    # Are ISLs up?
   show fcdomain domain-list vsan 100  # All domains reachable?

5. Flow Control Check
   show interface fc1/1 bbcredit  # BB credit state
   show interface fc1/1 counters  # Check TxBBcredit_zero_count

6. RSCN Check
   show rscn statistics vsan 100  # Excessive RSCNs?
```

### 10.2 Common Issues and Resolution

| Symptom                    | Likely Cause                | Check                           |
|----------------------------|-----------------------------|---------------------------------|
| Port stuck in initializing | Speed mismatch, bad SFP     | show interface, check SFP       |
| Device not in FLOGI DB     | Port offline, cable issue   | show interface, check cable     |
| Device in FLOGI but not NS | Zoning prevents NS entry    | show zone active, verify member |
| Path exists but slow I/O   | BB credit exhaustion        | show counters, check zero_count |
| Intermittent connectivity  | RSCN storms, unstable link  | show rscn stats, show logging   |
| Zone merge failure on ISL  | Mismatched zone databases   | show zone merge-check           |

## Prerequisites

- Understanding of storage concepts (LUNs, SCSI commands, block I/O)
- Familiarity with OSI model and general networking principles
- Access to FC switch management (Cisco MDS NX-OS or Brocade FOS)
- Knowledge of server HBA configuration (QLogic/Broadcom/Emulex drivers)
- Understanding of virtualization if using NPIV (VMware, KVM, Hyper-V)

## References

- [INCITS T11 Technical Committee -- FC Standards](https://www.t11.org/)
- [Cisco MDS 9000 NX-OS Configuration Guide](https://www.cisco.com/c/en/us/support/storage-networking/mds-9000-nx-os-san-os-software/products-installation-and-configuration-guides-list.html)
- [Brocade Fabric OS Administration Guide](https://www.broadcom.com/products/fibre-channel-networking)
- [Fibre Channel Industry Association](https://fibrechannel.org/)
- [RFC 4338 -- Transmission of IPv6, IPv4, and ARP over FC](https://www.rfc-editor.org/rfc/rfc4338)
- [RFC 4625 -- FC MIB](https://www.rfc-editor.org/rfc/rfc4625)
- [FC-FS-5: Fibre Channel Framing and Signaling](https://www.t11.org/ftp/t11/pub/fc/fs-5/)
- [FC-SW-7: Fibre Channel Switch Fabric](https://www.t11.org/ftp/t11/pub/fc/sw-7/)
- [FC-PI-7: Fibre Channel Physical Interface](https://www.t11.org/ftp/t11/pub/fc/pi-7/)
- Clark, Tom. "Designing Storage Area Networks." Addison-Wesley, 2003.
- Poelker, Christopher & Nikitin, Alex. "Storage Area Networks for Dummies." Wiley, 2009.
