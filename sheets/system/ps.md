# ps (process status)

Report a snapshot of current processes.

## Common Invocations

### Standard Overviews

```bash
# BSD-style: all processes with user info
ps aux

# System V-style: full format
ps -ef

# Full format, extra wide (no truncation)
ps auxww
```

## Tree View

### Process Hierarchy

```bash
# ASCII tree
ps auxf

# pstree (dedicated tool)
pstree
pstree -p       # show PIDs
pstree -u       # show user transitions
pstree deploy   # tree for one user
```

## Custom Format

### Select Columns

```bash
# PID, user, CPU, memory, command
ps -eo pid,user,%cpu,%mem,cmd

# Add start time and elapsed time
ps -eo pid,user,%cpu,%mem,etime,cmd

# Include PPID (parent PID)
ps -eo pid,ppid,user,%cpu,%mem,cmd

# Nice value and state
ps -eo pid,ni,stat,%cpu,%mem,cmd

# RSS (resident memory) in KB
ps -eo pid,user,rss,vsz,cmd --sort=-rss
```

## Sorting

### Sort Output

```bash
# Sort by CPU descending
ps aux --sort=-%cpu

# Sort by memory descending
ps aux --sort=-%mem

# Sort by RSS descending (custom format)
ps -eo pid,user,rss,cmd --sort=-rss

# Sort by start time
ps -eo pid,lstart,cmd --sort=lstart
```

## Filtering

### By User

```bash
ps -u deploy
ps -U root -u root

# All processes except root
ps -N -u root
```

### By PID

```bash
ps -p 1234
ps -p 1234,5678,9012
```

### By Command Name

```bash
ps -C nginx
ps -C nginx -o pid,%cpu,%mem,cmd
```

## Threads

### Show Threads

```bash
# Show threads as separate entries
ps -eLf

# Thread count per process
ps -eo pid,nlwp,cmd --sort=-nlwp

# Threads for a specific PID
ps -T -p 1234
```

## pgrep and pkill

### Pattern-Based Process Lookup

```bash
# Find PIDs by name
pgrep nginx

# Full command match
pgrep -f "python manage.py"

# List with process names
pgrep -a nginx

# By user
pgrep -u deploy python

# Newest match only
pgrep -n nginx

# Oldest match only
pgrep -o nginx

# Count matches
pgrep -c nginx
```

## Process States

### State Codes

```bash
# STAT column values:
#   R  — Running
#   S  — Sleeping (interruptible)
#   D  — Uninterruptible sleep (usually I/O)
#   T  — Stopped (signal or debugger)
#   Z  — Zombie
#   <  — High priority
#   N  — Low priority (nice)
#   s  — Session leader
#   l  — Multi-threaded
#   +  — Foreground process group

# Find zombie processes
ps aux | awk '$8 == "Z"'
```

## Tips

- `ps aux` (BSD-style, no dash) and `ps -ef` (SysV-style, with dash) show the same data in different formats -- pick one and be consistent.
- `ps auxww` prevents command truncation which happens when the terminal is narrow.
- `--sort=-%cpu` is descending; `--sort=%cpu` is ascending.
- `pgrep -f` matches the full command line, not just the process name -- essential for Python/Java/Node where the binary name is generic.
- Zombie processes (`Z` state) cannot be killed -- they are already dead. Kill or restart their parent process instead.
- `ps` shows a point-in-time snapshot; use `top`/`htop` for continuous monitoring.

## References

- [man ps(1)](https://man7.org/linux/man-pages/man1/ps.1.html)
- [man proc(5) — /proc filesystem](https://man7.org/linux/man-pages/man5/proc.5.html)
- [man pgrep(1)](https://man7.org/linux/man-pages/man1/pgrep.1.html)
- [man top(1)](https://man7.org/linux/man-pages/man1/top.1.html)
- [Arch Wiki — Process Management](https://wiki.archlinux.org/title/Process)
- [Kernel /proc/PID Documentation](https://www.kernel.org/doc/html/latest/filesystems/proc.html)
- [Red Hat — Monitoring Processes](https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux/9/html/monitoring_and_managing_system_status_and_performance/monitoring-processes_monitoring-and-managing-system-status-and-performance)
- [Ubuntu Manpage — ps](https://manpages.ubuntu.com/manpages/noble/man1/ps.1.html)
