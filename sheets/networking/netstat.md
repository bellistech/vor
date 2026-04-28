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

## netstat → ss Translation Cheatsheet

`netstat` is deprecated on modern Linux distros (Debian 13, RHEL 9+, Ubuntu 24.04+ no longer install it by default). The `ss` (socket statistics) tool from `iproute2` is the supported replacement and is **3-10x faster** because it reads `/proc/net/*` directly via netlink instead of parsing text files.

| netstat | ss equivalent | What it does |
|---------|--------------|--------------|
| `netstat -tlnp` | `ss -tlnp` | TCP listening, numeric, with PIDs |
| `netstat -aunp` | `ss -aunp` | UDP all, numeric, with PIDs |
| `netstat -an` | `ss -an` | all sockets, numeric |
| `netstat -t` | `ss -t` | established TCP only |
| `netstat -ta` | `ss -ta` | all TCP (listening + established) |
| `netstat -lx` | `ss -lx` | Unix socket listening |
| `netstat -anp \| grep :443` | `ss -anp '( sport = :443 or dport = :443 )'` | filter by port |
| `netstat -ant \| grep ESTABLISHED` | `ss -tn state established` | filter by state |
| `netstat -ant \| awk '{print $6}' \| sort -u` | `ss -tan -o state` | enumerate states |
| `netstat -i` | `ip -s link` | interface counters |
| `netstat -r` | `ip route` | routing table |
| `netstat -g` | `ip maddress show` | multicast groups |
| `netstat -s` | `nstat -a` | aggregated /proc/net/snmp counters |
| `netstat -c 1` | `watch -n 1 ss ...` | continuous refresh |

`ss` adds capabilities `netstat` lacks:

```bash
# Filter by socket state
ss -tan state established
ss -tan state syn-sent
ss -tan state time-wait
ss -tan state close-wait

# Show socket memory usage
ss -m

# Show internal TCP info (cwnd, rtt, ss-thresh, retrans)
ss -i

# Process info without root for your own processes
ss -lpnt

# UNIX socket peers
ss -px

# Time information (when socket was created)
ss -tan -o

# Quic / SCTP / DCCP (where the kernel supports them)
ss --vsock
ss --sctp
```

## State Reference

```
ESTABLISHED   active connection, both sides exchanging data
SYN_SENT      we sent SYN, waiting for SYN+ACK
SYN_RECV      received SYN, sent SYN+ACK, waiting for ACK
FIN_WAIT1     we initiated close, waiting for ACK or FIN
FIN_WAIT2     our FIN was ACKed, waiting for peer's FIN
CLOSE_WAIT    peer initiated close, our app hasn't called close() yet
LAST_ACK      we sent FIN after our app called close() in CLOSE_WAIT
TIME_WAIT     after both sides sent FIN; held for 2*MSL (typically 60-120s)
CLOSING       both sides sent FIN simultaneously
CLOSED        socket is gone (rarely seen — usually means it's already deallocated)
LISTEN        accept()-ing connections
```

**Common state pathologies:**
- Many `CLOSE_WAIT` → application has bugs leaking sockets (forgot to close())
- Many `TIME_WAIT` → high churn outbound; usually fine, kernel reaps them
- Many `SYN_RECV` → SYN flood or backlog overflow
- Many `LAST_ACK` → peer disappeared mid-close; will time out

## TCP Stats Deep Dive (-s and nstat)

```bash
# netstat -st key counters worth watching:
netstat -st | grep -E "(retransmit|fail|reset|overflow|drop)"

# segments retransmitted     — ratio to "segments sent" should be < 0.5%
# bad segments received      — checksum failures (NIC offload bugs, cabling)
# resets sent                — counts RST emissions
# TCPListenOverflows         — full accept queue, connection dropped
# TCPListenDrops             — total accept-queue drops
# TCPSynRetrans              — initial SYN retried (peer slow or bad ECN)
# PruneCalled                — receive buffer collapsed under memory pressure
# RcvPruned                  — packets dropped due to receive buffer pruning
# OutOfWindowIcmps           — usually fine; counts spec-violation ICMP

# nstat for the modern, machine-readable equivalent:
nstat -a              # all counters
nstat -az             # include zero-valued counters
nstat -t 1            # one-second delta (the per-interval view)
nstat | column -t     # aligned columns

# Reset baseline (so subsequent calls are deltas):
nstat -r
```

## Top Connections by Remote IP

```bash
# Existing snippet works but here are deeper variants:

# Top 20 remote IPs by total connection count
ss -ant | awk 'NR>1 {print $5}' | cut -d: -f1 | sort | uniq -c | sort -rn | head -20

# Connections by state, grouped per remote
ss -ant | awk 'NR>1 {print $1, $5}' | cut -d: -f1 | sort | uniq -c | sort -rn

# Detect a single client hammering you (DDoS heuristic)
ss -ant state established | awk 'NR>1 {print $5}' | cut -d: -f1 \
  | sort | uniq -c | sort -rn | head -5
```

## Worked Diagnostic Recipes

### Recipe 1 — "port already in use" but I just stopped the service

```bash
# nginx died but `nginx -s reload` says address in use
ss -lntp | grep :443
# State    Local Address:Port      Process
# LISTEN   0.0.0.0:443             users:(("old-nginx",pid=12345,fd=8))

# kill the orphan
sudo kill -TERM 12345

# OR: it's in TIME_WAIT — wait, or set SO_REUSEADDR in your service config
ss -tan state time-wait | grep :443 | wc -l
```

### Recipe 2 — application hangs, suspect kernel-level packet drop

```bash
# Per-interface packet errors
ip -s link show eth0
# RX: bytes  packets  errors  dropped  overrun  mcast
#     5.2T   18M      0       12345    0        92K
#                              ^^^^^^^^ drops in /proc/net/dev

# Drill in:
ethtool -S eth0 | grep -iE "(drop|err|disc)"
# rx_no_buffer_count: 0
# rx_missed_errors: 0
# rx_drops: 12345                   # bingo

# Kernel softnet stats (per-CPU)
column -t /proc/net/softnet_stat
# total      dropped    squeezed   ...
# 8a3f9c11   00000000   00012345   ...   ← squeezed = budget exhausted
```

### Recipe 3 — TIME_WAIT explosion on outbound API client

```bash
# Symptom: client app suddenly throws "address in use" connecting outbound
ss -tn state time-wait | wc -l
# 28000

cat /proc/sys/net/ipv4/ip_local_port_range
# 32768   60999    → 28000+ ephemeral ports stuck in TIME_WAIT

# Mitigation 1: enable TIME_WAIT reuse for outbound (safe)
sudo sysctl -w net.ipv4.tcp_tw_reuse=1

# Mitigation 2: shorten FIN timeout (less safe; default is 60s)
sudo sysctl -w net.ipv4.tcp_fin_timeout=30

# Mitigation 3: HTTP keepalive in the app — fewer connections opened/closed
```

### Recipe 4 — listen-backlog overflow

```bash
# Symptom: connections refused or hung at SYN_RECV
netstat -s | grep -E "(LISTEN|listen|overflowed|dropped)"
#     12345 SYNs to LISTEN sockets dropped
#     12345 times the listen queue of a socket overflowed

# Show current accept queue depths
ss -lnt
# State  Recv-Q  Send-Q  Local Address:Port
# LISTEN 50      128     0.0.0.0:443
#        ^                ^
#        connections waiting accept()  | configured backlog

# Fix: raise SOMAXCONN (kernel ceiling)
sudo sysctl -w net.core.somaxconn=4096

# AND raise the per-socket backlog in the app:
#   nginx:    listen 443 backlog=4096;
#   Go:       net.Listen with explicit ListenConfig + Control()
#   Python:   socket.listen(4096)
```

## /proc Source of Truth

Both `netstat` and `ss` derive their data from these files. Understanding them helps when neither tool is installed (busybox / minimal images):

```bash
cat /proc/net/tcp                # TCP IPv4 sockets (one line per)
cat /proc/net/tcp6               # TCP IPv6
cat /proc/net/udp                # UDP IPv4
cat /proc/net/unix               # Unix domain sockets
cat /proc/net/netstat            # ip+tcp protocol counters
cat /proc/net/snmp               # SNMP-style protocol counters
cat /proc/net/dev                # interface RX/TX (what `ip -s link` reads)
cat /proc/net/route              # routing table (for kernel)
cat /proc/net/igmp               # multicast memberships (IPv4)
cat /proc/net/sockstat           # global socket counts (TCP, UDP, RAW, FRAG)

# Decoding /proc/net/tcp local/remote address fields:
# they're hex-encoded, little-endian per byte:
#   sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode
#   0:  0100007F:1F90 00000000:0000 0A 00000000:00000000 ...
#
# 0100007F → 7F.00.00.01 → 127.0.0.1
# 1F90    → 0x1F90       → 8080
# st      → 0A           → 10 = LISTEN  (state codes in include/net/tcp_states.h)
```

## macOS / BSD Differences

### macOS-specific flags
```bash
netstat -p tcp          # TCP connections (macOS uses -p for protocol)
netstat -p udp          # UDP connections
netstat -nr             # routing table (same as Linux)
netstat -an -f inet     # IPv4 only
netstat -an -f inet6    # IPv6 only

# macOS does NOT have ss; use netstat or:
lsof -i -P -n           # closest BSD-side equivalent — sockets per process
sudo lsof -iTCP -sTCP:LISTEN -P -n   # TCP listening
sudo lsof -iUDP -P -n                # UDP

# macOS netstat statistics:
netstat -s -p tcp       # TCP-only stats
netstat -m              # mbuf cluster usage (BSD-specific)
```

## Common Errors and Fixes

```bash
# "command not found: netstat"
# Cause: deprecated, no longer pulled in by default.
sudo apt install net-tools            # Debian/Ubuntu
sudo dnf install net-tools            # RHEL/Fedora
# Better: just learn ss (already installed in iproute2).

# "PID/Program name" column shows '-'
# Cause: not running as root, can't read other users' /proc/<pid>/fd.
# Fix: sudo netstat -tlnp

# Output stops mid-list / huge wait
# Cause: DNS resolution on every IP. ss/netstat needs -n almost always.
# Fix: use -n (numeric).

# Numbers in netstat -s look frozen
# Cause: counters are since boot. Use nstat -t 1 for interval deltas.

# ss -tlnp shows nothing for a service you know is listening
# Cause: the service is in a different network namespace (containers, VPN).
sudo ip netns list
sudo ip netns exec <ns-name> ss -tlnp
```

## Tips

- `netstat` is deprecated on Linux — prefer `ss` for speed and features.
- `-n` is almost always what you want; DNS lookups are slow on busy systems.
- `-p` requires root to see PIDs for processes owned by other users.
- On macOS/BSD, `-p` means PROTOCOL (tcp/udp), not process; there is no process flag.
- `netstat -s` is still useful even on modern systems — `ss -s` shows less detail; cross-reference with `nstat -a`.
- `netstat -i` output is cumulative since boot; use `sar -n DEV` or `nstat` for interval stats.
- Watch for high `RX-DRP` or `RX-OVR` in `netstat -i` (or `dropped`/`overrun` in `ip -s link`) — indicates kernel is dropping packets.
- Many TIME_WAITs are usually fine; many CLOSE_WAITs almost always indicate an app bug (forgot close()).
- For RFC-correct port scans, prefer `nmap`; netstat tells you what your OWN box is doing, not what's reachable from elsewhere.
- The `lsof -i` family is the cross-platform alternative — works on macOS, Linux, and BSD with the same semantics.

## See Also

- networking/ss, networking/tcp, networking/udp, networking/ip, networking/ethtool

## References

- [man netstat](https://man7.org/linux/man-pages/man8/netstat.8.html)
- [man ss — socket statistics (modern replacement)](https://man7.org/linux/man-pages/man8/ss.8.html)
- [man proc — /proc/net/* entries used by netstat](https://man7.org/linux/man-pages/man5/proc.5.html)
- [net-tools Source Repository](https://sourceforge.net/projects/net-tools/)
- [iproute2 — Linux Foundation Wiki](https://wiki.linuxfoundation.org/networking/iproute2)
- [Linux Kernel — Networking Statistics](https://www.kernel.org/doc/html/latest/networking/statistics.html)
- [Red Hat — Monitoring Network Traffic](https://docs.redhat.com/en/documentation/red_hat_enterprise_linux/9/html/configuring_and_managing_networking/index)
