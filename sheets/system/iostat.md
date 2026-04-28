# iostat (I/O and CPU statistics)

Report CPU and device I/O statistics. Part of the sysstat package.

## Basic Usage

### Default Report

```bash
# Single snapshot (since boot)
iostat

# Repeat every 2 seconds
iostat 2

# 5 reports at 2-second intervals
iostat 2 5
```

## CPU Statistics

### CPU Only

```bash
iostat -c

# CPU stats every 1 second
iostat -c 1
```

## Device Statistics

### Device I/O

```bash
# Device stats only (no CPU)
iostat -d

# Extended device stats (the most useful view)
iostat -dx

# Specific device
iostat -dx sda

# NVMe device
iostat -dx nvme0n1
```

### Extended Stats Columns

```bash
# iostat -dx output columns:
#   r/s     — reads per second
#   w/s     — writes per second
#   rkB/s   — kB read per second
#   wkB/s   — kB written per second
#   rrqm/s  — read requests merged per second
#   wrqm/s  — write requests merged per second
#   await   — average wait time (ms) including queue
#   r_await — average read wait (ms)
#   w_await — average write wait (ms)
#   %util   — percent of time device was busy
```

## Output Formats

### Human-Readable and JSON

```bash
# Use megabytes instead of kilobytes
iostat -m

# Show in megabytes with extended stats
iostat -dxm 1

# JSON output (sysstat 12+)
iostat -o JSON -dx 1 1

# Omit first report (which is since-boot average)
iostat -dx -y 1
```

## Monitoring Patterns

### Continuous Monitoring

```bash
# Watch all devices every second, megabytes, extended
iostat -dxm 1

# Watch a specific device with timestamps
iostat -dx -t sda 1

# Quick check: 3 reports, skip the boot-average first one
iostat -dxm -y 1 3
```

## Tips

- The first report from `iostat` is always an average since boot -- use `-y` to skip it and see live data.
- `%util` near 100% does not always mean saturation on SSDs/NVMe; those devices handle parallel I/O. Check `await` instead.
- `await` above 10ms on SSDs or above 20ms on HDDs usually indicates I/O pressure.
- Install with `apt install sysstat` or `yum install sysstat`.
- `iostat -p ALL` shows per-partition stats (not just whole devices).
- Pair with `iotop` for per-process I/O breakdown.

## All Flags Reference

```bash
iostat [options] [interval [count]]

# CPU + device control
-c             CPU stats only (no device section)
-d             device stats only (no CPU section)
-x             extended device stats (the column set you usually want)
-h             human-readable: K/M/G suffixes, aligned columns
-N             show device-mapper / LVM logical names (dm-0 → vg-root-lv)
-p [device|ALL]  per-partition stats (default: whole devices only)
-z             omit devices with zero activity in the interval
-y             skip the first since-boot summary report

# Units
-k             kilobytes (default)
-m             megabytes
-H             group filesystems by mountpoint (sysstat 12+)

# Time / timestamping
-t             prefix each report with a timestamp
-y             omit the first since-boot report
[interval]     seconds between reports (e.g. iostat 2)
[count]        number of reports total (e.g. iostat 2 5)

# Output formats (sysstat 12+)
-o JSON        machine-parseable JSON
-o XML         (deprecated)
--pretty       wider, more readable extended view
--human        like -h but for the JSON renderer too
--dec=N        decimal places (default 2)
```

## Reading Extended Stats Like a Pro

```bash
$ iostat -dxh -y 1 3
Device   r/s    w/s   rkB/s   wkB/s   rrqm/s  wrqm/s  %rrqm  %wrqm  r_await  w_await  aqu-sz  rareq-sz  wareq-sz  svctm  %util
sda     12.0   89.4   192k    1.8M     0.0    142.0    0.0   61.4    0.42     2.83     0.27     16.0      20.6     0.18   2.31
nvme0n1  4.5  1003.4  72k    62.4M     0.0     12.3    0.0    1.2    0.04     0.08     0.08     16.0      63.7     0.01   0.78
```

Column meanings — what to actually look at:

| Column | Means | Yellow flag | Red flag |
|--------|-------|-------------|----------|
| `r/s`, `w/s` | IOPS | (depends on workload) | (depends on workload) |
| `rkB/s`, `wkB/s` | bandwidth | nearing device max | sustained at device max |
| `rrqm/s`, `wrqm/s` | merged requests | high merge ratio = good (sequential) | very high may indicate scheduler delay |
| `%rrqm`, `%wrqm` | merge percentage | <5% on sequential workload (suspect random) | n/a |
| `r_await`, `w_await` | total time per request (ms, includes queue) | HDD: >20 / SSD: >5 / NVMe: >2 | HDD: >50 / SSD: >10 / NVMe: >5 |
| `aqu-sz` | average queue depth | >1.0 sustained | >5.0 sustained |
| `rareq-sz`, `wareq-sz` | avg request size (sectors usually) | n/a | n/a (informational) |
| `svctm` | service time (ms) — **deprecated/unreliable**, ignore | — | — |
| `%util` | % of wall-clock time the device had at least one request in flight | >70% on HDD | >90% sustained on HDD |

**Important:** `%util` is misleading on modern SSDs/NVMe. They handle many parallel requests; 100% util does NOT mean saturation. Trust `await` and `aqu-sz` instead.

## Performance Debugging Recipes

### Recipe 1 — "the disk feels slow"

```bash
# Step 1: confirm with extended stats, skip the boot average
iostat -dxh -y 1 5

# Step 2: identify the offending device (highest await / aqu-sz)
# Step 3: find the culprit process
sudo iotop -o -P -d 1
# -o = only show processes doing I/O
# -P = aggregate by PID (not threads)

# Step 4: drill into syscalls if needed
sudo strace -c -p <pid>            # syscall summary
sudo perf trace -p <pid>           # syscall timing live

# Step 5: confirm at block layer
cat /proc/diskstats                # raw kernel counters
```

### Recipe 2 — "is this disk dying?"

```bash
# Sustained high await with low IOPS = symptom of remap-storms / dying media
iostat -dxht -y 5 12               # 1 minute of samples

# Cross-reference with SMART
sudo smartctl -A /dev/sda | grep -E "Reallocated|Pending|Uncorrect"

# Check error counters in dmesg
sudo dmesg --human --level=warn,err | grep -iE "ata|scsi|nvme|i/o error"
```

### Recipe 3 — "sequential vs random workload"

```bash
# Sequential reads merge well (high %rrqm), random reads do not.
# rrqm/s / r/s ratio > 0.5 means most reads are getting merged → sequential.

# Force the test: dd vs fio random
fio --name=seq --rw=read --bs=1m --size=1G --filename=/dev/sda
fio --name=rand --rw=randread --bs=4k --size=1G --filename=/dev/sda

# Compare iostat output during each. Random workloads show:
#   - low rrqm/s (no merges)
#   - small rareq-sz (one block per request)
#   - higher await (each seek costs)
```

### Recipe 4 — comparing two NVMes

```bash
# Show only NVMe devices, every second, omit idle
iostat -dxh -y -p nvme0n1,nvme1n1 1

# Filter further to only show busy devices
iostat -dxh -y -z 1
```

## sysstat Cousins

`iostat` is one tool in the sysstat package. Its siblings:

```bash
sar -d 1 5            # historical I/O stats (replays from /var/log/sysstat/)
sar -u 1 5            # CPU
sar -r 1 5            # memory
sar -n DEV 1 5        # network per-interface
sar -B 1 5            # paging activity
sar -W 1 5            # swap activity
sar -q 1 5            # load average + run queue

mpstat -P ALL 1       # per-CPU stats (mpstat is the CPU-focused sibling)

pidstat -d 1          # per-process I/O (read/write KB/s, kB_ccwr/s, iodelay)
pidstat -u 1          # per-process CPU
pidstat -r 1          # per-process memory
pidstat -dl 1         # per-process I/O with command line
```

## JSON Output for Pipelines

```bash
# Single snapshot of extended stats as JSON
iostat -dxh -y -o JSON 1 1

# Pipe into jq to extract just %util per device
iostat -dxh -y -o JSON 1 1 | jq '.sysstat.hosts[0].statistics[0].disk[]
                                | {name, util: .["%util"]}'

# Continuous JSON stream (one report per line) for ingestion into a TSDB
iostat -dxh -o JSON 5 | jq -c '.sysstat.hosts[0].statistics[]'
```

## Common Errors and Fixes

```bash
# "iostat: command not found"
# Fix:
sudo apt install sysstat              # Debian/Ubuntu
sudo dnf install sysstat              # RHEL/Fedora
sudo apk add sysstat                  # Alpine
brew install sysstat                  # macOS (limited; Linux-only kernel hooks)

# Always-zero stats on Debian/Ubuntu
# Cause: sysstat collection daemon disabled by default.
$ sudo systemctl status sysstat
inactive (dead)
# Fix: enable historical collection (needed for `sar`, not iostat itself)
sudo systemctl enable --now sysstat

# Empty extended stats on a clean container/cgroup
# Cause: container can't read /proc/diskstats for the host devices.
# Fix: use --privileged (with care) or mount /proc/diskstats read-only.

# Numbers seem absurdly high
# Cause: the FIRST report after start is the since-boot AVERAGE — it can show
#        years of accumulated I/O.
# Fix: always use `-y` to skip it, OR run `iostat 2 5` and ignore the first line.

# Per-partition stats missing
# Cause: -p needed, and on some kernels block-layer stats are off.
$ iostat -dxp ALL 1
# If stats are still all zeros, check:
$ cat /proc/sys/kernel/sched_rt_runtime_us  # not the issue, but related
$ ls /sys/block/*/stat                       # ensure block stats are populated
```

## Tips

- The first report from `iostat` is always an average since boot — use `-y` to skip it and see live data.
- `%util` near 100% does NOT always mean saturation on SSDs/NVMe; those devices handle parallel I/O. Check `await` and `aqu-sz` instead.
- `await` above 10ms on SSDs or above 20ms on HDDs usually indicates I/O pressure.
- Install with `apt install sysstat` or `yum install sysstat`.
- `iostat -p ALL` shows per-partition stats (not just whole devices).
- Pair with `iotop` (or `pidstat -d 1`) for per-process I/O breakdown.
- The `svctm` column is documented as "service time" but is unreliable on modern kernels — sysstat's man page says ignore it.
- `iostat -N` resolves device-mapper names (dm-0 → vg-root-lv) — extremely useful on LVM systems.
- `iostat -o JSON` is sysstat 12+. Older versions (sysstat 11 ships on RHEL 7) lack JSON entirely; use awk/sed.
- For long-term trending, configure sysstat's data collection (`/etc/cron.d/sysstat` on Debian) and use `sar -d -f /var/log/sysstat/saYYYYMMDD` to replay any past day.
- On NVMe, the queue depth that matters is per-namespace; check `cat /sys/class/nvme/nvme0/nvme0n1/queue/nr_requests`.

## See Also

- system/vmstat, system/sar, system/htop, system/lsof, troubleshooting/linux-errors

## References

- [man iostat(1)](https://man7.org/linux/man-pages/man1/iostat.1.html)
- [man sar(1)](https://man7.org/linux/man-pages/man1/sar.1.html)
- [man pidstat(1)](https://man7.org/linux/man-pages/man1/pidstat.1.html)
- [sysstat Project Site](https://sysstat.github.io/)
- [sysstat GitHub](https://github.com/sysstat/sysstat)
- [Kernel Block Layer Statistics](https://www.kernel.org/doc/html/latest/block/stat.html)
- [Arch Wiki — Sysstat](https://wiki.archlinux.org/title/Sysstat)
- [Red Hat — Monitoring I/O Performance](https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux/9/html/monitoring_and_managing_system_status_and_performance/monitoring-disk-i-o-performance-with-iostat_monitoring-and-managing-system-status-and-performance)
