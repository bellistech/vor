# df (Disk Free)

Report filesystem disk space usage.

## Basic Usage

```bash
df                                       # all mounted filesystems
df /                                     # specific mountpoint
df /dev/sda1                             # specific device
```

## Human-Readable Output

```bash
df -h                                    # powers of 1024 (KiB, MiB, GiB)
df -H                                    # powers of 1000 (KB, MB, GB)
```

## Filesystem Type

### Show Filesystem Type

```bash
df -T                                    # adds Type column
df -Th                                   # human-readable with type
```

### Filter by Type

```bash
df -t ext4                               # only ext4
df -t xfs -t ext4                        # ext4 and xfs
```

### Exclude by Type

```bash
df -x tmpfs                              # hide tmpfs
df -x tmpfs -x devtmpfs -x squashfs     # hide virtual filesystems
df -Th -x tmpfs -x devtmpfs -x squashfs # clean output on typical server
```

## Inode Usage

```bash
df -i                                    # inode usage (instead of blocks)
df -ih                                   # human-readable inodes
```

## Output Formatting

### Specific Columns

```bash
df --output=source,fstype,size,used,avail,pcent,target
df --output=source,pcent,target -h       # minimal view
```

### Available Output Fields

```bash
# source  - device or remote path
# fstype  - filesystem type
# size    - total size
# used    - used space
# avail   - available space
# pcent   - usage percentage
# target  - mount point
# itotal  - total inodes
# iused   - used inodes
# iavail  - available inodes
# ipcent  - inode usage percentage
```

## Filtering

### Local Filesystems Only

```bash
df -hl                                   # exclude network mounts
```

### Specific Mountpoints

```bash
df -h / /home /var
```

## Common Useful Combinations

```bash
# Clean server overview (hide virtual filesystems)
df -Th -x tmpfs -x devtmpfs -x squashfs -x overlay

# Check if any filesystem is over 80%
df -h | awk 'NR>1 && int($5)>80'

# Just percentage used for a specific mount
df --output=pcent / | tail -1

# Total disk usage across all real filesystems
df -hl --total | tail -1
```

## Tips

- `df` shows filesystem-level usage, not physical disk size; LVM, RAID, and thin provisioning add layers
- Inode exhaustion (`df -i` shows 100%) can make a filesystem "full" even with free space -- common with many small files
- On ext4, `tune2fs -m 0 /dev/sdb1` releases the reserved 5% space (reserved for root); use cautiously
- `df` can show different "available" than `used + avail = size` because ext4 reserves blocks by default
- For container environments, `df` inside a container shows the host filesystem unless the mount namespace differs
- Network filesystems (NFS, CIFS) appear in `df` output; use `-l` to exclude them
- `--output` is a GNU extension; on macOS/BSD, use column filtering with `awk` instead
- If `df` hangs, a network mount is likely unresponsive; use `df -l` to skip network mounts

## References

- [df(1) Man Page](https://man7.org/linux/man-pages/man1/df.1.html)
- [GNU Coreutils — df](https://www.gnu.org/software/coreutils/manual/html_node/df-invocation.html)
- [statvfs(3) Man Page](https://man7.org/linux/man-pages/man3/statvfs.3.html)
- [Arch Wiki — File Systems](https://wiki.archlinux.org/title/File_systems)
- [Ubuntu Manpage — df](https://manpages.ubuntu.com/manpages/noble/man1/df.1.html)
