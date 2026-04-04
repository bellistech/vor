# kill (signal processes)

Send signals to processes by PID, name, or pattern.

## kill (by PID)

### Send Signals

```bash
# Default signal is SIGTERM (15) — graceful shutdown
kill 1234

# Explicit SIGTERM
kill -TERM 1234
kill -15 1234

# Force kill (cannot be caught or ignored)
kill -KILL 1234
kill -9 1234

# Reload configuration
kill -HUP 1234
kill -1 1234

# Send to multiple PIDs
kill 1234 5678 9012

# User-defined signal (e.g. log rotation)
kill -USR1 1234
kill -USR2 1234
```

### Signal a Process Group

```bash
# Negative PID sends to the entire process group
kill -TERM -1234

# Kill all processes you own (careful!)
kill -TERM -1
```

## killall (by name)

### Kill by Process Name

```bash
# Kill all processes named nginx
killall nginx

# Force kill
killall -9 nginx

# Send HUP to all sshd processes
killall -HUP sshd

# Interactive (confirm each)
killall -i nginx

# Only kill processes owned by a specific user
killall -u deploy python3

# Only kill processes older than 1 hour
killall -o 1h python3

# Only kill processes newer than 10 minutes
killall -y 10m python3
```

## pkill (by pattern)

### Pattern-Based Signaling

```bash
# Kill by name pattern
pkill nginx

# Match full command line (not just process name)
pkill -f "python manage.py runserver"

# Signal a specific user's processes
pkill -u deploy python

# Send HUP
pkill -HUP -f "gunicorn"

# Exact match on process name
pkill -x nginx

# Kill newest matching process
pkill -n -f "worker"

# Kill oldest matching process
pkill -o -f "worker"
```

## Common Signals

### Signal Reference

```bash
# List all signals
kill -l

# Key signals:
#   1  SIGHUP    — Hangup; reload config (nginx, sshd, Apache)
#   2  SIGINT    — Interrupt; same as Ctrl+C
#   3  SIGQUIT   — Quit with core dump
#   9  SIGKILL   — Force kill; cannot be caught
#   15 SIGTERM   — Graceful termination (default)
#   10 SIGUSR1   — User-defined (log reopen, status dump)
#   12 SIGUSR2   — User-defined
#   18 SIGCONT   — Resume stopped process
#   19 SIGSTOP   — Pause process; cannot be caught
#   20 SIGTSTP   — Suspend; same as Ctrl+Z
```

## Practical Patterns

### Common Workflows

```bash
# Graceful nginx reload
kill -HUP $(cat /var/run/nginx.pid)

# Force-kill a stuck process
kill -9 $(pgrep -f "stuck_script")

# Kill all background jobs in current shell
kill $(jobs -p)

# Gracefully stop, wait, then force kill
kill 1234 && sleep 5 && kill -9 1234

# Kill everything running as a user (root only)
pkill -u baduser

# Reopen log files (common for daemons)
kill -USR1 $(cat /var/run/myapp.pid)
```

## Tips

- Always try SIGTERM (15) before SIGKILL (9) -- SIGTERM lets the process clean up, close files, and release locks.
- SIGKILL (9) cannot be caught or blocked -- the kernel terminates the process immediately. Use it only as a last resort.
- `killall` matches exact process names; `pkill` matches patterns. Use `pkill -f` to match full command lines (essential for Java, Python, Node).
- `killall` on Solaris kills ALL processes -- it does not filter by name. Use `pkill` on Solaris.
- Zombie processes (`Z` state) cannot be killed because they are already dead. Kill their parent to reap them.
- SIGHUP to a shell session kills all child processes. For daemons, SIGHUP conventionally means "reload config."
- `kill -0 PID` sends no signal but checks if the process exists (exit code 0 = alive).

## See Also

- ps, htop, nice, lsof, strace

## References

- [man kill(1)](https://man7.org/linux/man-pages/man1/kill.1.html)
- [man kill(2) — System Call](https://man7.org/linux/man-pages/man2/kill.2.html)
- [man signal(7) — Signal List](https://man7.org/linux/man-pages/man7/signal.7.html)
- [man signal(2)](https://man7.org/linux/man-pages/man2/signal.2.html)
- [man killall(1)](https://man7.org/linux/man-pages/man1/killall.1.html)
- [man pkill(1)](https://man7.org/linux/man-pages/man1/pkill.1.html)
- [Kernel Signal Documentation](https://www.kernel.org/doc/html/latest/process/signal.html)
- [Arch Wiki — Process Management](https://wiki.archlinux.org/title/Process)
- [Ubuntu Manpage — kill](https://manpages.ubuntu.com/manpages/noble/man1/kill.1.html)
