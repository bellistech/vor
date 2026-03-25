# ZFS (Zettabyte File System)

Combined filesystem and volume manager with built-in RAID, snapshots, compression, and data integrity.

## Pool Management

### Create Pool

```bash
# Single disk
sudo zpool create tank /dev/sdb

# Mirror (RAID1)
sudo zpool create tank mirror /dev/sdb /dev/sdc

# RAIDZ1 (single parity, like RAID5)
sudo zpool create tank raidz /dev/sdb /dev/sdc /dev/sdd

# RAIDZ2 (double parity, like RAID6)
sudo zpool create tank raidz2 /dev/sdb /dev/sdc /dev/sdd /dev/sde

# Use disk IDs instead of /dev/sd* (survives reboot)
sudo zpool create tank mirror /dev/disk/by-id/scsi-SATA_disk1 /dev/disk/by-id/scsi-SATA_disk2
```

### Pool Status

```bash
sudo zpool status                        # all pools with health
sudo zpool status tank                   # specific pool
sudo zpool list                          # usage summary
sudo zpool list -v                       # per-vdev breakdown
sudo zpool history tank                  # command history
```

### Destroy Pool

```bash
sudo zpool destroy tank
```

### Import and Export

```bash
sudo zpool export tank                   # safe to disconnect disks
sudo zpool import                        # list importable pools
sudo zpool import tank                   # import by name
sudo zpool import -d /dev/disk/by-id tank
```

## Datasets (Filesystems)

### Create Dataset

```bash
sudo zfs create tank/data
sudo zfs create tank/data/projects
sudo zfs create -o mountpoint=/var/lib/postgres tank/postgres
```

### List Datasets

```bash
sudo zfs list
sudo zfs list -r tank                    # recursive
sudo zfs list -t all                     # include snapshots
```

### Destroy Dataset

```bash
sudo zfs destroy tank/data/old
sudo zfs destroy -r tank/data            # recursive (children too)
```

### Set Mount Point

```bash
sudo zfs set mountpoint=/mnt/data tank/data
sudo zfs mount tank/data
sudo zfs unmount tank/data
```

## Properties

### Get and Set

```bash
sudo zfs get all tank/data
sudo zfs get compression,compressratio tank/data
sudo zfs set compression=lz4 tank/data
sudo zfs set atime=off tank/data
sudo zfs set quota=100G tank/data
sudo zfs set reservation=50G tank/data
```

### Common Properties

```bash
sudo zfs set compression=lz4 tank           # enable compression (inherited)
sudo zfs set atime=off tank                 # disable access time updates
sudo zfs set recordsize=1M tank/media       # large records for big files
sudo zfs set recordsize=8K tank/postgres    # small records for databases
sudo zfs set dedup=on tank/backups          # deduplication (RAM hungry)
sudo zfs set sync=disabled tank/scratch     # async writes (data loss risk)
```

## Snapshots

### Create Snapshot

```bash
sudo zfs snapshot tank/data@2024-01-15
sudo zfs snapshot -r tank@daily            # recursive (all datasets)
```

### List Snapshots

```bash
sudo zfs list -t snapshot
sudo zfs list -t snapshot -r tank/data
```

### Rollback to Snapshot

```bash
sudo zfs rollback tank/data@2024-01-15

# Rollback past intermediate snapshots (destroys them)
sudo zfs rollback -r tank/data@2024-01-10
```

### Destroy Snapshot

```bash
sudo zfs destroy tank/data@2024-01-15
```

### Access Snapshot Contents (Without Rollback)

```bash
ls /tank/data/.zfs/snapshot/2024-01-15/
```

### Clone (Writable Copy from Snapshot)

```bash
sudo zfs clone tank/data@2024-01-15 tank/data-clone
```

## Send and Receive (Backup / Replication)

### Full Send

```bash
sudo zfs send tank/data@snap1 | ssh backup-server sudo zfs recv pool/data
```

### Incremental Send

```bash
sudo zfs send -i tank/data@snap1 tank/data@snap2 | \
  ssh backup-server sudo zfs recv pool/data
```

### Send to File

```bash
sudo zfs send tank/data@snap1 > /backup/data-snap1.zfs
sudo zfs send tank/data@snap1 | gzip > /backup/data-snap1.zfs.gz
```

### Receive

```bash
sudo zfs recv tank/restored < /backup/data-snap1.zfs
sudo zfs recv -F tank/data < /backup/data-snap1.zfs   # force overwrite
```

## Scrub and Resilver

```bash
# Check data integrity (run monthly)
sudo zpool scrub tank

# Check scrub progress
sudo zpool status tank

# Cancel scrub
sudo zpool scrub -s tank
```

## Disk Replacement

### Replace a Failed Disk

```bash
# Check which disk failed
sudo zpool status tank

# Replace it
sudo zpool replace tank /dev/sdb /dev/sde

# If disk was already removed
sudo zpool replace tank old-disk-id /dev/sde
```

### Add Hot Spare

```bash
sudo zpool add tank spare /dev/sdf
```

### Add Cache (L2ARC) and Log (SLOG)

```bash
sudo zpool add tank cache /dev/nvme0n1p1
sudo zpool add tank log mirror /dev/nvme0n1p2 /dev/nvme1n1p2
```

## Quotas and Reservations

```bash
# Hard limit on dataset size
sudo zfs set quota=100G tank/data

# Guaranteed space for dataset
sudo zfs set reservation=50G tank/data

# Per-user and per-group quotas
sudo zfs set userquota@alice=50G tank/data
sudo zfs set groupquota@developers=200G tank/data

# Check usage
sudo zfs get userused@alice tank/data
```

## Tips

- Always use `lz4` compression; it is nearly free in CPU cost and saves substantial space
- Use disk IDs (`/dev/disk/by-id/`) not device names (`/dev/sd*`) to prevent pool corruption on disk reorder
- `dedup=on` requires ~5 GB of RAM per TB of data; avoid unless you have verified high deduplication ratios
- Never use `raidz` with SSDs that lack power-loss protection; use mirrors instead
- ZFS needs a minimum of 1 GB RAM per TB of storage for the ARC cache; 2 GB+ per TB is recommended
- ECC RAM is strongly recommended; ZFS checksums detect corruption but cannot fix it if RAM is the source
- `atime=off` is a significant performance improvement and should be set on every pool
- Snapshots are free until data diverges; they only consume space as blocks change
- `zpool scrub` is not a backup; it verifies on-disk integrity but does not protect against catastrophic loss
