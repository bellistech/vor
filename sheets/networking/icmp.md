# ICMP (Internet Control Message Protocol)

Network-layer diagnostic and error reporting protocol carried inside IP datagrams (protocol number 1) used by ping, traceroute, path MTU discovery, and routers to signal unreachable destinations, redirects, and TTL expiration.

## ICMP Header Format

```
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|     Type      |     Code      |          Checksum             |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                    Type-Specific Data                          |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+

- Type: 8 bits — message category
- Code: 8 bits — sub-type within category
- Checksum: 16 bits — covers entire ICMP message
- Type-Specific: varies (identifier + sequence for echo, MTU for frag needed, etc.)
```

## ICMP Types and Codes

### Common Types

```
Type  Name                      Direction    Used By
──────────────────────────────────────────────────────────────
0     Echo Reply                ← response   ping
3     Destination Unreachable   ← error      routing, firewalls
4     Source Quench (deprecated) ← error      (obsolete, RFC 6633)
5     Redirect                  ← advisory   routers
8     Echo Request              → probe      ping
9     Router Advertisement      ← info       IRDP
10    Router Solicitation        → request    IRDP
11    Time Exceeded             ← error      traceroute
12    Parameter Problem         ← error      malformed headers
13    Timestamp Request          → probe      time sync (rare)
14    Timestamp Reply           ← response   time sync (rare)
```

### Type 3 — Destination Unreachable Codes

```
Code  Meaning                          Common Cause
────────────────────────────────────────────────────────────
0     Network Unreachable              No route to destination network
1     Host Unreachable                 ARP failed, host down
2     Protocol Unreachable             No handler for IP protocol
3     Port Unreachable                 No process listening (UDP)
4     Fragmentation Needed + DF set    MTU too small, PMTUD
5     Source Route Failed              Strict source routing failed
6     Destination Network Unknown      (obsolete)
9     Network Administratively Prohibited   Firewall/ACL block
10    Host Administratively Prohibited      Firewall/ACL block
13    Communication Administratively Prohibited  Firewall
```

### Type 11 — Time Exceeded Codes

```
Code  Meaning
──────────────────────────
0     TTL exceeded in transit (used by traceroute)
1     Fragment reassembly time exceeded
```

### Type 5 — Redirect Codes

```
Code  Meaning
──────────────────────────
0     Redirect for the Network
1     Redirect for the Host
2     Redirect for TOS and Network
3     Redirect for TOS and Host
```

## Echo Request / Reply (Ping)

```bash
# Basic ping
ping 8.8.8.8
ping -c 4 8.8.8.8                     # send 4 packets only
ping -i 0.2 8.8.8.8                   # 200ms interval (needs root for < 0.2)
ping -s 1472 8.8.8.8                  # specific payload size (1472 + 28 = 1500 MTU)
ping -W 2 8.8.8.8                     # 2-second timeout per packet
ping -t 64 8.8.8.8                    # set TTL

# Flood ping (needs root)
ping -f 8.8.8.8                       # send as fast as possible
ping -f -c 10000 8.8.8.8             # flood 10K packets

# IPv6 ping
ping6 2001:4860:4860::8888
ping -6 2001:4860:4860::8888          # some systems

# Ping with timestamp
ping -D 8.8.8.8                       # prefix each line with UNIX timestamp

# Ping specific interface
ping -I eth0 8.8.8.8
ping -I 192.168.1.10 8.8.8.8         # source IP
```

## Path MTU Discovery (PMTUD)

```bash
# PMTUD works by setting DF (Don't Fragment) bit and sending increasingly
# large packets. Routers that can't forward return ICMP Type 3, Code 4
# with the next-hop MTU in the ICMP header.

# Test PMTUD manually
ping -M do -s 1472 192.168.1.1       # DF set, 1472 payload = 1500 total
# If response: OK, MTU >= 1500
# If "Frag needed": MTU < 1500, reduce size

# Binary search for path MTU
ping -M do -s 1400 192.168.1.1       # try 1428 total
ping -M do -s 1450 192.168.1.1       # try 1478 total
# ... narrow down

# tracepath finds PMTU automatically
tracepath 8.8.8.8

# Check interface MTU
ip link show eth0 | grep mtu

# Common MTU values
# 1500  — Ethernet default
# 1492  — PPPoE (1500 - 8 byte PPPoE header)
# 1480  — IPv6-in-IPv4 tunnel (1500 - 20 byte outer IP)
# 1400  — common VPN/tunnel conservative value
# 9000  — jumbo frames (datacenter)
```

## ICMP Redirect

```bash
# Router sends Type 5 when a better next-hop exists on the same network
#
# Host A ─── Router 1 ──── Router 2 ──── Destination
#     └──────── same LAN ────────┘
#
# Host A sends to Router 1, Router 1 knows Router 2 is better
# Router 1 sends ICMP Redirect to Host A: "use Router 2 for this destination"

# Accept ICMP redirects (default varies)
sysctl net.ipv4.conf.all.accept_redirects          # check
sysctl -w net.ipv4.conf.all.accept_redirects=0     # disable (security)

# Send ICMP redirects (routers)
sysctl net.ipv4.conf.all.send_redirects
sysctl -w net.ipv4.conf.all.send_redirects=0       # disable

# Secure redirects only (from default gateway)
sysctl -w net.ipv4.conf.all.secure_redirects=1
```

## Firewall Rules for ICMP

```bash
# iptables — allow essential ICMP
iptables -A INPUT -p icmp --icmp-type echo-request -j ACCEPT
iptables -A INPUT -p icmp --icmp-type echo-reply -j ACCEPT
iptables -A INPUT -p icmp --icmp-type destination-unreachable -j ACCEPT
iptables -A INPUT -p icmp --icmp-type time-exceeded -j ACCEPT
iptables -A INPUT -p icmp --icmp-type parameter-problem -j ACCEPT

# Rate limit ping to prevent ping flood
iptables -A INPUT -p icmp --icmp-type echo-request \
    -m limit --limit 10/s --limit-burst 20 -j ACCEPT
iptables -A INPUT -p icmp --icmp-type echo-request -j DROP

# nftables — allow essential ICMP
nft add rule inet filter input icmp type { echo-request, echo-reply, \
    destination-unreachable, time-exceeded } accept

# NEVER block ICMP Type 3 Code 4 (Fragmentation Needed)
# Blocking it breaks PMTUD and causes mysterious connection hangs
# This is the #1 ICMP firewall mistake
```

## ICMPv6

```bash
# ICMPv6 is CRITICAL for IPv6 — much more important than ICMP for IPv4
# IPv6 uses ICMPv6 for: NDP, PMTUD, SLAAC, MLD

# Key ICMPv6 types
# 1    Destination Unreachable
# 2    Packet Too Big (PMTUD — MUST NOT be filtered)
# 3    Time Exceeded
# 128  Echo Request
# 129  Echo Reply
# 133  Router Solicitation (NDP)
# 134  Router Advertisement (NDP)
# 135  Neighbor Solicitation (NDP — replaces ARP)
# 136  Neighbor Advertisement (NDP)
# 137  Redirect

# Blocking ICMPv6 types 133-136 completely breaks IPv6 networking
```

## Monitoring & Troubleshooting

```bash
# Capture all ICMP traffic
tcpdump -i eth0 icmp
tcpdump -i eth0 icmp -v                # verbose (show type/code)

# Capture specific ICMP types
tcpdump -i eth0 'icmp[icmptype] == 3'  # destination unreachable
tcpdump -i eth0 'icmp[icmptype] == 11' # time exceeded
tcpdump -i eth0 'icmp[icmptype] == 5'  # redirect

# Filter ICMP unreachable with specific code
tcpdump -i eth0 'icmp[icmptype] == 3 and icmp[icmpcode] == 4'  # frag needed

# Count ICMP errors per type
tcpdump -i eth0 -c 1000 icmp 2>/dev/null | \
    awk '/ICMP/{print $0}' | sort | uniq -c | sort -rn

# Check kernel ICMP statistics
nstat -az | grep Icmp
cat /proc/net/snmp | grep Icmp

# ICMP rate limiting (kernel)
sysctl net.ipv4.icmp_ratelimit           # default 1000 ms
sysctl net.ipv4.icmp_ratemask            # which types are rate limited
```

## Tips

- Never block ICMP Type 3 Code 4 (Fragmentation Needed / Packet Too Big). Blocking it creates a "black hole" where TCP connections hang after the handshake because PMTUD cannot negotiate the correct MSS. This is the most common ICMP-related outage.
- ICMP error messages always include the original IP header plus the first 8 bytes of the original datagram. This lets the sender match errors to specific flows (source/dest port for TCP/UDP).
- `ping -f` (flood ping) is useful for stress testing but requires root. It sends packets as fast as replies come back and only prints dots/backspaces. Use `-c` to limit packet count.
- ICMP redirects are a security risk on untrusted networks. Disable `accept_redirects` on servers and workstations. Only routers should send redirects, and only on trusted internal segments.
- The TTL field in the IP header decrements at each router hop. When it reaches 0, the router drops the packet and sends ICMP Type 11 Code 0 (Time Exceeded). This is how traceroute works.
- ICMPv6 is not optional. Filtering all ICMPv6 breaks Neighbor Discovery (replacing ARP), Router Discovery (replacing DHCP gateway), and PMTUD. At minimum, allow types 1-4 and 128-137.
- Source Quench (Type 4) was deprecated by RFC 6633 because it was ineffective and abusable. Modern systems should neither send nor act on Source Quench messages.
- Some firewalls silently drop ICMP, making ping appear to fail while TCP works. Use `tcpdump` on both ends to determine if the issue is ICMP filtering or actual reachability.
- ICMP timestamp requests (Type 13/14) can reveal system clocks for timing attacks. Most modern systems do not respond to them by default.
- The Linux kernel rate-limits outgoing ICMP error messages (default 1000ms between errors). This prevents ICMP floods from amplifying but can make debugging intermittent — errors may be silently suppressed.

## See Also

- ip, ipv4, ipv6, traceroute, mtr, tcpdump, iptables

## References

- [RFC 792 — Internet Control Message Protocol](https://www.rfc-editor.org/rfc/rfc792)
- [RFC 4443 — ICMPv6](https://www.rfc-editor.org/rfc/rfc4443)
- [RFC 1191 — Path MTU Discovery](https://www.rfc-editor.org/rfc/rfc1191)
- [RFC 8201 — Path MTU Discovery for IPv6](https://www.rfc-editor.org/rfc/rfc8201)
- [RFC 6633 — Deprecation of ICMP Source Quench](https://www.rfc-editor.org/rfc/rfc6633)
- [RFC 4884 — Extended ICMP for Multi-Part Messages](https://www.rfc-editor.org/rfc/rfc4884)
- [man ping](https://linux.die.net/man/8/ping)
