# vmstat (virtual memory statistics)

Report virtual memory, swap, I/O, CPU, and system activity.

## Basic Usage

### Default Report

```bash
# Single snapshot
vmstat

# Repeat every 2 seconds
vmstat 2

# 5 reports at 1-second intervals
vmstat 1 5
```

## Memory

### Memory and Swap Info

```bash
# Default output includes:
#   swpd  — swap used (KB)
#   free  — free memory (KB)
#   buff  — buffer memory (KB)
#   cache — cache memory (KB)

vmstat 1
```

### Wide Output (Full Column Names)

```bash
# Wider display with separate buff and cache columns
vmstat -w 1
```

## Swap Activity

### Swap In/Out

```bash
# si = swap in from disk (KB/s)
# so = swap out to disk (KB/s)
# Non-zero so means the system is under memory pressure
vmstat 1

# Show swap summary
vmstat -s
```

## I/O

### Block I/O

```bash
# bi = blocks received from disk (blocks/s)
# bo = blocks sent to disk (blocks/s)
vmstat 1

# Disk-specific stats
vmstat -d

# Partition-specific stats
vmstat -p sda1
```

## CPU

### CPU Breakdown

```bash
# Columns: us (user), sy (system), id (idle), wa (iowait), st (steal)
vmstat 1

# High wa% = processes blocked on disk I/O
# High sy% = kernel overhead (context switches, interrupts)
# Non-zero st = hypervisor stealing CPU (VM contention)
```

## System Activity

### Context Switches and Interrupts

```bash
# in = interrupts per second
# cs = context switches per second
vmstat 1
```

## Active/Inactive Memory

### Detailed Memory Stats

```bash
# Include active/inactive memory columns
vmstat -a 1

# Full memory statistics summary
vmstat -s
```

## Output Formats

### Timestamps and Units

```bash
# Add timestamps
vmstat -t 1

# Show in megabytes (not all versions)
vmstat -S M 1

# Show in kilobytes (default)
vmstat -S k 1
```

## Tips

- Like iostat, the first line of vmstat output is an average since boot -- ignore it and watch subsequent lines.
- `wa` (iowait) above 10% usually indicates an I/O bottleneck.
- `si`/`so` should both be 0 under normal conditions -- any sustained swap activity means you need more RAM or have a memory leak.
- `cs` in the thousands is normal; in the hundreds of thousands suggests excessive context switching.
- `vmstat -s` gives a one-shot memory summary that is easier to read than `/proc/meminfo`.
- `vmstat -d` output is cumulative (since boot), not per-second.

## References

- [man vmstat(8)](https://man7.org/linux/man-pages/man8/vmstat.8.html)
- [man proc(5) — /proc/meminfo, /proc/stat](https://man7.org/linux/man-pages/man5/proc.5.html)
- [man free(1)](https://man7.org/linux/man-pages/man1/free.1.html)
- [man sar(1)](https://man7.org/linux/man-pages/man1/sar.1.html)
- [procps-ng Project (vmstat source)](https://gitlab.com/procps-ng/procps)
- [Kernel /proc/meminfo Documentation](https://www.kernel.org/doc/html/latest/filesystems/proc.html)
- [Kernel VM Subsystem Documentation](https://www.kernel.org/doc/html/latest/admin-guide/sysctl/vm.html)
- [Arch Wiki — Memory Management](https://wiki.archlinux.org/title/Memory)
- [Red Hat — Monitoring Virtual Memory](https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux/9/html/monitoring_and_managing_system_status_and_performance/monitoring-memory-usage_monitoring-and-managing-system-status-and-performance)
