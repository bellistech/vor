# Cisco IOS XR -- Microkernel Network OS Architecture

> *IOS XR is a fundamentally different operating system from classic IOS. Built on a microkernel (QNX Neutrino, later Linux), it provides process isolation, memory protection, and independent process restart — capabilities that transform a network router from a monolithic single-point-of-failure into a resilient distributed system where control plane processes can crash and restart without interrupting packet forwarding.*

---

## 1. Microkernel Architecture (The Foundation)

### The Problem

Classic IOS runs as a single monolithic process. A bug in any subsystem — BGP parser, SNMP agent, CLI handler — can corrupt shared memory and crash the entire router. Recovery requires a full reload: 3-10 minutes of downtime during which all forwarding stops.

### The Microkernel Solution

IOS XR separates the system into three layers:

**Kernel layer** — The microkernel provides only:
- Process scheduling and dispatch
- Inter-process communication (IPC) via message passing
- Memory management with hardware-enforced protection
- Interrupt handling

Everything else — device drivers, protocol stacks, management agents, filesystem — runs in user space as independent processes.

**System infrastructure** — Shared services running as user-space processes:
- SysDB (System Database) — centralized configuration store
- CERRNO — error propagation framework
- Group Services — process group management and heartbeat
- Packet I/O — data plane interface

**Application processes** — Each routing protocol, management agent, and service runs as an isolated process:

```
PID   Process             Memory   State
---   -------             ------   -----
1001  bgp                 256MB    Running
1002  ospf                 64MB    Running
1003  isis                 48MB    Running
1004  mpls_ldp             32MB    Running
1005  netconf_agent        16MB    Running
1006  cli_agent            12MB    Running
```

### Process Isolation Guarantees

Each process runs in its own virtual address space. The microkernel enforces:

$$\text{Process}_A.\text{memory} \cap \text{Process}_B.\text{memory} = \emptyset$$

A buffer overflow in BGP cannot corrupt OSPF's data structures. A null pointer dereference in the SNMP agent does not affect packet forwarding. This isolation is the single most important architectural difference from classic IOS.

### QNX Neutrino vs. Linux (Classic XR vs. eXR)

IOS XR has two kernel variants:

| Property | Classic XR (QNX) | eXR (Linux) |
|:---|:---|:---|
| Kernel | QNX Neutrino 6.x | Wind River Linux |
| Platforms | CRS, ASR 9000 (early) | ASR 9000 (later), NCS 5500, NCS 540, XRd |
| Package format | PIE | RPM |
| Container support | No | Yes (LXC) |
| Third-party apps | Limited | Full Linux userspace |
| Install model | PIE add/activate | `install` with RPM |

The shift from QNX to Linux (eXR) preserved the process isolation model while gaining access to the Linux ecosystem: standard packaging (RPM/YUM), container isolation (LXC), and familiar tooling (bash, tcpdump, ip).

---

## 2. SysDB (The Configuration Database)

### The Problem

In a multi-process system, configuration cannot live inside a single process's memory. BGP configuration must be accessible to the BGP process, but also to the CLI for display, to NETCONF for programmatic access, and to the commit infrastructure for validation. A centralized, transactional configuration store is required.

### SysDB Architecture

SysDB is a hierarchical key-value database that serves as the single source of truth for all configuration and operational state:

```
SysDB Tree Structure:
/
+-- config/
|   +-- global/
|   |   +-- router/
|   |   |   +-- bgp/
|   |   |   |   +-- as/65000/
|   |   |   |   +-- neighbor/10.0.0.1/
|   |   |   +-- ospf/
|   |   |   +-- isis/
|   |   +-- interface/
|   |       +-- GigabitEthernet0_0_0_0/
|   +-- admin/
+-- oper/
    +-- bgp/
    +-- interface/
    +-- platform/
```

### Transaction Model

SysDB implements ACID transactions for configuration changes:

1. **Atomicity** — A commit either fully succeeds or fully fails. Partial configuration is never applied.
2. **Consistency** — Semantic validation occurs before any change is applied. Invalid configurations are rejected at commit time.
3. **Isolation** — Multiple users can edit configuration simultaneously in separate candidate configurations. Changes are merged at commit time.
4. **Durability** — Committed configuration survives process restarts and, after `install commit`, survives reloads.

### The Candidate Configuration

When a user enters `configure terminal`, IOS XR creates a candidate configuration — a copy-on-write fork of the running configuration. Changes are applied to the candidate only:

```
Running Config:   interface Gi0/0/0/0, ip 10.0.0.1/30
Candidate Config: interface Gi0/0/0/0, ip 10.0.0.5/30  (modified)

show configuration:  Shows the DIFF between running and candidate
commit:              Merges candidate into running atomically
abort:               Discards candidate entirely
```

This model eliminates the classic IOS problem where a syntax error mid-configuration leaves the router in a partially configured state.

### Configuration Sessions

Multiple configuration sessions can coexist:

| Session Type | Command | Behavior |
|:---|:---|:---|
| **Shared** | `configure terminal` | Multiple users can edit simultaneously; conflicts resolved at commit |
| **Exclusive** | `configure exclusive` | Locks configuration; other users cannot enter config mode |

When two shared sessions modify the same configuration element, the second commit may fail with a conflict error. The user must then refresh their candidate and retry.

---

## 3. Commit Model (Transactional Configuration)

### The Problem

Classic IOS applies each configuration command immediately and independently. There is no way to:
- Preview changes before applying
- Apply multiple changes atomically
- Roll back to a previous configuration state
- Automatically revert if a change causes connectivity loss

### Commit Semantics

The commit operation in IOS XR is the atomic transition from candidate to running:

$$\text{Running}_{n+1} = \text{merge}(\text{Running}_n, \Delta_{\text{candidate}})$$

Where $\Delta_{\text{candidate}}$ is the set of changes in the candidate configuration.

### Commit History and Rollback

Every commit creates a numbered checkpoint:

```
Commit ID    Timestamp                  User     Comment
---------    ---------                  ----     -------
1000000005   2026-04-01 14:30:00 UTC    admin    Added BGP peer
1000000004   2026-04-01 12:15:00 UTC    admin    Updated OSPF costs
1000000003   2026-03-28 09:00:00 UTC    noc      Interface config
1000000002   2026-03-25 16:45:00 UTC    admin    Initial VRF setup
1000000001   2026-03-20 08:00:00 UTC    admin    Base config
```

Rollback reconstructs the configuration state at any previous commit point:

$$\text{Running}_{\text{target}} = \text{Running}_0 + \sum_{i=1}^{\text{target}} \Delta_i$$

In practice, IOS XR stores complete configuration snapshots (not incremental deltas), so rollback is a direct replacement operation.

### Confirmed Commit (The Safety Net)

The confirmed commit is a two-phase commit with automatic rollback:

```
Phase 1: commit confirmed 300
  - Changes are applied to running config
  - A 300-second rollback timer starts
  - If the timer expires without confirmation, the config reverts

Phase 2: commit (confirmation)
  - Cancels the rollback timer
  - Makes the changes permanent
```

This is critical for remote management: if a configuration change breaks connectivity to the router, the confirmed commit automatically reverts the change after the timeout, restoring access.

### Commit Replace vs. Commit Merge

| Operation | Behavior | Risk |
|:---|:---|:---|
| `commit` (merge) | Applies candidate changes on top of running | Low — only modified elements change |
| `commit replace` | Replaces entire running config with candidate | High — any element not in candidate is removed |

`commit replace` is the IOS XR equivalent of "load override" — it implements a full declarative configuration model where the candidate represents the complete desired state.

---

## 4. Process Restart and High Availability

### The Problem

In carrier networks, downtime is measured in revenue loss per minute. The ability to recover from software faults without interrupting packet forwarding is a business requirement, not a luxury.

### Process Restart Hierarchy

IOS XR implements three levels of recovery:

**Level 1: Process Restart** — The faulted process is restarted by the process manager. Its state is reconstructed from SysDB (configuration) and peer processes (operational state). Forwarding continues using the existing FIB.

```
Example: BGP process crashes
1. Process manager detects BGP absence (heartbeat failure)
2. BGP process is restarted (new PID, fresh memory)
3. BGP reads its configuration from SysDB
4. BGP initiates graceful restart with neighbors (RFC 4724)
5. Neighbors maintain routes during restart timer
6. BGP re-converges without forwarding interruption
```

Recovery time: 5-30 seconds (process-dependent).

**Level 2: RSP Switchover** — If the entire RSP (Route Switch Processor) fails, the standby RSP takes over. This is similar to classic IOS SSO (Stateful Switchover) but enhanced by the process model:

```
Active RSP failure detected
  |
  v
Standby RSP becomes active
  |
  v
All processes re-register with new active RSP
  |
  v
Forwarding continues (FIB on line cards unaffected)
```

Recovery time: 1-5 seconds for switchover, 30-120 seconds for full protocol convergence.

**Level 3: Full Reload** — Only required for kernel faults or hardware failures. The entire system reboots.

### Non-Stop Routing (NSR)

NSR maintains routing protocol state across RSP switchovers by synchronizing protocol state from active to standby RSP in real-time:

```
Active RSP                    Standby RSP
+-----------+                 +-----------+
| BGP state | ----sync------> | BGP state |
| OSPF state| ----sync------> | OSPF state|
| IS-IS     | ----sync------> | IS-IS     |
| LDP       | ----sync------> | LDP       |
+-----------+                 +-----------+
```

With NSR, an RSP switchover is invisible to routing neighbors — no graceful restart is needed because the standby RSP already has the complete routing state.

### Non-Stop Forwarding (NSF)

NSF ensures the data plane continues forwarding packets during control plane recovery. The FIB on line cards is preserved:

$$\text{FIB}_{\text{during\_restart}} = \text{FIB}_{\text{before\_restart}}$$

Stale FIB entries are marked and purged only after the control plane has fully reconverged and pushed updated FIB entries.

---

## 5. Install and ISSU (Software Lifecycle)

### The Problem

Classic IOS uses monolithic images: upgrading any component requires replacing the entire image and reloading. This means every security patch, bug fix, or feature addition causes a full outage.

### Package-Based Software Model

IOS XR decomposes the software into packages:

```
Base Package (required):
  - Kernel, SysDB, process manager, CLI, basic forwarding

Optional Packages:
  - routing (BGP, OSPF, IS-IS, etc.)
  - mpls (LDP, RSVP-TE, L2VPN, L3VPN)
  - multicast (PIM, IGMP, MSDP)
  - security (ACL extensions, IPsec, MACsec)
  - management (NETCONF, gRPC, telemetry)
  - k9 (crypto)
```

Each package can be independently installed, activated, deactivated, and removed.

### Install Lifecycle

```
Repository               Active Set              Committed Set
(available)              (running)                (survives reload)
    |                        |                         |
    |-- install add -------->|                         |
    |                        |-- install activate ---->|
    |                        |                         |
    |                        |   (running but not      |
    |                        |    reload-safe)         |
    |                        |                         |
    |                        |-- install commit ------>|
    |                        |                         |
    |                        |   (permanent)           |
```

The three-stage model (add, activate, commit) provides safety:

1. **Add** — Downloads and extracts the package. No impact on running system.
2. **Activate** — Loads the package into the running system. Processes may restart. Forwarding impact depends on the package.
3. **Commit** — Makes the activation persistent across reloads. Without commit, a reload reverts to the previously committed software.

### SMU (Software Maintenance Update)

SMUs are targeted patches for specific bugs. They modify individual packages without replacing them:

```
SMU lifecycle:
  install add smu-patch.rpm       -- add to repository
  install activate smu-patch      -- apply (may restart affected process)
  install commit                  -- make persistent

SMU properties:
  - Fixes exactly one defect (one Cisco bug ID)
  - Minimal footprint (patches only affected binaries)
  - Can be removed without full image reinstall
  - Dependencies are tracked and enforced
```

### ISSU (In-Service Software Upgrade)

ISSU upgrades the entire software image without forwarding interruption on dual-RSP platforms:

```
Phase 1: Prepare
  - New image loaded on standby RSP
  - Standby RSP reboots with new image
  - Version mismatch is tolerated between RSPs

Phase 2: Activate
  - Switchover: standby (new image) becomes active
  - Former active RSP reboots with new image
  - Both RSPs now run new image

Phase 3: Commit
  - New image is committed as boot default
```

ISSU requires:
- Dual RSPs (active + standby)
- NSR enabled for all routing protocols
- Source and target images must be ISSU-compatible
- Line card firmware compatibility

---

## 6. RSP and Line Card Separation

### The Problem

On modular platforms, the control plane (RSP) and data plane (line cards) are physically separate. The operating system must manage this distributed system — distributing FIB entries, synchronizing state, and handling partial failures.

### Distributed Architecture

```
                    +----------------+
                    |   Active RSP   |
                    | (Control Plane)|
                    |  BGP, OSPF,    |
                    |  IS-IS, CLI    |
                    +-------+--------+
                            |
                    +-------+--------+
                    |   Fabric       |
                    |   (Crossbar)   |
                    +--+----+----+---+
                       |    |    |
              +--------+  +--------+  +--------+
              |  LC 0  |  |  LC 1  |  |  LC 2  |
              | (Data  |  | (Data  |  | (Data  |
              |  Plane)|  |  Plane)|  |  Plane)|
              | FIB,   |  | FIB,   |  | FIB,   |
              | ACL,   |  | ACL,   |  | ACL,   |
              | QoS    |  | QoS    |  | QoS    |
              +--------+  +--------+  +--------+
```

Each line card has:
- Its own CPU (for FIB programming, statistics collection)
- Its own memory (for local FIB copy, counters)
- Its own forwarding ASICs (for hardware-accelerated forwarding)
- IPC connection to RSP (for FIB updates, configuration)

### FIB Distribution

When the control plane computes a routing change:

1. RIB (Routing Information Base) is updated on the RSP
2. The best path is selected per prefix
3. FIB update is generated and sent to all line cards via IPC
4. Each line card programs its local forwarding ASIC

This distribution introduces a window where different line cards may have different FIB entries:

$$T_{\text{convergence}} = T_{\text{rib\_update}} + T_{\text{fib\_distribution}} + \max_{\text{LC}} T_{\text{asic\_program}}$$

The maximum ASIC programming time across all line cards determines the worst-case forwarding convergence.

### Line Card Independence

A line card crash does not affect other line cards or the RSP. The RSP detects the line card failure and:

1. Removes the line card's interfaces from routing
2. Withdraws routes that were reachable only through that line card
3. Updates FIB on remaining line cards
4. Waits for the line card to restart and re-programs its FIB

---

## 7. Management Interfaces (CLI, NETCONF, gRPC)

### The Problem

Service provider networks require programmatic management at scale. CLI screen-scraping is fragile. SNMP SET operations are limited. A structured, transactional management interface is needed.

### CLI (Command Line Interface)

IOS XR CLI is structurally different from IOS CLI:

| Aspect | IOS CLI | IOS XR CLI |
|:---|:---|:---|
| Config application | Immediate | On commit |
| Config hierarchy | Flat (indentation is cosmetic) | Hierarchical (indentation is semantic) |
| Negation | `no <command>` | `no <command>` (same syntax, different scope) |
| Pipe modifiers | `include`, `exclude`, `begin` | Same plus `utility`, `file`, `xml` |
| Output format | Text only | Text, XML, JSON |

### NETCONF (RFC 6241)

IOS XR implements NETCONF over SSH (port 830). The protocol provides:

- **get-config** — retrieve configuration (running, candidate, or startup)
- **edit-config** — modify configuration with merge, replace, or delete operations
- **commit** — apply candidate to running (maps directly to IOS XR commit)
- **validate** — check candidate without applying
- **lock/unlock** — exclusive configuration access

NETCONF operations map naturally to IOS XR's commit model:

```
NETCONF Operation          IOS XR Equivalent
-----------------          -----------------
<edit-config>              configure terminal + commands
<validate>                 show configuration (semantic check)
<commit>                   commit
<discard-changes>          abort
<get-config>               show running-config
<copy-config>              commit replace
```

### gRPC and Model-Driven Telemetry

IOS XR supports gRPC for both configuration (gNMI) and telemetry (gNMI Subscribe):

**gNMI Operations:**
- `Get` — retrieve operational or configuration data
- `Set` (Update/Replace/Delete) — modify configuration
- `Subscribe` — stream operational data at defined intervals

**Model-Driven Telemetry (MDT)** replaces SNMP polling with push-based streaming:

```
Traditional SNMP:
  Collector --poll--> Router --response--> Collector
  (every 5 minutes, high CPU on router)

MDT:
  Router --stream--> Collector
  (every 30 seconds, minimal CPU, near-real-time)
```

MDT advantages:
- Sub-minute granularity (30-second intervals common)
- Lower router CPU (no SNMP walk processing)
- Structured data (YANG models, not MIB OIDs)
- Efficient encoding (GPB or JSON)
- Scale to thousands of counters per device

### YANG Data Models

IOS XR uses YANG (RFC 7950) data models for all programmatic interfaces:

| Model Type | Scope | Example |
|:---|:---|:---|
| OpenConfig | Vendor-neutral | `openconfig-interfaces` |
| Cisco native | XR-specific | `Cisco-IOS-XR-ifmgr-cfg` |
| IETF | Standards-track | `ietf-interfaces` |

Native models expose the full IOS XR feature set. OpenConfig models provide a common abstraction but may not cover all features.

---

## 8. Route Policy Language (RPL)

### The Problem

IOS uses route-maps, prefix-lists, and community-lists as separate objects that are referenced from each other. This fragmented model is difficult to maintain at scale and lacks programming constructs.

### RPL Design

RPL unifies all route filtering and manipulation into a single policy language:

```
route-policy <NAME>
  if <condition> then
    <action>
  elseif <condition> then
    <action>
  else
    <action>
  endif
end-policy
```

### RPL Building Blocks

**Prefix Sets:**
```
prefix-set CUSTOMER-PREFIXES
  10.0.0.0/8 le 24,
  172.16.0.0/12 le 24,
  192.168.0.0/16 le 24
end-set
```

**Community Sets:**
```
community-set BLACKHOLE
  65000:666
end-set

community-set CUSTOMER-COMM
  65000:100,
  65000:200
end-set
```

**AS-Path Sets:**
```
as-path-set DIRECT-PEERS
  originates-from '65001',
  originates-from '65002'
end-as-path-set
```

### RPL vs. IOS Route-Maps

| Feature | IOS Route-Map | IOS XR RPL |
|:---|:---|:---|
| Conditional logic | Implicit (match + set) | Explicit (if/then/else) |
| Nesting | Not supported | Nested if/elseif/else |
| Inline sets | No (separate prefix-list) | Yes (inline or named) |
| Default action | Implicit deny | Must be explicit (pass/drop) |
| Parameterization | No | Yes (parameterized policies) |
| Boolean operators | No | AND, OR, NOT |

### Parameterized Policies

RPL supports parameterized policies — reusable templates with arguments:

```
route-policy SET-COMM($comm_val)
  set community ($comm_val) additive
  pass
end-policy

! Apply with different parameters:
router bgp 65000
 neighbor 10.0.0.1
  address-family ipv4 unicast
   route-policy SET-COMM(65000:100) in
  !
 !
 neighbor 10.0.0.2
  address-family ipv4 unicast
   route-policy SET-COMM(65000:200) in
  !
 !
!
```

This eliminates policy duplication and ensures consistency.

---

## 9. Data Plane Architecture (CEF/FIB)

### The Problem

At carrier scale, the forwarding plane must process millions of packets per second with deterministic latency. Software-based forwarding cannot meet these requirements. The hardware forwarding infrastructure must be tightly integrated with the OS.

### CEF on IOS XR

Cisco Express Forwarding (CEF) on IOS XR operates in distributed mode:

```
Control Plane (RSP):
  RIB (all routes) --> Best path selection --> FIB generation

Data Plane (Line Card):
  FIB (forwarding entries) --> Adjacency table --> ASIC forwarding
```

The FIB on each line card is a compressed representation of the RIB optimized for longest-prefix match in hardware:

| Structure | Location | Purpose |
|:---|:---|:---|
| RIB | RSP memory | Complete routing table, all paths |
| FIB | LC ASIC/TCAM | Best path per prefix, hardware-optimized |
| Adjacency table | LC memory | Next-hop MAC, interface, encapsulation |
| Label table | LC ASIC/TCAM | MPLS label switching entries |

### Hardware Forwarding Pipeline

On modern ASR 9000 line cards (Memory or Memory with External TCAM):

```
Packet In
  |
  v
Ingress parsing (L2/L3/L4 header decode)
  |
  v
Ingress ACL lookup (TCAM)
  |
  v
FIB lookup (longest prefix match in TCAM or algorithmic LPM)
  |
  v
Adjacency resolution (next-hop, rewrite info)
  |
  v
QoS classification and marking
  |
  v
Fabric interface (to egress line card)
  |
  v
Egress QoS (scheduling, shaping, policing)
  |
  v
Egress rewrite (MAC rewrite, VLAN tag, encapsulation)
  |
  v
Packet Out
```

The entire pipeline executes in hardware at line rate. The RSP CPU is never involved in forwarding — it only handles control plane protocols and management.

### FIB Scale

FIB capacity depends on the line card hardware:

| Line Card Generation | IPv4 FIB | IPv6 FIB | MPLS Labels |
|:---|:---|:---|:---|
| Typhoon (1st gen) | 1M | 512K | 512K |
| Tomahawk (2nd gen) | 2M | 1M | 1M |
| Lightspeed (3rd gen) | 4M | 2M | 2M |
| Lightspeed+ | 5M+ | 4M+ | 4M+ |

Internet full table (currently approximately 1M IPv4 prefixes) fits in all but first-generation cards.

---

## 10. Security Architecture (AAA and Task-Based Authorization)

### The Problem

IOS uses privilege levels (0-15) for authorization — a coarse-grained model that conflates commands into numeric tiers. Service providers need fine-grained control: an operator who can view BGP routes but cannot modify BGP configuration.

### Task-Based Authorization

IOS XR replaces privilege levels with a task-based model:

```
Task: a specific operational capability
  Format: task <permission> <domain>
  Example: task read bgp
           task write interface
           task execute basic-services

Permissions:
  read     — view configuration and operational state
  write    — modify configuration
  execute  — run operational commands (ping, traceroute, clear)
  debug    — enable debugging
```

### Authorization Hierarchy

```
User --> User Group --> Task Group --> Tasks

Example:
  username "operator"
    |
    v
  usergroup "NOC-TEAM"
    |
    v
  taskgroup "MONITORING"
    |
    v
  task read bgp
  task read ospf
  task read interface
  task execute basic-services
```

### Built-in Groups

| Group | Capabilities |
|:---|:---|
| `root-lr` | Full access to one logical router |
| `root-system` | Full access to entire system (admin plane) |
| `cisco-support` | TAC-level debugging access |
| `operator` | Basic monitoring and troubleshooting |
| `sysadmin` | System administration (non-routing) |

### AAA Integration

IOS XR AAA supports TACACS+ and RADIUS with task-based authorization:

```
TACACS+ server returns:
  task = "{ read bgp, read ospf, write interface, execute basic-services }"

Router maps returned tasks to user's effective permissions.
```

This allows centralized, per-user, per-command authorization at a granularity impossible with IOS privilege levels.

---

## Prerequisites

- Familiarity with Cisco IOS CLI and configuration concepts
- Understanding of IP routing fundamentals (BGP, OSPF, IS-IS)
- Basic knowledge of MPLS and VPN services (L2VPN, L3VPN)
- Understanding of high availability concepts (SSO, NSR, NSF, graceful restart)
- For programmability sections: familiarity with XML, YANG, NETCONF, and gRPC concepts

---

## References

- Cisco IOS XR Fundamentals — Mobeen Tahir, Mark Ghattas, Dawit Birhanu, Cisco Press
- Cisco IOS XR Configuration Fundamentals Guide — https://www.cisco.com/c/en/us/td/docs/iosxr/configuration-guide.html
- Cisco ASR 9000 Series Aggregation Services Router System Management Configuration Guide — https://www.cisco.com/c/en/us/td/docs/routers/asr9000/software/system-management/configuration/guide.html
- Cisco IOS XR Programmability Configuration Guide — https://www.cisco.com/c/en/us/td/docs/iosxr/programmability/netconf-yang.html
- RFC 6241 — Network Configuration Protocol (NETCONF)
- RFC 7950 — The YANG 1.1 Data Modeling Language
- RFC 4724 — Graceful Restart Mechanism for BGP
- RFC 8040 — RESTCONF Protocol
- QNX Neutrino RTOS System Architecture — https://www.qnx.com/developers/docs/
