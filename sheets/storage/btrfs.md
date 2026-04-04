# Btrfs (B-tree File System)

Copy-on-write filesystem with subvolumes, snapshots, RAID, compression, and online management.

## Filesystem Creation

```bash
# Single disk
sudo mkfs.btrfs /dev/sdb

# Force overwrite existing filesystem
sudo mkfs.btrfs -f /dev/sdb

# Multiple disks with RAID1 metadata, RAID0 data
sudo mkfs.btrfs -m raid1 -d raid0 /dev/sdb /dev/sdc

# RAID1 for both metadata and data
sudo mkfs.btrfs -m raid1 -d raid1 /dev/sdb /dev/sdc

# With label
sudo mkfs.btrfs -L mydata /dev/sdb
```

## Mount and Info

```bash
sudo mount /dev/sdb /mnt/data
sudo mount -o compress=zstd /dev/sdb /mnt/data

sudo btrfs filesystem show
sudo btrfs filesystem show /mnt/data
sudo btrfs filesystem df /mnt/data       # actual usage by data/metadata/system
sudo btrfs filesystem usage /mnt/data    # comprehensive usage report
```

## Subvolumes

### Create Subvolume

```bash
sudo btrfs subvolume create /mnt/data/home
sudo btrfs subvolume create /mnt/data/var
sudo btrfs subvolume create /mnt/data/snapshots
```

### List Subvolumes

```bash
sudo btrfs subvolume list /mnt/data
sudo btrfs subvolume list -t /mnt/data   # table format with gen info
```

### Delete Subvolume

```bash
sudo btrfs subvolume delete /mnt/data/old_subvol
```

### Mount Specific Subvolume

```bash
sudo mount -o subvol=home /dev/sdb /home
sudo mount -o subvolid=256 /dev/sdb /home

# fstab entry
# UUID=xxxx  /home  btrfs  subvol=home,defaults  0  0
```

### Set Default Subvolume

```bash
# Get subvolume ID
sudo btrfs subvolume list /mnt/data

sudo btrfs subvolume set-default 256 /mnt/data
```

## Snapshots

### Create Snapshot

```bash
# Read-only snapshot
sudo btrfs subvolume snapshot -r /mnt/data/home /mnt/data/snapshots/home-2024-01-15

# Writable snapshot
sudo btrfs subvolume snapshot /mnt/data/home /mnt/data/snapshots/home-writable
```

### Restore from Snapshot

```bash
# Replace current subvolume with snapshot
sudo btrfs subvolume delete /mnt/data/home
sudo btrfs subvolume snapshot /mnt/data/snapshots/home-2024-01-15 /mnt/data/home
```

### Delete Snapshot

```bash
sudo btrfs subvolume delete /mnt/data/snapshots/home-2024-01-15
```

## Send and Receive (Backup / Replication)

### Send Snapshot to Another Disk

```bash
# Full send (snapshot must be read-only)
sudo btrfs send /mnt/data/snapshots/home-2024-01-15 | \
  sudo btrfs receive /mnt/backup/

# Incremental send
sudo btrfs send -p /mnt/data/snapshots/home-2024-01-15 \
  /mnt/data/snapshots/home-2024-01-16 | \
  sudo btrfs receive /mnt/backup/
```

### Send Over SSH

```bash
sudo btrfs send /mnt/data/snapshots/home-snap | \
  ssh backup-server sudo btrfs receive /mnt/backup/
```

### Send to File

```bash
sudo btrfs send /mnt/data/snapshots/home-snap | gzip > /backup/home.btrfs.gz
```

## Compression

```bash
# Mount with compression
sudo mount -o compress=zstd /dev/sdb /mnt/data
sudo mount -o compress=zstd:3 /dev/sdb /mnt/data    # zstd level 3

# Set per-subvolume (takes effect for new writes)
sudo btrfs property set /mnt/data/home compression zstd

# Compress existing data in-place
sudo btrfs filesystem defragment -r -czstd /mnt/data/home

# Check compression ratio
sudo compsize /mnt/data                  # needs compsize package
```

## Device Management

### Add Device

```bash
sudo btrfs device add /dev/sdc /mnt/data
sudo btrfs balance start /mnt/data       # redistribute data
```

### Remove Device

```bash
sudo btrfs device remove /dev/sdb /mnt/data
```

### Replace Device

```bash
sudo btrfs replace start /dev/sdb /dev/sdd /mnt/data
sudo btrfs replace status /mnt/data
```

### Convert RAID Profile

```bash
# Convert single disk to RAID1 after adding a device
sudo btrfs balance start -dconvert=raid1 -mconvert=raid1 /mnt/data
```

## Balance (Redistribute Data)

```bash
# Full balance
sudo btrfs balance start /mnt/data

# Balance only data chunks under 50% usage
sudo btrfs balance start -dusage=50 /mnt/data

# Check balance progress
sudo btrfs balance status /mnt/data

# Cancel balance
sudo btrfs balance cancel /mnt/data
```

## Scrub (Integrity Check)

```bash
# Start scrub (online, reads all data and verifies checksums)
sudo btrfs scrub start /mnt/data

# Check progress
sudo btrfs scrub status /mnt/data

# Cancel
sudo btrfs scrub cancel /mnt/data
```

## Repair and Check

```bash
# Check filesystem (must be unmounted or mounted read-only)
sudo btrfs check /dev/sdb

# Repair (last resort -- try check first)
sudo btrfs check --repair /dev/sdb

# Rescue: recover from common failures
sudo btrfs rescue super-recover /dev/sdb
sudo btrfs rescue chunk-recover /dev/sdb
```

## Quotas

```bash
# Enable quotas
sudo btrfs quota enable /mnt/data

# Set quota on subvolume
sudo btrfs qgroup limit 50G /mnt/data/home

# Show quota usage
sudo btrfs qgroup show /mnt/data
sudo btrfs qgroup show -reF /mnt/data   # human-readable, exclusive data
```

## Tips

- Always keep metadata as RAID1 (even on single-disk setups with `-m dup`) to prevent metadata loss
- `zstd` compression is the best default; it offers excellent ratio with minimal CPU overhead
- Snapshots are instant and nearly free; use them before risky operations (package upgrades, config changes)
- `btrfs check --repair` is a last resort and can make things worse; always try a read-only `btrfs check` first
- RAID5/6 on Btrfs still has known write-hole issues -- use RAID1 or RAID10 for production
- `balance` is not a repair tool; it redistributes data across chunks and is needed after adding/removing devices
- Btrfs can run out of metadata space before data space; `btrfs filesystem usage` shows both
- For root filesystem snapshots, use tools like `snapper` or `timeshift` which automate snapshot rotation
- Mount options like `noatime,compress=zstd,space_cache=v2` are recommended for general use

## See Also

- zfs
- lvm
- mdadm
- ext4
- xfs

## References

- [Btrfs Documentation](https://btrfs.readthedocs.io/)
- [btrfs(8) Man Page](https://man7.org/linux/man-pages/man8/btrfs.8.html)
- [btrfs-subvolume(8) Man Page](https://man7.org/linux/man-pages/man8/btrfs-subvolume.8.html)
- [btrfs-balance(8) Man Page](https://man7.org/linux/man-pages/man8/btrfs-balance.8.html)
- [btrfs-scrub(8) Man Page](https://man7.org/linux/man-pages/man8/btrfs-scrub.8.html)
- [Kernel Btrfs Documentation](https://www.kernel.org/doc/html/latest/filesystems/btrfs.html)
- [Arch Wiki — Btrfs](https://wiki.archlinux.org/title/Btrfs)
- [SUSE — Managing Btrfs File Systems](https://documentation.suse.com/sles/15-SP5/html/SLES-all/cha-filesystems.html)
- [Red Hat RHEL 9 — Btrfs](https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux/9/html/managing_file_systems/creating-a-btrfs-file-system_managing-file-systems)
- [Btrfs Wiki — FAQ](https://btrfs.wiki.kernel.org/index.php/FAQ)
