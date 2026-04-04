# Proc-Sys (/proc and /sys Virtual Filesystems)

The /proc and /sys virtual filesystems expose kernel data structures, process information, and tunable parameters as readable files, providing the primary interface for inspecting system state and configuring kernel behavior at runtime.

## /proc Filesystem Overview

```bash
# /proc contains per-process and system-wide information
ls /proc/
# 1/  2/  3/ ... (per-PID directories)
# cpuinfo  meminfo  version  uptime  loadavg  stat  vmstat
# filesystems  mounts  partitions  interrupts  modules
# sys/  net/  bus/  driver/  fs/  irq/

# Mount type
mount | grep proc
# proc on /proc type proc (rw,nosuid,nodev,noexec,relatime)
```

## Process Information (/proc/pid/*)

```bash
# Essential per-process files
PID=$$

# Command line arguments
cat /proc/$PID/cmdline | tr '\0' ' '

# Environment variables
cat /proc/$PID/environ | tr '\0' '\n'

# Current working directory
readlink /proc/$PID/cwd

# Executable path
readlink /proc/$PID/exe

# File descriptors
ls -la /proc/$PID/fd/

# Memory map
cat /proc/$PID/maps
# 7f8a00000000-7f8a00021000 rw-p ...  [heap]
# 7fff12345000-7fff12366000 rw-p ...  [stack]

# Memory usage summary
cat /proc/$PID/status | grep -E "^(Vm|Rss|Threads|State)"
# VmPeak:   123456 kB     (peak virtual memory)
# VmSize:   120000 kB     (current virtual memory)
# VmRSS:     45000 kB     (resident set size)
# VmSwap:     1024 kB     (swapped out)
# RssAnon:   30000 kB     (anonymous RSS)
# RssFile:   15000 kB     (file-backed RSS)
# Threads:       4

# Detailed memory map with sizes
cat /proc/$PID/smaps_rollup
# Rss:           45000 kB
# Pss:           40000 kB    (proportional share)
# Shared_Clean:  10000 kB
# Shared_Dirty:   2000 kB
# Private_Clean:  8000 kB
# Private_Dirty: 25000 kB

# Process limits
cat /proc/$PID/limits

# I/O statistics
cat /proc/$PID/io
# rchar: 1234567890        (bytes read)
# wchar: 987654321         (bytes written)
# read_bytes: 123456789    (actual disk reads)
# write_bytes: 98765432    (actual disk writes)

# cgroup membership
cat /proc/$PID/cgroup

# Namespace IDs
ls -la /proc/$PID/ns/

# Scheduling info
cat /proc/$PID/sched | head -10

# Stack trace (kernel)
cat /proc/$PID/stack

# Open file descriptors with details
ls -la /proc/$PID/fd/
# 0 -> /dev/pts/0         (stdin)
# 1 -> /dev/pts/0         (stdout)
# 3 -> socket:[12345]     (network socket)
# 4 -> /var/log/app.log   (open file)
```

## System Memory (/proc/meminfo)

```bash
cat /proc/meminfo
# MemTotal:       16384000 kB
# MemFree:         2048000 kB
# MemAvailable:    8192000 kB   (estimated available for new apps)
# Buffers:          512000 kB
# Cached:          5120000 kB   (page cache)
# SwapCached:       102400 kB
# Active:          6144000 kB
# Inactive:        4096000 kB
# Active(anon):    3072000 kB
# Inactive(anon):  1024000 kB
# Active(file):    3072000 kB
# Inactive(file):  3072000 kB
# Dirty:             51200 kB   (pages waiting to be written)
# Writeback:             0 kB
# AnonPages:       4096000 kB   (anonymous pages)
# Mapped:          1024000 kB   (mmap'd files)
# Shmem:            512000 kB   (shared memory / tmpfs)
# KReclaimable:     512000 kB   (kernel reclaimable)
# Slab:             768000 kB   (kernel slab allocator)
# SReclaimable:     512000 kB
# SUnreclaim:       256000 kB
# PageTables:       102400 kB
# HugePages_Total:       0
# HugePages_Free:        0
# Hugepagesize:       2048 kB

# Available memory calculation:
# MemAvailable ≈ MemFree + Buffers + Cached - min(Cached/2, low_watermark)
```

## CPU Information (/proc/cpuinfo)

```bash
# CPU details
cat /proc/cpuinfo | head -30
# processor  : 0
# model name : Intel(R) Core(TM) i7-10700K
# cpu MHz    : 3800.000
# cache size : 16384 KB
# cpu cores  : 8
# siblings   : 16
# flags      : ... avx avx2 aes ...

# Quick counts
grep -c processor /proc/cpuinfo    # Total logical CPUs
grep "physical id" /proc/cpuinfo | sort -u | wc -l  # Physical sockets
grep "cpu cores" /proc/cpuinfo | head -1  # Cores per socket

# CPU load
cat /proc/loadavg
# 0.50 0.75 1.00 2/350 12345
# 1min  5min  15min  running/total  last_pid

# Per-CPU statistics
cat /proc/stat | head -5
# cpu  12345 678 91011 121314 1516 0 1718 0 0 0
# cpu0 1234  67  9101  12131  151  0 171  0 0 0
# Fields: user nice system idle iowait irq softirq steal guest guest_nice
```

## /proc/sys Tunables

### Networking

```bash
# TCP tuning
cat /proc/sys/net/core/somaxconn          # Listen backlog (default 4096)
echo 65535 > /proc/sys/net/core/somaxconn

cat /proc/sys/net/ipv4/tcp_max_syn_backlog  # SYN queue size
echo 65535 > /proc/sys/net/ipv4/tcp_max_syn_backlog

# TCP keepalive
cat /proc/sys/net/ipv4/tcp_keepalive_time    # 7200 (seconds)
cat /proc/sys/net/ipv4/tcp_keepalive_intvl   # 75 (seconds)
cat /proc/sys/net/ipv4/tcp_keepalive_probes  # 9

# Port range
cat /proc/sys/net/ipv4/ip_local_port_range   # 32768 60999
echo "1024 65535" > /proc/sys/net/ipv4/ip_local_port_range

# TCP buffer sizes (min default max)
cat /proc/sys/net/ipv4/tcp_rmem   # 4096 131072 6291456
cat /proc/sys/net/ipv4/tcp_wmem   # 4096 16384  4194304

# Connection tracking
cat /proc/sys/net/netfilter/nf_conntrack_max  # 262144

# IP forwarding
cat /proc/sys/net/ipv4/ip_forward             # 0 or 1
echo 1 > /proc/sys/net/ipv4/ip_forward

# TIME_WAIT reuse
echo 1 > /proc/sys/net/ipv4/tcp_tw_reuse
```

### Kernel

```bash
# PID max
cat /proc/sys/kernel/pid_max              # 4194304

# Shared memory max
cat /proc/sys/kernel/shmmax               # bytes
cat /proc/sys/kernel/shmall               # pages

# Message queue limits
cat /proc/sys/kernel/msgmax               # max message size
cat /proc/sys/kernel/msgmnb               # max queue size

# Core dump pattern
cat /proc/sys/kernel/core_pattern

# Kernel panic behavior
cat /proc/sys/kernel/panic                # seconds before reboot (0=hang)

# Hostname
cat /proc/sys/kernel/hostname

# Random entropy
cat /proc/sys/kernel/random/entropy_avail
```

### Virtual Memory

```bash
# Swappiness
cat /proc/sys/vm/swappiness               # 60

# Overcommit mode
cat /proc/sys/vm/overcommit_memory        # 0=heuristic 1=always 2=never
cat /proc/sys/vm/overcommit_ratio         # 50 (% of RAM, for mode 2)

# Dirty page writeback
cat /proc/sys/vm/dirty_ratio              # 20 (% of RAM, sync writeback)
cat /proc/sys/vm/dirty_background_ratio   # 10 (% of RAM, async writeback)
cat /proc/sys/vm/dirty_expire_centisecs   # 3000 (30 seconds)

# Min free kbytes
cat /proc/sys/vm/min_free_kbytes          # 67584

# Memory compaction
cat /proc/sys/vm/compact_memory           # Write 1 to trigger

# OOM controls
cat /proc/sys/vm/panic_on_oom             # 0
cat /proc/sys/vm/oom_kill_allocating_task # 0
```

### Filesystem

```bash
# File descriptor limits
cat /proc/sys/fs/file-max                 # System-wide FD limit
cat /proc/sys/fs/file-nr                  # allocated  free  max

# Inotify limits
cat /proc/sys/fs/inotify/max_user_watches    # 65536
cat /proc/sys/fs/inotify/max_user_instances  # 128
cat /proc/sys/fs/inotify/max_queued_events   # 16384

# AIO limits
cat /proc/sys/fs/aio-max-nr              # 1048576

# Dentry cache pressure
cat /proc/sys/fs/lease-break-time         # 45 seconds
```

## /sys Filesystem (sysfs)

```bash
# Block device information
ls /sys/block/
# sda  sdb  nvme0n1

# Disk scheduler
cat /sys/block/sda/queue/scheduler
# [mq-deadline] kyber bfq none

echo mq-deadline > /sys/block/sda/queue/scheduler

# Block device queue depth
cat /sys/block/sda/queue/nr_requests      # 256

# CPU frequency governor
cat /sys/devices/system/cpu/cpu0/cpufreq/scaling_governor
# performance  powersave  schedutil

# Network interface info
cat /sys/class/net/eth0/speed             # 1000 (Mbps)
cat /sys/class/net/eth0/mtu              # 1500
cat /sys/class/net/eth0/address          # MAC address
cat /sys/class/net/eth0/operstate        # up/down

# Power management
cat /sys/power/state
# freeze mem disk

# NUMA topology
ls /sys/devices/system/node/
cat /sys/devices/system/node/node0/meminfo

# Huge pages
cat /sys/kernel/mm/hugepages/hugepages-2048kB/nr_hugepages
echo 1024 > /sys/kernel/mm/hugepages/hugepages-2048kB/nr_hugepages

# Transparent Huge Pages
cat /sys/kernel/mm/transparent_hugepage/enabled
# [always] madvise never
echo never > /sys/kernel/mm/transparent_hugepage/enabled
```

## Using sysctl

```bash
# View all tunables
sysctl -a

# Get a specific value
sysctl net.ipv4.ip_forward

# Set a value
sysctl -w net.ipv4.ip_forward=1

# Load from config file
sysctl -p /etc/sysctl.conf

# Persistent configuration
# /etc/sysctl.d/99-custom.conf
# net.core.somaxconn = 65535
# vm.swappiness = 10
# net.ipv4.ip_forward = 1
# fs.inotify.max_user_watches = 524288
```

## Tips

- `MemAvailable` in `/proc/meminfo` is the best single metric for "how much memory can I allocate"
- `/proc/$PID/smaps_rollup` gives you PSS (Proportional Set Size) which is more accurate than RSS for shared libraries
- Always use `sysctl -p` or drop-in files in `/etc/sysctl.d/` for persistent kernel tuning
- `/proc/$PID/io` shows actual disk I/O per process, unlike `top` which shows CPU/memory only
- The `/sys/block/*/queue/scheduler` setting dramatically impacts I/O performance; use `mq-deadline` for databases
- Disable Transparent Huge Pages (`echo never > /sys/kernel/mm/transparent_hugepage/enabled`) for databases like Redis and MongoDB
- `/proc/sys/net/ipv4/ip_local_port_range` determines ephemeral ports; widen it for high-connection servers
- Use `/proc/$PID/fd/` to count open file descriptors and detect FD leaks in running processes
- `cat /proc/$PID/stack` shows the kernel stack trace, invaluable for debugging hung processes
- `/proc/vmstat` provides global page fault, swap, and reclaim counters for performance analysis
- Check `/proc/sys/fs/file-nr` to see how many FDs are currently allocated system-wide
- Use `/proc/pressure/` (PSI) files for early warning of CPU, memory, and I/O saturation

## See Also

cgroups, ulimit, oom-killer, swap, inotify

## References

- [proc(5) Man Page](https://man7.org/linux/man-pages/man5/proc.5.html)
- [sysfs(5) Man Page](https://man7.org/linux/man-pages/man5/sysfs.5.html)
- [Linux Kernel /proc Documentation](https://docs.kernel.org/filesystems/proc.html)
- [Linux Kernel Sysctl Documentation](https://docs.kernel.org/admin-guide/sysctl/index.html)
- [Red Hat Performance Tuning Guide](https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux/9/html/monitoring_and_managing_system_status_and_performance)
