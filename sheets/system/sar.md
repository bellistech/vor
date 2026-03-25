# sar (system activity reporter)

Collect, report, and save system activity data. Part of the sysstat package.

## CPU

### CPU Utilization

```bash
# Current CPU stats, 1-second interval, 5 reports
sar -u 1 5

# All CPUs individually
sar -u ALL 1 5

# Per-core breakdown
sar -P ALL 1 5

# Specific core
sar -P 0 1 5
```

## Memory

### Memory Utilization

```bash
# Memory usage
sar -r 1 5

# Includes %memused, kbmemfree, kbbuffers, kbcached
# Add ALL for extra fields
sar -r ALL 1 5

# Swap usage
sar -S 1 5
```

## Disk

### Disk I/O

```bash
# All block devices
sar -d 1 5

# Pretty device names (use dev mapper names)
sar -d -p 1 5
```

## Network

### Network Statistics

```bash
# Network interface stats (packets, bytes, errors)
sar -n DEV 1 5

# Network errors
sar -n EDEV 1 5

# Socket statistics
sar -n SOCK 1 5

# TCP stats (segments, retransmits)
sar -n TCP 1 5

# All network stats
sar -n ALL 1 5
```

## Load Average

### System Load

```bash
# Run queue length and load averages
sar -q 1 5
```

## Historical Data

### Read From Saved Files

```bash
# sar saves daily files in /var/log/sa/ (or /var/log/sysstat/)
# Today's file
sar -u -f /var/log/sa/sa$(date +%d)

# Yesterday's file
sar -u -f /var/log/sa/sa$(date -d yesterday +%d)

# Specific day
sar -u -f /var/log/sa/sa15
```

### Time Range

```bash
# CPU data between 09:00 and 12:00 from today's file
sar -u -s 09:00:00 -e 12:00:00

# Memory from yesterday, 14:00-16:00
sar -r -f /var/log/sa/sa$(date -d yesterday +%d) -s 14:00:00 -e 16:00:00
```

### Binary Data Collection

```bash
# Collect data to a binary file (10 samples, 1-sec interval)
sar -o /tmp/sar_capture.dat 1 10

# Read it back
sar -u -f /tmp/sar_capture.dat

# Collect everything
sar -A -o /tmp/full_capture.dat 1 60
```

## All-in-One

### Full Report

```bash
# Everything sar can report
sar -A 1 1
```

## Tips

- `sar` needs the sysstat service running to collect historical data -- enable it with `systemctl enable --now sysstat`.
- Data collection is done by `sa1` (called by cron or systemd timer every 10 minutes) and summarized by `sa2` daily.
- Historical data files are in `/var/log/sa/saDD` (binary) and `/var/log/sa/sarDD` (text).
- Default retention is 7-28 days depending on distro; change `HISTORY` in `/etc/sysstat/sysstat`.
- `sar -n DEV` is the quickest way to check network throughput historically without having had a monitoring agent running.
- On RHEL/CentOS the config file is `/etc/sysconfig/sysstat`; on Debian it is `/etc/sysstat/sysstat`.
