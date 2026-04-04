# Traceroute (Network Path Discovery)

Discovers the layer-3 hop-by-hop path to a destination by sending probes with incrementing TTL values, eliciting ICMP Time Exceeded messages from each router, revealing latency, routing, and path asymmetry.

## How Traceroute Works

```
Host (TTL=1) ──→ Router 1 → drops, sends ICMP Time Exceeded
Host (TTL=2) ──→ Router 1 → Router 2 → drops, sends ICMP Time Exceeded
Host (TTL=3) ──→ Router 1 → Router 2 → Router 3 → drops, sends ICMP Time Exceeded
...
Host (TTL=N) ──→ Router 1 → ... → Destination → sends ICMP Echo Reply (or Port Unreachable)

# Each hop sends 3 probes by default
# TTL decrements at each router; when it hits 0, router drops and sends ICMP Type 11
```

## Probe Modes

### ICMP Mode (Default on Windows)

```bash
# Uses ICMP Echo Request packets
traceroute -I 8.8.8.8                  # Linux
tracert 8.8.8.8                        # Windows (ICMP by default)

# Pros: Simple, widely understood
# Cons: Often filtered by firewalls
```

### UDP Mode (Default on Linux)

```bash
# Uses UDP packets to high ports (33434+)
traceroute 8.8.8.8                     # default mode
traceroute -U 8.8.8.8                  # explicit UDP

# Destination sends ICMP Port Unreachable when packet arrives
# (because nothing listens on port 33434+)

# Pros: Default on Unix, bypasses some ICMP filters
# Cons: UDP to high ports sometimes filtered too
```

### TCP Mode

```bash
# Uses TCP SYN packets (often to port 80 or 443)
traceroute -T -p 80 8.8.8.8           # TCP SYN to port 80
traceroute -T -p 443 8.8.8.8          # TCP SYN to port 443
tcptraceroute 8.8.8.8 443             # dedicated tool

# Pros: Gets through most firewalls (port 80/443 usually allowed)
# Cons: Destination responds with SYN-ACK or RST, not ICMP
```

## traceroute Flags

```bash
# Common options
traceroute -n 8.8.8.8                  # numeric only (no DNS reverse lookups)
traceroute -q 1 8.8.8.8               # 1 probe per hop (default 3)
traceroute -w 3 8.8.8.8               # 3-second wait for response (default 5)
traceroute -m 30 8.8.8.8              # max 30 hops (default 30)
traceroute -f 5 8.8.8.8               # start at TTL 5 (skip first 4 hops)
traceroute -s 192.168.1.10 8.8.8.8    # source address
traceroute -i eth0 8.8.8.8            # source interface
traceroute -A 8.8.8.8                 # show AS numbers (if supported)

# Set packet size
traceroute -q 1 8.8.8.8 1400          # 1400-byte packets

# Set source port (for firewall bypass)
traceroute -T -p 443 --sport=443 8.8.8.8
```

## Interpreting Output

```
 1  192.168.1.1 (192.168.1.1)  1.234 ms  1.125 ms  1.089 ms
 2  10.0.0.1 (10.0.0.1)  5.678 ms  5.234 ms  5.567 ms
 3  * * *
 4  72.14.215.85 (72.14.215.85)  15.234 ms  14.987 ms  15.123 ms
 5  8.8.8.8 (8.8.8.8)  20.123 ms  19.876 ms  20.234 ms

Symbols:
  *         — No response (filtered, rate-limited, or timeout)
  !N        — ICMP Network Unreachable
  !H        — ICMP Host Unreachable
  !X        — ICMP Communication Administratively Prohibited
  !P        — ICMP Protocol Unreachable
  !F-<mtu>  — ICMP Fragmentation Needed (shows next-hop MTU)
  !S        — ICMP Source Route Failed
```

### Reading Latency

```
# Per-hop latency is NOT the link latency
# It is the RTT from source to that hop
# Link latency between hops = hop_n_RTT - hop_(n-1)_RTT (approximately)

# Example:
#  Hop 2: 5 ms
#  Hop 3: 15 ms
#  Link 2→3: ~10 ms

# CAUTION: Router ICMP generation is low-priority
# A hop showing high latency may just be slow at generating ICMP TTL exceeded
# while forwarding data traffic at full speed (control plane vs data plane)
```

## Paris Traceroute

```bash
# Classic traceroute varies source port per probe
# This causes ECMP (Equal-Cost Multi-Path) routers to send probes down different paths
# Result: diamond-shaped paths, confusing output

# Paris traceroute keeps flow identifier constant
# All probes follow the same path through ECMP routers

paris-traceroute 8.8.8.8
paris-traceroute -n 8.8.8.8           # numeric
paris-traceroute -a mda 8.8.8.8       # Multi-path Discovery Algorithm
                                       # discovers ALL paths through ECMP

# mtr uses Paris traceroute internally
mtr --report 8.8.8.8
```

## Asymmetric Paths

```
# The path FROM you TO destination may differ from the RETURN path
# traceroute only shows the FORWARD path (routers that decrement TTL)
# ICMP replies may take a completely different route back

# Detecting asymmetry:
# 1. Run traceroute from both ends
# 2. Compare paths — they often differ
# 3. Latency differences between consecutive hops can indicate
#    asymmetric routing (return path goes through different links)

# Reverse traceroute (from remote to you)
# Requires access to the remote end, or use looking glass servers
# Many ISPs provide looking glass: https://www.traceroute.org/
```

## Advanced: mtr (Combined ping + traceroute)

```bash
# mtr continuously sends probes and builds statistics per hop
mtr 8.8.8.8                           # interactive mode
mtr -r -c 100 8.8.8.8                # report mode, 100 cycles
mtr -rw -c 100 8.8.8.8               # wide report (full hostnames)
mtr -T -P 443 8.8.8.8                # TCP mode, port 443
mtr -u 8.8.8.8                        # UDP mode
mtr -n 8.8.8.8                        # numeric (no DNS)
mtr --json 8.8.8.8                    # JSON output

# mtr output columns:
# Loss%   — packet loss at this hop
# Snt     — packets sent
# Last    — last RTT
# Avg     — average RTT
# Best    — minimum RTT
# Wrst    — maximum RTT
# StDev   — standard deviation
```

## Troubleshooting Patterns

### Pattern: Stars at One Hop

```
 3  10.0.0.1  5 ms  5 ms  5 ms
 4  * * *
 5  72.14.215.85  15 ms  15 ms  15 ms

# Router at hop 4 is not responding to probes
# But traffic passes through — subsequent hops respond
# This is normal: many routers filter/rate-limit ICMP
# NOT a problem unless everything after is also * * *
```

### Pattern: Latency Spike at One Hop

```
 3  10.0.0.1     5 ms    5 ms    5 ms
 4  172.16.0.1   150 ms  148 ms  152 ms
 5  8.8.8.8      20 ms   19 ms   20 ms

# Hop 4 shows 150 ms but hop 5 shows 20 ms
# The router at hop 4 is slow at GENERATING ICMP (control plane busy)
# It forwards actual data traffic fine — hop 5 proves this
# Only worry if ALL subsequent hops show elevated latency
```

### Pattern: Increasing Loss at a Hop

```
 Host      Loss%  Snt  Avg
 1. gw      0.0%  100  1.0
 2. isp     0.0%  100  5.0
 3. peer    5.0%  100  15.0
 4. dest    5.0%  100  20.0

# Loss at hop 3 AND all subsequent hops = real loss at hop 3
# Loss ONLY at hop 3 = ICMP rate limiting, not real loss
```

## Tips

- Stars (`* * *`) at intermediate hops are usually ICMP rate limiting, not a problem. Only worry if all subsequent hops are also stars, which indicates a real block in the path.
- Per-hop latency in traceroute is the RTT from source to that router, not the latency of the link. A spike at one hop that disappears at the next is almost always slow ICMP generation on the router, not a congested link.
- Use TCP mode (`traceroute -T -p 443`) when ICMP and UDP are blocked. Port 443 is rarely filtered because it would break HTTPS. The destination responds with RST or SYN-ACK instead of ICMP.
- Paris traceroute is essential for accurate path discovery on networks with ECMP. Classic traceroute varies the flow hash per probe, causing probes to follow different ECMP paths and producing a misleading diamond-shaped topology.
- mtr is strictly superior to traceroute for diagnosing intermittent issues because it continuously samples, showing loss percentage and latency statistics per hop. Always use mtr with at least 100 cycles (`-c 100`) for meaningful statistics.
- Traceroute latency includes ICMP generation delay on the router. Core routers with huge FIBs and busy control planes often show inflated ICMP response times. Look at the trend across subsequent hops, not individual hop values.
- When two traceroutes from different sources converge on the same destination through different paths but show similar latency from the convergence point, the issue is likely on the shared segment.
- The `-f` flag (first TTL) lets you skip known hops and focus on a specific segment of the path. Useful when debugging issues between your ISP and a remote provider.
- Traceroute to the same destination from multiple vantage points (your host, your ISP's looking glass, RIPE Atlas probes) gives the clearest picture of where an issue lies. Single-source traceroute is inherently incomplete.
- MPLS label-switched paths often show up as asterisks because MPLS routers may not decrement TTL on labeled packets. Some show as `MPLS Label X TTL=Y` if the router sends ICMP extensions (RFC 4950).

## See Also

- icmp, mtr, ip, tcp, udp, tcpdump

## References

- [RFC 1393 — Traceroute Using an IP Option](https://www.rfc-editor.org/rfc/rfc1393)
- [RFC 4950 — ICMP Extensions for Traceroute (MPLS)](https://www.rfc-editor.org/rfc/rfc4950)
- [Paris Traceroute](https://paris-traceroute.net/)
- [man traceroute](https://linux.die.net/man/8/traceroute)
- [man mtr](https://linux.die.net/man/8/mtr)
- [RIPE Atlas — Network Measurement Platform](https://atlas.ripe.net/)
