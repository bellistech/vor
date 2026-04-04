# lsof (list open files)

List open files, network connections, and what processes are using them.

## Open Files

### All Open Files

```bash
# All open files (very verbose)
lsof

# Open files for a specific process
lsof -p 1234

# Open files by command name
lsof -c nginx

# Open files by multiple commands
lsof -c nginx -c php-fpm
```

## By User

### Filter by User

```bash
lsof -u deploy

# Exclude a user
lsof -u ^root

# Files opened by root
lsof -u root
```

## By File

### Who Has a File Open

```bash
lsof /var/log/syslog

# Who is using a mount point (why umount fails)
lsof +D /mnt/usb

# Recursive directory (can be slow)
lsof +D /var/log/

# Non-recursive (faster)
lsof +d /var/log/
```

## Network Connections

### By Port

```bash
# What process is using port 8080
lsof -i :8080

# TCP connections only
lsof -i TCP

# UDP connections only
lsof -i UDP

# Specific protocol and port
lsof -i TCP:443

# Port range
lsof -i TCP:8000-9000

# Connections to a specific host
lsof -i @192.168.1.100

# Connections to host on specific port
lsof -i TCP@192.168.1.100:22
```

### Connection State

```bash
# Only LISTEN sockets
lsof -i -sTCP:LISTEN

# Only ESTABLISHED connections
lsof -i -sTCP:ESTABLISHED

# All internet connections (IPv4 and IPv6)
lsof -i 4    # IPv4 only
lsof -i 6    # IPv6 only
```

## By PID

### Process Details

```bash
lsof -p 1234

# Multiple PIDs
lsof -p 1234,5678

# Exclude a PID
lsof -p ^1
```

## Combining Filters

### AND vs OR Logic

```bash
# OR logic (default) — files matching user OR command
lsof -u deploy -c nginx

# AND logic — files matching user AND command
lsof -u deploy -c nginx -a

# AND: nginx processes listening on port 80
lsof -c nginx -i :80 -a
```

## Output Control

### Formatting

```bash
# No header
lsof +c0 -i :80    # +c0 shows full command name

# Repeat every 2 seconds
lsof -i :80 -r 2

# Terse output (PIDs only, for scripting)
lsof -t -i :80

# Kill whatever is on port 8080
kill $(lsof -t -i :8080)
```

## Tips

- `lsof -i :PORT` is the fastest way to find what is using a port -- faster than parsing `ss` or `netstat`.
- `-a` for AND logic is critical; without it, filters are ORed and you get too many results.
- `lsof +D` can be very slow on large directories; use `+d` for non-recursive.
- `lsof -t` returns just PIDs which pipes cleanly into `kill` or `xargs`.
- On macOS, `lsof -i` works the same way but some flags like `-sTCP:LISTEN` require newer versions.
- `lsof` needs root to see other users' processes on some systems; use `sudo` when results look incomplete.

## See Also

- ps, strace, find, htop, kill

## References

- [man lsof(8)](https://man7.org/linux/man-pages/man8/lsof.8.html)
- [man fuser(1)](https://man7.org/linux/man-pages/man1/fuser.1.html)
- [man proc(5) — /proc/PID/fd](https://man7.org/linux/man-pages/man5/proc.5.html)
- [lsof FAQ](https://github.com/lsof-org/lsof/blob/master/00FAQ)
- [lsof GitHub Repository](https://github.com/lsof-org/lsof)
- [Arch Wiki — lsof](https://wiki.archlinux.org/title/Lsof)
- [Kernel /proc/PID/fd Documentation](https://www.kernel.org/doc/html/latest/filesystems/proc.html)
- [Ubuntu Manpage — lsof](https://manpages.ubuntu.com/manpages/noble/man8/lsof.8.html)
