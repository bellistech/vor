# nice (process priority)

Control CPU and I/O scheduling priority for processes.

## CPU Priority (nice)

### Launch with Modified Priority

```bash
# Nice values range from -20 (highest priority) to 19 (lowest)
# Default nice value is 0

# Run with lower priority (nice to other processes)
nice -n 10 /usr/local/bin/heavy_job.sh

# Run with even lower priority
nice -n 19 make -j8

# Run with higher priority (requires root)
nice -n -5 /usr/local/bin/critical.sh

# Highest priority (requires root)
nice -n -20 /usr/local/bin/realtime_critical.sh
```

### Change Running Process Priority

```bash
# Renice a running process
renice -n 10 -p 1234

# Renice all processes of a user
renice -n 15 -u deploy

# Renice a process group
renice -n 5 -g 5678

# Make a running process higher priority (requires root)
renice -n -5 -p 1234
```

### Check Current Nice Value

```bash
# Show nice value in ps output
ps -eo pid,ni,comm --sort=-ni

# Check a specific process
ps -o pid,ni,comm -p 1234
```

## I/O Priority (ionice)

### Scheduling Classes

```bash
# Classes:
#   0 — None (use CPU nice to derive I/O priority)
#   1 — Realtime (highest, requires root, priority 0-7)
#   2 — Best-effort (default, priority 0-7)
#   3 — Idle (only when no other I/O, no priority level)

# Run with idle I/O priority (won't compete for disk)
ionice -c 3 rsync -a /data/ /backup/

# Run with best-effort, low priority
ionice -c 2 -n 7 tar czf backup.tar.gz /var/data/

# Run with realtime I/O (requires root)
ionice -c 1 -n 0 dd if=/dev/sda of=/dev/sdb

# Check I/O priority of a running process
ionice -p 1234

# Change I/O class of a running process
ionice -c 3 -p 1234
```

## Combining CPU and I/O

### Low-Impact Background Jobs

```bash
# Low CPU priority + idle I/O = minimal system impact
nice -n 19 ionice -c 3 find / -type f -name "*.log" -mtime +30 -delete

# Background compilation
nice -n 15 ionice -c 2 -n 7 make -j$(nproc)

# Low-priority backup
nice -n 19 ionice -c 3 rsync -a --bwlimit=10m /data/ /backup/
```

## Tips

- Regular users can only increase nice values (lower priority). Only root can set negative nice values.
- Nice value affects CPU scheduling share, not hard limits -- a process at nice 19 will still use 100% CPU if nothing else wants it.
- `ionice -c 3` (idle) is extremely useful for backups and maintenance tasks -- they only get I/O when the disk is otherwise idle.
- The CFQ I/O scheduler respects ionice classes. If you use `mq-deadline` or `none` (common on NVMe), ionice has no effect.
- `chrt` is the tool for real-time CPU scheduling (FIFO, round-robin) -- different from nice values.
- On systemd-managed services, use `Nice=` and `IOSchedulingClass=` in the unit file instead of wrapping with nice/ionice.
