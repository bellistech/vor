# Swap (Virtual Memory Swap Space)

Swap extends physical memory by paging inactive memory regions to disk, allowing the system to handle memory overcommitment and survive temporary memory spikes at the cost of significantly increased latency for swapped-out pages.

## Checking Swap Status

```bash
# View swap usage summary
free -h
#               total       used       free
# Swap:          4.0G       512M       3.5G

# Detailed swap device info
swapon --show
# NAME      TYPE       SIZE  USED PRIO
# /dev/sda2 partition  4G    512M   -2
# /swapfile file       2G      0B   -3

# From /proc
cat /proc/swaps
cat /proc/meminfo | grep -i swap
# SwapTotal:       6291456 kB
# SwapFree:        5767168 kB
# SwapCached:       131072 kB

# Per-process swap usage
grep VmSwap /proc/$PID/status
# VmSwap:     1024 kB

# Find top swap consumers
for pid in /proc/[0-9]*/; do
  swap=$(grep VmSwap "${pid}status" 2>/dev/null | awk '{print $2}')
  [ "${swap:-0}" -gt 0 ] && echo "$swap $(cat ${pid}cmdline 2>/dev/null | tr '\0' ' ')"
done | sort -rn | head -20
```

## Creating Swap

### Swap Partition

```bash
# Create swap partition (using fdisk/gdisk first to create partition)
mkswap /dev/sdb1

# Set a label
mkswap -L myswap /dev/sdb1

# Enable swap partition
swapon /dev/sdb1

# Add to /etc/fstab for persistence
echo "/dev/sdb1 none swap sw 0 0" >> /etc/fstab
# Or with UUID:
echo "UUID=$(blkid -s UUID -o value /dev/sdb1) none swap sw 0 0" >> /etc/fstab
```

### Swap File

```bash
# Create swap file (preferred method with fallocate)
fallocate -l 4G /swapfile

# Alternative: dd (works on all filesystems including XFS with older kernels)
dd if=/dev/zero of=/swapfile bs=1M count=4096

# Set permissions (MUST be 600)
chmod 600 /swapfile

# Format as swap
mkswap /swapfile

# Enable
swapon /swapfile

# Add to /etc/fstab
echo "/swapfile none swap sw 0 0" >> /etc/fstab

# Verify
swapon --show
```

### Swap File Restrictions

```bash
# Swap files do NOT work on:
# - Btrfs (before kernel 5.0, requires nocow)
# - NFS
# - Files with holes (use dd, not fallocate on some FSes)

# For Btrfs (kernel 5.0+):
truncate -s 0 /swapfile
chattr +C /swapfile          # Set nocow attribute
fallocate -l 4G /swapfile
chmod 600 /swapfile
mkswap /swapfile
swapon /swapfile
```

## Managing Swap

```bash
# Disable a specific swap device
swapoff /swapfile

# Disable all swap
swapoff -a

# Re-enable all swap from /etc/fstab
swapon -a

# Set swap priority (higher = preferred, -1 to 32767)
swapon -p 10 /dev/sdb1
swapon -p 5 /swapfile

# In /etc/fstab with priority
# /dev/sdb1  none  swap  sw,pri=10  0  0
# /swapfile  none  swap  sw,pri=5   0  0

# Resize swap file
swapoff /swapfile
fallocate -l 8G /swapfile
mkswap /swapfile
swapon /swapfile
```

## Swappiness

```bash
# Check current swappiness (0-200, default 60)
cat /proc/sys/vm/swappiness
# 60

# Set temporarily
echo 10 > /proc/sys/vm/swappiness
# Or:
sysctl vm.swappiness=10

# Set permanently
echo "vm.swappiness=10" >> /etc/sysctl.d/99-swap.conf
sysctl -p /etc/sysctl.d/99-swap.conf

# Swappiness values:
# 0   = Only swap to avoid OOM (kernel 3.5+)
# 1   = Minimum swapping
# 10  = Recommended for SSDs with enough RAM
# 60  = Default (balanced)
# 100 = Aggressively swap anonymous pages
# 200 = Maximum swap aggressiveness (cgroups v2)

# Per-cgroup swappiness (v1 only)
echo 0 > /sys/fs/cgroup/memory/myapp/memory.swappiness
```

## Zswap (Compressed Swap Cache)

```bash
# Check if zswap is enabled
cat /sys/module/zswap/parameters/enabled
# Y

# Enable zswap at boot (GRUB)
# GRUB_CMDLINE_LINUX="zswap.enabled=1"

# Enable at runtime
echo 1 > /sys/module/zswap/parameters/enabled

# Configure compression algorithm
echo lz4 > /sys/module/zswap/parameters/compressor
# Options: lzo, lz4, zstd, lzo-rle (default)

# Configure backing allocator
echo z3fold > /sys/module/zswap/parameters/zpool
# Options: zbud (default, 2:1 ratio), z3fold (3:1), zsmalloc

# Set max pool size (% of RAM)
echo 20 > /sys/module/zswap/parameters/max_pool_percent

# Check zswap statistics
grep -r . /sys/kernel/debug/zswap/ 2>/dev/null
# pool_total_size: 134217728
# stored_pages: 32768
# written_back_pages: 0
# reject_compress_poor: 128
# same_filled_pages: 1024
```

## Zram (Compressed RAM Disk as Swap)

```bash
# Load zram module
modprobe zram num_devices=1

# Set compression algorithm
echo lz4 > /sys/block/zram0/comp_algorithm

# Set disk size (half of RAM is typical)
echo 4G > /sys/block/zram0/disksize

# Format and enable as swap with high priority
mkswap /dev/zram0
swapon -p 100 /dev/zram0

# Check zram stats
cat /sys/block/zram0/mm_stat
# orig_data_size  compr_data_size  mem_used  mem_limit  mem_used_max
# 1073741824      268435456        280000000 0          290000000

# Systemd zram-generator (modern approach)
# /etc/systemd/zram-generator.conf
# [zram0]
# zram-size = ram / 2
# compression-algorithm = zstd

# Remove zram device
swapoff /dev/zram0
echo 1 > /sys/block/zram0/reset
```

## Swap Encryption

```bash
# Encrypted swap with random key (re-encrypted each boot)
# /etc/crypttab:
# cryptswap /dev/sdb1 /dev/urandom swap,cipher=aes-xts-plain64,size=256

# /etc/fstab:
# /dev/mapper/cryptswap none swap sw 0 0

# Or use dm-crypt manually
cryptsetup open --type plain /dev/sdb1 cryptswap \
  --cipher aes-xts-plain64 --key-size 256 --key-file /dev/urandom
mkswap /dev/mapper/cryptswap
swapon /dev/mapper/cryptswap
```

## Monitoring and Performance

```bash
# Watch swap I/O in real time
vmstat 1
# procs -----------memory---------- ---swap-- -----io----
#  r  b   swpd   free   buff  cache   si   so    bi    bo
#  1  0 524288 102400  51200 204800    0    0    10     5

# si = swap in (pages/s from disk to RAM)
# so = swap out (pages/s from RAM to disk)

# Swap I/O stats
cat /proc/vmstat | grep pswp
# pswpin 12345     (pages swapped in total)
# pswpout 67890    (pages swapped out total)

# Page scan rate (indicates memory pressure)
cat /proc/vmstat | grep pgsteal
# pgsteal_kswapd 100000
# pgsteal_direct 5000

# Check if swapping is causing latency
sar -W 1 10
# pswpin/s  pswpout/s
# 0.00      50.00      <-- swapping out, may cause issues

# Memory pressure (cgroups v2 PSI)
cat /proc/pressure/memory
# some avg10=5.00 avg60=3.00 avg300=1.00 total=500000
```

## Tips

- Use `vm.swappiness=10` on servers with SSDs and adequate RAM; 60 is too aggressive for most workloads
- Swap on SSD is 100x faster than HDD but still 1000x slower than RAM; do not rely on it for performance
- Zswap compresses pages in RAM before writing to disk, reducing swap I/O by 2-3x
- Zram provides a compressed RAM block device that is faster than any disk-backed swap
- Always use `chmod 600` on swap files; readable swap is a security vulnerability
- `fallocate` is instant but does not work on all filesystems for swap; use `dd` as fallback
- Set higher priority on faster swap devices so the kernel prefers them
- Monitor `pswpin/pswpout` in `/proc/vmstat`; sustained swap-out activity means you need more RAM
- `swapoff` will fail if there is not enough free RAM to absorb all swapped-out pages
- In Kubernetes, disable swap (`swapoff -a`) or use the `--fail-swap-on=false` kubelet flag
- Encrypted swap with a random key prevents sensitive data from persisting across reboots
- Use `vm.swappiness=0` for database servers where any swapping is unacceptable

## See Also

oom-killer, proc-sys, cgroups, ulimit

## References

- [Linux Kernel Swap Documentation](https://docs.kernel.org/admin-guide/sysctl/vm.html)
- [Zswap Documentation](https://docs.kernel.org/admin-guide/mm/zswap.html)
- [Zram Documentation](https://docs.kernel.org/admin-guide/blockdev/zram.html)
- [Red Hat Swap Guide](https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux/9/html/managing_storage_devices/getting-started-with-swap_managing-storage-devices)
- [ArchWiki: Swap](https://wiki.archlinux.org/title/Swap)
