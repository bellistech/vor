# Junos OS Architecture (JNCIA-Junos)

Junos OS software architecture: FreeBSD-based modular network operating system with strict separation of control and forwarding planes, process isolation, and a single unified codebase across all Juniper platforms.

## Software Architecture

### FreeBSD Kernel Foundation

```
# Junos OS is built on a modified FreeBSD kernel
# Single codebase across all platforms (routers, switches, firewalls)
# Same Junos release runs on MX, EX, SRX, QFX, PTX series

# Check Junos version and kernel info
show version
show version detail    # includes FreeBSD kernel version
show system uptime
```

### Modular Process Architecture

```
# Every major function runs as a separate protected process
# Processes run in their own memory space — one crash does not affect others
# Key Junos daemons:

rpd     # Routing Protocol Daemon — runs BGP, OSPF, IS-IS, RIP, static routes
         # Builds the routing table (RIB) and pushes forwarding table (FIB) to PFE

chassisd # Chassis Daemon — monitors hardware: fans, power, temp, line cards
          # Manages alarms and environmental conditions

pfed    # Packet Forwarding Engine Daemon — manages PFE microcode and state
         # Interface between RE software and PFE hardware

dcd     # Device Control Daemon — manages physical/logical interface config
         # Handles speed, duplex, MTU, encapsulation settings

mgd     # Management Daemon — handles CLI and NETCONF/XML API sessions
         # Parses commit operations, manages candidate config

snmpd   # SNMP Daemon — handles SNMP polling and trap generation

mib2d   # MIB-II Daemon — collects interface stats for SNMP MIB-II

alarmd  # Alarm Daemon — manages system alarms (yellow/red)

craftd   # Craft Interface Daemon — manages front-panel LCD and LEDs

eventd  # Event Daemon — system logging and event processing
```

### Process Architecture Diagram

```
+-------------------------------------------------------------------+
|                     ROUTING ENGINE (RE)                             |
|                                                                     |
|  +--------+  +--------+  +------+  +------+  +-------+  +-------+ |
|  |  rpd   |  |chassisd|  | mgd  |  | dcd  |  | snmpd |  |alarmd | |
|  |routing |  |hardware|  | CLI/ |  |device|  | SNMP  |  |alarms | |
|  |protocol|  |monitor |  |NETCNF|  |contrl|  |agent  |  |       | |
|  +--------+  +--------+  +------+  +------+  +-------+  +-------+ |
|                                                                     |
|  +-------------------+    +-------------------------------------+  |
|  | Junos Kernel      |    | Routing Table (RIB)                 |  |
|  | (Modified FreeBSD)|    | Forwarding Table (FIB) — master copy|  |
|  +-------------------+    +-------------------------------------+  |
|                                                                     |
+------------------------------+--------------------------------------+
                               |
                    Internal Ethernet Link
                    (RE-to-PFE communication)
                               |
+------------------------------+--------------------------------------+
|                   PACKET FORWARDING ENGINE (PFE)                    |
|                                                                     |
|  +----------------------------+  +-------------------------------+  |
|  | Forwarding Table (FIB)     |  | ASIC-based Forwarding Logic   |  |
|  | (synchronized copy from RE)|  | Line-rate packet processing   |  |
|  +----------------------------+  +-------------------------------+  |
|                                                                     |
|  +----------+  +----------+  +----------+  +----------+            |
|  |Interface |  |Interface |  |Interface |  |Interface |            |
|  |  ge-0/0/0|  |  ge-0/0/1|  |  ge-0/0/2|  |  ge-0/0/3|            |
|  +----------+  +----------+  +----------+  +----------+            |
+-------------------------------------------------------------------+
```

### Viewing Running Processes

```
# Show all running Junos daemons
show system processes extensive

# Check specific daemon status
show system processes extensive | match rpd
show system processes extensive | match chassisd

# Check system core dumps (crashed processes)
show system core-dumps
```

## Control Plane vs Forwarding Plane

### Control Plane (Routing Engine)

```
# The RE handles all control plane functions:
# - Routing protocol processing (BGP, OSPF, IS-IS, RIP)
# - Route selection and routing table (RIB) management
# - Forwarding table (FIB) computation and push to PFE
# - CLI and management access (SSH, NETCONF, SNMP)
# - System logging and monitoring
# - Software upgrades and configuration management
# - Keepalive and hello protocol processing

# Show routing table (control plane view)
show route
show route protocol bgp
show route table inet.0 summary
```

### Forwarding Plane (Packet Forwarding Engine)

```
# The PFE handles all data plane functions:
# - Packet forwarding at line rate using ASICs
# - Forwarding table (FIB) lookups
# - Firewall filter (ACL) processing in hardware
# - QoS classification, queuing, scheduling
# - Packet sampling and flow accounting
# - Policer enforcement
# - Encapsulation/decapsulation

# Show forwarding table (PFE view — what the hardware uses)
show route forwarding-table
show route forwarding-table destination 10.0.0.0/24
show route forwarding-table family inet

# Show PFE statistics
show pfe statistics traffic
```

### Separation Benefits

```
# 1. Stability: PFE continues forwarding even if RE is busy or restarting
# 2. Security: management plane isolated from forwarding plane
# 3. Performance: RE load does not affect forwarding throughput
# 4. Maintenance: RE software can be upgraded without stopping forwarding
# 5. Scalability: PFE handles millions of pps without RE involvement
```

## Routing Engine (RE)

```
# The RE is the brain of the device — a general-purpose CPU running Junos OS
# Physically: separate board with CPU, memory, storage (CF/SSD)
# Runs all daemons, manages config, computes routes

# Dual RE support (redundancy)
show chassis routing-engine     # shows RE0 and RE1 status
show chassis routing-engine 0   # primary RE details
show chassis routing-engine 1   # backup RE details

# RE hardware info
show chassis hardware           # CPU, memory, storage
show chassis environment        # temperature, fans, power

# RE mastership
show route summary              # only master RE has active routes
request chassis routing-engine master switch   # manual switchover
```

## Packet Forwarding Engine (PFE)

```
# ASIC-based hardware forwarding engine
# Receives FIB from RE and performs all packet forwarding in hardware
# Handles transit traffic at line rate without RE involvement

# PFE types vary by platform:
# - Trio chipset (MX Series)
# - Paradise/Express (EX Series)
# - Custom ASICs per platform generation

# Show PFE hardware
show chassis fpc                 # Flexible PIC Concentrators (line cards)
show chassis fpc detail
show chassis pic                 # Physical Interface Cards
show chassis mic                 # Modular Interface Cards

# PFE forwarding table
show route forwarding-table
show route forwarding-table summary

# PFE must match RE routing table — if not, traffic may blackhole
# Compare: show route vs show route forwarding-table
```

## Transit Traffic Processing

```
# Transit traffic: packets passing THROUGH the device (not destined to it)
# Handled ENTIRELY by the PFE — never touches the Routing Engine
#
# Flow:
# 1. Packet arrives on ingress interface (PFE)
# 2. PFE performs forwarding table (FIB) lookup
# 3. PFE applies ingress firewall filters
# 4. PFE applies QoS classification
# 5. PFE determines egress interface from FIB
# 6. PFE applies egress firewall filters
# 7. PFE rewrites Layer 2 header (next-hop MAC)
# 8. PFE decrements TTL, recalculates checksum
# 9. PFE transmits packet on egress interface
#
# Key point: the RE is NOT involved — this is why Junos devices
# maintain line-rate forwarding even under heavy management load

# Ingress PFE --> FIB Lookup --> Egress PFE
#   (all in hardware, microsecond-level latency)

# Show transit traffic statistics
show interfaces ge-0/0/0 statistics
show interfaces ge-0/0/0 extensive | match "packets"
```

## Exception Traffic

```
# Exception traffic: packets destined TO the device itself
# Examples:
# - SSH/Telnet to the device management IP
# - BGP, OSPF, IS-IS control protocol packets
# - ICMP echo (ping) to the device
# - SNMP queries to the device
# - ARP requests for the device
# - TTL-expired packets (traceroute)
# - Packets needing fragmentation
# - Packets with IP options

# Flow:
# 1. Packet arrives on ingress interface (PFE)
# 2. PFE determines packet is destined to the device
# 3. PFE sends packet to RE via internal link
# 4. RE processes the packet (rpd, sshd, snmpd, etc.)
# 5. RE generates response
# 6. Response sent back down to PFE for transmission

# Why lo0 filters matter:
# - Exception traffic consumes RE CPU cycles
# - RE has limited processing capacity (not line-rate)
# - DDoS attacks can overwhelm the RE
# - lo0 is the loopback representing the RE itself
# - Firewall filter on lo0 protects the RE from unwanted traffic

# Protecting the RE with a lo0 filter
set firewall filter PROTECT-RE term ALLOW-BGP from protocol tcp
set firewall filter PROTECT-RE term ALLOW-BGP from port bgp
set firewall filter PROTECT-RE term ALLOW-BGP then accept

set firewall filter PROTECT-RE term ALLOW-OSPF from protocol ospf
set firewall filter PROTECT-RE term ALLOW-OSPF then accept

set firewall filter PROTECT-RE term ALLOW-SSH from source-address 10.0.0.0/8
set firewall filter PROTECT-RE term ALLOW-SSH from protocol tcp
set firewall filter PROTECT-RE term ALLOW-SSH from port ssh
set firewall filter PROTECT-RE term ALLOW-SSH then accept

set firewall filter PROTECT-RE term ALLOW-ICMP from protocol icmp
set firewall filter PROTECT-RE term ALLOW-ICMP from icmp-type echo-request
set firewall filter PROTECT-RE term ALLOW-ICMP then policer ICMP-LIMIT
set firewall filter PROTECT-RE term ALLOW-ICMP then accept

set firewall filter PROTECT-RE term DENY-ALL then discard
set firewall filter PROTECT-RE term DENY-ALL then count DENIED-TO-RE
set firewall filter PROTECT-RE term DENY-ALL then log

# Apply to lo0
set interfaces lo0 unit 0 family inet filter input PROTECT-RE
```

## RE-PFE Communication

```
# RE and PFE communicate over a dedicated internal Ethernet link
# This is a physical connection on the backplane (not a front-panel port)

# What travels over the internal link:
# - Forwarding table (FIB) updates from RE to PFE
# - Exception traffic from PFE to RE (packets destined to device)
# - Interface state notifications
# - Statistics and counters from PFE to RE

# Kernel synchronization:
# - rpd computes routes and installs them in the kernel routing table
# - Kernel pushes FIB entries to PFE via internal link
# - PFE acknowledges receipt — ensures consistency
# - If PFE forwarding table diverges from RE, traffic may be misrouted

# Verify synchronization
show route summary           # RE view
show route forwarding-table summary   # PFE view
# These should be consistent
```

## GRES and NSR

### Graceful Routing Engine Switchover (GRES)

```
# GRES: when primary RE fails, backup RE takes over seamlessly
# - Preserves interface state and forwarding table
# - PFE continues forwarding during switchover (no traffic loss)
# - Kernel state synchronized from primary to backup RE
# - Does NOT preserve routing protocol sessions (peers will notice)

# Enable GRES
set chassis redundancy graceful-switchover
set routing-options nonstop-forwarding   # PFE keeps forwarding during switch

# Verify GRES readiness
show system switchover
show chassis routing-engine   # both RE0 and RE1 must be present
```

### Nonstop Active Routing (NSR)

```
# NSR: extends GRES by also preserving routing protocol sessions
# - BGP, OSPF, IS-IS sessions maintained during RE switchover
# - Protocol state synchronized from primary to backup RE in real-time
# - Neighbors do NOT detect the switchover (no adjacency flaps)
# - More resource-intensive than GRES alone

# Enable NSR (requires GRES first)
set routing-options nonstop-routing

# Verify NSR state
show task replication
show bgp neighbor | match "NSR"

# GRES alone:  forwarding preserved, protocols restart
# GRES + NSR:  forwarding preserved, protocols preserved
```

## Tips

- Every Junos daemon runs as a separate protected process; a single daemon crash does not bring down the entire system -- this is a key JNCIA exam topic.
- Transit traffic (through the device) is handled entirely by the PFE in hardware; the RE is never involved in normal forwarding.
- Exception traffic (to the device) must traverse from PFE to RE via the internal link; always protect the RE with a lo0 firewall filter.
- GRES preserves forwarding but not routing protocol sessions; add NSR to also preserve BGP/OSPF/IS-IS adjacencies across RE switchover.
- The RE maintains the master copy of the routing table (RIB); the PFE holds a synchronized copy of the forwarding table (FIB).
- `show route` displays the RE routing table; `show route forwarding-table` displays the PFE forwarding table -- know the difference for the exam.
- On dual-RE systems, only the master RE runs routing protocols and manages configuration; the backup RE synchronizes state via GRES/NSR.
- Junos uses a single software release across all platforms (MX, EX, SRX, QFX); this is "one OS" philosophy and a common exam question.
- The internal Ethernet link between RE and PFE is a dedicated backplane connection, not a front-panel interface.

## See Also

- bgp, ospf, is-is

## References

- [Juniper JNCIA-Junos Study Guide](https://www.juniper.net/us/en/training/certification/tracks/junos/jncia-junos.html)
- [Junos OS Architecture Overview — Juniper TechLibrary](https://www.juniper.net/documentation/us/en/software/junos/junos-overview/topics/concept/junos-software-architecture.html)
- [Junos OS Routing Engine and Packet Forwarding Engine](https://www.juniper.net/documentation/us/en/software/junos/junos-overview/topics/concept/routing-engine-packet-forwarding-engine.html)
- [Junos OS Process Overview — TechLibrary](https://www.juniper.net/documentation/us/en/software/junos/junos-overview/topics/concept/junos-software-process-overview.html)
- [Protecting the Routing Engine — Juniper TechLibrary](https://www.juniper.net/documentation/us/en/software/junos/routing-policy/topics/example/firewall-filter-re-protect.html)
- [GRES and NSR — Juniper TechLibrary](https://www.juniper.net/documentation/us/en/software/junos/high-availability/topics/concept/gres-overview.html)
- Day One: Junos for IOS Engineers (Juniper Books)
