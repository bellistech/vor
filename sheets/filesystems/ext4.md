# ext4 (Fourth Extended Filesystem)

Default Linux filesystem with journaling, extents, and mature tooling.

## Create Filesystem

```bash
sudo mkfs.ext4 /dev/sdb1

# With label
sudo mkfs.ext4 -L mydata /dev/sdb1

# With specific block size
sudo mkfs.ext4 -b 4096 /dev/sdb1

# With specific inode ratio (more inodes for many small files)
sudo mkfs.ext4 -i 8192 /dev/sdb1        # one inode per 8K of space

# Lazy initialization (faster mkfs, init happens in background)
sudo mkfs.ext4 -E lazy_itable_init=1,lazy_journal_init=1 /dev/sdb1
```

## Filesystem Info

```bash
sudo dumpe2fs /dev/sdb1                  # full superblock and group info
sudo dumpe2fs -h /dev/sdb1              # superblock only (header)
sudo tune2fs -l /dev/sdb1               # same as dumpe2fs -h
```

## Tune Filesystem (tune2fs)

### Set Label

```bash
sudo tune2fs -L mydata /dev/sdb1
```

### Set Reserved Blocks

```bash
# Default reserves 5% for root -- reduce on large data partitions
sudo tune2fs -m 1 /dev/sdb1             # reserve 1%
sudo tune2fs -m 0 /dev/sdb1             # reserve 0% (data-only partition)
```

### Set Mount Count / Time Interval for fsck

```bash
sudo tune2fs -c 30 /dev/sdb1            # fsck every 30 mounts
sudo tune2fs -i 6m /dev/sdb1            # fsck every 6 months
sudo tune2fs -c 0 -i 0 /dev/sdb1        # disable periodic fsck
```

### Enable/Disable Features

```bash
# Enable directory indexing (dir_index)
sudo tune2fs -O dir_index /dev/sdb1

# Enable large file support
sudo tune2fs -O large_file /dev/sdb1

# Enable metadata checksums
sudo tune2fs -O metadata_csum /dev/sdb1

# List current features
sudo tune2fs -l /dev/sdb1 | grep features
```

### Set Default Mount Options

```bash
# These apply even if not specified in fstab
sudo tune2fs -o journal_data_writeback /dev/sdb1
sudo tune2fs -o acl,user_xattr /dev/sdb1
```

### Convert ext3 to ext4

```bash
sudo tune2fs -O extents,uninit_bg,dir_index,flex_bg /dev/sdb1
sudo e2fsck -fD /dev/sdb1
```

## Resize Filesystem

### Grow (Online, Filesystem Mounted)

```bash
# After growing the partition or LV
sudo resize2fs /dev/sdb1                 # grow to fill partition
sudo resize2fs /dev/sdb1 100G            # grow to specific size
```

### Shrink (Offline, Filesystem Unmounted)

```bash
sudo umount /mnt/data
sudo e2fsck -f /dev/sdb1                 # required before shrink
sudo resize2fs /dev/sdb1 50G             # shrink to 50G
# Then shrink the partition to match
```

## Check and Repair (e2fsck)

```bash
# Check filesystem (must be unmounted)
sudo e2fsck /dev/sdb1

# Force check even if clean
sudo e2fsck -f /dev/sdb1

# Automatically fix errors
sudo e2fsck -y /dev/sdb1                 # answer "yes" to all
sudo e2fsck -p /dev/sdb1                 # auto-fix safe problems only

# Check and optimize directories
sudo e2fsck -fD /dev/sdb1

# Check a mounted root filesystem (read-only, limited)
sudo e2fsck -n /dev/sda1
```

### Force fsck on Next Boot

```bash
sudo tune2fs -C 31 -c 30 /dev/sda1      # set mount count past check interval
# Or:
sudo touch /forcefsck                    # some distros honor this
```

## Mount Options

```bash
# Recommended for general use
sudo mount -o defaults,noatime /dev/sdb1 /mnt/data

# Performance-oriented
sudo mount -o noatime,data=writeback,barrier=0 /dev/sdb1 /mnt/data

# Data safety (default)
sudo mount -o data=ordered /dev/sdb1 /mnt/data
```

### Journal Modes

```bash
data=journal      # safest: data and metadata journaled (slowest)
data=ordered      # default: metadata journaled, data written before metadata
data=writeback    # fastest: metadata journaled, data order not guaranteed
```

### SSD Options

```bash
# In fstab for SSDs
UUID=xxx  /data  ext4  defaults,noatime,discard  0  2

# Or use periodic TRIM instead of continuous discard
sudo systemctl enable fstrim.timer
```

## Debugging

```bash
# Show superblock backup locations
sudo dumpe2fs /dev/sdb1 | grep -i superblock

# Mount using backup superblock (recovery)
sudo mount -o sb=32768 /dev/sdb1 /mnt/recovery

# Dump filesystem statistics
sudo dumpe2fs -h /dev/sdb1 | grep -E "Block count|Free blocks|Inode count|Free inodes"
```

## Tips

- ext4 supports volumes up to 1 EiB and files up to 16 TiB; for larger needs, use XFS
- The default 5% reserved space on a 4 TB drive wastes 200 GB; use `tune2fs -m 1` on data partitions
- Online resize (grow) works without unmounting; shrink always requires unmount and fsck
- `data=ordered` (default) is the best balance of safety and performance for most workloads
- `noatime` is the single biggest performance win as a mount option; `relatime` (default since 2.6.30) is a good compromise
- ext4 lazy initialization means a freshly formatted large disk may show high I/O for minutes after first mount
- Always run `e2fsck -f` before shrinking; `resize2fs` refuses to shrink without a clean fsck
- `metadata_csum` (enabled by default since e2fsprogs 1.44) adds per-block checksums; older kernels may not support it
- Journal recovery is automatic on mount; if it fails, `e2fsck` is the next step

## See Also

- xfs
- btrfs
- mount
- fstab
- lvm
- fdisk

## References

- [ext4 Wiki](https://ext4.wiki.kernel.org/)
- [Kernel ext4 Documentation](https://www.kernel.org/doc/html/latest/filesystems/ext4/)
- [mke2fs(8) Man Page](https://man7.org/linux/man-pages/man8/mke2fs.8.html)
- [tune2fs(8) Man Page](https://man7.org/linux/man-pages/man8/tune2fs.8.html)
- [e2fsck(8) Man Page](https://man7.org/linux/man-pages/man8/e2fsck.8.html)
- [dumpe2fs(8) Man Page](https://man7.org/linux/man-pages/man8/dumpe2fs.8.html)
- [resize2fs(8) Man Page](https://man7.org/linux/man-pages/man8/resize2fs.8.html)
- [ext4(5) Man Page](https://man7.org/linux/man-pages/man5/ext4.5.html)
- [Arch Wiki — ext4](https://wiki.archlinux.org/title/Ext4)
- [Red Hat RHEL 9 — Managing ext4 File Systems](https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux/9/html/managing_file_systems/getting-started-with-an-ext4-file-system_managing-file-systems)
