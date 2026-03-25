# ss (Socket Statistics)

Modern replacement for `netstat` — dumps socket statistics from the kernel via netlink.

## Listing Sockets

### Show all connections
```bash
ss -a                  # all sockets (listening + non-listening)
ss -t -a               # all TCP sockets
ss -u -a               # all UDP sockets
ss -x -a               # all Unix domain sockets
ss -w -a               # all RAW sockets
```

### Show listening sockets
```bash
ss -l                  # all listening sockets
ss -lt                 # TCP only
ss -lu                 # UDP only
ss -lx                 # Unix domain only
```

### Show process info
```bash
ss -tlnp               # listening TCP with process names and PIDs
ss -tulnp              # listening TCP + UDP with process info
ss -tlnp | grep :8080  # find what's on port 8080
```

## Filtering

### By port
```bash
ss -t sport = :443              # source port 443
ss -t dport = :443              # destination port 443
ss -t 'sport = :http'           # by service name
ss -t '( sport = :80 or sport = :443 )'  # multiple ports
```

### By address
```bash
ss -t dst 10.0.0.1              # destination address
ss -t src 192.168.1.0/24        # source subnet
ss -t dst 10.0.0.1:443          # address + port
```

### By state
```bash
ss -t state established         # established connections
ss -t state time-wait           # TIME_WAIT sockets
ss -t state close-wait          # CLOSE_WAIT sockets
ss -t state listening           # same as -l
ss -t state syn-sent            # outgoing connection attempts
ss -t state fin-wait-1          # closing sockets
```

### Exclude states
```bash
ss -t exclude listening         # non-listening TCP
ss -t exclude time-wait         # skip TIME_WAIT noise
```

## Output Control

### Numeric output
```bash
ss -n                  # don't resolve service names (faster)
ss -r                  # resolve addresses to hostnames
```

### Detailed info
```bash
ss -e                  # extended info (UID, inode, cookie)
ss -i                  # internal TCP info (RTT, cwnd, ssthresh)
ss -m                  # memory usage per socket
ss -o                  # timer info (keepalive, retransmit)
ss -Z                  # SELinux context
```

### Summary
```bash
ss -s                  # socket summary statistics
```

## Common Combos

### Find all connections to a remote host
```bash
ss -tn dst 10.0.0.5
```

### Count connections per state
```bash
ss -t state established | awk '{print $4}' | sort | uniq -c | sort -rn
```

### Find processes with most connections
```bash
ss -tnp state established | awk '{print $NF}' | sort | uniq -c | sort -rn
```

### Watch for new connections
```bash
watch -n 1 'ss -tn state established | wc -l'
```

### IPv4 vs IPv6
```bash
ss -4 -tln              # IPv4 only, listening TCP
ss -6 -tln              # IPv6 only, listening TCP
```

### DCCP and SCTP
```bash
ss -d -a                # DCCP sockets
ss --sctp -a            # SCTP sockets
```

## Tips

- `ss` is faster than `netstat` because it reads directly from kernel netlink — use it on busy servers
- `-n` avoids DNS lookups, making output much faster on systems with many connections
- `-p` requires root to see process info for sockets owned by other users
- State filters use `state` keyword; port/address filters use `sport`/`dport`/`src`/`dst`
- Quote complex filter expressions to prevent shell interpretation
- `ss -i` shows TCP internal state (congestion window, RTT) — invaluable for debugging performance
- On older systems without `ss`, fall back to `netstat`

## References

- [man ss](https://man7.org/linux/man-pages/man8/ss.8.html)
- [man ip](https://man7.org/linux/man-pages/man8/ip.8.html)
- [iproute2 — Linux Foundation Wiki](https://wiki.linuxfoundation.org/networking/iproute2)
- [Linux Kernel — TCP Sysctl Documentation](https://www.kernel.org/doc/html/latest/networking/ip-sysctl.html)
- [man proc — /proc/net socket entries](https://man7.org/linux/man-pages/man5/proc.5.html)
- [man netstat — Legacy Socket Statistics](https://man7.org/linux/man-pages/man8/netstat.8.html)
- [Red Hat — Monitoring Network Sockets](https://docs.redhat.com/en/documentation/red_hat_enterprise_linux/9/html/configuring_and_managing_networking/index)
