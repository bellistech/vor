# SAN Storage (Storage Area Networking)

> Dedicated high-speed network providing block-level access to consolidated storage, decoupling storage from individual servers for performance, scalability, and centralized management.

---

## Storage Architecture Comparison

### SAN vs NAS vs DAS

| Feature          | DAS                        | NAS                         | SAN                              |
|------------------|----------------------------|-----------------------------|----------------------------------|
| Access type      | Block                      | File (NFS, SMB/CIFS)        | Block                            |
| Network          | Direct attach (SAS, SATA)  | Ethernet (TCP/IP)           | FC, iSCSI, NVMe-oF              |
| Protocol         | SCSI, NVMe                 | NFS, SMB, AFP               | FCP, iSCSI, NVMe-oF             |
| Sharing          | Single server              | Multi-client                | Multi-server                     |
| Filesystem       | Host manages               | NAS appliance manages       | Host manages                     |
| Scalability      | Limited                    | Moderate                    | High                             |
| Typical use      | Local workstation          | File shares, home dirs      | Databases, VMs, mission-critical |
| Latency          | Lowest (local bus)         | Moderate                    | Low (dedicated fabric)           |
| Cost             | Low                        | Moderate                    | High                             |

### Block vs File vs Object

| Attribute      | Block                     | File                       | Object                          |
|----------------|---------------------------|----------------------------|---------------------------------|
| Unit           | Fixed-size blocks (LBAs)  | Files in directories       | Objects in flat namespace       |
| Metadata       | Minimal (LBA only)        | POSIX attrs, ACLs          | Rich custom metadata            |
| Protocol       | SCSI, NVMe, iSCSI, FC     | NFS, SMB                   | S3, Swift                       |
| Performance    | Highest IOPS              | Good throughput             | High throughput, higher latency |
| Best for       | Databases, VMs            | Shared files, media        | Backups, archives, cloud-native |

---

## SCSI Concepts

### Core Terminology

```
Initiator     Host HBA or software driver that sends SCSI commands
Target        Storage controller or port that receives commands
LUN           Logical Unit Number — addressable storage unit on a target
WWNN          World Wide Node Name — unique identifier for a node
WWPN          World Wide Port Name — unique identifier for a port
CDB           Command Descriptor Block — the SCSI command structure
```

### SCSI Command Flow

```
Initiator  ---->  CDB (Read/Write)  ---->  Target
           <----  Status + Data     <----
```

### Common SCSI Commands

```
INQUIRY                   Identify device type and capabilities
TEST UNIT READY           Check if LUN is accessible
READ(10) / READ(16)       Read blocks from LUN
WRITE(10) / WRITE(16)     Write blocks to LUN
REPORT LUNS               List available LUNs on a target
MODE SENSE / SELECT       Query or set device parameters
PERSISTENT RESERVE        Cluster-aware locking (SCSI-3 PR)
```

---

## Fibre Channel SAN

### Architecture Layers

```
FC-4    Upper Layer Protocol (FCP for SCSI, FICON for mainframe)
FC-3    Common services (multicast, striping — mostly unused)
FC-2    Framing, flow control, classes of service
FC-1    Encoding (8b/10b for 1-8G, 64b/66b for 16G+)
FC-0    Physical layer (optics, cables, connectors)
```

### FC Speeds

```
1 GFC      1.0625 Gbps       ~100 MB/s
2 GFC      2.125  Gbps       ~200 MB/s
4 GFC      4.25   Gbps       ~400 MB/s
8 GFC      8.5    Gbps       ~800 MB/s
16 GFC     14.025 Gbps       ~1.6 GB/s
32 GFC     28.05  Gbps       ~3.2 GB/s
64 GFC     57.2   Gbps       ~6.4 GB/s
```

### FC Topologies

```
Point-to-Point (N_Port)     Direct connection between two devices
Arbitrated Loop (NL_Port)   Legacy shared loop (max 127 devices, avoid)
Switched Fabric (F_Port)    Standard production topology using FC switches
```

### Zoning

```
# Zoning types
Hard zoning         Enforced in switch ASIC hardware — most secure
Soft zoning         Enforced in name server — less secure
Mixed               Hard + soft together

# Zoning methods
WWPN zoning         Zone by port world-wide name (preferred, portable)
Port zoning         Zone by switch port number (breaks on cable moves)
Device alias        Friendly names mapped to WWPNs for readability

# Best practices
- Single initiator zoning: one initiator + one or more targets per zone
- Never mix initiators in the same zone
- Use device aliases for maintainability
- Document every zone with naming convention: host_hba_array_port
```

### Zoning Configuration (Brocade Example)

```bash
# Create alias
alicreate "srv01_hba0", "50:00:00:00:00:00:00:01"
alicreate "array01_p0", "50:00:00:00:00:00:01:00"

# Create zone
zonecreate "srv01_hba0_array01_p0", "srv01_hba0;array01_p0"

# Add zone to config
cfgadd "production_cfg", "srv01_hba0_array01_p0"

# Enable config
cfgenable "production_cfg"

# Save config
cfgsave
```

### Zoning Configuration (Cisco MDS Example)

```
# Device aliases
device-alias database
  device-alias name srv01_hba0 pwwn 50:00:00:00:00:00:00:01
  device-alias name array01_p0 pwwn 50:00:00:00:00:00:01:00

# Create zoneset and zone
zone name srv01_to_array01 vsan 100
  member device-alias srv01_hba0
  member device-alias array01_p0

zoneset name production vsan 100
  member srv01_to_array01

zoneset activate name production vsan 100
```

---

## iSCSI

### Architecture

```
iSCSI Initiator  -->  IP Network (Ethernet)  -->  iSCSI Target
  (host)                 (switch)                  (storage array)
```

### Key Terminology

```
IQN           iSCSI Qualified Name (iqn.yyyy-mm.com.domain:identifier)
EUI           Extended Unique Identifier (eui.xxxxxxxxxxxxxxxx)
Portal        IP:port combination for target access (default port 3260)
TPG           Target Portal Group — set of portals on a target
ISID          Initiator Session ID
TSIH          Target Session Identifying Handle
```

### Linux iSCSI Initiator (open-iscsi)

```bash
# Install
apt install open-iscsi        # Debian/Ubuntu
yum install iscsi-initiator-utils  # RHEL/CentOS

# Set initiator name
echo "InitiatorName=iqn.2026-01.com.example:srv01" > /etc/iscsi/initiatorname.iscsi

# Discover targets
iscsiadm -m discovery -t sendtargets -p 192.168.10.100:3260

# List discovered targets
iscsiadm -m node -o show

# Login to a target
iscsiadm -m node -T iqn.2026-01.com.example:array01 -p 192.168.10.100 --login

# Set automatic login on boot
iscsiadm -m node -T iqn.2026-01.com.example:array01 -p 192.168.10.100 \
  -o update -n node.startup -v automatic

# Logout
iscsiadm -m node -T iqn.2026-01.com.example:array01 -p 192.168.10.100 --logout

# Show active sessions
iscsiadm -m session -o show

# Rescan for new LUNs
iscsiadm -m session --rescan
```

### CHAP Authentication

```bash
# One-way CHAP (target authenticates initiator)
iscsiadm -m node -T iqn.2026-01.com.example:array01 -p 192.168.10.100 \
  -o update -n node.session.auth.authmethod -v CHAP
iscsiadm -m node -T iqn.2026-01.com.example:array01 -p 192.168.10.100 \
  -o update -n node.session.auth.username -v initiator_user
iscsiadm -m node -T iqn.2026-01.com.example:array01 -p 192.168.10.100 \
  -o update -n node.session.auth.password -v s3cretP@ss

# Mutual CHAP (bidirectional authentication)
iscsiadm -m node -T iqn.2026-01.com.example:array01 -p 192.168.10.100 \
  -o update -n node.session.auth.username_in -v target_user
iscsiadm -m node -T iqn.2026-01.com.example:array01 -p 192.168.10.100 \
  -o update -n node.session.auth.password_in -v t@rgetS3cret

# Discovery CHAP (authenticate during discovery phase)
iscsiadm -m discovery -t sendtargets -p 192.168.10.100 \
  --op update -n discovery.sendtargets.auth.authmethod -v CHAP
iscsiadm -m discovery -t sendtargets -p 192.168.10.100 \
  --op update -n discovery.sendtargets.auth.username -v disc_user
iscsiadm -m discovery -t sendtargets -p 192.168.10.100 \
  --op update -n discovery.sendtargets.auth.password -v d1scP@ss
```

### Linux iSCSI Target (targetcli)

```bash
# Install
apt install targetcli-fb

# Launch targetcli
targetcli

# Create a backing store
/backstores/block create lun0 /dev/sdb
/backstores/fileio create lun1 /var/iscsi/disk1.img 10G

# Create iSCSI target
/iscsi create iqn.2026-01.com.example:array01

# Create LUN
/iscsi/iqn.2026-01.com.example:array01/tpg1/luns create /backstores/block/lun0

# Create ACL (restrict to specific initiator)
/iscsi/iqn.2026-01.com.example:array01/tpg1/acls create iqn.2026-01.com.example:srv01

# Set CHAP credentials on ACL
cd /iscsi/iqn.2026-01.com.example:array01/tpg1/acls/iqn.2026-01.com.example:srv01/
set auth userid=initiator_user
set auth password=s3cretP@ss

# Set portal (bind to specific IP)
/iscsi/iqn.2026-01.com.example:array01/tpg1/portals create 192.168.10.100

# Save and exit
saveconfig
exit
```

---

## NVMe-oF (NVMe over Fabrics)

### Transport Types

```
NVMe/FC       NVMe over Fibre Channel — leverages existing FC fabric
NVMe/RDMA     NVMe over RDMA (RoCE v2 or InfiniBand) — lowest latency
NVMe/TCP      NVMe over TCP — no special hardware needed, widest adoption
```

### NVMe Terminology

```
NQN             NVMe Qualified Name (nqn.yyyy-mm.com.vendor:identifier)
Subsystem       NVMe target namespace container
Namespace (NS)  Addressable storage unit (like a LUN)
NSID            Namespace ID (integer)
Controller      Initiator-to-subsystem session
ANA             Asymmetric Namespace Access (like ALUA for NVMe)
```

### Linux NVMe-oF Initiator

```bash
# Install nvme-cli
apt install nvme-cli

# Load kernel modules
modprobe nvme-fabrics
modprobe nvme-tcp         # for NVMe/TCP
modprobe nvme-rdma        # for NVMe/RDMA
modprobe nvme-fc          # for NVMe/FC

# Discover subsystems
nvme discover -t tcp -a 192.168.10.200 -s 4420

# Connect to a subsystem
nvme connect -t tcp -n nqn.2026-01.com.example:nvme-array \
  -a 192.168.10.200 -s 4420

# List connected controllers
nvme list

# Disconnect
nvme disconnect -n nqn.2026-01.com.example:nvme-array

# Show controller details
nvme id-ctrl /dev/nvme0

# Show namespace details
nvme id-ns /dev/nvme0n1
```

### Linux NVMe-oF Target (nvmet)

```bash
# Load target modules
modprobe nvmet
modprobe nvmet-tcp

# Create subsystem
mkdir -p /sys/kernel/config/nvmet/subsystems/nqn.2026-01.com.example:nvme-array
echo 1 > /sys/kernel/config/nvmet/subsystems/nqn.2026-01.com.example:nvme-array/attr_allow_any_host

# Create namespace
mkdir /sys/kernel/config/nvmet/subsystems/nqn.2026-01.com.example:nvme-array/namespaces/1
echo /dev/nvme0n1 > /sys/kernel/config/nvmet/subsystems/nqn.2026-01.com.example:nvme-array/namespaces/1/device_path
echo 1 > /sys/kernel/config/nvmet/subsystems/nqn.2026-01.com.example:nvme-array/namespaces/1/enable

# Create port (TCP transport)
mkdir /sys/kernel/config/nvmet/ports/1
echo 192.168.10.200 > /sys/kernel/config/nvmet/ports/1/addr_traddr
echo 4420 > /sys/kernel/config/nvmet/ports/1/addr_trsvcid
echo tcp > /sys/kernel/config/nvmet/ports/1/addr_trtype
echo ipv4 > /sys/kernel/config/nvmet/ports/1/addr_adrfam

# Link subsystem to port
ln -s /sys/kernel/config/nvmet/subsystems/nqn.2026-01.com.example:nvme-array \
  /sys/kernel/config/nvmet/ports/1/subsystems/
```

---

## Multipathing

### MPIO Concepts

```
Active/Active        All paths carry I/O simultaneously (round-robin)
Active/Passive       One path active, others standby (failover only)
ALUA                 Asymmetric Logical Unit Access — paths have priorities
                     (optimized, non-optimized, standby, unavailable)
Path group           Set of paths with same priority
```

### Linux Device Mapper Multipath

```bash
# Install
apt install multipath-tools

# Detect multipath devices
multipath -ll

# Show all paths
multipathd show paths

# Reconfigure
multipathd reconfigure

# Flush a specific map
multipath -f mpath0

# Force rescan
multipath -r
```

### /etc/multipath.conf

```
defaults {
    user_friendly_names  yes
    find_multipaths      yes
    path_grouping_policy multibus
    path_selector        "round-robin 0"
    failback             immediate
    no_path_retry        5
    polling_interval     5
}

blacklist {
    devnode "^sd[a]$"       # boot disk
    devnode "^(ram|raw|loop|fd|md|dm-|sr|scd|st)[0-9]*"
}

devices {
    device {
        vendor           "VENDOR"
        product          "PRODUCT"
        path_grouping_policy  group_by_prio
        prio             alua
        path_selector    "round-robin 0"
        hardware_handler "1 alua"
        failback         immediate
        rr_weight        uniform
        no_path_retry    queue
    }
}

multipaths {
    multipath {
        wwid     360000000000000001
        alias    data_lun0
    }
}
```

---

## Storage Tiering

```
Tier 0    NVMe / SSD (flash)        Highest IOPS, lowest latency
Tier 1    SAS 15K / 10K HDD         High performance spinning disk
Tier 2    NL-SAS / SATA 7.2K HDD    Capacity tier
Tier 3    Tape / Object / Archive    Cold storage, lowest cost

Auto-tiering     Array moves blocks between tiers based on heat map
Sub-LUN tiering  Granularity at chunk level (e.g., 256 KB - 1 MB extents)
Policy-based     Admin sets rules (e.g., DB volumes always Tier 0)
```

---

## Thin Provisioning

```bash
# Concept
Physical capacity:   10 TB allocated to array pool
Virtual capacity:    50 TB presented to hosts as LUNs
Actual written:       3 TB on disk

# Key mechanisms
Space reclamation    UNMAP/TRIM returns freed blocks to the pool
Threshold alerts     Warn at 70%, 80%, 90% pool usage
Over-provisioning    Ratio of virtual to physical (common: 3:1 to 5:1)

# Linux SCSI UNMAP
sg_unmap --lba=0 --num=1024 /dev/sdb
fstrim /mount/point          # filesystem-level UNMAP
fstrim -a                    # trim all mounted filesystems
```

---

## Snapshots and Replication

### Snapshots

```
Copy-on-Write (CoW)     Only changed blocks copied; fast creation, read penalty
Redirect-on-Write (RoW) New writes go to new location; no read penalty
Consistency group        Snapshot multiple LUNs atomically (for multi-volume apps)
Application-consistent   Quiesce app/filesystem before snap (VSS, fsfreeze)
```

### Replication

```
Synchronous replication
  - Write acknowledged only after both sites confirm
  - RPO = 0 (zero data loss)
  - Distance limited (~100 km due to latency)
  - Performance impact on write latency

Asynchronous replication
  - Write acknowledged after local commit; replicated in background
  - RPO > 0 (seconds to minutes of potential data loss)
  - No distance limitation
  - Minimal performance impact

Metro cluster / stretch cluster
  - Active/active across two sites
  - Synchronous replication underneath
  - Automatic failover

3-site replication
  - Site A <-> Site B (sync) + Site B -> Site C (async)
  - Protects against regional disaster
```

---

## RAID in SAN Context

```
RAID 1      Mirror               50% usable, best small random I/O
RAID 5      Single parity        (N-1) usable, good read, slow rebuild
RAID 6      Double parity        (N-2) usable, survives 2 disk failures
RAID 10     Striped mirrors      50% usable, best all-around performance
RAID-DP     NetApp dual parity   Similar to RAID 6, optimized for WAFL
RAID-TEC    NetApp triple parity Survives 3 disk failures
DDP         Dynamic Disk Pools   NetApp E-Series — distributed parity
Erasure coding                   Object storage alternative to RAID

# Modern arrays often use proprietary wide striping:
# - VRAID (Pure Storage)
# - ADAPT (HPE Alletra/Nimble)
# - RAID-Z (ZFS-based arrays)
# - Distributed RAID (Dell PowerStore, Hitachi VSP)
```

---

## SAN Design Best Practices

### Redundancy

```
Dual fabric          Fabric A + Fabric B — completely independent
Dual HBAs            One HBA per fabric per host
Dual controllers     Active/active or active/standby on array
Dual paths           Minimum 2 paths per LUN (4 preferred)
No single point      Every component from host to disk is redundant
ISL redundancy       Multiple inter-switch links between core/edge
```

### Fabric Isolation

```
Separate physical fabrics    Fabric A never touches Fabric B hardware
Separate VLANs (iSCSI)      Dedicated storage VLAN, no shared traffic
Jumbo frames (iSCSI)         MTU 9000 end-to-end for throughput
Flow control                 Enable PFC (Priority Flow Control) for lossless
Dedicated switches           Never converge SAN traffic on general LAN switches
```

### Zoning Best Practices

```
Single initiator zoning       1 host HBA + target ports per zone
Naming convention             site_host_hba_array_port
Device aliases                Map every WWPN to human-readable name
Document everything           Spreadsheet or DCIM of every zone and WWPN
```

---

## Tips

- Always use dual-fabric design; a single fabric is a single point of failure
- WWPN zoning is preferred over port zoning because it survives cable moves
- For iSCSI, use dedicated VLANs with jumbo frames (MTU 9000) end-to-end
- Enable CHAP authentication on iSCSI to prevent unauthorized LUN access
- Monitor thin provisioning pools; unexpected 100% full causes outages for all tenants
- Test failover paths regularly; a path you never tested is a path that will not work
- Use SCSI-3 Persistent Reservations (not SCSI-2 reserves) for clustered environments
- Keep firmware levels consistent across all switches in a fabric
- NVMe/TCP is the easiest NVMe-oF transport to deploy (no RDMA NICs or FC HBAs required)
- Snapshot != backup; always replicate snapshots to a separate system
- Async replication RPO depends on bandwidth; size your WAN link for peak write rate
- Label every cable, port, and HBA; SAN troubleshooting without labels is miserable
- Use multipath `no_path_retry queue` for critical LUNs so I/O waits instead of failing
- Run `multipathd show paths` after any fabric change to verify all paths are active

---

## See Also

- `sheets/storage/nas-storage.md` -- NAS protocols (NFS, SMB/CIFS)
- `sheets/storage/zfs.md` -- ZFS administration
- `sheets/storage/lvm.md` -- LVM logical volume management
- `sheets/networking/vlans.md` -- VLAN configuration for iSCSI networks
- `sheets/linux/block-devices.md` -- Block device management

---

## References

- SNIA (Storage Networking Industry Association): https://www.snia.org/
- T10 SCSI Standards: https://www.t10.org/
- NVMe Specification: https://nvmexpress.org/specifications/
- Fibre Channel (T11): https://www.t11.org/
- RFC 7143 — iSCSI Protocol: https://www.rfc-editor.org/rfc/rfc7143
- RFC 3720 — iSCSI (original): https://www.rfc-editor.org/rfc/rfc3720
- Linux SCSI Target Wiki: http://linux-iscsi.org/
- Open-iSCSI Project: https://github.com/open-iscsi/open-iscsi
- Linux Multipath Documentation: https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux/9/html/managing_storage_devices/assembly_multipath-device-mapper-multipathing_managing-storage-devices
- Brocade FOS Admin Guide: https://www.broadcom.com/
- Cisco MDS Configuration Guides: https://www.cisco.com/c/en/us/support/storage-networking/mds-9000-series-multilayer-switches/series.html
