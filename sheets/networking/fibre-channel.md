# Fibre Channel (Storage Area Networking Protocol)

High-speed network technology primarily used for Storage Area Networks (SANs), providing lossless, ordered delivery of raw block data between servers and storage arrays at speeds from 1 to 64 Gbps.

## Protocol Layers

Fibre Channel defines five protocol layers (FC-0 through FC-4), similar to the OSI model but purpose-built for storage transport.

```
+----------------------------------------------+
|  FC-4   Upper Layer Protocols (SCSI, IP, NVMe)|
+----------------------------------------------+
|  FC-3   Common Services (multicast, hunt grp) |
+----------------------------------------------+
|  FC-2   Signaling / Framing & Flow Control    |
+----------------------------------------------+
|  FC-1   Encode/Decode (8b/10b, 64b/66b)      |
+----------------------------------------------+
|  FC-0   Physical (optics, cables, connectors) |
+----------------------------------------------+
```

| Layer | Name            | Function                                    |
|-------|-----------------|---------------------------------------------|
| FC-0  | Physical        | Media, transceivers, cables, signal levels   |
| FC-1  | Encode/Decode   | 8b/10b (up to 8G), 64b/66b (16G+), word sync|
| FC-2  | Framing/FC      | Frame structure, flow control, classes of svc|
| FC-3  | Common Services | Striping, hunt groups, multicast (mostly thin)|
| FC-4  | Protocol Mapping| Maps SCSI (FCP), IP (IPFC), NVMe-oF to FC   |

### FC-0 Speeds and Media

| Generation | Speed   | Line Rate  | Encoding  | Max Distance (MMF/SMF) |
|------------|---------|------------|-----------|------------------------|
| 1GFC       | 1 Gbps  | 1.0625 Gbd | 8b/10b   | 500m / 10km            |
| 2GFC       | 2 Gbps  | 2.125 Gbd  | 8b/10b   | 300m / 10km            |
| 4GFC       | 4 Gbps  | 4.25 Gbd   | 8b/10b   | 150m / 10km            |
| 8GFC       | 8 Gbps  | 8.5 Gbd    | 8b/10b   | 150m / 10km            |
| 16GFC      | 16 Gbps | 14.025 Gbd | 64b/66b  | 100m / 10km            |
| 32GFC      | 32 Gbps | 28.05 Gbd  | 64b/66b  | 100m / 10km            |
| 64GFC      | 64 Gbps | 57.8 Gbd   | 256b/257b| 100m / 10km            |

## Topologies

### Point-to-Point (FC-P2P)

Direct connection between two N_Ports. Simplest topology; used for direct-attached storage.

```
+--------+          +--------+
| Server |--N_Port--| Storage|
| (HBA)  |          | (Array)|
+--------+          +--------+
```

### Arbitrated Loop (FC-AL)

Shared-loop topology (up to 126 NL_Ports + 1 FL_Port). Legacy; rarely deployed new.

```
     +----NL_Port----+
     |               |
  NL_Port         NL_Port
     |               |
  NL_Port---FL_Port--+
             |
          (Switch)
```

- Devices share bandwidth (half-duplex)
- One device transmits at a time after winning arbitration
- Loop Initialization Primitive (LIP) resets the loop
- AL_PA (Arbitrated Loop Physical Address): 1-byte, 126 valid values

### Switched Fabric (FC-SW)

Full-duplex, non-blocking switch fabric. Standard for modern SANs.

```
+--------+     +---------+     +---------+     +--------+
| Server |--F--| Switch  |--E--| Switch  |--F--| Storage|
| N_Port |     | F_Port  |     | F_Port  |     | N_Port |
+--------+     +---------+     +---------+     +--------+
```

**Port types:**

| Port  | Description                                      |
|-------|--------------------------------------------------|
| N_Port| Node port (HBA on server or storage)             |
| F_Port| Fabric port (switch port connected to N_Port)    |
| E_Port| Expansion port (inter-switch link / ISL)         |
| FL_Port| Fabric Loop port (switch to FC-AL loop)         |
| NL_Port| Node Loop port (device on FC-AL)                |
| TE_Port| Trunking E_Port (ISL with VSAN trunking)        |
| NP_Port| Proxy N_Port (NPIV on NPV switch)               |

## Addressing

### FCID (Fibre Channel Identifier)

24-bit address assigned dynamically during fabric login.

```
+----------+----------+----------+
|  Domain  |   Area   |   Port   |
|  (8 bit) |  (8 bit) |  (8 bit) |
+----------+----------+----------+
  0xDD        0xAA       0xPP

Example: 0x0A0201
  Domain = 0x0A (10)  -- identifies the switch
  Area   = 0x02 (2)   -- identifies the line card/group
  Port   = 0x01 (1)   -- identifies the physical port
```

- Domain ID: unique per switch (1-239), assigned by principal switch
- Well-known addresses: 0xFFFFFE (Fabric Login), 0xFFFFFC (Name Server), 0xFFFFFD (Fabric Controller)

### WWN (World Wide Name)

64-bit globally unique identifier, analogous to MAC addresses.

| Type | Format         | Usage                         |
|------|----------------|-------------------------------|
| WWNN | 2x:xx:xx:xx:xx:xx:xx:xx | Identifies the node (HBA)   |
| WWPN | 2x:xx:xx:xx:xx:xx:xx:xx | Identifies the port on a node|

```
Example WWPN: 21:00:00:e0:8b:05:05:04
  NAA type (2) : format identifier
  21:00        : port identifier
  00:e0:8b     : OUI (vendor, e.g., QLogic)
  05:05:04     : vendor-assigned serial
```

## Login Process

```
Server HBA                  FC Switch              Storage
    |                           |                      |
    |---FLOGI (to 0xFFFFFE)--->|                      |
    |<--FLOGI ACC (FCID)-------|                      |
    |                           |                      |
    |---PLOGI (to target FCID)----------------------->|
    |<--PLOGI ACC---------------------------------------|
    |                           |                      |
    |---PRLI (FCP/SCSI)------------------------------>|
    |<--PRLI ACC---------------------------------------|
    |                           |                      |
    |===== SCSI I/O begins ===========================|
```

| Step  | Full Name                     | Purpose                            |
|-------|-------------------------------|------------------------------------|
| FLOGI | Fabric Login                  | Register with fabric, get FCID     |
| PLOGI | Port Login                    | Establish session with remote port  |
| PRLI  | Process Login                 | Negotiate upper-layer protocol (FCP)|
| FDISC | Fabric Discover               | NPIV additional virtual port login  |
| LOGO  | Logout                        | Graceful session teardown           |

## Zoning

Zoning controls which initiators can see which targets. Enforced by the fabric.

### Zoning Types

| Type          | Enforced At      | Key By        | Pros                    | Cons                     |
|---------------|------------------|---------------|-------------------------|--------------------------|
| Soft Zoning   | Name Server only | WWPN          | Flexible                | Bypassable with known FCID|
| Hard Zoning   | Hardware (ASIC)  | Port / Domain,Port | Very secure       | Less flexible            |
| WWN-based     | Name Server      | WWPN/WWNN     | Survives port moves     | Can be spoofed           |
| Port-based    | Hardware         | Switch/Port#  | Secure, cannot spoof    | Must rezone on port move |
| Smart Zoning  | Both             | Device-type aware | Reduces IT-IT zones | Requires newer firmware  |

### Zoning Configuration (Cisco MDS)

```
! Create VSAN
vsan database
  vsan 100 name PROD_SAN

! Create zone
zone name ESX01_ARRAY01 vsan 100
  member pwwn 21:00:00:e0:8b:05:05:04   ! ESX host HBA
  member pwwn 50:06:01:60:c7:e0:00:1a   ! Array port

! Create zoneset and add zone
zoneset name PROD_ZONESET vsan 100
  member ESX01_ARRAY01

! Activate
zoneset activate name PROD_ZONESET vsan 100

! Verify
show zone active vsan 100
show zoneset active vsan 100
```

### Smart Zoning

Automatically identifies initiator vs target roles to suppress unnecessary IT-IT zone members:

```
zone name ESX_CLUSTER_ARRAY01 vsan 100
  member pwwn 21:00:00:e0:8b:05:05:04 init   ! initiator
  member pwwn 21:00:00:e0:8b:05:05:05 init   ! initiator
  member pwwn 50:06:01:60:c7:e0:00:1a target ! target
  member pwwn 50:06:01:60:c7:e0:00:1b target ! target
```

## VSANs (Virtual SANs)

Logical segmentation of a physical SAN fabric, similar to VLANs in Ethernet.

```
+-------Physical Switch Fabric-------+
|                                     |
|  +--VSAN 100--+   +--VSAN 200--+   |
|  | Prod SAN   |   | Dev SAN    |   |
|  | Zone A     |   | Zone X     |   |
|  | Zone B     |   | Zone Y     |   |
|  +------------+   +------------+   |
|                                     |
+-------------------------------------+
```

### VSAN Trunking (Cisco MDS)

```
! Enable trunking on ISL ports
interface fc1/1
  switchport trunk mode on
  switchport trunk allowed vsan 100,200

! Verify
show interface fc1/1 trunk
show vsan membership
```

### IVR (Inter-VSAN Routing)

Routes traffic between VSANs without merging fabrics:

```
ivr zone name CROSS_VSAN_ZONE
  member pwwn 21:00:00:e0:8b:05:05:04 vsan 100
  member pwwn 50:06:01:60:c7:e0:00:1a vsan 200

ivr zoneset name IVR_SET
  member CROSS_VSAN_ZONE

ivr zoneset activate name IVR_SET
```

## Flow Control

### Buffer-to-Buffer Credits (BB Credits)

Hop-by-hop flow control between adjacent ports. Prevents frame loss.

```
Switch A                     Switch B
  |                              |
  |--Frame 1--> (BB_Credit - 1) |
  |--Frame 2--> (BB_Credit - 1) |
  |<--R_RDY--- (BB_Credit + 1)  |
  |<--R_RDY--- (BB_Credit + 1)  |
  |                              |
```

**BB credit calculation for long-distance links:**

```
BB_Credits = (Link_Speed_Bps * RTT_sec) / (Frame_Size_bits)

Example: 16 Gbps link, 10 km (RTT ~ 0.1 ms), 2112-byte frames
  = (16e9 * 0.0001) / (2112 * 8)
  = 1,600,000 / 16,896
  ~ 95 credits needed
```

### End-to-End Credits (EE Credits)

Flow control between source and destination N_Ports across the fabric:

- Managed by ACK frames
- Less commonly tuned than BB credits
- Ensures destination can accept frames

## FC Frame Structure

```
+------+----------------+---------+--------+------+
| SOF  | Frame Header   | Payload | CRC    | EOF  |
| 4B   | 24 bytes       | 0-2112B | 4B     | 4B   |
+------+----------------+---------+--------+------+

Frame Header (24 bytes):
+--------+--------+--------+--------+--------+--------+
| R_CTL  |  D_ID  | CS_CTL |  S_ID  |  TYPE  | F_CTL  |
| 1B     |  3B    | 1B     |  3B    | 1B     |  3B    |
+--------+--------+--------+--------+--------+--------+
| SEQ_ID | DF_CTL | SEQ_CNT| OX_ID  |  RX_ID | PARAM  |
| 1B     | 1B     | 2B     | 2B     |  2B    |  4B    |
+--------+--------+--------+--------+--------+--------+
```

**SOF/EOF Delimiters:**

| Delimiter | Meaning                          |
|-----------|----------------------------------|
| SOFi3     | Start of Frame Initiate, Class 3 |
| SOFn3     | Start of Frame Normal, Class 3   |
| EOFt      | End of Frame Terminate           |
| EOFn      | End of Frame Normal              |
| EOFa      | End of Frame Abort               |

## Name Server and RSCN

### Name Server (dNS / FCNS)

Fabric-internal directory at well-known address `0xFFFFFC`. Stores:

- FCID-to-WWPN/WWNN mappings
- Port type, FC-4 protocol support, class of service
- Symbolic names

### RSCN (Registered State Change Notification)

Fabric sends RSCNs when topology changes occur (device login/logout, zone changes).

```
Fabric Controller (0xFFFFFD)
    |
    |---RSCN (device 0x0A0201 removed)---> Host A
    |---RSCN (device 0x0A0201 removed)---> Host B
    |<--RSCN ACC-------------------------------- Host A
    |<--RSCN ACC-------------------------------- Host B
```

- Hosts re-query name server after RSCN
- Can cause I/O disruption if excessive
- Filtering: `rscn suppress-domain-swchange` on MDS

## Common Commands (Cisco MDS / Nexus)

### Fabric Discovery

```
show flogi database                    # Logged-in devices
show flogi database vsan 100           # FLOGI entries for VSAN 100
show fcns database                     # Name server entries
show fcns database detail              # Detailed NS with symbolic names
show fcdomain domain-list vsan 100     # Domain IDs in fabric
show topology                          # ISL topology map
```

### Zoning

```
show zone active vsan 100              # Active zone members
show zoneset active vsan 100           # Active zoneset
show zone status vsan 100              # Zone database status / checksum
show zone member vsan 100              # All zone members
zone merge-check vsan 100              # Pre-check ISL zone merge
```

### VSAN and Interfaces

```
show vsan                              # All VSANs
show vsan usage                        # VSAN ID allocation
show vsan membership                   # Port-to-VSAN mapping
show interface fc1/1                   # FC interface status
show interface fc1/1 counters          # Frame/error counters
show interface fc1/1 trunk             # Trunk status
show interface fc1/1 bbcredit          # BB credit status
show port-channel summary              # Port-channel (ISL bundles)
```

### Troubleshooting

```
show logging last 50                   # Recent logs
show fcs ie                            # Fabric Config Server
show fcdomain                          # Domain manager state
show rscn statistics                   # RSCN event counts
show analytics query "select all from fc-scsi.port" # FC analytics (if licensed)
show tech-support zone                 # Full zone tech-support dump
debug zone all                         # Zone debugging (use sparingly)
```

## NPIV and NPV

**NPIV (N_Port ID Virtualization):** Allows a single physical HBA to present multiple WWPNs/FCIDs. Essential for virtualization (each VM gets its own WWPN).

**NPV (N_Port Virtualizer):** Access-layer switch acts as an HBA proxy, reducing domain count in fabric.

```
+------+     +----------+     +---------+
| VMs  |-----|  NPV     |-----|  Core   |
| NPIV |     |  Switch  |     |  Switch |
| HBA  |     | (no domain)    | (domain)|
+------+     +----------+     +---------+
```

```
! Enable NPIV on core switch
feature npiv

! Enable NPV mode on access switch (requires reload)
feature npv
npv enable
```

## Tips

- Always use single-initiator/single-target zoning for production SANs to isolate failure domains
- Name zones descriptively: `HostName_HBA#_ArrayName_Port#` (e.g., `ESX01_HBA0_VNX5400_SPA0`)
- Keep peer zone databases identical across all switches -- use `zone merge-check` before ISL activation
- Monitor BB credit zero counts (`show interface counters`) -- non-zero values indicate congestion
- Use device-alias for human-readable WWPN mapping instead of raw hex in zone configs
- Back up zone databases before changes: `copy running-config startup-config` and `show zoneset active` to file
- Avoid FC-AL in new deployments; always use switched fabric
- For long-distance ISLs, calculate and configure BB credits explicitly
- RSCN storms can hammer hosts -- enable RSCN suppression features where appropriate
- Use port-channels for ISL redundancy and bandwidth aggregation (up to 16 member ports)

## See Also

- iscsi
- ethernet
- vlan
- sctp
- mpls

## References

- [INCITS T11 FC Standards](https://www.t11.org/)
- [Cisco MDS 9000 Configuration Guide](https://www.cisco.com/c/en/us/support/storage-networking/mds-9000-nx-os-san-os-software/products-installation-and-configuration-guides-list.html)
- [Brocade FOS Admin Guide](https://www.broadcom.com/products/fibre-channel-networking)
- [Fibre Channel Industry Association](https://fibrechannel.org/)
- [RFC 4338 - Transmission of IPv6 over FC](https://www.rfc-editor.org/rfc/rfc4338)
- [FC-FS-5 (Fibre Channel Framing & Signaling)](https://www.t11.org/ftp/t11/pub/fc/fs-5/)
