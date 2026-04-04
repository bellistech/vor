# mount (Mount Filesystems)

Attach filesystems, network shares, and virtual filesystems to the directory tree.

## Basic Mount and Unmount

```bash
sudo mount /dev/sdb1 /mnt/data
sudo umount /mnt/data
sudo umount /dev/sdb1                    # can also unmount by device

# Lazy unmount (detach now, cleanup when not busy)
sudo umount -l /mnt/data

# Force unmount (dangerous, use for unresponsive NFS)
sudo umount -f /mnt/nfs
```

## Show Mounted Filesystems

```bash
mount                                    # all mounts
mount | grep sdb                         # filter
findmnt                                  # tree view (preferred)
findmnt -t ext4                          # filter by type
findmnt /mnt/data                        # check specific mountpoint
```

## Mount by UUID or Label

```bash
# Find UUID
sudo blkid /dev/sdb1

sudo mount UUID="a1b2c3d4-e5f6-7890-abcd-ef1234567890" /mnt/data
sudo mount LABEL="mydata" /mnt/data
```

## Mount Options

```bash
# Read-only
sudo mount -o ro /dev/sdb1 /mnt/data

# Read-write with specific options
sudo mount -o rw,noatime,noexec /dev/sdb1 /mnt/data

# Remount with different options (no unmount needed)
sudo mount -o remount,rw /mnt/data
```

### Common Options

```bash
# Performance
noatime      # don't update access time (big performance boost)
nodiratime   # don't update directory access time
relatime     # update atime only if older than mtime (default)

# Security
noexec       # prevent binary execution
nosuid       # ignore SUID/SGID bits
nodev        # ignore device files

# Error handling
errors=remount-ro   # remount read-only on error (ext4 default)
nofail       # don't fail boot if device is missing

# User access
user         # allow non-root users to mount
users        # allow any user to mount and unmount
```

## Bind Mounts

```bash
# Mount a directory to another location
sudo mount --bind /var/log /mnt/logs

# Read-only bind mount (two steps)
sudo mount --bind /var/log /mnt/logs
sudo mount -o remount,ro,bind /mnt/logs

# Recursive bind (includes sub-mounts)
sudo mount --rbind /home /mnt/home
```

## Loop Devices (ISO / Disk Images)

```bash
# Mount an ISO
sudo mount -o loop disk.iso /mnt/iso

# Mount a raw disk image
sudo mount -o loop,offset=1048576 disk.img /mnt/img
# offset = start_sector * 512 (use fdisk -l disk.img to find it)

# Mount a partition inside a disk image
sudo losetup --partscan --find --show disk.img
# Creates /dev/loop0p1, /dev/loop0p2, etc.
sudo mount /dev/loop0p1 /mnt/img
```

## Network Filesystems

### NFS

```bash
# Mount NFS share
sudo mount -t nfs nfs-server:/export/data /mnt/nfs

# With options
sudo mount -t nfs -o rw,hard,intr,timeo=600 nfs-server:/export/data /mnt/nfs

# NFSv4 specific
sudo mount -t nfs4 nfs-server:/data /mnt/nfs
```

### CIFS/SMB (Windows Shares)

```bash
# Basic mount
sudo mount -t cifs //fileserver/share /mnt/smb -o username=alice,password=secret

# With credentials file (more secure)
sudo mount -t cifs //fileserver/share /mnt/smb -o credentials=/root/.smbcredentials

# Credentials file format:
# username=alice
# password=secret
# domain=ACME
```

### SSHFS (FUSE)

```bash
sshfs alice@server:/home/alice /mnt/ssh
fusermount -u /mnt/ssh                   # unmount
```

## tmpfs (RAM Disk)

```bash
sudo mount -t tmpfs -o size=2G tmpfs /mnt/ramdisk

# Resize existing tmpfs
sudo mount -o remount,size=4G /mnt/ramdisk
```

## Propagation (for Containers)

```bash
# Private (default): sub-mounts not shared
sudo mount --make-private /mnt/data

# Shared: sub-mounts propagate bidirectionally
sudo mount --make-shared /mnt/data

# Slave: receives propagation, doesn't send
sudo mount --make-slave /mnt/data
```

## Troubleshooting

```bash
# Find what is using a mount point
sudo lsof +f -- /mnt/data
sudo fuser -vm /mnt/data

# Check filesystem before mounting
sudo fsck /dev/sdb1

# Mount all entries from fstab
sudo mount -a

# Debug mount issues
sudo mount -v /dev/sdb1 /mnt/data       # verbose output
dmesg | tail                             # kernel messages
```

## Tips

- Always use `findmnt` instead of parsing `mount` output; it handles edge cases and is more readable
- `noatime` is the single most impactful mount option for performance on read-heavy workloads
- `umount -l` (lazy) is useful for stuck mounts but can cause data loss if writes are pending
- Bind mounts do not survive reboot unless added to fstab with the `bind` option
- For NFS, `hard` mount is safer than `soft`; soft mounts can return corrupt data on timeout
- CIFS credentials files must be `chmod 600` to prevent password exposure
- tmpfs is lost on reboot by design; size defaults to 50% of RAM if not specified
- `mount --rbind` inside containers is how container runtimes expose host paths
- If `umount` says "target is busy", use `lsof` or `fuser` to find and stop the offending process

## See Also

- fstab
- fdisk
- parted
- df
- lvm
- ext4

## References

- [mount(8) Man Page](https://man7.org/linux/man-pages/man8/mount.8.html)
- [umount(8) Man Page](https://man7.org/linux/man-pages/man8/umount.8.html)
- [findmnt(8) Man Page](https://man7.org/linux/man-pages/man8/findmnt.8.html)
- [mount_namespaces(7) Man Page](https://man7.org/linux/man-pages/man7/mount_namespaces.7.html)
- [fstab(5) Man Page](https://man7.org/linux/man-pages/man5/fstab.5.html)
- [Arch Wiki — File Systems](https://wiki.archlinux.org/title/File_systems)
- [Red Hat RHEL 9 — Mounting File Systems](https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux/9/html/managing_file_systems/mounting-file-systems_managing-file-systems)
- [Kernel Filesystems Documentation](https://www.kernel.org/doc/html/latest/filesystems/)
