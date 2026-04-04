# netstat (Network Statistics)

Legacy tool for displaying network connections, routing tables, interface stats, and multicast memberships.

## Connections

### Show active connections
```bash
netstat -t              # TCP connections
netstat -u              # UDP connections
netstat -a              # all (listening + established)
netstat -an             # all, numeric (no DNS lookups)
netstat -atn            # all TCP, numeric
netstat -aunp           # all UDP, numeric, with PIDs
```

### Show listening sockets
```bash
netstat -l              # all listening
netstat -lt             # TCP listening
netstat -lu             # UDP listening
netstat -lx             # Unix socket listening
netstat -tlnp           # TCP listening, numeric, with process info
```

### Filter by process
```bash
netstat -tlnp | grep nginx
netstat -anp | grep :3306       # find MySQL connections
netstat -anp | grep python      # find Python sockets
```

## Routing

### Show routing table
```bash
netstat -r              # display kernel routing table
netstat -rn             # numeric (skip DNS resolution)
netstat -rn -4          # IPv4 routes only
netstat -rn -6          # IPv6 routes only
```

## Statistics

### Protocol statistics
```bash
netstat -s              # summary stats for all protocols
netstat -st             # TCP statistics only
netstat -su             # UDP statistics only
netstat -sw             # RAW socket statistics
```

### Key TCP stats to watch
```bash
netstat -st | grep -i retrans    # retransmissions (packet loss indicator)
netstat -st | grep -i reset      # connection resets
netstat -st | grep -i overflow   # listen queue overflows
```

## Interface Statistics

### Show interface stats
```bash
netstat -i              # interface table (RX/TX packets, errors)
netstat -ie             # extended — same as ifconfig
```

### Continuous monitoring
```bash
netstat -c              # refresh output every second
netstat -ct             # continuous TCP connection listing
```

## Multicast and Groups

### Show multicast group memberships
```bash
netstat -g              # multicast group info
```

## Connection Counting

### Count connections by state
```bash
netstat -ant | awk '{print $6}' | sort | uniq -c | sort -rn
```

### Count connections per remote IP
```bash
netstat -ant | awk '{print $5}' | cut -d: -f1 | sort | uniq -c | sort -rn | head -20
```

### Count TIME_WAIT sockets
```bash
netstat -ant | grep TIME_WAIT | wc -l
```

### Find port usage
```bash
netstat -tlnp | awk '{print $4}' | rev | cut -d: -f1 | rev | sort -n | uniq
```

## macOS / BSD Differences

### macOS-specific flags
```bash
netstat -p tcp          # TCP connections (macOS uses -p for protocol)
netstat -p udp          # UDP connections
netstat -nr             # routing table (same as Linux)
netstat -an -f inet     # IPv4 only
netstat -an -f inet6    # IPv6 only
```

## Tips

- `netstat` is deprecated on Linux — prefer `ss` for speed and features
- `-n` is almost always what you want; DNS lookups are slow on busy systems
- `-p` requires root to see PIDs for processes owned by other users
- On macOS/BSD, `-p` means protocol (tcp/udp), not process; there is no process flag
- `netstat -s` is still useful even on modern systems — `ss -s` shows less detail
- `netstat -i` output is cumulative since boot; use `sar` or `nstat` for interval stats
- Watch for high `RX-DRP` or `RX-OVR` in `netstat -i` — indicates kernel is dropping packets

## See Also

- ss, tcp, udp, ip, nmap

## References

- [man netstat](https://man7.org/linux/man-pages/man8/netstat.8.html)
- [man ss — socket statistics (modern replacement)](https://man7.org/linux/man-pages/man8/ss.8.html)
- [man proc — /proc/net/* entries used by netstat](https://man7.org/linux/man-pages/man5/proc.5.html)
- [net-tools Source Repository](https://sourceforge.net/projects/net-tools/)
- [iproute2 — Linux Foundation Wiki](https://wiki.linuxfoundation.org/networking/iproute2)
- [Linux Kernel — Networking Statistics](https://www.kernel.org/doc/html/latest/networking/statistics.html)
- [Red Hat — Monitoring Network Traffic](https://docs.redhat.com/en/documentation/red_hat_enterprise_linux/9/html/configuring_and_managing_networking/index)
