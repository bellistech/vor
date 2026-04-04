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

## See Also

- vmstat, sar, htop, perf, strace

## References

- [man iostat(1)](https://man7.org/linux/man-pages/man1/iostat.1.html)
- [man sar(1)](https://man7.org/linux/man-pages/man1/sar.1.html)
- [man pidstat(1)](https://man7.org/linux/man-pages/man1/pidstat.1.html)
- [sysstat Project Site](https://sysstat.github.io/)
- [sysstat GitHub](https://github.com/sysstat/sysstat)
- [Kernel Block Layer Statistics](https://www.kernel.org/doc/html/latest/block/stat.html)
- [Arch Wiki — Sysstat](https://wiki.archlinux.org/title/Sysstat)
- [Red Hat — Monitoring I/O Performance](https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux/9/html/monitoring_and_managing_system_status_and_performance/monitoring-disk-i-o-performance-with-iostat_monitoring-and-managing-system-status-and-performance)
