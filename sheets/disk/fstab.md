# fstab (Filesystem Table)

Configure persistent mounts in `/etc/fstab` that apply at boot.

## Format

```bash
# /etc/fstab
# <device>                                 <mountpoint>  <type>  <options>        <dump> <pass>
UUID=a1b2c3d4-e5f6-7890-abcd-ef1234567890  /             ext4    defaults         0      1
UUID=b2c3d4e5-f6a7-8901-bcde-f12345678901  /home         ext4    defaults,noatime 0      2
UUID=c3d4e5f6-a7b8-9012-cdef-123456789012  none          swap    sw               0      0
```

### Field Descriptions

```bash
# <device>     - UUID, LABEL, or device path
# <mountpoint> - where to mount (or "none" for swap)
# <type>       - filesystem type (ext4, xfs, btrfs, nfs, cifs, swap, tmpfs)
# <options>    - mount options (comma-separated)
# <dump>       - backup with dump (0=no, 1=yes) -- almost always 0
# <pass>       - fsck order (0=skip, 1=root, 2=other)
```

## Identify Devices

```bash
# Find UUID (preferred over /dev/sd* paths)
sudo blkid
sudo blkid /dev/sdb1

# Find LABEL
sudo blkid -s LABEL -o value /dev/sdb1

# List by-uuid symlinks
ls -la /dev/disk/by-uuid/
```

## Common Entries

### Root and Home (ext4)

```bash
UUID=a1b2c3d4-e5f6-7890-abcd-ef1234567890  /       ext4  errors=remount-ro  0  1
UUID=b2c3d4e5-f6a7-8901-bcde-f12345678901  /home    ext4  defaults,noatime   0  2
```

### XFS

```bash
UUID=d4e5f6a7-b890-1234-cdef-567890abcdef  /data   xfs   defaults,noatime   0  2
```

### Btrfs with Subvolume

```bash
UUID=e5f6a7b8-9012-3456-def0-1234567890ab  /       btrfs  subvol=@,defaults,compress=zstd  0  0
UUID=e5f6a7b8-9012-3456-def0-1234567890ab  /home   btrfs  subvol=@home,defaults,compress=zstd  0  0
```

### Swap

```bash
UUID=f6a7b890-1234-5678-ef01-234567890abc  none    swap   sw                 0  0

# Swap file
/swapfile                                   none    swap   sw                 0  0
```

### tmpfs (RAM Disk)

```bash
tmpfs  /tmp        tmpfs  defaults,noatime,size=2G      0  0
tmpfs  /run/shm    tmpfs  defaults,noatime,nosuid,nodev 0  0
```

### NFS

```bash
nfs-server:/export/data  /mnt/nfs  nfs  defaults,hard,intr,timeo=600  0  0

# NFSv4
nfs-server:/data  /mnt/nfs  nfs4  defaults,_netdev  0  0
```

### CIFS/SMB

```bash
//fileserver/share  /mnt/smb  cifs  credentials=/root/.smbcredentials,uid=1000,gid=1000,_netdev  0  0
```

### Bind Mount

```bash
/var/log  /mnt/logs  none  bind  0  0
```

## Important Options

### Safety Options

```bash
nofail       # don't halt boot if device is missing (essential for removable disks)
noauto       # don't mount at boot (mount manually with "mount /mnt/usb")
_netdev      # wait for network before mounting (NFS, CIFS, iSCSI)
x-systemd.automount  # mount on first access (systemd)
```

### Performance Options

```bash
noatime      # don't update access times
nodiratime   # don't update directory access times
commit=60    # ext4: flush data every 60 seconds (default 5)
discard      # enable TRIM for SSDs (or use fstrim.timer instead)
```

### Security Options

```bash
nosuid       # ignore SUID/SGID bits
noexec       # prevent executing binaries
nodev        # ignore device special files
ro           # read-only
```

### Combined Defaults

```bash
defaults     # rw,suid,dev,exec,auto,nouser,async
```

## Validate and Apply

### Test fstab Before Rebooting

```bash
# Mount everything in fstab
sudo mount -a

# Verify no errors (most important step)
echo $?                                  # should be 0

# Check specific entry
sudo mount -fav                          # fake mount (dry run)
```

### Find Problems

```bash
# Show what fstab would mount
sudo findmnt --fstab

# Verify fstab
sudo findmnt --verify
```

## Label-Based Entries

```bash
LABEL=data    /mnt/data   ext4  defaults,noatime  0  2
LABEL=backup  /mnt/backup xfs   defaults,nofail   0  2
```

## Tips

- Always use UUID instead of `/dev/sd*` paths; device names can change between boots when disks are added or removed
- `nofail` is critical for non-essential disks; without it, a missing disk prevents the system from booting
- `_netdev` is required for network filesystems; without it, mount attempts happen before networking is up
- Always run `mount -a` after editing fstab to catch errors before the next reboot
- `pass` field (column 6): use `1` for root only, `2` for other filesystems, `0` to skip fsck (btrfs, xfs, network mounts)
- For SSDs, prefer `fstrim.timer` over the `discard` mount option; continuous TRIM has a performance cost
- `noatime` subsumes `nodiratime`; you only need `noatime`
- A broken fstab can prevent boot; keep a live USB ready for recovery, or use `nofail` liberally
- systemd's `x-systemd.automount` is useful for slow or unreliable mounts (NFS, USB) as it defers until first access
