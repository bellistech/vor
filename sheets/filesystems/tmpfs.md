# tmpfs (RAM-Backed Filesystem)

tmpfs is a memory-backed filesystem that stores files in RAM and swap, providing extremely fast I/O with no disk writes, automatic sizing, and contents that vanish on reboot.

## Mount tmpfs

### Basic Mount

```bash
# Mount a tmpfs at /mnt/ramdisk
sudo mount -t tmpfs tmpfs /mnt/ramdisk

# Mount with size limit (default is half of RAM)
sudo mount -t tmpfs -o size=2G tmpfs /mnt/ramdisk

# Mount with percentage of RAM
sudo mount -t tmpfs -o size=25% tmpfs /mnt/ramdisk

# Mount with specific permissions
sudo mount -t tmpfs -o size=1G,mode=1777 tmpfs /mnt/ramdisk

# Mount with uid/gid
sudo mount -t tmpfs -o size=512M,uid=1000,gid=1000,mode=0700 tmpfs /home/user/scratch
```

### Mount Options

```bash
size=2G             # maximum size (bytes, k, m, g, or %)
nr_inodes=1000000   # maximum number of inodes (files/dirs)
mode=1777           # permissions (1777 = sticky + rwxrwxrwx, like /tmp)
uid=1000            # owner user ID
gid=1000            # owner group ID
noatime             # don't update access times (marginal benefit on tmpfs)
nodev               # don't allow device files
nosuid              # don't honor setuid/setgid bits
noexec              # don't allow execution (security for /tmp)
```

### Persistent Mount (fstab)

```bash
# /etc/fstab entries
tmpfs  /tmp        tmpfs  defaults,noatime,nosuid,nodev,noexec,size=2G,mode=1777  0 0
tmpfs  /var/tmp    tmpfs  defaults,noatime,nosuid,nodev,size=1G,mode=1777         0 0
tmpfs  /run/cache  tmpfs  defaults,noatime,size=512M,uid=www-data,gid=www-data    0 0
```

## Common tmpfs Locations

### System tmpfs Mounts

```bash
# These are typically tmpfs by default on modern Linux:
/dev/shm            # POSIX shared memory (usually half of RAM)
/run                # runtime data (PID files, sockets)
/run/lock           # lock files
/sys/fs/cgroup      # cgroup filesystem
/tmp                # temporary files (if configured)

# Check current tmpfs mounts
mount | grep tmpfs
df -h --type=tmpfs
findmnt -t tmpfs
```

### /dev/shm (Shared Memory)

```bash
# /dev/shm is always tmpfs — used for POSIX shared memory
ls -la /dev/shm/

# Applications using /dev/shm:
# - PostgreSQL (shared buffers)
# - Chrome/Chromium (shared renderer memory)
# - Docker (for --shm-size)
# - Oracle DB (SGA)

# Resize /dev/shm (runtime)
sudo mount -o remount,size=4G /dev/shm

# Docker: increase shared memory
docker run --shm-size=2g myapp
```

### /tmp on tmpfs

```bash
# Check if /tmp is tmpfs
findmnt /tmp

# Enable /tmp as tmpfs via systemd
sudo systemctl enable tmp.mount

# Or create override
sudo cp /usr/share/systemd/tmp.mount /etc/systemd/system/
sudo systemctl enable tmp.mount
sudo systemctl start tmp.mount

# Disable /tmp as tmpfs (use disk-backed /tmp)
sudo systemctl disable tmp.mount
sudo systemctl mask tmp.mount

# Custom tmp.mount size
sudo systemctl edit tmp.mount
# [Mount]
# Options=mode=1777,strictatime,nosuid,nodev,size=4G
```

## Sizing and Capacity

### Check Usage

```bash
# Show tmpfs usage
df -h /tmp
df -h /dev/shm
df -h --type=tmpfs                     # all tmpfs mounts

# Detailed info
findmnt -t tmpfs -o TARGET,SIZE,USED,AVAIL,USE%

# Check what is using space
du -sh /tmp/*
du -sh /dev/shm/*
```

### Resize at Runtime

```bash
# Grow tmpfs (no data loss)
sudo mount -o remount,size=4G /tmp

# Shrink tmpfs (safe if data fits)
sudo mount -o remount,size=1G /tmp
# mount: /tmp: mount point not mounted or bad option.
# (fails if current usage exceeds new size)
```

### Memory Accounting

```bash
# tmpfs pages are counted against RAM + swap
# Check how much RAM tmpfs is using
cat /proc/meminfo | grep -i shmem
# Shmem:           1234 kB    ← tmpfs + shared memory

# tmpfs does NOT pre-allocate memory
# Pages are allocated on write, freed on delete
# Unused tmpfs space costs nothing

# Check if tmpfs data is in RAM or swap
cat /proc/meminfo | grep -E '(SwapTotal|SwapFree|Shmem)'
```

## Use Cases

### Build Directory (Fast Compilation)

```bash
# Mount tmpfs for build artifacts
sudo mount -t tmpfs -o size=4G tmpfs /home/user/project/build

# Build in RAM — dramatically faster for I/O heavy builds
cd /home/user/project
make -j$(nproc) BUILD_DIR=/home/user/project/build

# Go build cache in tmpfs
export GOCACHE=/dev/shm/go-cache
go build ./...

# Rust target in tmpfs
mkdir -p /dev/shm/rust-target
ln -s /dev/shm/rust-target ./target
cargo build
```

### Test Databases

```bash
# PostgreSQL test database in tmpfs
sudo mount -t tmpfs -o size=2G tmpfs /var/lib/postgresql/test-data
sudo -u postgres initdb -D /var/lib/postgresql/test-data
sudo -u postgres pg_ctl -D /var/lib/postgresql/test-data start

# SQLite in tmpfs
sqlite3 /dev/shm/test.db "CREATE TABLE t (id INTEGER PRIMARY KEY);"

# MySQL/MariaDB tmpdir
# /etc/mysql/my.cnf:
# [mysqld]
# tmpdir = /dev/shm/mysql-tmp
```

### CI/CD Pipelines

```bash
# Fast workspace for CI jobs
sudo mount -t tmpfs -o size=8G tmpfs /workspace
git clone --depth 1 repo /workspace/repo
cd /workspace/repo
make test
# Workspace vanishes after job (or reboot)
```

### Browser and Application Caches

```bash
# Chrome cache in tmpfs (reduces SSD writes)
mkdir -p /dev/shm/chrome-cache
google-chrome --disk-cache-dir=/dev/shm/chrome-cache

# Firefox profile in tmpfs
# Use profile-sync-daemon (psd) for automatic sync
```

### IPC and Shared Memory

```bash
# Create shared memory segment (POSIX)
# C: shm_open("/my_segment", O_CREAT | O_RDWR, 0666)
# Python:
python3 -c "
import mmap, os
fd = os.open('/dev/shm/my_segment', os.O_CREAT | os.O_RDWR)
os.ftruncate(fd, 4096)
mm = mmap.mmap(fd, 4096)
mm.write(b'hello from shared memory')
mm.close()
os.close(fd)
"

# Check it
cat /dev/shm/my_segment
```

## Swap Interaction

```bash
# tmpfs can be swapped out when RAM is under pressure
# This means tmpfs is NOT guaranteed to be in RAM

# To prevent swapping of tmpfs data (performance-critical):
sudo sysctl vm.swappiness=0            # reduce swap tendency (not specific to tmpfs)

# Or use ramfs instead (never swaps, but no size limit — dangerous)
sudo mount -t ramfs ramfs /mnt/ramfs   # WARNING: can consume all RAM

# tmpfs vs ramfs comparison:
# tmpfs: size limit, can swap, shows in df, preferred
# ramfs: no size limit, never swaps, invisible to df, dangerous
```

## Security Considerations

```bash
# /tmp as tmpfs with hardened options
sudo mount -t tmpfs -o size=2G,noexec,nosuid,nodev,mode=1777 tmpfs /tmp

# noexec — prevent execution of binaries (blocks many exploits)
# nosuid — ignore setuid bits
# nodev  — prevent device files

# Encrypt tmpfs with dm-crypt (paranoid mode)
# Not common — tmpfs is volatile, but swap could persist data
# Encrypt swap instead:
sudo cryptsetup open --type plain /dev/sdX swap_crypt
sudo mkswap /dev/mapper/swap_crypt
sudo swapon /dev/mapper/swap_crypt
```

## Tips

- tmpfs does not pre-allocate memory -- files consume RAM only when written, and memory is freed instantly when files are deleted.
- The default size is half of total RAM, but setting an explicit `size=` is recommended to prevent runaway processes from consuming all memory.
- tmpfs data can be swapped to disk under memory pressure -- if you need guaranteed RAM residency, use `ramfs` (but with extreme caution as it has no size limit).
- `/dev/shm` is the standard location for POSIX shared memory -- do not unmount it or applications like PostgreSQL and Chrome will break.
- Use `noexec,nosuid,nodev` on `/tmp` tmpfs mounts for security -- this blocks most `/tmp`-based exploit techniques.
- Resizing tmpfs at runtime with `mount -o remount,size=N` is live and non-destructive -- no data is lost when growing.
- Build directories on tmpfs can speed up compilation by 2-10x for I/O-bound builds (Go, Rust, C++) by eliminating disk latency.
- tmpfs memory usage shows up as `Shmem` in `/proc/meminfo`, not in process RSS -- tools like `top` may not show it.
- Docker's `--shm-size` flag controls the size of `/dev/shm` inside the container (default 64MB, often too small for databases).
- Do not store anything important on tmpfs -- all data is lost on reboot, kernel panic, or power loss.
- Use `systemctl enable tmp.mount` to make `/tmp` a tmpfs on systemd systems -- this is increasingly the default on modern distributions.
- Monitor tmpfs usage with `df -h --type=tmpfs` -- running out of tmpfs space causes write failures, not OOM kills.

## See Also

ext4, xfs, nfs, fstab, swap, systemd

## References

- [tmpfs kernel documentation](https://www.kernel.org/doc/html/latest/filesystems/tmpfs.html) -- official kernel docs
- [man tmpfs(5)](https://man7.org/linux/man-pages/man5/tmpfs.5.html) -- tmpfs man page
- [man mount(8)](https://man7.org/linux/man-pages/man8/mount.8.html) -- mount command and options
- [man shm_overview(7)](https://man7.org/linux/man-pages/man7/shm_overview.7.html) -- POSIX shared memory overview
- [systemd tmp.mount](https://www.freedesktop.org/software/systemd/man/tmp.mount.html) -- systemd unit for /tmp as tmpfs
- [Arch Wiki: tmpfs](https://wiki.archlinux.org/title/Tmpfs) -- comprehensive tmpfs guide
- [man fstab(5)](https://man7.org/linux/man-pages/man5/fstab.5.html) -- filesystem table configuration
