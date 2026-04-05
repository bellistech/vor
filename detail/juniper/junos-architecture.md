# Junos OS Architecture -- Hardware Design, Forwarding Theory, and Vendor Comparison

> *Deep dive into the Junos control/forwarding plane split: RE and PFE hardware internals, microkernel design philosophy, FIB programming mechanics, exception traffic rate-limiting, and how Junos architecture compares to Cisco IOS-XR and Arista EOS.*

---

## 1. Routing Engine Hardware Architecture

### CPU and Memory

The Routing Engine is a general-purpose computing platform running the Junos OS
(modified FreeBSD). Its hardware resources determine control plane capacity:

| Component | Role | Typical Specs (MX Series) |
|:---|:---|:---|
| CPU | Runs all Junos daemons (rpd, mgd, chassisd, etc.) | Intel multi-core x86_64, 2-4 cores |
| DRAM | Stores routing table (RIB), config, process memory | 16-64 GB DDR4 |
| Storage | Junos OS image, logs, core dumps | Dual SSD/CF, 64-256 GB |
| Management Ethernet | Out-of-band management (em0/fxp0) | 1 GbE copper |
| Console/AUX | Serial console access | RS-232 |
| USB | Software image loading | USB 2.0/3.0 |

### Memory Breakdown

RE memory is partitioned across several consumers:

```
+-------------------------------------------------+
|              RE Memory Layout                    |
+-------------------------------------------------+
| Junos Kernel (FreeBSD)        |  ~512 MB - 1 GB |
| rpd (Routing Protocol Daemon) |  variable        |
|   - RIB (routing table)      |  scales with      |
|   - Policy evaluation cache   |  route count      |
|   - Protocol state (BGP, OSPF)|                   |
| mgd (Management Daemon)       |  ~128-256 MB     |
| chassisd (Chassis Daemon)     |  ~64-128 MB      |
| snmpd + mib2d                 |  ~64-128 MB      |
| Other daemons                 |  ~256-512 MB     |
| System buffers / cache        |  remainder        |
+-------------------------------------------------+
```

The dominant consumer is rpd. A full Internet routing table (1M+ IPv4 prefixes,
200k+ IPv6 prefixes) with multiple BGP peers can consume 4-8 GB of rpd memory.
Each additional BGP peer holding a full table adds roughly 1-2 GB due to per-peer
Adj-RIB-In storage.

### Dual RE Configuration

High-end platforms (MX240/480/960/10000, PTX Series) support two Routing Engines
in active/standby configuration:

- **RE0** (master): runs all active daemons, owns the configuration
- **RE1** (backup): receives synchronized state via GRES/NSR
- Hardware-level switchover takes 1-3 seconds
- PFE continues forwarding from its local FIB copy throughout switchover

---

## 2. Packet Forwarding Engine Hardware Architecture

### ASIC Design

The PFE is purpose-built forwarding silicon -- not a general-purpose CPU. Juniper
designs custom ASICs optimized for packet processing:

| ASIC Family | Platform | Capabilities |
|:---|:---|:---|
| Trio | MX Series | Programmable, inline services, deep buffering |
| Paradise | EX4300 | Fixed-pipeline, L2/L3 at line rate |
| Express | EX9200 | High-density, deep buffering |
| Memory Complex | All | Stores FIB, filter rules, counters |

### Trio ASIC Architecture (MX Series)

```
+----------------------------------------------------------+
|                    Trio Chipset                            |
|                                                           |
|  +------------------+    +----------------------------+   |
|  | Memory Complex   |    | Packet Processing Pipeline |   |
|  | - FIB (LPM tree) |    | - Ingress parsing          |   |
|  | - Firewall filters|   | - FIB lookup               |   |
|  | - Policers       |    | - Filter evaluation        |   |
|  | - Counters       |    | - QoS classification       |   |
|  | - CoS queues     |    | - Header rewrite           |   |
|  +------------------+    | - Egress scheduling        |   |
|                          +----------------------------+   |
|                                                           |
|  +------------------+    +----------------------------+   |
|  | Memory (RLDRAM)  |    | I/O (SerDes to interfaces) |   |
|  | - Packet buffer  |    | - 10G/40G/100G/400G lanes  |   |
|  | - Soberton cells |    +----------------------------+   |
|  +------------------+                                     |
+----------------------------------------------------------+
```

### PFE Memory Breakdown

| Memory Type | Purpose | Typical Size |
|:---|:---|:---|
| RLDRAM (packet buffer) | Stores packets during processing and queuing | 256 MB - 4 GB per ASIC |
| TCAM | Firewall filter / ACL lookups (exact + wildcard) | 8-64K entries |
| SRAM | FIB prefix storage (LPM trie), counters | 128-512 MB |
| On-chip SRAM | Pipeline registers, header scratch space | KB-range |

The FIB is stored as a Longest Prefix Match (LPM) trie in SRAM. The lookup
time is O(W) where W is the address width (32 for IPv4, 128 for IPv6), but
the ASIC pipeline is fully pipelined so one lookup completes every clock cycle
regardless of trie depth.

---

## 3. Microkernel vs Monolithic Network OS Design

### Monolithic Architecture (Legacy IOS)

In a monolithic network OS (e.g., classic Cisco IOS), all functions run in a
single memory space as a single process:

```
+-------------------------------------------+
|          Single Process / Image            |
|                                           |
|  Routing | CLI | Forwarding | SNMP | ...  |
|  All share one memory space               |
|  One bug can crash the entire system      |
+-------------------------------------------+
```

Problems with monolithic design:

- **No memory protection**: a buffer overflow in SNMP can corrupt BGP state
- **No process isolation**: one hung function blocks the entire system
- **Reboots required**: any software update requires full system restart
- **No graceful degradation**: a single bug crashes everything

### Junos Microkernel-Inspired Architecture

Junos is not a pure microkernel (it uses a modified FreeBSD monolithic kernel
underneath), but it applies microkernel *principles* at the daemon level:

```
+-------------------------------------------+
|  FreeBSD Kernel (hardware abstraction)    |
+-------------------------------------------+
|  rpd   | mgd  | chassisd | dcd | snmpd   |
|  Each daemon is a separate Unix process   |
|  Protected memory space per process       |
|  IPC via Unix domain sockets / shared mem |
+-------------------------------------------+
```

Benefits of the Junos approach:

| Property | Monolithic | Junos (Process-Separated) |
|:---|:---|:---|
| Memory protection | None | Per-process |
| Fault isolation | None -- crash = full outage | Single daemon restarts |
| Software upgrades | Full reboot | ISSU (in some cases) |
| Security | Any vuln = full control | Privilege separation |
| Debugging | Difficult (shared state) | Per-process core dumps |

### How Daemon Restart Works

When a Junos daemon crashes, the kernel detects the failure and restarts it
automatically. The PFE continues forwarding throughout because it operates
from its own FIB copy:

```
1. rpd crashes (e.g., malformed BGP update triggers bug)
2. Kernel detects rpd exit, logs core dump
3. Kernel restarts rpd (new process, fresh memory)
4. rpd re-reads configuration
5. rpd re-establishes routing protocol adjacencies
6. rpd recomputes RIB and pushes updated FIB to PFE
7. PFE was forwarding on stale-but-valid FIB the entire time
   --> Transit traffic was NOT interrupted
```

---

## 4. Forwarding Table Programming (FIB Push)

### From Route to Forwarding Entry

The path from routing protocol advertisement to actual hardware forwarding
involves several stages:

```
BGP/OSPF/IS-IS advertisement
        |
        v
  rpd receives update
        |
        v
  rpd runs route selection (best path algorithm)
        |
        v
  rpd installs best route in RIB (inet.0, inet6.0, etc.)
        |
        v
  rpd pushes active routes to kernel forwarding table (krt)
        |
        v
  Kernel serializes FIB entries
        |
        v
  Kernel sends FIB update over internal Ethernet link to PFE
        |
        v
  PFE ASIC installs entry in hardware LPM trie (SRAM)
        |
        v
  PFE sends acknowledgement back to kernel
        |
        v
  Route is now active in hardware forwarding path
```

### RIB vs FIB

The Routing Information Base (RIB) and Forwarding Information Base (FIB) serve
different purposes and contain different data:

| Property | RIB (Routing Table) | FIB (Forwarding Table) |
|:---|:---|:---|
| Location | RE (rpd memory) | PFE (ASIC SRAM) |
| Contents | All learned routes + attributes | Only active/best routes |
| Size | Millions of entries possible | Subset of RIB |
| Format | Rich (AS-path, communities, etc.) | Minimal (prefix, next-hop, interface) |
| Purpose | Route selection and policy | Packet forwarding |
| Command | `show route` | `show route forwarding-table` |

### FIB Push Mechanisms

Junos uses two FIB synchronization methods:

**Incremental updates** (normal operation):
- When a route changes, only the delta is sent to PFE
- Typical update latency: single-digit milliseconds
- PFE processes updates while continuing to forward at line rate

**Full synchronization** (after RE restart or PFE reset):
- Complete FIB is pushed from RE to PFE
- Time depends on table size: a full Internet table takes 10-30 seconds
- During sync, PFE forwards on whatever partial FIB it has

### Kernel Route Table (krt) Queue

The kernel maintains a queue of pending FIB updates to the PFE. You can monitor
this queue to detect synchronization delays:

```
show krt queue
show krt state
```

If the krt queue backs up, it means the PFE cannot install routes as fast as
rpd is computing them. This can happen during large-scale route churn (e.g.,
BGP session reset with a full-table peer).

---

## 5. Exception Traffic Rate-Limiting

### The DDoS Problem for Network Devices

Exception traffic (packets destined to the device itself) must traverse from
PFE to RE via the internal link. The RE is a general-purpose CPU with limited
packet processing capacity:

| Component | Packet Processing Rate |
|:---|:---|
| PFE (transit, hardware) | Millions to billions pps |
| Internal link bandwidth | 1-10 Gbps (platform-dependent) |
| RE CPU (exception, software) | Thousands to low millions pps |

This asymmetry means a relatively small volume of exception traffic can
overwhelm the RE while being trivial for the PFE to receive. This is the
fundamental reason lo0 protection is critical.

### DDoS Protection (Junos Built-In)

Starting with Junos 11.2, Juniper includes a built-in DDoS protection feature
that automatically rate-limits control plane traffic by protocol type:

```
# View default DDoS protection settings
show ddos-protection protocols

# Customize DDoS protection per protocol
set system ddos-protection protocols bgp aggregate bandwidth 5000
set system ddos-protection protocols bgp aggregate burst 5000

set system ddos-protection protocols ospf aggregate bandwidth 10000
set system ddos-protection protocols icmp aggregate bandwidth 2000

# View violations (traffic exceeding rate limits)
show ddos-protection protocols violations
show ddos-protection statistics
```

DDoS protection operates at two levels:

1. **PFE policer** (hardware): rate-limits exception traffic before it reaches
   the internal link, preventing internal link saturation
2. **RE policer** (software): second layer of defense at the RE input queue

### lo0 Policer Design

A well-designed lo0 filter combines both accept/deny rules and rate-limiting
(policers) to protect the RE:

```
# Define policers for different traffic types
set firewall policer BGP-POLICER if-exceeding bandwidth-limit 5m burst-size-limit 625k
set firewall policer BGP-POLICER then discard

set firewall policer ICMP-POLICER if-exceeding bandwidth-limit 1m burst-size-limit 125k
set firewall policer ICMP-POLICER then discard

set firewall policer SSH-POLICER if-exceeding bandwidth-limit 2m burst-size-limit 250k
set firewall policer SSH-POLICER then discard

# Apply policers within lo0 filter terms
set firewall filter PROTECT-RE term BGP from protocol tcp
set firewall filter PROTECT-RE term BGP from port bgp
set firewall filter PROTECT-RE term BGP then policer BGP-POLICER
set firewall filter PROTECT-RE term BGP then accept

set firewall filter PROTECT-RE term ICMP from protocol icmp
set firewall filter PROTECT-RE term ICMP then policer ICMP-POLICER
set firewall filter PROTECT-RE term ICMP then accept

set firewall filter PROTECT-RE term SSH from protocol tcp
set firewall filter PROTECT-RE term SSH from port ssh
set firewall filter PROTECT-RE term SSH from source-address 10.0.0.0/8
set firewall filter PROTECT-RE term SSH then policer SSH-POLICER
set firewall filter PROTECT-RE term SSH then accept

set firewall filter PROTECT-RE term DEFAULT then discard
set firewall filter PROTECT-RE term DEFAULT then count DROPPED
set firewall filter PROTECT-RE term DEFAULT then log

set interfaces lo0 unit 0 family inet filter input PROTECT-RE
```

### Policer Math

A token-bucket policer has two parameters:

- **Bandwidth limit** ($r$): sustained rate in bits per second
- **Burst size** ($B$): maximum burst in bytes

The burst size should be at least:

$$B \geq r \times \frac{1}{8} \times t_{interface}$$

where $t_{interface}$ is the interface serialization time for the largest
expected packet. A common rule of thumb is:

$$B = r \times 5\text{ms} \div 8 = \frac{r}{1600}$$

For a 5 Mbps BGP policer:

$$B = \frac{5{,}000{,}000}{1600} = 3125 \text{ bytes (minimum)}$$

In practice, Junos requires burst sizes in powers of 2 and has platform-specific
minimums (typically 125 KB or higher).

---

## 6. Comparison with Other Vendor Architectures

### Cisco IOS-XR

IOS-XR is Cisco's microkernel-based network OS, running on ASR 9000, NCS 5500,
and 8000 series platforms:

| Aspect | Junos OS | IOS-XR |
|:---|:---|:---|
| Kernel | Modified FreeBSD | QNX microkernel (legacy) / Linux (eXR) |
| Process model | Separate Unix processes | Separate processes with IPC |
| Forwarding plane | Custom ASICs (Trio, etc.) | Custom ASICs (Memory Complex, Silicon One) |
| FIB sync | Kernel-mediated push to PFE | FIB Manager pushes to line cards |
| Configuration | Candidate + commit model | Candidate + commit model |
| HA mechanism | GRES + NSR | NSR + ISSU |
| In-service upgrade | Unified ISSU (limited platforms) | ISSU with process restart |
| Multi-chassis | Virtual Chassis (limited) | nV cluster |

Key differences:

- IOS-XR uses a true microkernel (QNX) in legacy releases and Linux in modern
  eXR releases, while Junos uses a monolithic FreeBSD kernel with process-level
  separation
- Both achieve similar fault isolation goals through different mechanisms
- IOS-XR supports third-party applications via containers (AppMgr) more
  extensively than Junos historically has
- Both use commit-based configuration (IOS-XR adopted this concept years after
  Junos pioneered it)

### Arista EOS

EOS (Extensible Operating System) runs on Arista switches, primarily in data
center environments:

| Aspect | Junos OS | Arista EOS |
|:---|:---|:---|
| Kernel | Modified FreeBSD | Unmodified Linux |
| Process model | Separate Unix processes | Separate Linux processes (Sysdb-centric) |
| Forwarding plane | Custom ASICs | Merchant silicon (Memory Complex, Memory Complex2) |
| Central state | Distributed across daemons | Sysdb (centralized state database) |
| Programmability | Junos Automation (PyEZ, SLAX) | eAPI (JSON-RPC), Python native |
| Configuration | Candidate + commit | Running config (CLI-oriented) |
| HA mechanism | GRES + NSR | SSO (Stateful Switchover) |

Key differences:

- EOS runs on unmodified Linux, giving direct access to standard Linux tools
  (bash, tcpdump, Python) -- Junos restricts shell access
- EOS uses Sysdb, a centralized publish-subscribe state database that all
  daemons read from and write to, providing a single source of truth
- Junos daemons communicate via IPC (Unix sockets, shared memory) without
  a centralized state store
- EOS is built for merchant silicon (Broadcom Memory Complex/Memory Complex2), making it
  inherently data-center focused; Junos spans routers, switches, and firewalls
- Arista's advantage is programmability (Python on-box, eAPI); Juniper's
  advantage is routing protocol maturity and service provider feature depth

### Summary Comparison

```
+------------------+-------------------+-------------------+------------------+
|                  | Junos OS          | Cisco IOS-XR      | Arista EOS       |
+------------------+-------------------+-------------------+------------------+
| Base OS          | FreeBSD (modified)| QNX/Linux (eXR)   | Linux (unmod.)   |
| Process isolation| Yes (Unix procs)  | Yes (microkernel) | Yes (Linux procs)|
| Config model     | Candidate+commit  | Candidate+commit  | Running config   |
| ASIC type        | Custom (Trio)     | Custom (Si One)   | Merchant (Memory Complex) |
| Primary market   | SP + Enterprise   | SP + Large DC     | Data Center      |
| Programmability  | PyEZ, SLAX, JET   | Yang, gRPC, Apps  | eAPI, Python     |
| RE/PFE split     | Yes (dedicated)   | Yes (RSP/LC)      | Yes (Memory Complex/CPU)  |
+------------------+-------------------+-------------------+------------------+
```

---

## Prerequisites

- junos-architecture (cheat sheet)
- bgp, ospf, is-is (routing protocol fundamentals)
- ipv4, ipv6 (IP addressing and forwarding concepts)

## References

- Juniper Networks JNCIA-Junos Study Guide (Official Certification Guide)
- Junos OS Architecture Documentation (Juniper TechLibrary)
- "JUNOS Enterprise Routing" by Doug Marschke & Harry Reynolds (O'Reilly)
- "Day One: Junos for IOS Engineers" (Juniper Books)
- Cisco IOS-XR Architecture Documentation (cisco.com)
- Arista EOS Architecture White Paper (arista.com)
- RFC 6241 -- NETCONF Configuration Protocol
- Memory Complex Architecture Guide (memory-complex.com)
- "Network Programmability with YANG" by Benoit Claise et al. (Addison-Wesley)
