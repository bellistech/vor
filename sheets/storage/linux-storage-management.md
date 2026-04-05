# Linux Storage Management

Advanced storage configuration: device mapper, multipath, iSCSI, NFS, Stratis, VDO, LUKS.

## Device Mapper

### dmsetup Basics

```bash
# List all device-mapper devices
dmsetup ls

# Show table for a device
dmsetup table /dev/mapper/vg0-lv0

# Show status
dmsetup status

# Show device info
dmsetup info /dev/mapper/vg0-lv0

# Create a linear target manually
echo "0 $(blockdev --getsz /dev/sdb) linear /dev/sdb 0" | dmsetup create mydev

# Remove a device-mapper device
dmsetup remove mydev

# Suspend / resume (for table swaps)
dmsetup suspend mydev
dmsetup load mydev --table "0 2097152 linear /dev/sdc 0"
dmsetup resume mydev
```

### Device Mapper Striped Target

```bash
# Create 2-disk stripe (64K chunk)
echo "0 4194304 striped 2 128 /dev/sdb 0 /dev/sdc 0" | dmsetup create stripe0
# Fields: start_sector num_sectors striped num_stripes chunk_sectors dev1 offset1 dev2 offset2
# chunk_sectors = chunk_size_bytes / 512 (128 sectors = 64K)

# Create mirrored target (dm-raid1)
echo "0 2097152 mirror core 1 1024 2 /dev/sdb 0 /dev/sdc 0 1 handle_errors" | \
  dmsetup create mirror0
```

## Multipath I/O

### Installation and Service

```bash
# Install
dnf install device-mapper-multipath

# Generate default config
mpathconf --enable

# Start/enable
systemctl enable --now multipathd

# Show topology
multipath -ll

# Show all paths
multipath -v3

# Flush all unused multipath maps
multipath -F

# Reconfigure
multipath -r
```

### multipath.conf

```bash
# /etc/multipath.conf

defaults {
    polling_interval     5
    path_selector        "round-robin 0"
    path_grouping_policy multibus
    failback             immediate
    no_path_retry        5
    user_friendly_names  yes
}

blacklist {
    devnode "^(ram|raw|loop|fd|md|dm-|sr|scd|st)[0-9]*"
    devnode "^sd[a]$"   # boot disk
}

devices {
    device {
        vendor               "NETAPP"
        product              "LUN"
        path_grouping_policy group_by_prio
        path_selector        "round-robin 0"
        prio                 alua
        failback             immediate
    }
}

multipaths {
    multipath {
        wwid    3600508b4000c4a37000009000010000
        alias   data_lun0
    }
}
```

### Path Grouping Policies

```bash
# multibus         — all paths in one group (best throughput)
# failover         — one path per group (simple failover)
# group_by_serial  — group by storage controller serial
# group_by_prio    — group by ALUA priority
# group_by_node_name — group by target node name

# Path selectors
# "round-robin 0"       — rotate through paths
# "queue-length 0"      — least outstanding I/O
# "service-time 0"      — estimated shortest service time
```

### multipathd Interactive

```bash
multipathd -k
# Interactive commands:
#   show paths
#   show maps
#   show config
#   reconfigure
#   fail path sda
#   reinstate path sda
```

## iSCSI Initiator

### Installation

```bash
dnf install iscsi-initiator-utils

# Set initiator name
cat /etc/iscsi/initiatorname.iscsi
# InitiatorName=iqn.2024-01.com.example:client01

systemctl enable --now iscsid
```

### Discovery and Login

```bash
# Discover targets on portal
iscsiadm -m discovery -t sendtargets -p 192.168.1.100:3260

# List discovered targets
iscsiadm -m node

# Login to specific target
iscsiadm -m node \
  -T iqn.2024-01.com.example:storage.lun0 \
  -p 192.168.1.100:3260 \
  --login

# Login to all discovered targets
iscsiadm -m node --loginall=all

# Show active sessions
iscsiadm -m session

# Show session detail
iscsiadm -m session -P 3
```

### Persistent Configuration

```bash
# Enable automatic login on boot
iscsiadm -m node \
  -T iqn.2024-01.com.example:storage.lun0 \
  -p 192.168.1.100:3260 \
  --op update -n node.startup -v automatic

# Set CHAP authentication
iscsiadm -m node \
  -T iqn.2024-01.com.example:storage.lun0 \
  -p 192.168.1.100:3260 \
  --op update -n node.session.auth.authmethod -v CHAP
iscsiadm -m node \
  -T iqn.2024-01.com.example:storage.lun0 \
  -p 192.168.1.100:3260 \
  --op update -n node.session.auth.username -v myuser
iscsiadm -m node \
  -T iqn.2024-01.com.example:storage.lun0 \
  -p 192.168.1.100:3260 \
  --op update -n node.session.auth.password -v mypassword

# Logout from target
iscsiadm -m node \
  -T iqn.2024-01.com.example:storage.lun0 \
  -p 192.168.1.100:3260 \
  --logout

# Delete discovered target record
iscsiadm -m node \
  -T iqn.2024-01.com.example:storage.lun0 \
  -p 192.168.1.100:3260 \
  --op delete
```

## iSCSI Target (LIO/targetcli)

### Installation

```bash
dnf install targetcli

systemctl enable --now target
```

### targetcli Configuration

```bash
targetcli

# Create a backstore (block device)
/backstores/block create disk0 /dev/sdb

# Create a backstore (file-backed)
/backstores/fileio create file0 /var/lib/iscsi/file0.img 10G

# Create iSCSI target
/iscsi create iqn.2024-01.com.example:storage

# Create portal (listener)
/iscsi/iqn.2024-01.com.example:storage/tpg1/portals create 192.168.1.100 3260

# Create LUN
/iscsi/iqn.2024-01.com.example:storage/tpg1/luns create /backstores/block/disk0

# Create ACL (initiator access)
/iscsi/iqn.2024-01.com.example:storage/tpg1/acls create iqn.2024-01.com.example:client01

# Set CHAP on ACL
/iscsi/iqn.2024-01.com.example:storage/tpg1/acls/iqn.2024-01.com.example:client01 set auth userid=myuser
/iscsi/iqn.2024-01.com.example:storage/tpg1/acls/iqn.2024-01.com.example:client01 set auth password=mypassword

# Save and exit
saveconfig
exit
```

### Verify Target

```bash
# Show full config tree
targetcli ls

# Saved config location
cat /etc/target/saveconfig.json

# Firewall
firewall-cmd --permanent --add-service=iscsi-target
firewall-cmd --reload
```

## NFS

### Server (Exports)

```bash
# Install
dnf install nfs-utils

# /etc/exports
/data       192.168.1.0/24(rw,sync,no_subtree_check,root_squash)
/shared     *(ro,sync,no_subtree_check)
/home       192.168.1.0/24(rw,sync,no_root_squash)

# Export options explained:
#   rw/ro            — read-write or read-only
#   sync/async       — sync writes to disk before replying
#   root_squash      — map root to nobody (default, secure)
#   no_root_squash   — allow remote root (use sparingly)
#   all_squash       — map all users to nobody
#   anonuid/anongid  — UID/GID for squashed users
#   no_subtree_check — disable subtree checking (performance)

# Apply exports
exportfs -a

# Show current exports
exportfs -v

# Unexport a single share
exportfs -u 192.168.1.0/24:/data

# Enable and start
systemctl enable --now nfs-server

# Firewall
firewall-cmd --permanent --add-service=nfs
firewall-cmd --permanent --add-service=mountd
firewall-cmd --permanent --add-service=rpc-bind
firewall-cmd --reload
```

### Client (Mounts)

```bash
# Show exports from server
showmount -e 192.168.1.100

# Mount NFS share
mount -t nfs 192.168.1.100:/data /mnt/data

# Mount NFSv4 specifically
mount -t nfs4 192.168.1.100:/data /mnt/data

# Mount with options
mount -t nfs -o rw,hard,intr,timeo=600,retrans=2 192.168.1.100:/data /mnt/data

# fstab entry
# 192.168.1.100:/data  /mnt/data  nfs  defaults,_netdev  0 0

# autofs (on-demand)
# /etc/auto.master:
#   /mnt/nfs  /etc/auto.nfs
# /etc/auto.nfs:
#   data  -rw,sync  192.168.1.100:/data
```

## Stratis

### Pool and Filesystem Management

```bash
# Install
dnf install stratisd stratis-cli
systemctl enable --now stratisd

# Create pool
stratis pool create mypool /dev/sdb /dev/sdc

# Add device to pool (data tier)
stratis pool add-data mypool /dev/sdd

# Add cache device
stratis pool add-cache mypool /dev/nvme0n1

# List pools
stratis pool list

# Create filesystem
stratis filesystem create mypool myfs

# List filesystems
stratis filesystem list

# Snapshot
stratis filesystem snapshot mypool myfs myfs-snap

# Mount (uses /dev/stratis/mypool/myfs)
mount /dev/stratis/mypool/myfs /mnt/myfs

# fstab (use x-systemd.requires=stratisd.service)
# /dev/stratis/mypool/myfs  /mnt/myfs  xfs  defaults,x-systemd.requires=stratisd.service  0 0

# Destroy filesystem
stratis filesystem destroy mypool myfs

# Destroy pool
stratis pool destroy mypool
```

## VDO (Virtual Data Optimizer)

### Create and Manage

```bash
# Install
dnf install vdo kmod-kvdo

# Create VDO volume
vdo create --name=vdo0 \
  --device=/dev/sdb \
  --vdoLogicalSize=100G \
  --writePolicy=auto

# Format and mount
mkfs.xfs -K /dev/mapper/vdo0
mount /dev/mapper/vdo0 /mnt/vdo

# Show stats (dedup and compression)
vdostats --human-readable

# Sample output:
# Device         1K-blocks  Used    Available  Use%  Space saving%
# /dev/mapper/vdo0  50G     12G      38G       24%       65%

# Start/stop
vdo start --name=vdo0
vdo stop --name=vdo0

# Show status
vdo status --name=vdo0

# Enable/disable compression
vdo enableCompression --name=vdo0
vdo disableCompression --name=vdo0

# Enable/disable deduplication
vdo enableDeduplication --name=vdo0
vdo disableDeduplication --name=vdo0

# fstab
# /dev/mapper/vdo0  /mnt/vdo  xfs  defaults,x-systemd.requires=vdo.service  0 0

# Remove VDO volume
vdo remove --name=vdo0
```

## LUKS Encryption

### Create and Manage

```bash
# Format device with LUKS
cryptsetup luksFormat /dev/sdb

# Open (decrypt)
cryptsetup luksOpen /dev/sdb secret_disk
# Creates /dev/mapper/secret_disk

# Format and mount
mkfs.xfs /dev/mapper/secret_disk
mount /dev/mapper/secret_disk /mnt/encrypted

# Close
umount /mnt/encrypted
cryptsetup luksClose secret_disk

# Add additional key
cryptsetup luksAddKey /dev/sdb

# Remove a key slot
cryptsetup luksRemoveKey /dev/sdb

# Show key slot info
cryptsetup luksDump /dev/sdb

# Persistent: /etc/crypttab
# secret_disk  /dev/sdb  none  luks
# Then add fstab entry for /dev/mapper/secret_disk

# Key file instead of passphrase
dd if=/dev/urandom of=/root/luks-key bs=256 count=1
chmod 400 /root/luks-key
cryptsetup luksAddKey /dev/sdb /root/luks-key

# /etc/crypttab with key file
# secret_disk  /dev/sdb  /root/luks-key  luks
```

## fstab and systemd.mount

### fstab

```bash
# /etc/fstab format:
# device          mountpoint  fstype  options         dump  fsck
/dev/sda1          /boot       xfs     defaults        0     1
/dev/mapper/vg-root /          xfs     defaults        0     1
UUID=abc-123       /data       ext4    defaults,noatime 0    2
LABEL=backup       /backup     xfs     defaults        0     2

# Find UUID
blkid /dev/sdb1

# Find LABEL
blkid -s LABEL /dev/sdb1

# Mount all fstab entries
mount -a

# Common options:
#   defaults     — rw,suid,dev,exec,auto,nouser,async
#   noatime      — don't update access time
#   nodiratime   — don't update dir access time
#   nofail       — don't fail boot if missing
#   _netdev      — wait for network (NFS, iSCSI)
#   x-systemd.requires=unit — depend on systemd unit
```

### systemd.mount

```bash
# /etc/systemd/system/mnt-data.mount
[Unit]
Description=Mount /mnt/data
After=local-fs.target

[Mount]
What=/dev/sdb1
Where=/mnt/data
Type=xfs
Options=defaults,noatime

[Install]
WantedBy=multi-user.target

# Enable
systemctl enable --now mnt-data.mount

# Automount (mount on first access)
# /etc/systemd/system/mnt-data.automount
[Unit]
Description=Automount /mnt/data

[Automount]
Where=/mnt/data
TimeoutIdleSec=600

[Install]
WantedBy=multi-user.target
```

## Storage Troubleshooting

### Common Issues

```bash
# Device not showing
lsblk
lsscsi
cat /proc/partitions
dmesg | tail -30

# Rescan SCSI bus
echo "- - -" > /sys/class/scsi_host/host0/scan

# Rescan all hosts
for host in /sys/class/scsi_host/host*/scan; do echo "- - -" > "$host"; done

# Resize online device (after SAN LUN expansion)
echo 1 > /sys/block/sdb/device/rescan

# Multipath not detecting paths
multipath -v3
multipathd show paths
systemctl restart multipathd

# iSCSI session issues
iscsiadm -m session -P 3
iscsiadm -m node -T <target> -p <portal> --rescan

# NFS mount hanging
mount -v -t nfs 192.168.1.100:/data /mnt/data
rpcinfo -p 192.168.1.100
showmount -e 192.168.1.100

# Filesystem check (unmount first!)
umount /dev/sdb1
fsck -y /dev/sdb1
xfs_repair /dev/sdb1

# Check LUKS status
cryptsetup status secret_disk
cryptsetup luksDump /dev/sdb

# VDO recovery
vdo status --name=vdo0
vdo start --name=vdo0 --forceRebuild
```

## See Also

- lvm
- filesystems
- disk-management
- luks

## References

- man dmsetup, multipath, multipath.conf, iscsiadm, targetcli
- man cryptsetup, vdo, stratis
- Red Hat Storage Administration Guide
- kernel.org Device Mapper documentation
