# mtr (My Traceroute)

Combines traceroute and ping into a single network diagnostic tool — shows real-time hop-by-hop stats.

## Basic Usage

### Interactive mode
```bash
mtr example.com                       # interactive, continuous updates
mtr 8.8.8.8                          # trace to IP
mtr -4 example.com                   # force IPv4
mtr -6 example.com                   # force IPv6
```

### Report mode (non-interactive)
```bash
mtr -r example.com                    # report mode — run then print summary
mtr -r -c 100 example.com            # 100 cycles then report
mtr -r -c 50 -n example.com          # 50 cycles, no DNS resolution
mtr -rw example.com                  # wide report (full hostnames)
mtr -rwc 100 example.com             # wide report, 100 cycles
```

## Protocol Selection

### TCP and UDP modes
```bash
mtr --tcp example.com                 # TCP SYN instead of ICMP
mtr --tcp -P 443 example.com         # TCP to port 443
mtr --tcp -P 80 example.com          # TCP to port 80
mtr --udp example.com                # UDP instead of ICMP
mtr --udp -P 53 example.com          # UDP to port 53 (DNS)
mtr --sctp example.com               # SCTP mode
```

## Output Formats

### Structured output
```bash
mtr -r --csv example.com             # CSV output
mtr -r --json example.com            # JSON output
mtr -r --xml example.com             # XML output
mtr -r --raw example.com             # raw format (for further processing)
```

### Save to file
```bash
mtr -rwc 100 example.com > mtr-report.txt
mtr -r --json -c 100 example.com > mtr-report.json
```

## Display Options

### Control resolution and display
```bash
mtr -n example.com                    # no DNS resolution (faster)
mtr -b example.com                   # show both hostname and IP
mtr -o 'LSDR NBAW JMXI' example.com # custom field order
```

### Field codes for -o
```bash
# L = Loss%    S = Sent     D = Dropped   R = Received
# N = Newest   B = Best     A = Average   W = Worst
# J = Jitter   M = Mean     X = Best-RT   I = Interarrival
mtr -o 'LSBA W J' example.com
```

## Packet Configuration

### Packet size and interval
```bash
mtr -s 1400 example.com              # packet size (bytes) — test MTU
mtr -i 0.5 example.com               # send interval (0.5 seconds)
mtr -c 200 example.com               # number of pings per hop
mtr -m 30 example.com                # max hops (default 30)
```

### First TTL
```bash
mtr -f 5 example.com                 # start at TTL 5 (skip known hops)
```

## Reading Results

### Key columns
```bash
# Host        — router/hop hostname or IP
# Loss%       — packet loss percentage (most important metric)
# Snt         — packets sent
# Last        — last RTT in ms
# Avg         — average RTT
# Best        — minimum RTT
# Wrst        — maximum RTT
# StDev       — standard deviation (jitter indicator)
```

### What to look for
```bash
# Loss at a single hop but not beyond = ICMP rate limiting (not real loss)
# Loss that persists from one hop to the end = real packet loss at that hop
# Sudden RTT jump = congestion or long physical distance
# High StDev = inconsistent latency (jitter)
```

## Common Scenarios

### Diagnose packet loss
```bash
mtr -rwc 200 -n problematic-host.example.com
```

### Test through firewall (ICMP blocked)
```bash
mtr --tcp -P 443 example.com
```

### Check path MTU
```bash
mtr -s 1472 --no-dns example.com     # 1472 + 28 header = 1500 MTU
```

### Compare paths from different sources
```bash
mtr -r -c 100 example.com            # from current host
ssh gateway mtr -r -c 100 example.com  # from another host
```

## Tips

- Run at least 100 cycles (`-c 100`) for meaningful loss/jitter statistics; 10 (the default) is too few
- Loss at a single intermediate hop that doesn't continue to the destination is almost always ICMP rate limiting, not real loss
- Use `--tcp -P 443` when ICMP is blocked — many firewalls allow TCP but not ICMP
- `-n` (no DNS) makes results appear faster and avoids misleading reverse DNS names
- `-rw` (report + wide) is the best format for sharing with network engineers or ISPs
- High jitter (StDev) matters for VoIP and real-time applications even without packet loss
- `mtr` requires root (or `setuid`) for raw socket access; use `sudo mtr` if needed
- Compare reports from both ends of a connection to identify asymmetric routing issues
