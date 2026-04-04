# OverlayFS

Union filesystem that layers directories to present a merged view, enabling copy-on-write semantics.

## Core Concepts

```
merged/     — the unified view (mount point users see)
upperdir/   — writable layer (changes go here)
lowerdir/   — read-only layer(s) (original data)
workdir/    — internal bookkeeping (must be on same filesystem as upperdir)

Read:  file found in upperdir? use it. Otherwise, read from lowerdir.
Write: copy file from lowerdir to upperdir (copy-up), then modify.
Delete: create a whiteout in upperdir to hide the lowerdir file.
```

## Basic Mount

```bash
# Mount an overlay filesystem
mount -t overlay overlay \
    -o lowerdir=/lower,upperdir=/upper,workdir=/work \
    /merged

# With multiple lower layers (colon-separated, leftmost = top)
mount -t overlay overlay \
    -o lowerdir=/lower3:/lower2:/lower1,upperdir=/upper,workdir=/work \
    /merged

# Read-only overlay (no upperdir/workdir needed)
mount -t overlay overlay \
    -o lowerdir=/lower2:/lower1 \
    /merged

# Unmount
umount /merged
```

## Verify Mount

```bash
# Check mounted overlays
mount | grep overlay

# Detailed view
findmnt -t overlay

# View overlay options
cat /proc/mounts | grep overlay
```

## Whiteout Files

```bash
# When a file in lowerdir is deleted, a whiteout is created in upperdir
# Whiteout = character device with 0/0 major/minor

# Inspect whiteouts
ls -la /upper/
# c--------- 1 root root 0, 0 ... deleted_file   <-- whiteout

# Create a whiteout manually (requires root)
mknod /upper/filename c 0 0
```

## Opaque Directories

```bash
# When a directory in lowerdir is deleted and recreated,
# the upperdir version gets the "opaque" xattr.
# This hides ALL lowerdir contents below it.

# Check for opaque marker
getfattr -n trusted.overlay.opaque /upper/somedir/
# trusted.overlay.opaque="y"

# Set opaque manually
setfattr -n trusted.overlay.opaque -v y /upper/somedir/
```

## Docker Layer Implementation

```bash
# Docker uses overlayfs (overlay2 driver) for container layers
# Each image layer = a lowerdir
# Container writable layer = upperdir

# View Docker overlay mounts
docker inspect <container> --format '{{.GraphDriver.Data}}'
# Returns: LowerDir, UpperDir, MergedDir, WorkDir

# Typical Docker layer stack
# /var/lib/docker/overlay2/<layer-id>/diff   = layer content
# /var/lib/docker/overlay2/<layer-id>/merged = merged view
# /var/lib/docker/overlay2/<layer-id>/work   = workdir

# Examine layers
ls /var/lib/docker/overlay2/

# Check storage driver
docker info | grep "Storage Driver"
```

## fstab Entry

```bash
# Persistent mount via /etc/fstab
overlay  /merged  overlay  lowerdir=/lower,upperdir=/upper,workdir=/work  0  0

# With options
overlay  /merged  overlay  lowerdir=/lower,upperdir=/upper,workdir=/work,metacopy=on  0  0
```

## Mount Options

```bash
# redirect_dir: enables directory rename across layers
mount -t overlay overlay \
    -o lowerdir=/lower,upperdir=/upper,workdir=/work,redirect_dir=on \
    /merged

# metacopy: copy only metadata on copy-up, not data (saves space)
mount -t overlay overlay \
    -o lowerdir=/lower,upperdir=/upper,workdir=/work,metacopy=on \
    /merged

# index: enables hard link dedup across layers
mount -t overlay overlay \
    -o lowerdir=/lower,upperdir=/upper,workdir=/work,index=on \
    /merged

# volatile: skip fsync for faster writes (data loss risk on crash)
mount -t overlay overlay \
    -o lowerdir=/lower,upperdir=/upper,workdir=/work,volatile \
    /merged

# nfs_export: enable NFS exporting of overlay
mount -t overlay overlay \
    -o lowerdir=/lower,upperdir=/upper,workdir=/work,nfs_export=on \
    /merged

# Check kernel defaults
cat /sys/module/overlay/parameters/redirect_dir
cat /sys/module/overlay/parameters/metacopy
```

## Copy-Up Behavior

```bash
# When a lowerdir file is modified, entire file is copied to upperdir
# This is called "copy-up"

# View copy-up in action
echo "original" > /lower/file.txt
mount -t overlay overlay -o lowerdir=/lower,upperdir=/upper,workdir=/work /merged

# Modify through merged view
echo "modified" >> /merged/file.txt

# Now file exists in both layers
cat /lower/file.txt    # "original" (unchanged)
cat /upper/file.txt    # "original\nmodified" (full copy)

# With metacopy=on, only metadata is copied initially
# Data is copied lazily on first read through upperdir
```

## Namespace Usage (Rootless Containers)

```bash
# Unprivileged overlayfs in user namespaces (kernel 5.11+)
unshare --user --mount bash -c '
    mkdir -p /tmp/lower /tmp/upper /tmp/work /tmp/merged
    mount -t overlay overlay \
        -o lowerdir=/tmp/lower,upperdir=/tmp/upper,workdir=/tmp/work \
        /tmp/merged
'

# Podman uses fuse-overlayfs for older kernels
podman info | grep -i overlay
```

## Performance Characteristics

```bash
# Benchmark overlay vs direct filesystem
# First access to lowerdir file: ~same as direct
# Copy-up (first write): proportional to file size
# Subsequent writes after copy-up: ~same as direct
# Many layers (>10): slight lookup overhead per layer
# metacopy: eliminates copy-up data cost for metadata-only changes

# Monitor copy-ups
perf trace -e overlay:* -a

# Check inode usage
df -i /merged
```

## Troubleshooting

```bash
# "too many levels of symbolic links"
# upperdir/workdir must be on same filesystem
# workdir must be empty and on same fs as upperdir

# "filesystem not supported"
modprobe overlay

# Permission errors
# upperdir needs to be writable by the mount namespace's user

# Stale file handles after remount
# Use index=on to maintain inode consistency

# xattr not supported
# Underlying filesystem must support trusted.* xattrs
# ext4, xfs, btrfs all work; tmpfs does not (for trusted.*)
```

## Tips

- Upper and work directories must reside on the same filesystem; XFS or ext4 recommended
- The workdir must be an empty directory on the same filesystem as upperdir
- Copy-up copies the entire file, so large files incur significant cost on first modification
- Use `metacopy=on` to defer data copy-up and save disk space for metadata-only changes
- Docker limits overlay to ~128 layers; keep images lean to minimize lookup overhead
- Enable `redirect_dir=on` if you need to rename directories that span layers
- Read-only overlays (no upperdir) are useful for creating unified views of multiple directories
- Whiteout files are invisible through the merged view but visible in upperdir directly
- Always set `volatile` for ephemeral workloads (CI/CD builds) where crash recovery is unnecessary
- OverlayFS does not support POSIX ACLs on the overlay itself in older kernels (fixed in 5.11+)
- Use `index=on` when hard links across layers must maintain correct link counts
- The order of lowerdirs matters: the leftmost (first listed) has the highest priority

## See Also

- AUFS (earlier union filesystem, predecessor to overlayfs in Docker)
- Device Mapper (alternative Docker storage backend)
- Btrfs (copy-on-write filesystem with native snapshot support)
- SquashFS (read-only compressed filesystem, often used as lowerdir)
- FUSE (Filesystem in Userspace for custom filesystem implementations)

## References

- [OverlayFS Kernel Documentation](https://www.kernel.org/doc/html/latest/filesystems/overlayfs.html)
- [Docker Storage Drivers (overlay2)](https://docs.docker.com/storage/storagedriver/overlayfs-driver/)
- [Understanding Overlay Filesystem (Red Hat)](https://www.redhat.com/sysadmin/overlayfs)
- [Rootless Containers with Overlayfs](https://rootlesscontaine.rs/how-it-works/overlayfs/)
- [OCI Image Specification (Layer Format)](https://github.com/opencontainers/image-spec/blob/main/layer.md)
