# Cisco UCS (Unified Computing System)

Manage stateless x86 compute infrastructure through centralized service profiles that abstract server identity from hardware, enabling rapid provisioning, failover, and policy-driven datacenter operations via UCS Manager or Intersight.

## Architecture Overview

### Core Components

```
                        ┌─────────────────────────────┐
                        │       UCS Manager (UCSM)    │
                        │    or Intersight (Cloud)     │
                        └──────────┬──────────────────┘
                                   │
                    ┌──────────────┴──────────────┐
                    │                             │
              ┌─────┴─────┐                ┌─────┴─────┐
              │  Fabric    │                │  Fabric    │
              │ Interconn  │                │ Interconn  │
              │  (FI-A)    │────────────────│  (FI-B)    │
              └─────┬─────┘   Cluster HA   └─────┬─────┘
                    │          Link(s)           │
        ┌───────────┼───────────┐    ┌───────────┼───────────┐
        │           │           │    │           │           │
   ┌────┴───┐ ┌────┴───┐ ┌────┴┐  ┌┴────┐ ┌────┴───┐ ┌────┴───┐
   │ IOM-A  │ │ IOM-A  │ │ IOM │  │ IOM │ │ IOM-B  │ │ IOM-B  │
   │Chassis1│ │Chassis2│ │  A  │  │  B  │ │Chassis2│ │Chassis1│
   └────┬───┘ └────┬───┘ └────┘  └─────┘ └────┬───┘ └────┬───┘
        │          │                            │          │
   ┌────┴───┐ ┌────┴───┐                  ┌────┴───┐ ┌────┴───┐
   │Blade 1 │ │Blade 1 │    Rack Servers   │Blade 1 │ │Blade 1 │
   │Blade 2 │ │Blade 2 │   (C-Series via   │Blade 2 │ │Blade 2 │
   │  ...   │ │  ...   │    FEX/direct)    │  ...   │ │  ...   │
   │Blade 8 │ │Blade 8 │                  │Blade 8 │ │Blade 8 │
   └────────┘ └────────┘                  └────────┘ └────────┘
```

### Fabric Interconnect (FI)

```bash
# Show FI cluster state
UCS-FI-A# show cluster state
# Output: Subordinate / Primary roles, HA readiness, management VIP

# Show FI inventory
UCS-FI-A# show fabric-interconnect inventory

# Check FI firmware
UCS-FI-A# show firmware monitor

# Show FI interfaces
UCS-FI-A# show interface brief

# Show ethanalyzer (packet capture on FI)
UCS-FI-A# ethanalyzer local interface mgmt capture-filter "host 10.1.1.50"

# Connect to local management
UCS-FI-A# connect local-mgmt
UCS-FI-A(local-mgmt)# show mgmt-ip
```

### Chassis and IOM

```bash
# Show chassis inventory
UCS-FI-A# show chassis inventory

# Show IOM (IO Module) status
UCS-FI-A# show iom detail

# Check IOM backplane connectivity
UCS-FI-A# show chassis iom 1/1

# Show server slots in chassis
UCS-FI-A# show server inventory

# Check chassis power supply status
UCS-FI-A# show chassis 1 psu detail
```

## UCS Manager (UCSM)

### Initial Setup

```bash
# UCSM is accessed via the Fabric Interconnect management IP
# Default: https://<FI-cluster-VIP>

# Initial setup wizard sets:
#   - System name
#   - Management IP pool (for FI, CIMC)
#   - Admin password
#   - DNS, NTP, timezone

# CLI access via SSH
ssh admin@<UCSM-VIP>

# Enter scope for configuration
UCS-FI-A# scope org /
UCS-FI-A /org # show service-profile status
```

### UCSM CLI Navigation

```bash
# Top-level scopes
UCS-FI-A# scope fabric a          # Fabric A configuration
UCS-FI-A# scope fabric b          # Fabric B configuration
UCS-FI-A# scope server 1/1        # Blade in chassis 1, slot 1
UCS-FI-A# scope server 2          # Rack server 2
UCS-FI-A# scope org /             # Root organization
UCS-FI-A# scope eth-uplink        # Ethernet uplink configuration
UCS-FI-A# scope fc-uplink         # Fibre Channel uplink config

# Show running configuration
UCS-FI-A# show configuration pending
UCS-FI-A# commit-buffer            # Apply pending changes
UCS-FI-A# discard-buffer           # Discard pending changes

# Backup configuration
UCS-FI-A# scope system
UCS-FI-A /system # backup create type=config-all protocol=scp
```

### Firmware Management

```bash
# Download firmware bundle to UCSM
# Use Infrastructure Bundle + B-Series/C-Series Bundle

# Check current firmware versions
UCS-FI-A# scope firmware
UCS-FI-A /firmware # show package

# Create host firmware policy
UCS-FI-A# scope org /
UCS-FI-A /org # create fw-host-pack FirmwarePolicy
UCS-FI-A /org/fw-host-pack # set blade-vers <bundle-version>
UCS-FI-A /org/fw-host-pack # commit-buffer

# Create infrastructure firmware pack
UCS-FI-A /org # create fw-infra-pack InfraFirmware
UCS-FI-A /org/fw-infra-pack # set infra-vers <bundle-version>
UCS-FI-A /org/fw-infra-pack # commit-buffer
```

## Service Profiles

### Concept

```
Service Profile = Server Identity Card
├── UUID          → Unique server identifier
├── MAC Addresses → Per-vNIC MAC
├── WWNN / WWPN   → Fibre Channel identities
├── Boot Policy   → Boot order and targets
├── BIOS Policy   → BIOS tuning parameters
├── vNICs         → Virtual network interfaces (LAN)
├── vHBAs         → Virtual host bus adapters (SAN)
├── Firmware      → Host firmware policy
├── Maintenance   → Reboot/acknowledge policy
├── Server Pool   → Which hardware to bind to
└── Storage       → Local disk, SAN LUN policies
```

### Creating a Service Profile

```bash
# Create a service profile from scratch
UCS-FI-A# scope org /
UCS-FI-A /org # create service-profile WebServer
UCS-FI-A /org/service-profile # set bios-policy HighPerformance
UCS-FI-A /org/service-profile # set boot-policy SAN-Boot
UCS-FI-A /org/service-profile # set host-fw-policy FirmwarePolicy
UCS-FI-A /org/service-profile # set uuid-pool UUIDPool
UCS-FI-A /org/service-profile # set maint-policy UserAck
UCS-FI-A /org/service-profile # commit-buffer

# Associate service profile to a specific blade
UCS-FI-A /org/service-profile # associate server 1/3
UCS-FI-A /org/service-profile # commit-buffer

# Associate using server pool
UCS-FI-A /org/service-profile # associate server-pool BladePool
UCS-FI-A /org/service-profile # commit-buffer

# Check association status
UCS-FI-A /org # show service-profile assoc
```

### Service Profile Templates

```bash
# Create an UPDATING template (changes push to derived profiles)
UCS-FI-A /org # create service-profile-template WebTemplate updating

# Create an INITIAL template (snapshot, no push)
UCS-FI-A /org # create service-profile-template DBTemplate initial

# Set template policies
UCS-FI-A /org/service-profile-template # set bios-policy HighPerformance
UCS-FI-A /org/service-profile-template # set boot-policy PXE-Local
UCS-FI-A /org/service-profile-template # set host-fw-policy FirmwarePolicy
UCS-FI-A /org/service-profile-template # commit-buffer

# Instantiate service profiles from template
UCS-FI-A /org # create service-profile-from-template Web01 WebTemplate
UCS-FI-A /org # create service-profile-from-template Web02 WebTemplate
UCS-FI-A /org # commit-buffer
```

## Pools

### UUID Pool

```bash
# Create UUID pool
UCS-FI-A /org # create uuid-suffix-pool UUIDPool
UCS-FI-A /org/uuid-suffix-pool # set assignment-order sequential
UCS-FI-A /org/uuid-suffix-pool # create block 0000-000000000001 0000-000000000050
UCS-FI-A /org/uuid-suffix-pool # commit-buffer

# Check pool usage
UCS-FI-A /org # show uuid-suffix-pool UUIDPool expand
```

### MAC Pool

```bash
# Create MAC address pool
UCS-FI-A /org # create mac-pool MAC-Pool-A
UCS-FI-A /org/mac-pool # create block 00:25:B5:A0:00:01 00:25:B5:A0:00:50
UCS-FI-A /org/mac-pool # commit-buffer

# Separate pool per fabric for traceability
UCS-FI-A /org # create mac-pool MAC-Pool-B
UCS-FI-A /org/mac-pool # create block 00:25:B5:B0:00:01 00:25:B5:B0:00:50
UCS-FI-A /org/mac-pool # commit-buffer
```

### WWNN and WWPN Pools

```bash
# Create WWNN pool (one per server)
UCS-FI-A /org # create wwn-pool WWNN-Pool node
UCS-FI-A /org/wwn-pool # create block 20:00:00:25:B5:00:00:01 20:00:00:25:B5:00:00:50
UCS-FI-A /org/wwn-pool # commit-buffer

# Create WWPN pool for Fabric A
UCS-FI-A /org # create wwn-pool WWPN-Pool-A port
UCS-FI-A /org/wwn-pool # create block 20:00:00:25:B5:A0:00:01 20:00:00:25:B5:A0:00:50
UCS-FI-A /org/wwn-pool # commit-buffer

# Create WWPN pool for Fabric B
UCS-FI-A /org # create wwn-pool WWPN-Pool-B port
UCS-FI-A /org/wwn-pool # create block 20:00:00:25:B5:B0:00:01 20:00:00:25:B5:B0:00:50
UCS-FI-A /org/wwn-pool # commit-buffer
```

### Server Pool

```bash
# Create a server pool
UCS-FI-A /org # create server-pool WebPool
UCS-FI-A /org/server-pool # create blade 1/1
UCS-FI-A /org/server-pool # create blade 1/2
UCS-FI-A /org/server-pool # create blade 1/3
UCS-FI-A /org/server-pool # create blade 1/4
UCS-FI-A /org/server-pool # commit-buffer

# Pool qualification (auto-assign by CPU, memory, disk)
UCS-FI-A /org # create server-pool-policy-qualif HighMemQual
UCS-FI-A /org/server-pool-policy-qualif # create memory min-cap 262144
UCS-FI-A /org/server-pool-policy-qualif # commit-buffer
```

## Policies

### BIOS Policy

```bash
# Create BIOS policy
UCS-FI-A /org # create bios-policy HighPerformance
UCS-FI-A /org/bios-policy # set hyperthreading enabled
UCS-FI-A /org/bios-policy # set turbo-mode enabled
UCS-FI-A /org/bios-policy # set power-technology performance
UCS-FI-A /org/bios-policy # set numa-optimized enabled
UCS-FI-A /org/bios-policy # set energy-perf-bias performance
UCS-FI-A /org/bios-policy # commit-buffer

# Virtualization-optimized BIOS
UCS-FI-A /org # create bios-policy Virtualization
UCS-FI-A /org/bios-policy # set intel-vt enabled
UCS-FI-A /org/bios-policy # set intel-vtd enabled
UCS-FI-A /org/bios-policy # set direct-cache-access enabled
UCS-FI-A /org/bios-policy # set vga-priority onboard
UCS-FI-A /org/bios-policy # commit-buffer
```

### Boot Policy

```bash
# SAN Boot policy
UCS-FI-A /org # create boot-policy SAN-Boot
UCS-FI-A /org/boot-policy # set boot-mode uefi
UCS-FI-A /org/boot-policy # create san primary
UCS-FI-A /org/boot-policy/san # create san-image primary vhba-name vHBA-A
UCS-FI-A /org/boot-policy/san/san-image # create boot-target 50:0A:09:82:89:AB:CD:EF lun 0
UCS-FI-A /org/boot-policy/san/san-image # commit-buffer

# PXE + Local Disk boot policy
UCS-FI-A /org # create boot-policy PXE-Local
UCS-FI-A /org/boot-policy # create lan primary order 1 vnic-name eth0
UCS-FI-A /org/boot-policy # create local-storage primary order 2
UCS-FI-A /org/boot-policy # commit-buffer

# iSCSI boot policy
UCS-FI-A /org # create boot-policy iSCSI-Boot
UCS-FI-A /org/boot-policy # create iscsi primary order 1 iscsi-vnic iSCSI-A
UCS-FI-A /org/boot-policy # commit-buffer
```

### Maintenance Policy

```bash
# User-acknowledge (manual reboot approval)
UCS-FI-A /org # create maint-policy UserAck
UCS-FI-A /org/maint-policy # set reboot-policy user-ack
UCS-FI-A /org/maint-policy # commit-buffer

# Immediate (auto-reboot on changes)
UCS-FI-A /org # create maint-policy Immediate
UCS-FI-A /org/maint-policy # set reboot-policy immediate
UCS-FI-A /org/maint-policy # commit-buffer

# Timer-based (scheduled maintenance windows)
UCS-FI-A /org # create maint-policy Scheduled
UCS-FI-A /org/maint-policy # set reboot-policy timer-automatic
UCS-FI-A /org/maint-policy # commit-buffer
```

## LAN Connectivity

### VLANs

```bash
# Create VLANs on FI
UCS-FI-A# scope eth-uplink
UCS-FI-A /eth-uplink # create vlan Production 100
UCS-FI-A /eth-uplink # create vlan Management 10
UCS-FI-A /eth-uplink # create vlan vMotion 200
UCS-FI-A /eth-uplink # create vlan Storage 300
UCS-FI-A /eth-uplink # commit-buffer
```

### vNIC Templates

```bash
# Create vNIC template for Fabric A
UCS-FI-A /org # create vnic-templ vNIC-Prod-A updating A
UCS-FI-A /org/vnic-templ # set mtu 9000
UCS-FI-A /org/vnic-templ # set mac-pool MAC-Pool-A
UCS-FI-A /org/vnic-templ # create eth-if Production default
UCS-FI-A /org/vnic-templ # set failover enabled            # Fabric failover
UCS-FI-A /org/vnic-templ # set cdn-source user-defined
UCS-FI-A /org/vnic-templ # set cdn-name ProdNetwork
UCS-FI-A /org/vnic-templ # commit-buffer

# Create vNIC template for Fabric B (failover pair)
UCS-FI-A /org # create vnic-templ vNIC-Prod-B updating B
UCS-FI-A /org/vnic-templ # set mtu 9000
UCS-FI-A /org/vnic-templ # set mac-pool MAC-Pool-B
UCS-FI-A /org/vnic-templ # create eth-if Production default
UCS-FI-A /org/vnic-templ # commit-buffer
```

### LAN Connectivity Policy

```bash
# Create LAN connectivity policy
UCS-FI-A /org # create lan-conn-policy LAN-Connectivity
UCS-FI-A /org/lan-conn-policy # create vnic eth0 vnic-templ vNIC-Prod-A adaptor-profile VMware
UCS-FI-A /org/lan-conn-policy # create vnic eth1 vnic-templ vNIC-Prod-B adaptor-profile VMware
UCS-FI-A /org/lan-conn-policy # commit-buffer
```

### Uplink Port Channels

```bash
# Create uplink port-channel
UCS-FI-A# scope fabric a
UCS-FI-A /fabric # create port-channel 10
UCS-FI-A /fabric/port-channel # create member-port 1/1
UCS-FI-A /fabric/port-channel # create member-port 1/2
UCS-FI-A /fabric/port-channel # commit-buffer

# Verify port-channel status
UCS-FI-A# show port-channel detail
```

## SAN Connectivity

### VSANs

```bash
# Create VSANs
UCS-FI-A# scope fc-uplink
UCS-FI-A /fc-uplink # create vsan VSAN-A 100 fabric a
UCS-FI-A /fc-uplink # create vsan VSAN-B 200 fabric b
UCS-FI-A /fc-uplink # commit-buffer

# Set FCOE VLAN IDs for VSANs
UCS-FI-A /fc-uplink # scope vsan VSAN-A
UCS-FI-A /fc-uplink/vsan # set fcoe-vlan 100
UCS-FI-A /fc-uplink/vsan # commit-buffer
```

### vHBA Templates

```bash
# Create vHBA template for Fabric A
UCS-FI-A /org # create vhba-templ vHBA-A updating A
UCS-FI-A /org/vhba-templ # set wwpn-pool WWPN-Pool-A
UCS-FI-A /org/vhba-templ # create fc-if VSAN-A
UCS-FI-A /org/vhba-templ # commit-buffer

# Create vHBA template for Fabric B
UCS-FI-A /org # create vhba-templ vHBA-B updating B
UCS-FI-A /org/vhba-templ # set wwpn-pool WWPN-Pool-B
UCS-FI-A /org/vhba-templ # create fc-if VSAN-B
UCS-FI-A /org/vhba-templ # commit-buffer
```

### SAN Connectivity Policy

```bash
# Create SAN connectivity policy
UCS-FI-A /org # create san-conn-policy SAN-Connectivity
UCS-FI-A /org/san-conn-policy # set wwnn-pool WWNN-Pool
UCS-FI-A /org/san-conn-policy # create vhba fc0 vhba-templ vHBA-A adaptor-profile VMware
UCS-FI-A /org/san-conn-policy # create vhba fc1 vhba-templ vHBA-B adaptor-profile VMware
UCS-FI-A /org/san-conn-policy # commit-buffer
```

## Fabric Failover

```bash
# Enable fabric failover on a vNIC
UCS-FI-A /org/vnic-templ # set failover enabled

# When FI-A path fails:
#   1. Traffic automatically shifts to FI-B path
#   2. MAC address migrates (same MAC, different physical path)
#   3. No OS-level reconfiguration required
#   4. Seamless to the hypervisor or OS

# Verify failover status
UCS-FI-A# show service-profile circuit server 1/1

# Check adapter paths
UCS-FI-A# scope server 1/1
UCS-FI-A /server # show adapter detail
```

## Server Provisioning Workflow

```bash
# Complete provisioning sequence:
#
# 1. Rack and cable hardware (chassis, blades, FIs)
# 2. Initial FI setup (cluster, management IP, NTP, DNS)
# 3. Discover chassis and blades in UCSM
# 4. Create pools (UUID, MAC, WWNN, WWPN, server)
# 5. Create policies (BIOS, boot, firmware, maintenance)
# 6. Create vNIC/vHBA templates
# 7. Create LAN/SAN connectivity policies
# 8. Create service profile template
# 9. Instantiate service profiles from template
# 10. Associate service profiles to blades (or pool)
# 11. Server reboots with new identity — ready for OS install

# Verify full provisioning status
UCS-FI-A /org # show service-profile status detail

# Check for faults
UCS-FI-A# show fault detail
UCS-FI-A# show fault | grep critical
```

## Intersight (Cloud Management)

```bash
# Intersight replaces UCSM for multi-domain management
# Claim device via device connector on each FI

# On FI: enable device connector
UCS-FI-A# scope device-connector
UCS-FI-A /device-connector # set admin-state enabled
UCS-FI-A /device-connector # commit-buffer

# Verify connectivity to Intersight cloud
UCS-FI-A /device-connector # show detail

# Intersight features beyond UCSM:
#   - Multi-domain management (many UCS domains, one pane)
#   - HyperFlex integration
#   - Kubernetes (IKS) orchestration
#   - Terraform provider (intersight_server_profile)
#   - REST API with OpenAPI spec
#   - Recommendation engine (firmware, workload optimization)
```

### Intersight API

```bash
# Intersight REST API examples (via curl with API key auth)

# List all servers
curl -s https://intersight.com/api/v1/compute/PhysicalSummaries \
  -H "Authorization: Bearer $INTERSIGHT_API_KEY" | jq '.Results[] | .Name'

# Get server profiles
curl -s https://intersight.com/api/v1/server/Profiles \
  -H "Authorization: Bearer $INTERSIGHT_API_KEY" | jq '.Results[] | {Name, Status}'

# Intersight Terraform provider
# provider "intersight" {
#   apikey    = var.api_key
#   secretkey = file(var.secret_key_path)
#   endpoint  = "https://intersight.com"
# }
```

## UCS Mini

```bash
# UCS Mini: FI-6324 integrated into UCS 5108 chassis
# - Up to 8 blades in single chassis
# - Built-in FI (no external FI required)
# - Ideal for remote/branch offices, edge compute
# - Same UCSM management as full-scale UCS
# - Supports up to 4 chassis (32 blades)

# UCS Mini limitations:
#   - Fewer uplink ports than 6300/6400 series FI
#   - No FCoE uplinks (FC direct only)
#   - Single chassis per FI pair (6324 specific)
#   - No unified ports on 6324
```

## Troubleshooting

### Common Diagnostics

```bash
# Show all faults sorted by severity
UCS-FI-A# show fault detail | head -100

# Show server health
UCS-FI-A# scope server 1/1
UCS-FI-A /server # show environment detail

# Check service profile FSM (Finite State Machine) status
UCS-FI-A /org # show service-profile fsm status

# Show SEL (System Event Log) for a blade
UCS-FI-A# scope server 1/1
UCS-FI-A /server # show sel

# Check discovery status
UCS-FI-A# show server discovery detail

# Verify adapter (CIMC) connectivity
UCS-FI-A# scope server 1/1
UCS-FI-A /server # show cimc detail

# Show tech-support (for TAC cases)
UCS-FI-A# show tech-support module 1
```

### Pool Exhaustion

```bash
# Check pool utilization
UCS-FI-A /org # show uuid-suffix-pool detail expand
UCS-FI-A /org # show mac-pool detail expand
UCS-FI-A /org # show wwn-pool detail expand

# Common fix: extend pool ranges
UCS-FI-A /org # scope mac-pool MAC-Pool-A
UCS-FI-A /org/mac-pool # create block 00:25:B5:A0:01:01 00:25:B5:A0:01:50
UCS-FI-A /org/mac-pool # commit-buffer
```

## Tips

- Always use separate MAC/WWPN pools per fabric (A vs B) for troubleshooting and traceability
- Use updating templates for environments where policy changes must propagate immediately; use initial templates for immutable deployments
- Set maintenance policy to user-ack in production to prevent unexpected reboots during service profile changes
- Enable fabric failover on vNICs to survive single FI failure without OS intervention
- Keep UUID, MAC, and WWN pools within the Cisco OUI range (00:25:B5) to avoid conflicts with physical NICs
- Back up the full UCSM configuration (all-config) before firmware upgrades
- Use server pool qualifications to auto-place workloads by hardware capability (CPU cores, memory, GPU)
- Pin vNICs to specific uplinks only when required by network design; unpinned gives better load distribution
- Create sub-organizations for multi-tenant environments to isolate pools and policies
- Monitor the FSM (Finite State Machine) during associations; a stuck FSM usually indicates a policy conflict or hardware fault

## See Also

- Cisco ACI
- VMware vSphere
- NetApp ONTAP
- Cisco Nexus
- Terraform

## References

- Cisco UCS Manager CLI Configuration Guide: https://www.cisco.com/c/en/us/support/servers-unified-computing/ucs-manager/products-installation-and-configuration-guides-list.html
- Cisco Intersight Documentation: https://intersight.com/help/saas
- Cisco UCS Hardware Installation Guide: https://www.cisco.com/c/en/us/support/servers-unified-computing/ucs-5100-series-blade-server-chassis/products-installation-guides-list.html
- Cisco UCS Central Documentation: https://www.cisco.com/c/en/us/support/servers-unified-computing/ucs-central-software/tsd-products-support-series-home.html
- UCS PowerTool Suite (PowerShell): https://community.cisco.com/t5/cisco-developed-ucs-integrations/cisco-ucs-powertool-suite/ta-p/3639523
- Cisco UCS Python SDK: https://github.com/CiscoUcs/ucsmsdk
