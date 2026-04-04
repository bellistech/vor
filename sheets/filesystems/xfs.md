# XFS (XFS Filesystem)

High-performance 64-bit journaling filesystem designed for large files and parallel I/O.

## Create Filesystem

```bash
sudo mkfs.xfs /dev/sdb1

# With label
sudo mkfs.xfs -L mydata /dev/sdb1

# Force overwrite existing filesystem
sudo mkfs.xfs -f /dev/sdb1

# With specific block size
sudo mkfs.xfs -b size=4096 /dev/sdb1

# With specific sector size (for 4Kn drives)
sudo mkfs.xfs -s size=4096 /dev/sdb1

# Separate log device (external journal for performance)
sudo mkfs.xfs -l logdev=/dev/nvme0n1p1,size=512m /dev/sdb1
```

## Filesystem Info

```bash
# Filesystem geometry and features
sudo xfs_info /dev/sdb1
sudo xfs_info /mnt/data                  # can also use mount point
```

## Grow Filesystem (Online)

```bash
# XFS can only grow, never shrink
# After extending the partition or LV:
sudo xfs_growfs /mnt/data                # grow to fill partition
sudo xfs_growfs /mnt/data -D 200g        # grow to specific size

# Check new size
df -h /mnt/data
```

## Check and Repair (xfs_repair)

```bash
# Must be unmounted
sudo xfs_repair /dev/sdb1

# Dry run (check only, no changes)
sudo xfs_repair -n /dev/sdb1

# Force log zeroing (if journal is corrupted)
sudo xfs_repair -L /dev/sdb1            # WARNING: may lose recent data

# Verbose output
sudo xfs_repair -v /dev/sdb1
```

### Repair Workflow

```bash
# 1. Unmount
sudo umount /mnt/data

# 2. Try normal repair
sudo xfs_repair /dev/sdb1

# 3. If repair fails due to dirty log, mount and unmount to replay journal
sudo mount /dev/sdb1 /mnt/data && sudo umount /mnt/data
sudo xfs_repair /dev/sdb1

# 4. Last resort: zero the log (journal data lost)
sudo xfs_repair -L /dev/sdb1
```

## Administration (xfs_admin)

```bash
# Set or change label
sudo xfs_admin -L mydata /dev/sdb1

# Show UUID
sudo xfs_admin -u /dev/sdb1

# Generate new UUID
sudo xfs_admin -U generate /dev/sdb1

# Enable lazy counters (performance improvement)
sudo xfs_admin -c 1 /dev/sdb1
```

## Mount Options

```bash
# Recommended defaults
sudo mount -o defaults,noatime /dev/sdb1 /mnt/data

# High-performance (disable barriers for battery-backed RAID)
sudo mount -o noatime,nobarrier,logbufs=8,logbsize=256k /dev/sdb1 /mnt/data

# With external log device
sudo mount -o logdev=/dev/nvme0n1p1 /dev/sdb1 /mnt/data
```

### Key Mount Options

```bash
noatime       # don't update access times (performance)
inode64       # allocate inodes anywhere (default since kernel 3.7)
logbufs=8     # increase log buffers (2-8, default 8 for 256K log)
logbsize=256k # log buffer size (32K-256K)
allocsize=64m # preallocation size for streaming writes
nobarrier     # disable write barriers (only with battery-backed write cache)
discard       # enable continuous TRIM for SSDs
```

## Freeze and Thaw (Consistent Snapshots)

```bash
# Freeze filesystem (flush + halt new writes)
sudo xfs_freeze -f /mnt/data

# Take LVM snapshot, storage snapshot, etc.
sudo lvcreate -s -L 10G -n data_snap /dev/vg/data

# Thaw filesystem (resume writes)
sudo xfs_freeze -u /mnt/data
```

## Backup and Restore (xfsdump / xfsrestore)

```bash
# Full backup (level 0)
sudo xfsdump -l 0 -f /backup/data-full.dump /mnt/data

# Incremental backup (level 1)
sudo xfsdump -l 1 -f /backup/data-incr.dump /mnt/data

# Restore full backup
sudo xfsrestore -f /backup/data-full.dump /mnt/restore

# List contents of dump
sudo xfsrestore -t -f /backup/data-full.dump

# Restore single file
sudo xfsrestore -f /backup/data-full.dump -s path/to/file /mnt/restore
```

## Quotas

```bash
# Enable quotas at mount
sudo mount -o uquota,gquota /dev/sdb1 /mnt/data

# Or in fstab
# UUID=xxx  /data  xfs  defaults,uquota,gquota  0  2

# Set user quota (soft=80G, hard=100G)
sudo xfs_quota -x -c "limit bsoft=80g bhard=100g alice" /mnt/data

# Set group quota
sudo xfs_quota -x -c "limit -g bsoft=500g bhard=600g developers" /mnt/data

# Report usage
sudo xfs_quota -x -c "report -h" /mnt/data
sudo xfs_quota -x -c "report -uh" /mnt/data   # per-user
```

## Defragmentation

```bash
# Check fragmentation level
sudo xfs_db -c frag -r /dev/sdb1

# Defragment a single file
sudo xfs_fsr /mnt/data/largefile.db

# Defragment entire filesystem (runs for 2 hours by default)
sudo xfs_fsr /dev/sdb1

# Defragment for specific duration (seconds)
sudo xfs_fsr -t 3600 /dev/sdb1
```

## Debugging

```bash
# Interactive filesystem debugger
sudo xfs_db -r /dev/sdb1                 # read-only mode
# sb 0     -> show superblock
# frag     -> fragmentation report
# quit

# Filesystem metadata dump (for bug reports)
sudo xfs_metadump /dev/sdb1 metadata.dump
```

## Tips

- XFS cannot be shrunk, only grown; plan partition sizes accordingly or use LVM for flexibility
- `xfs_repair` must be run on an unmounted filesystem; try mount+umount first to replay the journal
- `xfs_repair -L` (zero log) is a last resort and can cause data loss; always try without `-L` first
- XFS excels at large files and parallel I/O; it is the default filesystem in RHEL/CentOS 7+
- Maximum filesystem size is 8 EiB; maximum file size is 8 EiB -- effectively unlimited
- `xfs_freeze` is called automatically by LVM when taking snapshots; no need to call it manually with LVM
- For SSDs, prefer `fstrim.timer` over the `discard` mount option; batch TRIM is more efficient
- `xfsdump` supports multi-level incremental backups (levels 0-9), unlike generic tools
- XFS allocates space in extents and uses delayed allocation by default, which improves performance for streaming writes
- The `allocsize` mount option pre-allocates space for write-heavy workloads; `64m` is a good starting point for large sequential writes

## See Also

- ext4
- btrfs
- mount
- fstab
- lvm
- zfs

## References

- [XFS Wiki](https://xfs.wiki.kernel.org/)
- [Kernel XFS Documentation](https://www.kernel.org/doc/html/latest/filesystems/xfs/)
- [mkfs.xfs(8) Man Page](https://man7.org/linux/man-pages/man8/mkfs.xfs.8.html)
- [xfs_repair(8) Man Page](https://man7.org/linux/man-pages/man8/xfs_repair.8.html)
- [xfs_growfs(8) Man Page](https://man7.org/linux/man-pages/man8/xfs_growfs.8.html)
- [xfs_info(8) Man Page](https://man7.org/linux/man-pages/man8/xfs_info.8.html)
- [xfsdump(8) Man Page](https://man7.org/linux/man-pages/man8/xfsdump.8.html)
- [xfs(5) Man Page](https://man7.org/linux/man-pages/man5/xfs.5.html)
- [Arch Wiki — XFS](https://wiki.archlinux.org/title/XFS)
- [Red Hat RHEL 9 — Managing XFS File Systems](https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux/9/html/managing_file_systems/getting-started-with-an-xfs-file-system_managing-file-systems)
