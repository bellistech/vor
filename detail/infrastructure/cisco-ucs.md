# Cisco UCS — Unified Computing Architecture

> *Cisco UCS redefines datacenter compute by severing the bond between server identity and physical hardware. Through service profiles, a server's UUID, MAC addresses, WWNN, WWPN, BIOS settings, firmware level, boot policy, and network connectivity become a portable software construct that can be instantiated on any compatible blade or rack server in seconds. This stateless computing model eliminates manual per-server configuration, transforms hardware into fungible capacity, and enables infrastructure-as-code patterns years before the term became mainstream. The architecture unifies LAN, SAN, and management traffic through a single pair of Fabric Interconnects, collapsing three separate networks into one converged fabric.*

---

## 1. The Fabric Interconnect — Convergence Point

### Design Philosophy

The Fabric Interconnect (FI) is the central nervous system of every UCS domain. Unlike traditional datacenter designs where compute, LAN switching, SAN switching, and management are four independent silos, the FI collapses all three data-plane networks plus management into a single device pair.

Each FI is simultaneously:

- A top-of-rack Ethernet switch (10/25/40/100GbE uplinks to campus or spine)
- A Fibre Channel switch (native FC or FCoE to storage arrays)
- A management controller (hosting UCS Manager as an embedded application)
- A unified fabric endpoint (carrying all traffic types over common backplane links)

### Clustering and High Availability

FIs operate as an active/standby cluster for the management plane (UCS Manager) and an active/active pair for the data plane. The cluster link between FI-A and FI-B carries:

- UCS Manager state synchronization (configuration database replication)
- Heartbeat messages for primary/subordinate election
- Management traffic failover

The management VIP (virtual IP) floats between the two FIs. When the primary FI fails, the subordinate promotes itself, assumes the VIP, and continues serving the UCSM GUI and CLI within seconds. The data plane is unaffected because each blade maintains independent paths through both FIs simultaneously.

### Port Types and Unified Ports

FIs from the 6200 series onward support unified ports — physical ports that can be configured as either Ethernet or Fibre Channel:

| Port Role | Function | Direction |
|:---|:---|:---|
| Server port | Connects to IOM or FEX | Downlink to chassis |
| Uplink port | Connects to upstream LAN switch | Uplink to network |
| FC uplink | Connects to SAN switch or storage | Uplink to SAN |
| FCoE uplink | Carries FC over Ethernet to upstream switch | Uplink to FCoE switch |
| Appliance port | Direct Ethernet to NAS/appliance | Uplink to storage |
| Monitor port | SPAN/ERSPAN destination | Monitoring |

Unified port reconfiguration requires an FI reboot for the affected port group, making it a day-zero planning decision. Ports are grouped in blocks of four; all ports in a block must be the same type.

---

## 2. IO Modules and the Chassis Midplane

### IOM Architecture

Each UCS 5108 chassis has two IOM slots at the rear. IOM-A connects exclusively to FI-A; IOM-B connects exclusively to FI-B. This creates two independent physical paths from every blade to the network.

The IOM is not a switch. In most operating modes it functions as a multiplexer (mux) that maps blade-side ports to FI-side uplinks. The FI itself makes all switching decisions. This design:

- Eliminates spanning tree within the chassis
- Centralizes all forwarding policy at the FI
- Simplifies firmware — IOMs have minimal intelligence
- Reduces failure domains — an IOM failure affects only one fabric path

### Backplane Connectivity

Each blade slot has a fixed number of backplane traces to each IOM. The bandwidth per blade depends on the IOM model:

| IOM Model | Backplane per Blade | Total Chassis Bandwidth | Notes |
|:---|:---|:---|:---|
| 2204XP | 4 x 10GbE | 320 Gbps | Legacy, 2200 series |
| 2208XP | 8 x 10GbE | 640 Gbps | Standard for B200 M4 |
| 2304 | 4 x 40GbE | 1.28 Tbps | UCS 6300 FI required |
| 2408 | 8 x 25GbE | 1.6 Tbps | UCS 6400 FI required |

The IOM model must match the FI generation. A 2408 IOM requires 6400-series FIs; it will not function with 6300-series FIs.

---

## 3. Blade and Rack Servers

### B-Series Blades

B-series blades are half-width servers that slide into the 5108 chassis (8 slots per chassis). Each blade contains:

- One or two CPUs (Intel Xeon Scalable)
- DDR4/DDR5 memory (up to 24 DIMM slots on B200 M6)
- A Virtual Interface Card (VIC) that presents virtual NICs and HBAs
- Optional mezzanine cards for additional VIC capacity or GPU
- Local storage (M.2 or optional SAS/NVMe via storage mezzanine)

The VIC is the critical component for the stateless computing model. A Cisco VIC 1440 or 1480 can present up to 256 virtual interfaces (vNICs + vHBAs) to the OS, each with its own MAC, VLAN trunk, QoS policy, and failover behavior. The OS sees standard Ethernet NICs and FC HBAs — no special drivers required beyond the VIC enic/fnic drivers.

### C-Series Rack Servers

C-series rack-mount servers can be UCS-managed in two ways:

1. **Direct-connect**: C-series with VIC connects directly to FI server ports via 10/25/40GbE. Full UCSM management and service profile support.
2. **FEX-attached**: Cisco Fabric Extender (2232PP or similar) acts as a remote line card of the FI, connecting multiple rack servers.

Rack servers in UCS-managed mode gain the same service profile capabilities as blades: pooled identities, policy-driven configuration, and stateless operation.

### S-Series Storage Servers

The S3260 is a dense storage server (56 drives per 2U chassis) that operates within UCS domains for software-defined storage, object stores, and big data workloads. It supports dual server modules and connects to FIs like C-series servers.

---

## 4. Stateless Computing — The Core Innovation

### The Problem with Traditional Servers

In traditional datacenter operations, a server's identity is permanently bound to its hardware:

1. The BIOS UUID is burned into the motherboard
2. MAC addresses are on the physical NIC
3. WWN addresses are on the physical HBA
4. BIOS settings are stored in NVRAM on the motherboard
5. Boot configuration is local to the server

When hardware fails, the replacement server has a completely different identity. This triggers cascading reconfiguration: update DHCP reservations (new MAC), update zoning (new WWPN), update monitoring (new UUID), update OS licenses (hardware-locked), and re-register with management tools.

### The Service Profile Solution

UCS service profiles decouple every identity element from hardware:

| Identity Element | Traditional | UCS Service Profile |
|:---|:---|:---|
| UUID | Burned in BIOS | Assigned from pool |
| MAC addresses | Physical NIC | Assigned from pool, per vNIC |
| WWNN | Physical HBA | Assigned from pool |
| WWPN | Physical HBA | Assigned from pool, per vHBA |
| BIOS settings | Local NVRAM | BIOS policy |
| Boot device/order | Local NVRAM | Boot policy |
| Firmware version | Flashed to hardware | Firmware policy |
| Network connectivity | Cable + switch config | LAN/SAN connectivity policy |

When a blade fails, the service profile is simply re-associated to a spare blade. The new blade boots with the exact same UUID, MAC addresses, WWPNs, BIOS settings, and boot targets. SAN zoning still works (same WWPN). DHCP still works (same MAC). OS licenses still work (same UUID). The replacement is invisible to everything above the hardware layer.

### Association Mechanics

When a service profile is associated to a blade, the following sequence executes (managed by the FSM — Finite State Machine):

1. **Identity programming**: VIC firmware receives pooled MAC, WWNN, WWPN values and programs them into hardware registers. The OS will see these as physical addresses.
2. **BIOS configuration**: CIMC (Cisco Integrated Management Controller) on the blade receives BIOS token values from the BIOS policy and writes them to NVRAM.
3. **Firmware alignment**: If the blade's current firmware differs from the host firmware policy, the FI stages the firmware and the blade reboots into the update process.
4. **vNIC/vHBA instantiation**: The VIC creates the specified number of virtual interfaces with the correct VLAN trunks, QoS settings, MTU, and failover configuration.
5. **Boot target configuration**: The VIC's boot ROM is configured with SAN boot targets (WWPN + LUN) or PXE/iSCSI parameters.
6. **Power on**: The blade boots with its complete software-defined identity.

The entire process takes 5-15 minutes, dominated by firmware updates and BIOS POST. Without firmware changes, association completes in 2-3 minutes.

---

## 5. Identity Pools — Managed Address Spaces

### Pool Architecture

UCS pools are pre-allocated ranges of identifiers that service profiles draw from. Each pool type serves a specific identity dimension:

**UUID Suffix Pool**: The UUID presented to the OS as the system UUID (visible in `dmidecode`). The prefix is typically derived from the UCS domain, and the suffix comes from the pool. Format: `xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx`.

**MAC Pool**: Layer 2 addresses for each vNIC. Best practice is one pool per fabric to simplify troubleshooting. When a packet arrives at the upstream switch, the OUI (first three octets) immediately identifies it as UCS-originated (00:25:B5 is Cisco's UCS OUI).

**WWNN Pool**: World Wide Node Name, one per server. Identifies the server node to the SAN fabric. All vHBAs on a server share the same WWNN.

**WWPN Pool**: World Wide Port Name, one per vHBA. Identifies each virtual HBA port to the SAN fabric. Used in SAN zoning. Separate pools per fabric ensure that Fabric-A vHBAs have traceable addresses distinct from Fabric-B vHBAs.

### Pool Sizing and Planning

Pool sizing must account for:

- Current server count plus growth headroom (typically 2x)
- Number of vNICs per server (each needs a unique MAC)
- Number of vHBAs per server (each needs a unique WWPN)
- Multiple organizations if using UCSM sub-orgs (pools are org-scoped)

A common formula for MAC pool sizing:

```
Pool size = (max_servers) x (vNICs_per_server) x (growth_factor)
Example: 160 blades x 4 vNICs x 2 = 1,280 MAC addresses per fabric
```

### Pool Overlap and Conflict Avoidance

Pools across different UCS domains must not overlap. When managing multiple domains, use a structured allocation scheme:

| UCS Domain | MAC Pool A (Prefix) | MAC Pool B (Prefix) | WWPN Pool A | WWPN Pool B |
|:---|:---|:---|:---|:---|
| Domain 1 | 00:25:B5:A0:01:xx | 00:25:B5:B0:01:xx | 20:00:00:25:B5:A1:xx:xx | 20:00:00:25:B5:B1:xx:xx |
| Domain 2 | 00:25:B5:A0:02:xx | 00:25:B5:B0:02:xx | 20:00:00:25:B5:A2:xx:xx | 20:00:00:25:B5:B2:xx:xx |
| Domain 3 | 00:25:B5:A0:03:xx | 00:25:B5:B0:03:xx | 20:00:00:25:B5:A3:xx:xx | 20:00:00:25:B5:B3:xx:xx |

Encoding the domain number into the address prefix makes packet-level debugging trivial: any tcpdump or SAN analytics trace immediately reveals which domain and fabric originated the traffic.

---

## 6. Policies — Declarative Hardware Configuration

### BIOS Policy

BIOS policies replace manual BIOS tuning. Each policy contains hundreds of BIOS tokens covering:

- **Performance**: C-states, P-states, turbo mode, power technology
- **Virtualization**: Intel VT-x, VT-d, SR-IOV, ACS
- **Memory**: NUMA optimization, interleaving, patrol scrub
- **Security**: TPM, Secure Boot, SGX
- **Boot**: boot order overrides, quiet boot

A single BIOS policy can be shared across hundreds of service profiles. When the policy changes (and an updating template propagates the change), the maintenance policy determines when the blades acknowledge the reboot required to apply new BIOS settings.

### Boot Policy

Boot policies define the ordered list of boot devices the server attempts during startup:

1. **Local disk**: Boot from RAID or single local drive
2. **SAN boot**: Boot from a Fibre Channel LUN (requires WWPN of target + LUN ID)
3. **iSCSI boot**: Boot from an iSCSI target (requires target IQN + IP)
4. **PXE boot**: Network boot from a PXE/TFTP server (via specific vNIC)
5. **UEFI shell**: Boot to UEFI environment
6. **Embedded local LUN**: Boot from M.2 or embedded SATA

SAN boot is the most common pattern in enterprise UCS deployments because it fully eliminates local storage dependencies, making the blade truly stateless. The boot LUN is pre-provisioned on the storage array, zoned to the service profile's WWPN, and the blade boots directly from SAN.

### Firmware Policy

Host firmware policies bind a specific firmware bundle version to service profiles. This ensures:

- All blades in a workload run identical firmware
- Firmware upgrades are staged and policy-driven, not ad-hoc
- New blades auto-align to the correct firmware during association
- Rollback is a policy change, not a manual flash

Infrastructure firmware policies separately control FI and IOM firmware, enabling independent upgrade cycles for infrastructure vs. compute.

### Network Control Policy

Controls Layer 2 behavior for vNICs:

- **CDP/LLDP**: Enable or disable Cisco Discovery Protocol and Link Layer Discovery Protocol for upstream switch visibility
- **MAC register mode**: How MAC addresses are registered with the FI (only-native-vlan or all-host-vlans)
- **Forge MAC**: Allow or deny MAC address spoofing (relevant for hypervisor environments where VMs have their own MACs)
- **Uplink fail action**: What happens to vNIC when all uplinks fail (link-down or warning)

---

## 7. Templates — Scaling Configuration

### Initial vs Updating Templates

This is one of the most important design decisions in UCS:

**Initial Template**: Creates a point-in-time snapshot. Service profiles instantiated from an initial template are independent copies. Subsequent changes to the template do not propagate to existing profiles. Use cases:

- Environments requiring per-server customization after initial deployment
- Regulatory environments where change propagation must be explicitly controlled
- Brownfield migrations where servers need individual tuning

**Updating Template**: Maintains a live binding. Any change to the template automatically propagates to all derived service profiles. The maintenance policy controls when the change takes effect (immediate reboot, user-acknowledge, or scheduled window). Use cases:

- Large-scale homogeneous environments (web tiers, compute farms)
- Environments requiring guaranteed configuration consistency
- Rapid policy rollout (firmware updates, BIOS changes, new VLANs)

### Template Composition

A service profile template references other templates and policies in a hierarchical structure:

```
Service Profile Template
├── BIOS Policy
├── Boot Policy
├── Host Firmware Policy
├── Maintenance Policy
├── IPMI Access Profile
├── Serial over LAN Policy
├── vNIC Template A (references MAC pool, QoS policy, network control policy)
├── vNIC Template B
├── vHBA Template A (references WWPN pool, VSAN)
├── vHBA Template B
├── LAN Connectivity Policy (alternative to individual vNIC references)
├── SAN Connectivity Policy (alternative to individual vHBA references)
├── UUID Pool
├── WWNN Pool
├── Server Pool (or manual assignment)
├── Server Pool Qualification
├── Power Control Policy
├── Scrub Policy
└── Storage Profile
```

Each referenced policy is independently versioned and reusable. A BIOS policy shared across five templates propagates changes to all five when modified (for updating templates).

---

## 8. vNIC and vHBA — Virtual Interface Architecture

### Cisco VIC Hardware

The Cisco Virtual Interface Card (VIC) is a custom ASIC-based adapter that enables the stateless computing model. Unlike commodity NICs that present a fixed number of physical ports, the VIC:

- Creates up to 256 virtual interfaces in hardware (not software SR-IOV)
- Programs MAC, VLAN, and QoS per interface at the hardware level
- Supports both Ethernet (vNIC) and Fibre Channel (vHBA) on the same physical ports
- Implements fabric failover in hardware (sub-millisecond path switchover)
- Provides NetQueue / RSS for multi-queue performance

The VIC connects to both IOMs through the chassis backplane, giving each virtual interface dual paths. The VIC firmware, controlled by the host firmware policy, is updated during service profile association.

### vNIC Properties

Each vNIC is a fully independent Ethernet interface with:

- **MAC address**: From the assigned MAC pool
- **VLAN trunk**: List of allowed VLANs and native VLAN
- **MTU**: Per-vNIC (supports jumbo frames up to 9216)
- **QoS policy**: Traffic class, burst size, rate limiting
- **Fabric pin group**: Optional pinning to specific uplink port-channels
- **Failover**: Enable/disable automatic fabric path failover
- **CDN (Consistent Device Naming)**: OS-visible interface name (e.g., "eth-prod" instead of "ens192")
- **Adapter policy**: Hardware offload settings (RSS, TSO, interrupt coalescing, ring size)
- **USNIC (User-space NIC)**: Optional user-space direct access for low-latency applications

### vHBA Properties

Each vHBA is a virtual Fibre Channel port with:

- **WWPN**: From the assigned WWPN pool
- **VSAN**: The virtual SAN this vHBA belongs to
- **QoS policy**: FC-specific traffic management
- **Adapter policy**: Queue depth, interrupt settings, FC error handling
- **Persistent binding**: Control target discovery persistence

### Fabric Failover Deep Dive

Fabric failover is implemented at the VIC hardware level. When enabled on a vNIC:

1. The vNIC has a preferred fabric (A or B) where it normally operates
2. The VIC maintains an active path through the preferred IOM and a standby path through the other IOM
3. If the active path fails (IOM failure, FI failure, uplink failure), the VIC switches the vNIC to the standby path
4. The MAC address moves with the vNIC — the upstream network sees the MAC appear on the other FI
5. Switchover is hardware-driven and completes in milliseconds
6. When the preferred path recovers, the vNIC can fail back (configurable)

This is distinct from NIC teaming in the OS. Fabric failover operates below the OS, requires no bonding configuration, and works with any operating system. The OS sees a single NIC that never goes down — only a brief traffic pause during switchover.

For dual-homed designs without fabric failover, the OS or hypervisor must manage both vNICs: NIC teaming in Windows, bonding in Linux, or vSwitch teaming in ESXi.

---

## 9. LAN and SAN Connectivity Architecture

### LAN Traffic Flow

The end-to-end path for Ethernet traffic from a VM to the upstream network:

```
VM → vSwitch → vNIC (VIC) → Backplane → IOM → FI → Uplink → Upstream Switch
```

Each segment:

1. **VM to vSwitch**: Standard virtual networking (VST or VGT mode)
2. **vSwitch to vNIC**: The VIC vNIC appears as a physical NIC to the hypervisor
3. **vNIC to backplane**: Traces on the chassis midplane carry traffic to the IOM slot
4. **Backplane to IOM**: The IOM multiplexes blade ports onto FI-facing uplinks
5. **IOM to FI**: 10/25/40GbE links between IOM and FI carry all blade traffic
6. **FI switching**: The FI performs Layer 2 switching, VLAN enforcement, QoS
7. **FI to uplink**: Port-channels to upstream LAN switches

### SAN Traffic Flow

Fibre Channel traffic takes a parallel but distinct path:

```
VM/OS → vHBA (VIC) → Backplane → IOM → FI (FC switching) → FC Uplink → SAN Switch → Storage
```

The FI operates as an NPV (N-Port Virtualizer) or full FC switch:

- **NPV mode** (default): The FI proxies FC logins to the upstream SAN switch. The SAN switch handles zoning and fabric services. Simpler, fewer FC domain IDs consumed.
- **FC switch mode**: The FI is a full FC switch with its own domain ID. Supports direct-attached storage without upstream SAN switches. More complex but eliminates a tier.

### FCoE (Fibre Channel over Ethernet)

Between the blade and the FI, all traffic (Ethernet + FC) travels as FCoE over the same physical backplane links. The IOM does not distinguish traffic types — it forwards everything to the FI. The FI then:

- Decapsulates FCoE frames destined for the FC network and forwards them as native FC out FC uplink ports
- Switches Ethernet frames normally out Ethernet uplink ports

This convergence is transparent to the blade. The VIC presents standard Ethernet vNICs and standard FC vHBAs; the blade OS is unaware that they share physical transport.

---

## 10. UCS Manager vs Intersight — Management Evolution

### UCS Manager (UCSM)

UCSM is the embedded management application running on the FI cluster. It manages a single UCS domain (one FI pair and all attached chassis/rack servers). Key characteristics:

- **On-premises**: No cloud dependency, runs entirely on the FI
- **Single domain**: One UCSM instance per FI pair
- **Full-featured GUI and CLI**: Java-based GUI (legacy), HTML5 GUI (newer), and comprehensive CLI
- **XML API**: Programmatic access for automation (used by UCS PowerTool, Python SDK)
- **Proven**: 15+ years of deployment history, extremely stable

Limitations:
- Cannot manage multiple UCS domains from a single pane
- No cross-domain policy consistency enforcement
- No cloud-based analytics or recommendation engine
- Requires UCS Central for multi-domain management (separate product)

### Cisco Intersight

Intersight is the cloud-based (SaaS) management platform that supersedes UCSM for new deployments:

- **Multi-domain**: Manages thousands of UCS domains, HyperFlex clusters, and third-party servers from one console
- **Cloud-delivered**: Firmware updates, policy recommendations, and analytics from Cisco's cloud
- **API-first**: REST API with OpenAPI specification, Terraform provider, Ansible modules
- **IMM (Intersight Managed Mode)**: Replaces UCSM entirely — FI is managed by Intersight, not local UCSM
- **Kubernetes**: Intersight Kubernetes Service (IKS) for container orchestration on UCS hardware
- **Private appliance**: For air-gapped environments, Intersight runs as an on-prem VM (Connected Virtual Appliance or Private Virtual Appliance)

### Migration Path

Organizations typically migrate from UCSM to Intersight in phases:

1. **Claim devices**: Connect FIs to Intersight via the device connector (UCSM still manages day-to-day)
2. **Monitor mode**: Intersight provides inventory, alerts, and recommendations while UCSM remains primary
3. **IMM conversion**: Convert the FI pair to Intersight Managed Mode (destructive — wipes UCSM config, requires reprovisioning service profiles as Intersight server profiles)
4. **Full IMM**: All policy management through Intersight

IMM conversion is a significant migration event. It requires rebuilding all service profiles as Intersight server profiles. While the concepts are similar (pools, policies, templates), the object model differs.

---

## 11. Server Provisioning Workflow — End to End

### Phase 1: Infrastructure Setup

1. Rack and cable chassis, blades, and FIs according to cabling guide
2. Power on FIs and complete initial setup wizard (management IPs, cluster config, NTP, DNS)
3. Chassis auto-discovery begins — UCSM detects IOMs and blades
4. Acknowledge chassis and blades in UCSM to complete discovery
5. Configure Ethernet uplinks (port-channels to upstream switches)
6. Configure FC uplinks (port-channels to SAN switches) if SAN boot is used
7. Create VLANs matching upstream network design
8. Create VSANs matching SAN fabric design

### Phase 2: Identity and Policy Foundation

1. Create UUID pool with sufficient range for all expected servers
2. Create MAC pools (one per fabric: MAC-A, MAC-B)
3. Create WWNN pool (one per domain)
4. Create WWPN pools (one per fabric: WWPN-A, WWPN-B)
5. Create server pool and optionally define qualification policies
6. Create IP pool for KVM access (out-of-band management IPs for CIMC)
7. Create BIOS policy for each workload type
8. Create host firmware policy referencing the approved firmware bundle
9. Create boot policy (SAN boot, local disk, PXE, or hybrid)
10. Create maintenance policy (user-ack for production)

### Phase 3: Connectivity Templates

1. Create QoS policies if differentiating traffic classes
2. Create network control policy (CDP/LLDP, forge MAC settings)
3. Create adapter policies for workload tuning (RSS queues, ring sizes)
4. Create vNIC templates (one per fabric per network role)
5. Create vHBA templates (one per fabric)
6. Create LAN connectivity policy referencing vNIC templates
7. Create SAN connectivity policy referencing vHBA templates and WWNN pool

### Phase 4: Service Profile Deployment

1. Create service profile template (updating for production workloads)
2. Reference all policies, pools, and connectivity templates
3. Instantiate N service profiles from the template
4. Associate each service profile to a blade (specific or pool-based)
5. Monitor FSM for association progress (UCSM GUI or CLI)
6. Verify: blade reboots with new identity, SAN targets visible, PXE reachable
7. Proceed with OS installation (PXE, SAN boot from pre-imaged LUN, or manual)

### Time Estimates

| Phase | Duration | Notes |
|:---|:---|:---|
| Physical cabling | 2-8 hours | Depends on chassis count |
| FI initial setup | 30 minutes | Per FI pair |
| Chassis discovery | 5-10 minutes | Automatic |
| Pool/policy creation | 1-2 hours | Per domain, one-time |
| Template creation | 30-60 minutes | Per workload type |
| Service profile association | 5-15 minutes per blade | Parallel across chassis |
| OS deployment | Varies | PXE/kickstart can be fully automated |

A well-prepared deployment of 64 blades (8 chassis) from power-on to OS-ready takes approximately one business day with automation, or 2-3 days manually.

---

## 12. UCS Mini — Branch and Edge Computing

### Architecture

UCS Mini integrates the Fabric Interconnect directly into a UCS 5108 chassis via the FI-6324 module. Instead of external FI appliances connected by cables, the 6324 occupies the IOM slots:

- **IOM slot 1**: FI-6324 module A (acts as both IOM and FI)
- **IOM slot 2**: FI-6324 module B (acts as both IOM and FI)

The result is a fully self-contained UCS domain in a single chassis. The same UCSM runs on the embedded 6324, providing identical management capabilities: service profiles, pools, policies, templates.

### Use Cases

- **Remote offices**: Full UCS capabilities without dedicated FI rack space
- **Edge compute**: Compact, self-contained compute for retail, manufacturing, healthcare
- **Lab environments**: Full-featured UCS lab in minimal footprint
- **Disaster recovery**: Standby compute at DR sites

### Scaling

UCS Mini supports connecting additional chassis to the 6324 via standard server ports. A single 6324 pair can manage up to 4 chassis (32 blades total), though the limited uplink capacity of the 6324 constrains aggregate throughput compared to 6300/6400 series FIs.

### Limitations vs Full-Scale UCS

| Capability | Full-Scale UCS (6300/6400) | UCS Mini (6324) |
|:---|:---|:---|
| Max chassis | 20+ per FI pair | 4 per 6324 pair |
| Uplink ports | 24-54 | 4 fixed |
| Unified ports | Yes (configurable) | No (fixed Ethernet) |
| FC uplinks | Native FC ports available | FCoE only (no native FC) |
| Max throughput | Multi-terabit | ~160 Gbps |
| FI redundancy | Separate appliances | Integrated in chassis |

---

## Prerequisites

- Understanding of Layer 2 and Layer 3 networking (VLANs, trunking, routing)
- Fibre Channel fundamentals (WWNN, WWPN, zoning, VSAN)
- Server hardware concepts (CPU, memory, PCIe, BIOS/UEFI)
- Virtualization basics (hypervisor, vSwitch, virtual machines)
- Basic familiarity with Cisco IOS/NX-OS CLI syntax (UCS CLI is similar)

## References

- Cisco UCS Architecture Whitepaper: https://www.cisco.com/c/en/us/products/collateral/servers-unified-computing/ucs-manager/whitepaper-c11-744018.html
- "Data Center Virtualization Fundamentals" by Gustavo Santana (Cisco Press) — Chapters on UCS architecture
- Cisco UCS Manager CLI Configuration Guide: https://www.cisco.com/c/en/us/support/servers-unified-computing/ucs-manager/products-installation-and-configuration-guides-list.html
- Cisco UCS Design Guides (CVDs): https://www.cisco.com/c/en/us/solutions/design-zone/data-center-design-guides/data-center-design-guides-702702.html
- Cisco Intersight Documentation: https://intersight.com/help/saas
- RFC 4338 — Transmission of IPv6 and IPv4 over Fibre Channel
- Cisco UCS Python SDK: https://github.com/CiscoUcs/ucsmsdk
- Cisco UCS Ansible Collection: https://galaxy.ansible.com/cisco/ucs
- "Cisco UCS Cookbook" by Victor Wu and Adrian Borja (Packt) — Practical service profile design
