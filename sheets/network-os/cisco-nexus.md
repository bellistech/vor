# Cisco Nexus (NX-OS Data Center Switching)

NX-OS is Cisco's modular, Linux-based network operating system purpose-built for data center switching across the Nexus platform family.

## Platform Overview

### Nexus Family Comparison

```
+----------+----------------+--------------------+---------------------------+
| Platform | Role           | Typical Position   | Key Features              |
+----------+----------------+--------------------+---------------------------+
| 9000     | Spine / Leaf   | ACI or standalone  | VXLAN EVPN, ACI, 400G     |
| 7000     | Core / Agg     | DC core, DCI       | VDC, OTV, FabricPath, L3  |
| 5000     | Access / Agg   | Unified fabric     | FCoE, FEX parent, vPC     |
| 3000     | ToR / Leaf     | Access layer       | Low latency, compact      |
| 2000     | FEX            | Server access      | Remote line card for 5K/9K|
+----------+----------------+--------------------+---------------------------+
```

### Platform Selection Quick Guide

| Need                        | Platform       |
|-----------------------------|----------------|
| ACI fabric                  | Nexus 9000     |
| Multi-tenant DC core        | Nexus 7000     |
| FCoE convergence            | Nexus 5000     |
| Low-latency ToR             | Nexus 3000     |
| High-density server access  | Nexus 2000 FEX |

## NX-OS Architecture

### Kernel and Modularity

NX-OS runs on a Linux kernel with a modular service architecture. Each protocol and feature runs as an independent process with its own protected memory space.

```
+-------------------------------------------------------+
|                    NX-OS User Space                    |
|  +----------+ +--------+ +------+ +-------+ +------+  |
|  | BGP (bgp)| |OSPF    | |STP   | |vPC    | |LLDP  |  |
|  | process  | |process | |proc  | |proc   | |proc  |  |
|  +----------+ +--------+ +------+ +-------+ +------+  |
|  +----------+ +--------+ +------+ +-------+           |
|  |   URIB   | |  UFDM  | |  MTS | | sysmgr|           |
|  +----------+ +--------+ +------+ +-------+           |
+-------------------------------------------------------+
|               Linux Kernel (Wind River)                |
+-------------------------------------------------------+
|             Hardware Abstraction Layer                 |
+-------------------------------------------------------+
|         ASICs (Memory, Memory, Memory!)                |
+-------------------------------------------------------+
```

### Key System Processes

| Process  | Function                                        |
|----------|-------------------------------------------------|
| sysmgr   | System manager, starts/monitors all services    |
| mts      | Message and Transaction Service (IPC)            |
| urib     | Unicast RIB — route table management             |
| u6rib    | IPv6 unicast RIB                                 |
| mrib     | Multicast RIB                                    |
| ufdm     | Unicast Forwarding Distribution Manager          |
| pixm     | Port Index Manager                               |
| ethpm    | Ethernet Port Manager                            |
| aclmgr   | ACL Manager                                      |

### Process Restart and HA

```bash
# Restart a process without affecting traffic
restart bgp

# Check process status
show processes | include bgp
show system internal sysmgr service name bgp

# View process crash history
show cores
show process log
```

## NX-OS CLI Differences from IOS

### Key Differences

```
+---------------------------+-----------------------------+----------------------------+
| Action                    | IOS                         | NX-OS                      |
+---------------------------+-----------------------------+----------------------------+
| Enable a protocol         | router ospf 1               | feature ospf               |
|                           |                             | router ospf 1              |
| Show running              | show running-config         | show running-config        |
| Section filter            | | section                   | | section                  |
| Copy to startup           | copy run start              | copy run start             |
| Rollback                  | N/A (archive)               | checkpoint / rollback      |
| VRF-aware commands        | show ip route vrf X         | show ip route vrf X        |
| Interface ranges          | interface range gi0/1 - 4   | interface e1/1-4           |
| Enable features           | (always available)          | feature <name> required    |
| Default config            | no shutdown (varies)        | shutdown (most intf)       |
+---------------------------+-----------------------------+----------------------------+
```

### Feature Enablement

Features must be explicitly enabled before use:

```
feature ospf
feature bgp
feature eigrp
feature hsrp
feature lacp
feature vpc
feature interface-vlan
feature lldp
feature nxapi
feature fabric forwarding
feature nv overlay
feature vn-segment-vlan-based
```

### Checkpoint and Rollback

```bash
# Create a checkpoint
checkpoint my-checkpoint

# List checkpoints
show checkpoint summary

# Compare running to checkpoint
show diff rollback-patch checkpoint my-checkpoint

# Rollback
rollback running-config checkpoint my-checkpoint
```

## Virtual Device Contexts (VDC)

### Overview

VDCs partition a single physical Nexus 7000 into multiple logical switches, each with its own config, admin, and fault domain.

```
+----------------------------------------------+
|           Physical Nexus 7000                |
|  +------------+ +------------+ +----------+  |
|  | VDC 1      | | VDC 2      | | VDC 3    |  |
|  | (default)  | | (tenant-a) | | (dmz)    |  |
|  | Admin VDC  | |            | |          |  |
|  | e1/1-8     | | e2/1-12    | | e3/1-4   |  |
|  +------------+ +------------+ +----------+  |
+----------------------------------------------+
```

### Creating and Managing VDCs

```bash
# Create a VDC
vdc tenant-a
  limit-resource vlan minimum 16 maximum 4094
  limit-resource vrf minimum 2 maximum 4096
  limit-resource u4route-mem minimum 8 maximum 8
  limit-resource u6route-mem minimum 4 maximum 4
  allocate interface ethernet 2/1-12

# Switch to a VDC
switchto vdc tenant-a

# Return to default VDC
switchback

# Show all VDCs
show vdc

# Show VDC resource usage
show vdc resource

# Show VDC membership of interfaces
show vdc membership
```

### VDC Resource Limits

| Resource      | Description                    | Default | Range         |
|---------------|--------------------------------|---------|---------------|
| vlan          | VLAN count                     | 4094    | 16 - 4094     |
| vrf           | VRF instances                  | 4096    | 2 - 4096      |
| u4route-mem   | IPv4 route memory (GB)         | 8       | 1 - 8         |
| u6route-mem   | IPv6 route memory (GB)         | 4       | 1 - 4         |
| port-channel  | Port-channel count             | 768     | 0 - 768       |
| m4route-mem   | IPv4 multicast route mem       | 2       | 1 - 2         |

## vPC (Virtual Port Channel)

### Architecture

```
        +----------+   peer-keepalive   +----------+
        | Nexus-A  |<==================>| Nexus-B  |
        | (primary)|   (mgmt or L3)    |(secondary)|
        +----+-----+                    +-----+----+
             |        peer-link (Po)          |
             +================================+
             |               |                |
        +----+----+     +----+----+      +----+----+
        | vPC 10  |     | vPC 20  |      | vPC 30  |
        | Server1 |     | Server2 |      | Switch  |
        +---------+     +---------+      +---------+
```

### vPC Configuration

```bash
# Step 1: Enable features
feature vpc
feature lacp

# Step 2: Create vPC domain
vpc domain 100
  role priority 1000              ! lower = primary
  peer-keepalive destination 10.1.1.2 source 10.1.1.1 vrf management
  peer-gateway
  ip arp synchronize
  auto-recovery
  delay restore 30
  delay restore interface-vlan 45

# Step 3: Configure peer-link
interface port-channel 1
  description vPC-PEER-LINK
  switchport
  switchport mode trunk
  switchport trunk allowed vlan 1-4094
  vpc peer-link
  spanning-tree port type network

# Step 4: Configure vPC member ports
interface port-channel 10
  description vPC-TO-SERVER1
  switchport
  switchport mode trunk
  vpc 10

interface ethernet 1/1
  channel-group 10 mode active

# Orphan port configuration
interface ethernet 1/48
  description ORPHAN-SINGLE-ATTACHED
  switchport
  vpc orphan-port suspend
```

### vPC Consistency Checks

```bash
# Check type-1 (must match — will suspend)
show vpc consistency-parameters global
show vpc consistency-parameters interface port-channel 10

# Key Type-1 parameters (must be identical):
#   - STP mode
#   - STP region config (MST)
#   - VLAN-to-STP mapping
#   - STP global settings (Bridge Assurance, loop guard)
#   - Interface VLAN membership

# Type-2 parameters (should match — warning only):
#   - VTP mode / domain
#   - STP port type
#   - BPDU filter/guard
```

### vPC Verification Commands

```bash
show vpc
show vpc brief
show vpc role
show vpc peer-keepalive
show vpc consistency-parameters global
show vpc orphan-port
show vpc statistics
show port-channel summary
```

## FabricPath

### Architecture

```
+----------+    FabricPath Core    +----------+
| Spine 1  |<=====================>| Spine 2  |
+-----+----+                      +----+-----+
      |  \                          /  |
      |   +---------+  +---------+   |
      |             |  |             |
+-----+----+  +-----+----+  +-----+----+
|  Leaf 1  |  |  Leaf 2  |  |  Leaf 3  |
| Switch-ID|  | Switch-ID|  | Switch-ID|
|    1     |  |    2     |  |    3     |
+-----+----+  +-----+----+  +-----+----+
      |             |             |
   [Hosts]       [Hosts]       [Hosts]

  CE Edge       FP Core       CE Edge
  (802.1Q)      (FP IS-IS)    (802.1Q)
```

### FabricPath Configuration

```bash
# Enable FabricPath
feature-set fabricpath
install feature-set fabricpath

# Assign switch ID
fabricpath switch-id 1

# Configure FabricPath VLANs
vlan 100
  mode fabricpath

# Configure FabricPath core interfaces
interface ethernet 1/1
  switchport mode fabricpath

# Verify FabricPath
show fabricpath switch-id
show fabricpath isis adjacency
show fabricpath isis route
show fabricpath topology
show fabricpath conflict
```

### FabricPath vs Traditional STP

| Feature               | STP                  | FabricPath            |
|-----------------------|----------------------|-----------------------|
| Loop prevention       | Block redundant paths| IS-IS equal-cost paths|
| Bandwidth utilization | 50% (active/standby) | 100% (all paths)      |
| Convergence           | Seconds              | Sub-second            |
| MAC learning          | Flood and learn      | Conversational        |
| Scale                 | Limited by STP domain| IS-IS routing scale   |

## OTV (Overlay Transport Virtualization)

### Architecture

```
+--Site A--+         IP Transport        +--Site B--+
|          |  (MPLS/Internet/Dark Fiber) |          |
| VLAN 100 +============================+ VLAN 100 |
| VLAN 200 |     OTV Overlay Tunnel      | VLAN 200 |
|          +============================+          |
| +------+ |                            | +------+ |
| |OTV   | |    Join Interface           | |OTV   | |
| |Edge   |<===========================>|Edge   | |
| |Device | |    (Loopback/Physical)     | |Device | |
| +------+ |                            | +------+ |
+-+--------+                            +--------+-+
  |                                              |
 Site VLAN                                  Site VLAN
 (local only)                               (local only)
```

### OTV Configuration

```bash
# Enable OTV
feature otv

# Configure OTV site VLAN (unique per site, never extended)
otv site-vlan 999

# Configure OTV overlay
otv site-identifier 0x1
interface overlay 1
  otv join-interface loopback 0
  otv control-group 239.1.1.1
  otv data-group 232.1.1.0/24
  otv extend-vlan 100,200
  no shutdown

# Site VLAN interface — must be up on a local trunk
interface vlan 999
  no shutdown

# Verify OTV
show otv
show otv overlay 1
show otv adjacency
show otv route
show otv vlan
show otv site
```

### OTV Key Concepts

| Concept          | Description                                                   |
|------------------|---------------------------------------------------------------|
| Site VLAN        | Local-only VLAN used to detect multi-homing within a site     |
| Join Interface   | Routable IP interface for OTV encapsulation (loopback/phys)   |
| AED              | Authoritative Edge Device — elected per VLAN for BUM traffic  |
| Overlay          | Logical tunnel carrying extended VLANs across transport       |
| Control Group    | Multicast group for OTV control plane (IS-IS)                 |
| Data Group       | Multicast range for encapsulated data traffic                 |

## Fabric Extender (FEX) — Nexus 2000

### Architecture

```
+--------------------+
|   Nexus 5000/9000  |   <-- Parent switch
|   (Management)     |
+--------+-----------+
         | Fabric Interface
         | (10G/40G uplinks)
+--------+-----------+
|   Nexus 2000 FEX   |   <-- Remote line card
|   FEX ID: 101      |
+--+-+-+-+-+-+-+--+--+
   | | | | | | |  |
  Host Interfaces (1G/10G)
  (Servers / endpoints)
```

### FEX Configuration

```bash
# Enable FEX feature
feature fex

# Pre-provision the FEX (before connecting)
slot 101
  provision model N2K-C2248TP

# Configure fabric (uplink) interface
interface ethernet 1/1
  switchport mode fex-fabric
  fex associate 101
  channel-group 101

interface port-channel 101
  switchport mode fex-fabric
  fex associate 101

# Configure host-facing interface on FEX
interface ethernet 101/1/1
  description SERVER-PORT
  switchport
  switchport mode access
  switchport access vlan 100

# Pinning — control which fabric link carries which host port
fex 101
  pinning max-links 1          ! static pinning
  description FEX-ROW-A-RACK1

# Verify FEX
show fex
show fex 101
show fex 101 detail
show interface ethernet 101/1/1
show fex version
```

### FEX Pinning Modes

| Mode                | Description                              |
|---------------------|------------------------------------------|
| max-links 1         | Static pinning, one fabric link per host |
| max-links (default) | Distribute across all fabric links       |

## NX-API

### Enabling NX-API

```bash
feature nxapi
nxapi http port 80
nxapi https port 443
nxapi sandbox            ! Enable the built-in API explorer
```

### NX-API Request Formats

```
+--------------------+------------------------------------------+
| Format             | Content-Type                             |
+--------------------+------------------------------------------+
| JSON-RPC           | application/json-rpc                     |
| CLI (JSON output)  | application/json                         |
| XML                | application/xml                          |
+--------------------+------------------------------------------+
```

### NX-API JSON-RPC Example

```json
{
  "jsonrpc": "2.0",
  "method": "cli",
  "params": {
    "cmd": "show vlan brief",
    "version": 1
  },
  "id": 1
}
```

### NX-API CLI via curl

```bash
# JSON-RPC show command
curl -k -u admin:password -X POST https://nexus-switch/ins \
  -H "Content-Type: application/json" \
  -d '{
    "ins_api": {
      "version": "1.0",
      "type": "cli_show",
      "chunk": "0",
      "sid": "1",
      "input": "show vlan brief",
      "output_format": "json"
    }
  }'

# Configuration via NX-API
curl -k -u admin:password -X POST https://nexus-switch/ins \
  -H "Content-Type: application/json" \
  -d '{
    "ins_api": {
      "version": "1.0",
      "type": "cli_conf",
      "chunk": "0",
      "sid": "1",
      "input": "interface loopback 99 ; ip address 10.99.99.1/32",
      "output_format": "json"
    }
  }'
```

### NX-API Python (nxapi-plumbing / requests)

```python
import requests, json, urllib3
urllib3.disable_warnings()

url = "https://nexus-switch/ins"
headers = {"Content-Type": "application/json"}
auth = ("admin", "password")

payload = {
    "ins_api": {
        "version": "1.0",
        "type": "cli_show",
        "chunk": "0",
        "sid": "1",
        "input": "show version",
        "output_format": "json"
    }
}

resp = requests.post(url, json=payload, headers=headers, auth=auth, verify=False)
data = resp.json()
print(json.dumps(data, indent=2))
```

## Key Show Commands

### System and Module

```bash
show version                        # NX-OS version, uptime, hardware
show module                         # Line cards, fabric modules, status
show inventory                      # Physical inventory (serial, PID)
show environment                    # Power, fans, temperature
show processes cpu sort             # CPU usage by process
show system resources               # Memory, CPU summary
show logging last 50                # Recent syslog messages
show feature                        # Enabled/disabled features
show running-config diff            # Changes since last save
show startup-config                 # Saved configuration
```

### vPC Commands

```bash
show vpc                            # vPC domain status, role, peer
show vpc brief                      # Summary of all vPC port-channels
show vpc role                       # Primary/secondary, priority
show vpc peer-keepalive             # Keepalive status and counters
show vpc consistency-parameters global  # Type-1 global checks
show vpc orphan-port                # Single-attached ports in vPC VLANs
show vpc statistics                 # vPC-related traffic stats
```

### VDC Commands

```bash
show vdc                            # All VDC names, IDs, state
show vdc current-vdc                # Which VDC you are in
show vdc resource                   # Resource allocation per VDC
show vdc membership                 # Interface-to-VDC mapping
show vdc resource template          # Available resource templates
```

### Interface and Routing

```bash
show interface brief                # One-line per interface status
show interface status               # Port status, VLAN, speed, duplex
show ip route vrf all               # All VRF routing tables
show ip bgp summary                 # BGP neighbor summary
show ip ospf neighbors              # OSPF adjacencies
show mac address-table              # MAC table
show cdp neighbors                  # CDP neighbor discovery
show lldp neighbors                 # LLDP neighbor discovery
```

### Spanning Tree and L2

```bash
show spanning-tree summary          # STP mode, root, port states
show vlan brief                     # VLAN list and assigned ports
show port-channel summary           # LAG status and members
show lacp counters                  # LACP PDU statistics
show interface trunk                # Trunk ports and allowed VLANs
```

## Tips

- Always enable features before configuring them; NX-OS will reject commands for disabled features
- Use `checkpoint` and `rollback` instead of relying solely on `copy run start` for safe changes
- The `show diff rollback-patch` command lets you preview exactly what a rollback will do
- vPC peer-keepalive should use the management VRF or a dedicated L3 link, never the peer-link
- Never carry the vPC peer-keepalive VLAN on the peer-link trunk
- For vPC, always configure `peer-gateway` to handle HSRP/VRRP traffic correctly
- FEX uplinks should always be port-channeled for redundancy
- Use `show vpc consistency-parameters global` after any vPC change to catch mismatches early
- NX-API sandbox (web GUI) is invaluable for testing API calls before scripting
- Use `| json` pipe on most show commands to get structured JSON output (e.g., `show vlan brief | json`)
- VDCs are only available on the Nexus 7000 platform
- OTV site VLAN must never be extended across the overlay
- FabricPath switch-IDs must be unique across the entire FabricPath domain
- Use `show forwarding adjacency` and `show forwarding route` to verify hardware programming

## See Also

- cisco-aci
- cisco-ios
- vxlan-evpn
- data-center-design
- network-automation

## References

- [Cisco NX-OS Configuration Guides](https://www.cisco.com/c/en/us/support/switches/nexus-9000-series-switches/products-installation-and-configuration-guides-list.html)
- [Cisco NX-OS Command Reference](https://www.cisco.com/c/en/us/support/switches/nexus-9000-series-switches/products-command-reference-list.html)
- [NX-API Documentation](https://developer.cisco.com/docs/nx-os/)
- [Cisco vPC Design Guide](https://www.cisco.com/c/en/us/products/collateral/switches/nexus-5000-series-switches/design_guide_c07-625857.html)
- [OTV Technology Overview](https://www.cisco.com/c/en/us/products/collateral/switches/nexus-7000-series-switches/white_paper_c11-729383.html)
- [FabricPath Design Guide](https://www.cisco.com/c/en/us/products/collateral/switches/nexus-7000-series-switches/guide_c07-690079.html)
