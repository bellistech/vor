# I/O Scheduler Tuning (Linux Block Layer Schedulers and Queue Optimization)

Configure and tune Linux multi-queue I/O schedulers, manage per-device scheduling policies, and optimize queue depths and read-ahead for different storage workloads.

## Multi-Queue Schedulers Overview

### Available Schedulers (blk-mq)

```bash
# List available schedulers for a device (active one in brackets)
cat /sys/block/sda/queue/scheduler
# [mq-deadline] kyber bfq none

# List for all block devices
for dev in /sys/block/*/queue/scheduler; do
  echo "$(dirname $(dirname $dev) | xargs basename): $(cat $dev)"
done

# Check if device uses multi-queue (blk-mq) — all modern kernels (5.0+)
cat /sys/block/sda/queue/nr_hw_queues
# >1 means hardware multi-queue (NVMe typically 32+)

# Check if rotational (HDD=1, SSD/NVMe=0)
cat /sys/block/sda/queue/rotational
```

## Checking and Setting Schedulers

### Runtime Changes

```bash
# Set scheduler for a single device
echo mq-deadline | sudo tee /sys/block/sda/queue/scheduler
echo bfq | sudo tee /sys/block/sdb/queue/scheduler
echo none | sudo tee /sys/block/nvme0n1/queue/scheduler

# Verify the change
cat /sys/block/sda/queue/scheduler
# [mq-deadline] kyber bfq none
```

### Persistent via udev

```bash
# Persist scheduler by device type (survives reboot)
cat <<'EOF' | sudo tee /etc/udev/rules.d/60-io-scheduler.rules
# NVMe — none (passthrough to hardware queues)
ACTION=="add|change", KERNEL=="nvme[0-9]*", ATTR{queue/scheduler}="none"

# SSD (non-rotational SCSI/SATA) — mq-deadline or kyber
ACTION=="add|change", KERNEL=="sd[a-z]*", ATTR{queue/rotational}=="0", ATTR{queue/scheduler}="kyber"

# HDD (rotational) — mq-deadline or bfq
ACTION=="add|change", KERNEL=="sd[a-z]*", ATTR{queue/rotational}=="1", ATTR{queue/scheduler}="mq-deadline"
EOF

sudo udevadm control --reload-rules && sudo udevadm trigger
```

### Persistent via Kernel Parameter

```bash
# Set default scheduler for all devices at boot (GRUB)
# Edit /etc/default/grub:
GRUB_CMDLINE_LINUX="elevator=mq-deadline"

# Apply
sudo update-grub        # Debian/Ubuntu
sudo grub2-mkconfig -o /boot/grub2/grub.cfg  # RHEL/Fedora
```

## mq-deadline

### Overview and Parameters

```bash
# mq-deadline — best general-purpose scheduler for SSDs and HDDs
# Guarantees request completion within a deadline (prevents starvation)
echo mq-deadline | sudo tee /sys/block/sda/queue/scheduler

# Read deadline — max time (ms) a read can wait before being dispatched
cat /sys/block/sda/queue/iosched/read_expire
# 500 (default)
echo 300 | sudo tee /sys/block/sda/queue/iosched/read_expire

# Write deadline — max time (ms) a write can wait
cat /sys/block/sda/queue/iosched/write_expire
# 5000 (default)
echo 3000 | sudo tee /sys/block/sda/queue/iosched/write_expire

# fifo_batch — number of requests to dispatch in one batch from the sorted queue
# Lower = more responsive but less throughput (more seeks on HDD)
cat /sys/block/sda/queue/iosched/fifo_batch
# 16 (default)
echo 8 | sudo tee /sys/block/sda/queue/iosched/fifo_batch

# front_merges — allow merging of new requests at the front of existing ones
# 1=enabled (default), 0=disabled (disable if random I/O dominant)
cat /sys/block/sda/queue/iosched/front_merges
# 1
echo 0 | sudo tee /sys/block/sda/queue/iosched/front_merges

# writes_starved — how many read batches before a write batch gets dispatched
# Higher = favor reads over writes
cat /sys/block/sda/queue/iosched/writes_starved
# 2 (default — 2 read batches per 1 write batch)
echo 4 | sudo tee /sys/block/sda/queue/iosched/writes_starved
```

### Tuning Profiles

```bash
# Database server — favor reads, tighter deadlines
echo 100 | sudo tee /sys/block/sda/queue/iosched/read_expire
echo 2000 | sudo tee /sys/block/sda/queue/iosched/write_expire
echo 4 | sudo tee /sys/block/sda/queue/iosched/writes_starved
echo 8 | sudo tee /sys/block/sda/queue/iosched/fifo_batch

# File server — balanced reads/writes
echo 500 | sudo tee /sys/block/sda/queue/iosched/read_expire
echo 5000 | sudo tee /sys/block/sda/queue/iosched/write_expire
echo 2 | sudo tee /sys/block/sda/queue/iosched/writes_starved
echo 16 | sudo tee /sys/block/sda/queue/iosched/fifo_batch
```

## BFQ (Budget Fair Queueing)

### Overview and Core Parameters

```bash
# BFQ — fair bandwidth allocation with weight-based priority
# Best for: desktops, mixed workloads, QoS requirements, HDDs
echo bfq | sudo tee /sys/block/sda/queue/scheduler

# low_latency — prioritize interactive I/O (1=enabled, 0=throughput mode)
cat /sys/block/sda/queue/iosched/low_latency
# 1 (default)
echo 1 | sudo tee /sys/block/sda/queue/iosched/low_latency

# timeout_sync — max time (ms) a sync process can hold the disk
cat /sys/block/sda/queue/iosched/timeout_sync
# 124 (default)

# timeout_async — max time (ms) for async (background) I/O
cat /sys/block/sda/queue/iosched/timeout_async
# 250 (default)

# max_budget — max number of sectors BFQ dispatches in one round
# 0 = auto (default, kernel selects optimal value)
cat /sys/block/sda/queue/iosched/max_budget

# strict_guarantees — enforce throughput distribution exactly per weight
# 0=off (default), 1=on (lower throughput but stricter fairness)
echo 1 | sudo tee /sys/block/sda/queue/iosched/strict_guarantees
```

### Weight-Based Priority (cgroups v2)

```bash
# BFQ uses cgroup-based weight assignment (100-10000, default 100)
# Higher weight = more bandwidth share

# Check current weight for a cgroup
cat /sys/fs/cgroup/my-app/io.bfq.weight
# default 100

# Set weight for a cgroup (gets proportionally more bandwidth)
echo "default 500" > /sys/fs/cgroup/my-app/io.bfq.weight

# Per-device weight
echo "8:0 300" > /sys/fs/cgroup/my-app/io.bfq.weight

# Example: database gets 5x bandwidth vs background backups
echo "default 500" > /sys/fs/cgroup/database/io.bfq.weight
echo "default 100" > /sys/fs/cgroup/backup/io.bfq.weight
# Database gets ~83% bandwidth (500/(500+100))
# Backup gets ~17% bandwidth (100/(500+100))
```

## Kyber

### Overview and Latency Targets

```bash
# Kyber — latency-oriented scheduler, minimal tuning knobs
# Best for: fast SSDs and NVMe where you want low latency
echo kyber | sudo tee /sys/block/sda/queue/scheduler

# read_lat_nsec — target read latency in nanoseconds
cat /sys/block/sda/queue/iosched/read_lat_nsec
# 2000000 (2ms default)
echo 1000000 | sudo tee /sys/block/sda/queue/iosched/read_lat_nsec  # 1ms

# write_lat_nsec — target write latency in nanoseconds
cat /sys/block/sda/queue/iosched/write_lat_nsec
# 10000000 (10ms default)
echo 5000000 | sudo tee /sys/block/sda/queue/iosched/write_lat_nsec  # 5ms

# Kyber auto-adjusts queue depths to meet latency targets
# It throttles submission queues when latency exceeds the target
# Two internal queues: KYBER_READ (latency-sensitive) and KYBER_OTHER
```

### Kyber for Low-Latency Storage

```bash
# Aggressive latency targets for NVMe
echo kyber | sudo tee /sys/block/nvme0n1/queue/scheduler
echo 500000 | sudo tee /sys/block/nvme0n1/queue/iosched/read_lat_nsec   # 500us
echo 2000000 | sudo tee /sys/block/nvme0n1/queue/iosched/write_lat_nsec  # 2ms
```

## none (noop)

### When to Use

```bash
# none — no scheduling, direct passthrough to hardware
# Best for: NVMe (has its own hardware scheduler), VM guests (host handles scheduling)
echo none | sudo tee /sys/block/nvme0n1/queue/scheduler

# Verify — should show [none]
cat /sys/block/nvme0n1/queue/scheduler
# mq-deadline kyber bfq [none]

# NVMe devices have their own internal schedulers with:
# - Hardware multi-queue (often 32-128 queues)
# - Internal reordering and merging
# - Adding a software scheduler adds overhead with no benefit

# Check NVMe hardware queue count
cat /sys/block/nvme0n1/queue/nr_hw_queues
# 32 (typical — one per CPU core)

# For virtual disks (VM guests)
echo none | sudo tee /sys/block/vda/queue/scheduler
# Host hypervisor already handles scheduling — double scheduling hurts
```

## ionice — I/O Priority Classes

### Setting Process I/O Priority

```bash
# ionice classes:
#   1 = Real-time  (highest, 0-7 priority levels)
#   2 = Best-effort (default, 0-7 priority levels)
#   3 = Idle        (only runs when no other I/O)

# Check current ionice of a process
ionice -p $(pidof mysqld)
# best-effort: prio 4

# Start a process with real-time I/O (class 1, priority 0 = highest)
sudo ionice -c 1 -n 0 ./database-server

# Start with best-effort, high priority (class 2, priority 0)
ionice -c 2 -n 0 ./important-app

# Start with idle I/O (class 3 — only when disk is idle)
ionice -c 3 tar czf backup.tar.gz /data

# Change ionice of a running process
sudo ionice -c 1 -n 0 -p $(pidof mysqld)

# Combine with nice (CPU + I/O priority)
nice -n 19 ionice -c 3 rsync -a /src /dst

# CFQ/BFQ respect ionice classes; mq-deadline and kyber do not
# For mq-deadline/kyber, use cgroup-based I/O throttling instead
```

### Persist via systemd

```bash
# In a systemd unit file:
# [Service]
# IOSchedulingClass=realtime
# IOSchedulingPriority=0
# Nice=0

# Or for background tasks:
# IOSchedulingClass=idle
# Nice=19
```

## Read-Ahead Tuning

### blockdev and sysfs

```bash
# Check current read-ahead in KB
cat /sys/block/sda/queue/read_ahead_kb
# 128 (default)

# Check via blockdev (shows 512-byte sectors)
sudo blockdev --getra /dev/sda
# 256 (= 128KB)

# Set read-ahead via sysfs (KB)
echo 2048 | sudo tee /sys/block/sda/queue/read_ahead_kb

# Set via blockdev (512-byte sectors)
sudo blockdev --setra 4096 /dev/sda  # = 2048KB

# Sequential HDD workloads — increase read-ahead
echo 4096 | sudo tee /sys/block/sda/queue/read_ahead_kb  # 4MB

# Random I/O (databases) — reduce read-ahead to avoid waste
echo 64 | sudo tee /sys/block/sda/queue/read_ahead_kb

# NVMe — lower read-ahead, hardware handles prefetch
echo 128 | sudo tee /sys/block/nvme0n1/queue/read_ahead_kb
```

### Persist via udev

```bash
cat <<'EOF' | sudo tee /etc/udev/rules.d/61-readahead.rules
# HDD — large read-ahead for sequential throughput
ACTION=="add|change", KERNEL=="sd[a-z]*", ATTR{queue/rotational}=="1", ATTR{queue/read_ahead_kb}="2048"

# SSD — moderate read-ahead
ACTION=="add|change", KERNEL=="sd[a-z]*", ATTR{queue/rotational}=="0", ATTR{queue/read_ahead_kb}="256"

# NVMe — default or small
ACTION=="add|change", KERNEL=="nvme[0-9]*", ATTR{queue/read_ahead_kb}="128"
EOF

sudo udevadm control --reload-rules && sudo udevadm trigger
```

## nr_requests — Queue Depth Tuning

### Adjusting Queue Depth

```bash
# nr_requests — max number of I/O requests queued per hardware queue
cat /sys/block/sda/queue/nr_requests
# 256 (typical default)

# Increase for throughput (batching, HDD sequential)
echo 512 | sudo tee /sys/block/sda/queue/nr_requests

# Decrease for latency (fewer queued = less waiting)
echo 64 | sudo tee /sys/block/sda/queue/nr_requests

# Check actual hardware queue depth
cat /sys/block/sda/device/queue_depth
# 32 (SATA NCQ), 1024+ (NVMe)

# Adjust hardware queue depth (SATA NCQ)
echo 1 | sudo tee /sys/block/sda/device/queue_depth   # serialize I/O
echo 32 | sudo tee /sys/block/sda/device/queue_depth  # max NCQ

# Check current I/O in flight
cat /sys/block/sda/inflight
# 0 0 (reads writes)
```

### Persist via udev

```bash
cat <<'EOF' | sudo tee /etc/udev/rules.d/62-queue-depth.rules
# HDD — moderate queue depth
ACTION=="add|change", KERNEL=="sd[a-z]*", ATTR{queue/rotational}=="1", ATTR{queue/nr_requests}="128"

# SSD — higher queue depth for parallelism
ACTION=="add|change", KERNEL=="sd[a-z]*", ATTR{queue/rotational}=="0", ATTR{queue/nr_requests}="256"
EOF

sudo udevadm control --reload-rules && sudo udevadm trigger
```

## Practical sysctl and sysfs Examples

### Complete I/O Tuning Script

```bash
#!/usr/bin/env bash
# io-tune.sh — Apply I/O tuning based on device type
set -euo pipefail

for dev in /sys/block/sd* /sys/block/nvme*; do
  [ -d "$dev" ] || continue
  name=$(basename "$dev")
  rotational=$(cat "$dev/queue/rotational" 2>/dev/null || echo 1)

  if [[ "$name" == nvme* ]]; then
    echo "NVMe: $name — scheduler=none, read_ahead=128KB"
    echo none > "$dev/queue/scheduler"
    echo 128 > "$dev/queue/read_ahead_kb"
  elif [[ "$rotational" == "0" ]]; then
    echo "SSD: $name — scheduler=mq-deadline, read_ahead=256KB"
    echo mq-deadline > "$dev/queue/scheduler"
    echo 256 > "$dev/queue/read_ahead_kb"
    echo 256 > "$dev/queue/nr_requests"
  else
    echo "HDD: $name — scheduler=bfq, read_ahead=2048KB"
    echo bfq > "$dev/queue/scheduler"
    echo 2048 > "$dev/queue/read_ahead_kb"
    echo 128 > "$dev/queue/nr_requests"
  fi
done
```

### Monitoring I/O Performance

```bash
# Real-time I/O stats per device
iostat -xz 1

# Key columns:
#   r/s, w/s      — reads/writes per second
#   rkB/s, wkB/s  — throughput
#   await          — average I/O latency (ms)
#   r_await, w_await — read/write latency separately
#   aqu-sz         — average queue size (should match queue depth)
#   %util          — device utilization (100% = saturated for single-queue)

# BPF-based I/O latency histogram
sudo biolatency -D 1

# Trace individual I/O requests
sudo biotrace

# blktrace — detailed block layer tracing
sudo blktrace -d /dev/sda -o - | blkparse -i -
```

### Verifying Scheduler Impact

```bash
# Quick fio benchmark — random read IOPS
fio --name=randread --ioengine=libaio --direct=1 --bs=4k \
    --iodepth=32 --rw=randread --size=1G --numjobs=4 \
    --runtime=30 --group_reporting --filename=/dev/sda

# Sequential read throughput
fio --name=seqread --ioengine=libaio --direct=1 --bs=128k \
    --iodepth=32 --rw=read --size=1G --numjobs=1 \
    --runtime=30 --group_reporting --filename=/dev/sda

# Compare schedulers — run fio, change scheduler, run again
for sched in mq-deadline bfq kyber none; do
  echo $sched | sudo tee /sys/block/sda/queue/scheduler
  echo "=== $sched ==="
  fio --name=test --ioengine=libaio --direct=1 --bs=4k \
      --iodepth=32 --rw=randread --size=1G --runtime=10 \
      --group_reporting --filename=/dev/sda 2>&1 | grep -E "iops|lat"
done
```

## Tips

- Use `none` for NVMe -- the device has its own hardware scheduler with multiple submission queues; software scheduling adds overhead
- Use `mq-deadline` as the safe default for SSDs and HDDs -- it prevents starvation and is simple to tune
- Use `bfq` when you need fair bandwidth sharing between workloads (desktops, mixed-use servers, cgroup-based QoS)
- Use `kyber` on fast SSDs when latency is the primary concern -- it auto-tunes queue depths to hit latency targets
- `ionice` only works with BFQ (and the legacy CFQ); mq-deadline and kyber ignore I/O priority classes
- Reduce `read_ahead_kb` for random I/O workloads (databases) to avoid reading data that will never be used
- Increase `read_ahead_kb` for sequential workloads (streaming, backups) especially on HDDs
- Lower `nr_requests` reduces latency but hurts throughput; higher allows more batching but increases queue wait time
- In VMs, use `none` on the guest -- the hypervisor host handles scheduling
- Always benchmark with `fio` before and after scheduler changes to measure actual impact
- udev rules are the cleanest way to persist `/sys` tunables -- they fire on device add/change events

## See Also

- kernel, iostat, fio, cgroups, systemd, blktrace

## References

- [Kernel Block Layer Documentation](https://www.kernel.org/doc/html/latest/block/)
- [BFQ I/O Scheduler Documentation](https://www.kernel.org/doc/html/latest/block/bfq-iosched.html)
- [Kyber I/O Scheduler Documentation](https://www.kernel.org/doc/html/latest/block/kyber-iosched.html)
- [blk-mq Documentation](https://www.kernel.org/doc/html/latest/block/blk-mq.html)
- [ionice(1) Man Page](https://man7.org/linux/man-pages/man1/ionice.1.html)
- [blockdev(8) Man Page](https://man7.org/linux/man-pages/man8/blockdev.8.html)
- [Arch Wiki — Improving Performance](https://wiki.archlinux.org/title/Improving_performance#Storage_devices)
- [Red Hat — I/O Schedulers](https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux/9/html/monitoring_and_managing_system_status_and_performance/setting-the-disk-scheduler_monitoring-and-managing-system-status-and-performance)
- [Jens Axboe — blk-mq Design](https://kernel.dk/blk-mq.pdf)
- [Paolo Valente — BFQ Algorithm](https://algo.ing.unimo.it/people/paolo/disk_sched/)
