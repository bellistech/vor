# Memory Tuning (Linux Kernel)

Tune the kernel's memory subsystem for throughput, latency, and stability.

## Write Caching — Dirty Page Ratios

### Dirty Ratio Tuning

```bash
# Check current dirty page settings
sysctl vm.dirty_ratio vm.dirty_background_ratio vm.dirty_expire_centisecs vm.dirty_writeback_centisecs

# Default: 20% total RAM triggers synchronous writeback
# Default: 10% total RAM triggers background writeback

# High-throughput I/O server (allow more dirty pages before flush)
sysctl -w vm.dirty_ratio=40
sysctl -w vm.dirty_background_ratio=10

# Low-latency / database workload (flush early, avoid write storms)
sysctl -w vm.dirty_ratio=5
sysctl -w vm.dirty_background_ratio=2

# Use absolute bytes instead of percentage (useful for large-RAM systems)
# Set 256MB background, 1GB sync threshold
sysctl -w vm.dirty_background_bytes=268435456
sysctl -w vm.dirty_bytes=1073741824

# Expire dirty pages after 15 seconds (default 30s = 3000 centisecs)
sysctl -w vm.dirty_expire_centisecs=1500

# Writeback thread wakes every 3 seconds (default 5s = 500 centisecs)
sysctl -w vm.dirty_writeback_centisecs=300
```

### Persist Dirty Tuning

```bash
cat >> /etc/sysctl.d/10-dirty.conf << 'EOF'
vm.dirty_ratio = 5
vm.dirty_background_ratio = 2
vm.dirty_expire_centisecs = 1500
vm.dirty_writeback_centisecs = 300
EOF
sysctl --system
```

## Huge Pages

### Transparent Huge Pages (THP)

```bash
# Check THP status
cat /sys/kernel/mm/transparent_hugepage/enabled

# Enable THP (default on most distros)
echo always > /sys/kernel/mm/transparent_hugepage/enabled

# Disable THP (recommended for databases: Redis, MongoDB, PostgreSQL)
echo never > /sys/kernel/mm/transparent_hugepage/enabled

# Set to madvise-only (apps opt in via madvise(MADV_HUGEPAGE))
echo madvise > /sys/kernel/mm/transparent_hugepage/enabled

# Control defragmentation behavior
echo defer+madvise > /sys/kernel/mm/transparent_hugepage/defrag

# Check THP usage statistics
grep -i huge /proc/meminfo
cat /proc/vmstat | grep thp
```

### Explicit Huge Pages (hugetlbfs)

```bash
# Reserve 1024 x 2MB huge pages (2GB total)
sysctl -w vm.nr_hugepages=1024

# Reserve huge pages on a specific NUMA node
echo 512 > /sys/devices/system/node/node0/hugepages/hugepages-2048kB/nr_hugepages
echo 512 > /sys/devices/system/node/node1/hugepages/hugepages-2048kB/nr_hugepages

# Reserve 1GB huge pages (must be set at boot via kernel cmdline)
# GRUB: hugepagesz=1G hugepages=4

# Check reservation status
cat /proc/meminfo | grep -i huge

# Mount hugetlbfs for application use
mkdir -p /mnt/hugepages
mount -t hugetlbfs nodev /mnt/hugepages

# Persistent mount
echo 'nodev /mnt/hugepages hugetlbfs defaults 0 0' >> /etc/fstab

# DPDK-style: reserve and mount in one shot
echo 2048 > /sys/kernel/mm/hugepages/hugepages-2048kB/nr_hugepages
mount -t hugetlbfs -o pagesize=2M nodev /mnt/hugepages-2M
```

## NUMA

### NUMA Topology and Inspection

```bash
# Show NUMA topology
numactl --hardware

# Show per-node memory stats
numastat

# Detailed per-node info
numastat -c -m

# Per-process NUMA stats
numastat -p $(pidof myapp)
```

### NUMA Memory Policies

```bash
# Run process on specific NUMA node (CPU + memory)
numactl --cpunodebind=0 --membind=0 ./myapp

# Interleave memory across all nodes (good for hash tables, caches)
numactl --interleave=all ./myapp

# Preferred node (fallback allowed)
numactl --preferred=0 ./myapp

# Zone reclaim mode
# 0 = disabled (allocate from remote nodes freely) — default, usually best
# 1 = enable zone reclaim (reclaim file-backed pages before going remote)
sysctl -w vm.zone_reclaim_mode=0

# For NUMA-heavy workloads: automatic NUMA balancing
sysctl -w kernel.numa_balancing=1
```

## OOM Tuning

### OOM Score Adjustment

```bash
# Check a process's OOM score (higher = more likely to be killed)
cat /proc/$(pidof myapp)/oom_score

# Protect critical process (lower score = less likely killed)
echo -1000 > /proc/$(pidof myapp)/oom_score_adj

# Make a process the prime OOM target
echo 1000 > /proc/$(pidof myapp)/oom_score_adj

# Systemd unit: protect from OOM
# [Service]
# OOMScoreAdjust=-900
```

### Overcommit and OOM Behavior

```bash
# Overcommit modes:
# 0 = heuristic (default — allows ~50% overcommit)
# 1 = always allow (never refuse malloc, dangerous)
# 2 = strict (commit limit = swap + RAM * overcommit_ratio/100)
sysctl -w vm.overcommit_memory=2
sysctl -w vm.overcommit_ratio=80

# Check commit limits
grep -i commit /proc/meminfo

# Panic on OOM instead of killing processes (for HA clusters)
sysctl -w vm.panic_on_oom=1

# Reboot N seconds after panic
sysctl -w kernel.panic=10

# Disable OOM killer entirely (system hangs on OOM — dangerous)
sysctl -w vm.oom-kill=0
```

## Page Cache

### Cache Pressure and Management

```bash
# vfs_cache_pressure controls tendency to reclaim inode/dentry cache
# Default: 100 (balanced)
# Lower: keep more directory/inode caches (file servers)
# Higher: reclaim caches more aggressively
sysctl -w vm.vfs_cache_pressure=50    # file server
sysctl -w vm.vfs_cache_pressure=200   # memory-constrained system

# Drop caches (diagnostic only, not for production tuning)
echo 1 > /proc/sys/vm/drop_caches   # page cache only
echo 2 > /proc/sys/vm/drop_caches   # dentries + inodes
echo 3 > /proc/sys/vm/drop_caches   # all of the above

# Sync before dropping caches
sync; echo 3 > /proc/sys/vm/drop_caches

# Monitor page cache hit rate
perf stat -e cache-misses,cache-references -a sleep 10

# Check page cache usage
free -h
vmstat 1 5
```

### Min Free Memory

```bash
# Reserve memory for the kernel (prevents allocation failures under pressure)
# Default: auto-calculated, usually ~64MB
sysctl -w vm.min_free_kbytes=131072   # 128MB

# For large-memory systems (512GB+), set higher
sysctl -w vm.min_free_kbytes=1048576  # 1GB

# Watermark scale factor (controls gap between min/low/high watermarks)
sysctl -w vm.watermark_scale_factor=200   # default 10, range 1-3000
```

## Swap Tuning

### Swappiness

```bash
# vm.swappiness: tendency to swap out anonymous pages vs drop file cache
# Range: 0-200 (default 60)
# 0 = avoid swapping unless absolutely necessary
# 100 = treat anonymous and file pages equally
# 200 = strongly prefer swapping over cache drop

# Database servers: minimize swap
sysctl -w vm.swappiness=10

# Desktop: balanced
sysctl -w vm.swappiness=60

# Disable swap entirely
swapoff -a

# Check swap usage by process
for f in /proc/[0-9]*/status; do
  awk '/VmSwap|Name/{printf "%s ", $2}' "$f" 2>/dev/null
  echo
done | sort -k2 -nr | head -20
```

### zswap (Compressed Swap Cache)

```bash
# Enable zswap (compressed page cache in front of swap)
echo 1 > /sys/module/zswap/parameters/enabled

# Set compressor (lz4 for speed, zstd for ratio)
echo lz4 > /sys/module/zswap/parameters/compressor

# Max pool size as percentage of RAM
echo 20 > /sys/module/zswap/parameters/max_pool_percent

# Set memory allocator
echo z3fold > /sys/module/zswap/parameters/zpool

# Check zswap stats
grep -r . /sys/kernel/debug/zswap/ 2>/dev/null
```

### zram (Compressed RAM Block Device)

```bash
# Load zram module
modprobe zram num_devices=1

# Set compression algorithm
echo lz4 > /sys/block/zram0/comp_algorithm

# Set disk size (usually 25-50% of RAM)
echo 8G > /sys/block/zram0/disksize

# Create swap on zram
mkswap /dev/zram0
swapon -p 100 /dev/zram0   # priority 100 (higher than disk swap)

# Check compression stats
cat /sys/block/zram0/mm_stat

# Systemd: use zram-generator
# /etc/systemd/zram-generator.conf:
# [zram0]
# zram-size = ram / 2
# compression-algorithm = zstd
```

## Memory Cgroups (cgroup v2)

### Setting Memory Limits

```bash
# Create a cgroup
mkdir -p /sys/fs/cgroup/myapp

# Hard memory limit (OOM kill above this)
echo 4G > /sys/fs/cgroup/myapp/memory.max

# Soft memory limit (reclaim pressure starts here)
echo 2G > /sys/fs/cgroup/myapp/memory.high

# Swap limit
echo 1G > /sys/fs/cgroup/myapp/memory.swap.max

# Disable swap for this cgroup
echo 0 > /sys/fs/cgroup/myapp/memory.swap.max

# Memory reservation (best-effort minimum)
echo 1G > /sys/fs/cgroup/myapp/memory.min

# Low watermark (protected from reclaim when under pressure)
echo 1536M > /sys/fs/cgroup/myapp/memory.low
```

### Monitoring Cgroup Memory

```bash
# Current memory usage
cat /sys/fs/cgroup/myapp/memory.current

# Detailed stats
cat /sys/fs/cgroup/myapp/memory.stat

# OOM events
cat /sys/fs/cgroup/myapp/memory.events

# Pressure stall info
cat /sys/fs/cgroup/myapp/memory.pressure

# Systemd unit memory limits
# [Service]
# MemoryMax=4G
# MemoryHigh=2G
# MemorySwapMax=1G

# List all cgroup memory usage (systemd)
systemd-cgtop -m
```

## SLUB/SLAB Allocator Tuning

### Inspecting Slab Caches

```bash
# Summary of slab cache usage
slabtop -o

# Detailed slab info
cat /proc/slabinfo

# Per-cache stats (SLUB)
ls /sys/kernel/slab/

# Check specific cache (e.g., dentry, inode_cache, task_struct)
cat /sys/kernel/slab/dentry/object_size
cat /sys/kernel/slab/dentry/objs_per_slab
cat /sys/kernel/slab/dentry/total_objects

# Check which allocator is in use
cat /proc/cmdline | grep -o 'slab_[a-z]*'
# or: dmesg | grep -i 'slab\|slub\|slob'
```

### SLUB Tuning

```bash
# Set minimum slab order (higher = larger slabs, fewer allocations)
echo 1 > /sys/kernel/slab/kmalloc-256/order

# Merge compatible caches (saves memory, default on)
# Boot param: slub_nomerge (to disable merging for debugging)

# Enable SLUB debug for a specific cache
echo 1 > /sys/kernel/slab/dentry/trace

# Boot-time SLUB debug
# GRUB: slub_debug=FZP slub_debug=FZP,dentry

# Free slab caches (force reclaim of reclaimable slabs)
echo 2 > /proc/sys/vm/drop_caches
```

## Practical Sysctl Profiles

### Web/Application Server

```bash
cat > /etc/sysctl.d/20-memory-webserver.conf << 'EOF'
vm.swappiness = 30
vm.dirty_ratio = 15
vm.dirty_background_ratio = 5
vm.vfs_cache_pressure = 75
vm.min_free_kbytes = 131072
vm.overcommit_memory = 0
vm.zone_reclaim_mode = 0
EOF
sysctl --system
```

### Database Server

```bash
cat > /etc/sysctl.d/20-memory-database.conf << 'EOF'
vm.swappiness = 5
vm.dirty_ratio = 5
vm.dirty_background_ratio = 2
vm.dirty_expire_centisecs = 500
vm.dirty_writeback_centisecs = 100
vm.vfs_cache_pressure = 50
vm.min_free_kbytes = 524288
vm.overcommit_memory = 2
vm.overcommit_ratio = 80
vm.zone_reclaim_mode = 0
EOF
sysctl --system
```

### High-Performance Computing

```bash
cat > /etc/sysctl.d/20-memory-hpc.conf << 'EOF'
vm.swappiness = 1
vm.dirty_ratio = 40
vm.dirty_background_ratio = 10
vm.vfs_cache_pressure = 50
vm.min_free_kbytes = 1048576
vm.nr_hugepages = 4096
vm.zone_reclaim_mode = 0
kernel.numa_balancing = 1
EOF
sysctl --system
```

## Tips

- Always benchmark before and after tuning; measure with `perf`, `vmstat`, `sar`, or `bpftrace`
- On NUMA systems, `vm.zone_reclaim_mode=0` is almost always correct; zone reclaim causes stalls
- For databases, disable THP (`echo never > /sys/.../enabled`) and use explicit huge pages
- `vm.min_free_kbytes` too high wastes RAM; too low causes allocation failures under load
- Use `vm.dirty_background_bytes` / `vm.dirty_bytes` on systems with 64GB+ RAM for predictable behavior
- zram with lz4 gives 2-3x compression; zstd gives 3-4x but uses more CPU
- Memory cgroup `memory.high` is preferable to `memory.max` for graceful throttling
- Check `/proc/buddyinfo` to monitor memory fragmentation
- Use `perf stat -e dTLB-load-misses` to quantify TLB miss overhead from page size choices

## See Also

- sysctl
- cgroups
- numa
- swap

## References

- Linux kernel documentation: `Documentation/admin-guide/sysctl/vm.rst`
- Linux kernel documentation: `Documentation/admin-guide/mm/`
- `man 5 proc` (/proc/meminfo, /proc/vmstat, /proc/buddyinfo)
- `man numactl`, `man numastat`
- Brendan Gregg, "Systems Performance", Chapter 7: Memory
- LKML: Transparent Huge Pages documentation
- Red Hat Performance Tuning Guide: Memory
