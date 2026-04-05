# JunOS Interfaces -- Deep Dive

> Beyond the cheat sheet: interface hierarchy internals, CoS queue mapping, error
> statistics interpretation, discovery protocols, troubleshooting methodology, and
> chassis slot numbering across Juniper platforms.

## Prerequisites

- Familiarity with JunOS CLI (operational and configuration modes).
- Understanding of basic interface naming (`type-FPC/PIC/Port.unit`).
- Comfort with `show interfaces` output and set-style configuration.
- Review the companion sheet: `sheets/juniper/junos-interfaces.md`.

## 1. Interface Hierarchy in the Junos Configuration Tree

The Junos configuration is a structured hierarchy. Interfaces live under the
`[edit interfaces]` stanza, but their behavior is influenced by configuration
scattered across multiple branches.

```
[edit]
+-- interfaces
|   +-- ge-0/0/0                          # physical interface
|   |   +-- description
|   |   +-- mtu
|   |   +-- speed / link-mode / duplex
|   |   +-- gigether-options
|   |   |   +-- 802.3ad ae0              # LAG membership
|   |   |   +-- no-auto-negotiation
|   |   +-- unit 0                        # logical interface
|   |       +-- family inet
|   |       |   +-- address 10.0.1.1/24
|   |       |   +-- filter
|   |       |       +-- input FILTER-NAME
|   |       |       +-- output FILTER-NAME
|   |       +-- family inet6
|   |       +-- family mpls
|   |       +-- vlan-id 100
|   +-- lo0
|   +-- ae0
|   +-- irb
|   +-- interface-range ACCESS-PORTS
|
+-- class-of-service
|   +-- interfaces ge-0/0/0              # CoS binding
|       +-- scheduler-map MAP-NAME
|       +-- unit 0
|           +-- classifiers ...
|           +-- rewrite-rules ...
|
+-- protocols
|   +-- ospf
|   |   +-- area 0
|   |       +-- interface ge-0/0/0.0
|   +-- lldp
|       +-- interface all
|
+-- vlans
|   +-- VLAN100
|       +-- vlan-id 100
|       +-- l3-interface irb.100
|
+-- chassis
    +-- aggregated-devices
        +-- ethernet
            +-- device-count 5
```

**Key points:**

- The physical interface holds hardware-level settings (speed, MTU, gigether-options).
- Each unit (logical interface) holds protocol families, addresses, and filters.
- CoS configuration binds to the interface from `[edit class-of-service]`, not from
  `[edit interfaces]` directly.
- Routing protocol participation is declared under `[edit protocols]`, referencing the
  logical interface name (e.g., `ge-0/0/0.0`).
- VLANs reference IRB interfaces via `l3-interface` under `[edit vlans]`.
- The `[edit chassis]` stanza controls aggregated device count, FPC/PIC hardware
  settings, and port channelization.

### Configuration Inheritance and Apply-Groups

When `apply-groups` is used, Junos merges the group template into the candidate
configuration at commit time. Inheritance order:

1. Explicit interface-level configuration (highest priority).
2. Apply-groups at the interface level.
3. Apply-groups at the top level (`[edit]`).
4. System defaults (lowest priority).

Use `show interfaces ge-0/0/0 | display inheritance` to see which values come
from groups versus explicit config.

## 2. CoS Queue Assignment per Interface

Junos maps traffic into forwarding classes, which are serviced by hardware queues
on each interface. The default mapping provides four queues, but platforms support
up to eight.

### Default Queue Mapping (4-Queue Model)

| Queue | Forwarding Class       | Typical Use             | Default Scheduler |
|-------|------------------------|-------------------------|-------------------|
| 0     | best-effort            | General data            | 95% bandwidth     |
| 1     | expedited-forwarding   | Voice / real-time       | Strict priority   |
| 2     | assured-forwarding     | Business-critical       | 5% bandwidth      |
| 3     | network-control        | Routing protocols       | 5% bandwidth      |

### Binding CoS to an Interface

```
# Assign a scheduler map to an interface
set class-of-service interfaces ge-0/0/0 scheduler-map MY-SCHEDULER

# Apply a DSCP classifier to incoming traffic on a unit
set class-of-service interfaces ge-0/0/0 unit 0 classifiers dscp MY-DSCP-MAP

# Apply rewrite rules for outbound DSCP marking
set class-of-service interfaces ge-0/0/0 unit 0 rewrite-rules dscp MY-REWRITE

# Verify queue statistics
show class-of-service interface ge-0/0/0
```

### Scheduler Map Example

```
set class-of-service schedulers BE-SCHED transmit-rate percent 60
set class-of-service schedulers BE-SCHED buffer-size percent 60
set class-of-service schedulers BE-SCHED priority low

set class-of-service schedulers EF-SCHED transmit-rate percent 20
set class-of-service schedulers EF-SCHED buffer-size percent 10
set class-of-service schedulers EF-SCHED priority strict-high

set class-of-service scheduler-maps MY-SCHEDULER forwarding-class best-effort scheduler BE-SCHED
set class-of-service scheduler-maps MY-SCHEDULER forwarding-class expedited-forwarding scheduler EF-SCHED
```

### Monitoring CoS per Interface

```
show class-of-service interface ge-0/0/0          # queue assignments
show interfaces queue ge-0/0/0                     # per-queue packet/byte counters
show class-of-service classifier                   # active classifiers
show class-of-service rewrite-rule                  # active rewrite rules
```

## 3. Interface Statistics Internals

The `show interfaces ge-0/0/0 extensive` command produces detailed counters. Here
is how to interpret the error fields.

### Input Errors

| Counter            | Meaning                                                    | Common Cause                          |
|--------------------|------------------------------------------------------------|---------------------------------------|
| Input errors       | Total of all input error sub-counters                      | (aggregate)                           |
| Runts              | Frames shorter than 64 bytes                               | Collisions, bad NIC, duplex mismatch  |
| Giants             | Frames larger than the configured MTU                      | MTU mismatch between neighbors        |
| CRC errors         | Frame check sequence failures                              | Bad cable, SFP, EMI, duplex mismatch  |
| Fifo errors        | Receive FIFO overflow (interface overwhelmed)              | Traffic burst exceeding port rate     |
| Framing errors     | Frames without proper start/end delimiters                 | Physical layer issue, bad cable       |
| Input discards     | Frames dropped due to policer, filter, or queue full       | Policer rate exceeded, CoS tail drop  |

### Output Errors

| Counter            | Meaning                                                    | Common Cause                          |
|--------------------|------------------------------------------------------------|---------------------------------------|
| Output errors      | Total of all output error sub-counters                     | (aggregate)                           |
| Carrier transitions| Link up/down flap count                                    | Unstable SFP, cable, or remote port   |
| Collisions         | Ethernet collisions (half-duplex)                          | Duplex mismatch, hub in path          |
| Output drops       | Frames dropped on egress (queue full)                      | Congestion, insufficient CoS config   |
| Aged packets       | Packets held too long in output queue                      | Severe congestion                     |

### Key Diagnostic Commands

```
# Full error counters
show interfaces ge-0/0/0 extensive | find "error|CRC|runt|giant"

# Per-second rate counters
monitor interface ge-0/0/0

# Clear counters for baselining
clear interfaces statistics ge-0/0/0

# Check optics for physical layer issues
show interfaces diagnostics optics ge-0/0/0
#   Look for: Rx power < -10 dBm, Tx power out of range, high temperature
```

### Error Triage Rules of Thumb

- **CRC + Runts increasing together**: Almost always duplex mismatch or bad cable.
- **Giants only**: MTU mismatch -- check both ends and any intermediate switches.
- **Input discards with zero CRC**: Policer or firewall filter dropping traffic (not a physical problem).
- **Carrier transitions climbing**: Flapping link -- check cable, SFP seating, remote device.
- **Fifo errors**: The interface is being overrun -- consider CoS or rate limiting upstream.

## 4. LLDP and CDP on Juniper Interfaces

### LLDP (Link Layer Discovery Protocol)

LLDP is the standard neighbor discovery protocol and is natively supported in Junos.

```
# Enable LLDP globally
set protocols lldp interface all

# Enable on specific interfaces only
set protocols lldp interface ge-0/0/0
set protocols lldp interface ge-0/0/1

# Disable on a specific interface
set protocols lldp interface ge-0/0/2 disable

# Set advertised management address
set protocols lldp management-address 10.0.1.1

# Configure LLDP timers
set protocols lldp advertisement-interval 30
set protocols lldp hold-multiplier 4
```

**Operational commands:**

```
show lldp                           # global LLDP status
show lldp neighbors                 # summary of all discovered neighbors
show lldp neighbors interface ge-0/0/0   # neighbor on a specific port
show lldp local-information         # what this device advertises
show lldp statistics                # LLDP PDU counters
```

**LLDP-MED** (Media Endpoint Discovery) extends LLDP for VoIP devices:

```
set protocols lldp-med interface all
```

### CDP (Cisco Discovery Protocol)

Junos supports CDP reception (and optionally transmission) for interoperability
with Cisco devices.

```
# Enable CDP (receive + transmit)
set protocols cdp interface ge-0/0/0

# Operational commands
show cdp neighbors
show cdp neighbors interface ge-0/0/0
```

CDP on Junos is limited compared to Cisco -- LLDP is preferred for Juniper-to-Juniper
environments.

## 5. Interface Troubleshooting Flowchart

Follow this systematic approach when an interface is not passing traffic.

```
START: Interface not working
  |
  v
[1] show interfaces terse
  |
  +-- Physical link = "down"
  |     |
  |     +-- Check: cable seated? SFP inserted? Remote end up?
  |     +-- Check: show interfaces diagnostics optics (Rx power?)
  |     +-- Check: "disable" in config? (show config interfaces ge-x/y/z)
  |     +-- Check: speed/duplex mismatch with remote end?
  |     +-- Check: show chassis alarms (hardware fault?)
  |     +-- Action: swap cable/SFP, verify remote config
  |
  +-- Physical = "up", Protocol = "down"
  |     |
  |     +-- Check: is a family configured on the unit?
  |     |     show interfaces ge-0/0/0.0 (look for "Protocol" lines)
  |     +-- Check: keepalive failure? (show interfaces ge-0/0/0 extensive)
  |     +-- Check: authentication failure? (PPP, 802.1X)
  |     +-- Action: add family inet/inet6, check L2 protocol config
  |
  +-- Physical = "up", Protocol = "up" (but no traffic)
        |
        +-- [2] Verify addressing
        |     show interfaces ge-0/0/0.0 (correct IP/mask?)
        |     ping <remote-ip> source <local-ip>
        |
        +-- [3] Check routing
        |     show route <destination>
        |     show route forwarding-table destination <ip>
        |
        +-- [4] Check firewall filters
        |     show firewall filter <name>
        |     show firewall log (if logging enabled)
        |
        +-- [5] Check ARP / ND
        |     show arp interface ge-0/0/0.0
        |     show ipv6 neighbors interface ge-0/0/0.0
        |
        +-- [6] Check interface errors
        |     show interfaces ge-0/0/0 extensive | find error
        |     (see Section 3 for error interpretation)
        |
        +-- [7] Check CoS / policer drops
              show class-of-service interface ge-0/0/0
              show policer
```

### Additional Troubleshooting Commands

```
# Test connectivity from a specific interface
ping 10.0.1.2 source 10.0.1.1 count 5 rapid

# Trace the forwarding path
traceroute 10.0.1.2 source 10.0.1.1

# Check for interface flapping
show log messages | match "ge-0/0/0.*LINK"

# Verify LACP on aggregated interfaces
show lacp interfaces ae0
show lacp statistics interfaces ae0

# Check spanning tree state (if L2)
show spanning-tree interface ge-0/0/0
```

## 6. Chassis Slot Numbering Across Platforms

The FPC/PIC/Port numbering maps differently to physical hardware depending on the
platform family.

### MX Series (Modular Routers)

```
Chassis layout (MX240/MX480/MX960):
  FPC = physical line card slot (0, 1, 2, ... up to 11 on MX960)
  PIC = sub-slot on the line card (MPC) -- typically 0-3
  Port = port on the PIC -- varies by MPC/MIC type

Example: xe-2/1/3
  FPC 2 = third line card slot
  PIC 1 = second MIC on that MPC
  Port 3 = fourth port on that MIC

Fixed-config (MX204, MX10003):
  FPC = 0 (always)
  PIC = 0 (always)
  Port = physical port number
```

### EX Series (Switches)

```
Standalone:
  FPC = 0 (always -- single chassis)
  PIC = 0 (fixed)
  Port = physical port number (0-47 typical)

  Example: ge-0/0/23 = port 24 on the switch

Virtual Chassis (VC):
  FPC = member ID in the VC (0, 1, 2, ...)
  PIC = 0
  Port = port number on that member

  Example: ge-2/0/15 = port 16 on VC member 2

EX4300/EX4600 uplink module:
  PIC = 1 (uplink module slot)
  Example: et-0/1/0 = first 40G uplink on the expansion module
```

### SRX Series (Firewalls)

```
Branch SRX (SRX300, SRX345, SRX380):
  FPC = 0 (always)
  PIC = 0 (fixed ports), PIC = 1 (expansion module, if present)
  Port = physical port number

  Example: ge-0/0/6 = seventh onboard port
  Example: xe-0/1/0 = first port on the expansion module

Data center SRX (SRX4600, SRX5400/5600/5800):
  FPC = slot number (modular IOC slots)
  PIC = sub-slot
  Port = port number

Cluster (HA):
  Interface naming adds node prefix in some contexts:
    node0: ge-0/0/0    node1: ge-5/0/0 (SRX5000 with offset)
    Redundant: reth0 (redundant Ethernet -- similar to ae but across cluster nodes)
```

### QFX Series (Data Center Switches)

```
QFX5100/5110/5120/5200/5210:
  FPC = 0 (fixed)
  PIC = 0 (front panel ports)
  Port = physical port number

  Example: et-0/0/48 = 49th port (first QSFP uplink on QFX5100-48T)

QFX10002/10008/10016 (modular):
  FPC = line card slot
  PIC = sub-slot on the line card
  Port = port number

Virtual Chassis:
  Same convention as EX -- FPC = member ID
```

### Quick Reference Table

| Platform     | FPC Meaning            | PIC Meaning             | Typical Port Range |
|--------------|------------------------|-------------------------|--------------------|
| MX240/480    | Line card slot (0-7)   | MIC on the MPC (0-3)   | 0-11 per MIC       |
| MX960        | Line card slot (0-11)  | MIC on the MPC (0-3)   | 0-11 per MIC       |
| MX204        | Always 0               | Always 0                | 0-3                |
| EX4300       | VC member ID (or 0)    | 0=fixed, 1=uplink mod  | 0-47               |
| SRX345       | Always 0               | 0=onboard, 1=expansion | 0-15               |
| SRX5600      | IOC slot               | Sub-slot                | Varies by IOC      |
| QFX5120      | VC member ID (or 0)    | Always 0                | 0-63               |
| QFX10008     | Line card slot (0-7)   | Sub-slot                | Varies by LC       |
