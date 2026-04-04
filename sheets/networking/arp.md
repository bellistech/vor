# ARP (Address Resolution Protocol)

Layer 2 protocol that maps IPv4 addresses to MAC addresses on a local network segment using broadcast requests and unicast replies, operating below IP with EtherType 0x0806.

## ARP Resolution Process

```
Host A (192.168.1.10)                           Host B (192.168.1.20)
  |                                                |
  |  -- ARP Request (broadcast) ----------------> |
  |     "Who has 192.168.1.20? Tell 192.168.1.10"  |
  |     Dst MAC: ff:ff:ff:ff:ff:ff                 |
  |     Src MAC: aa:bb:cc:dd:ee:01                 |
  |                                                |
  |  <- ARP Reply (unicast) ---------------------  |
  |     "192.168.1.20 is at aa:bb:cc:dd:ee:02"    |
  |     Dst MAC: aa:bb:cc:dd:ee:01                 |
  |     Src MAC: aa:bb:cc:dd:ee:02                 |
  |                                                |

# Both sides update their ARP cache after this exchange
```

## ARP Packet Format

```
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|         Hardware Type         |         Protocol Type         |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
| HW Addr Len   | Proto Addr Len|           Operation           |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                    Sender Hardware Address                     |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                    Sender Protocol Address                    |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                    Target Hardware Address                     |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                    Target Protocol Address                    |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+

Hardware Type:    1 = Ethernet
Protocol Type:    0x0800 = IPv4
HW Addr Length:   6 (MAC)
Proto Addr Length: 4 (IPv4)
Operation:        1 = Request, 2 = Reply
```

## ARP Types

### Standard ARP

```
# Normal resolution — "who has this IP?"
# Sent when a host needs to communicate with an IP on the same subnet
# and doesn't have the MAC in its cache
```

### Gratuitous ARP (GARP)

```
# Sender and target IP are the SAME address
# Purposes:
#   1. Announce IP address (after boot or IP change)
#   2. Detect IP conflicts (if someone replies, conflict exists)
#   3. Update switches/hosts after failover (VRRP, HSRP)

# Send gratuitous ARP manually
arping -U -I eth0 192.168.1.10       # ARP request form
arping -A -I eth0 192.168.1.10       # ARP reply form (unsolicited)
```

### Proxy ARP

```
# A router answers ARP requests on behalf of hosts on another subnet
# The router replies with its own MAC address

# Enable/disable proxy ARP on Linux
sysctl -w net.ipv4.conf.eth0.proxy_arp=1      # enable
sysctl -w net.ipv4.conf.eth0.proxy_arp=0      # disable

# Check current setting
sysctl net.ipv4.conf.eth0.proxy_arp

# Proxy ARP is sometimes used when subnetting is misconfigured
# or for VPN concentrators answering for remote clients
```

### Reverse ARP (RARP)

```
# Obsolete — host knows its MAC, asks for its IP
# Replaced by BOOTP, then DHCP
# Operation: 3 = RARP Request, 4 = RARP Reply
```

## ARP Cache Management

### Viewing the Cache

```bash
# Modern Linux (iproute2)
ip neigh show                            # show all ARP entries
ip neigh show dev eth0                   # entries on specific interface
ip -s neigh show                         # include statistics

# Legacy
arp -a                                   # BSD-style output
arp -an                                  # numeric (no DNS)
arp -e                                   # Linux-style table

# macOS
arp -a                                   # show ARP table
```

### Manipulating the Cache

```bash
# Add static entry
ip neigh add 192.168.1.50 lladdr 00:11:22:33:44:55 dev eth0

# Delete entry
ip neigh del 192.168.1.50 dev eth0

# Flush all entries on an interface
ip neigh flush dev eth0

# Change existing entry
ip neigh change 192.168.1.50 lladdr 00:11:22:33:44:66 dev eth0

# Legacy commands
arp -s 192.168.1.50 00:11:22:33:44:55   # add static
arp -d 192.168.1.50                      # delete entry
```

### ARP Cache States (Linux)

```
State       Description
──────────────────────────────────────────────────
REACHABLE   Recently confirmed (reply received)
STALE       Entry expired, will verify on next use
DELAY       Waiting for upper-layer confirmation
PROBE       Actively sending ARP to reconfirm
FAILED      Resolution failed after max probes
PERMANENT   Manually added static entry
NOARP       Interface doesn't use ARP (e.g., loopback)
INCOMPLETE  Request sent, no reply yet
```

## ARP Probing (arping)

```bash
# Send ARP request to specific IP
arping -I eth0 192.168.1.1

# Send N requests only
arping -c 3 -I eth0 192.168.1.1

# Detect duplicate IP addresses (DAD)
arping -D -I eth0 192.168.1.10          # returns 0 if duplicate found

# Set source IP for ARP request
arping -s 192.168.1.10 -I eth0 192.168.1.1

# Broadcast ARP reply (update everyone's cache)
arping -A -c 3 -I eth0 192.168.1.10

# Quiet mode (just exit code)
arping -q -c 1 -I eth0 192.168.1.1
```

## ARP Tuning (Linux)

```bash
# ARP cache timeout settings (in /proc or sysctl)
# base_reachable_time_ms — how long entries stay REACHABLE (default 30000 ms)
sysctl -w net.ipv4.neigh.eth0.base_reachable_time_ms=30000

# gc_stale_time — how long STALE entries persist before garbage collection (60s)
sysctl -w net.ipv4.neigh.eth0.gc_stale_time=60

# ARP table size limits
sysctl -w net.ipv4.neigh.default.gc_thresh1=128    # start GC above this
sysctl -w net.ipv4.neigh.default.gc_thresh2=512    # aggressive GC above this
sysctl -w net.ipv4.neigh.default.gc_thresh3=1024   # hard max entries

# Number of probes before marking FAILED
sysctl -w net.ipv4.neigh.eth0.ucast_solicit=3      # unicast probes
sysctl -w net.ipv4.neigh.eth0.mcast_solicit=3      # multicast probes

# ARP announce/filter modes
sysctl -w net.ipv4.conf.eth0.arp_announce=2         # use best local addr
sysctl -w net.ipv4.conf.eth0.arp_ignore=1           # only reply for local IPs
sysctl -w net.ipv4.conf.eth0.arp_filter=1           # filter by routing table
```

## ARP Poisoning Defense

```bash
# Static ARP entries for critical hosts (gateway, DNS)
ip neigh add 192.168.1.1 lladdr 00:11:22:33:44:55 nud permanent dev eth0

# Enable Dynamic ARP Inspection (DAI) on switches
# Validates ARP against DHCP snooping binding table

# arpwatch — monitor ARP changes
arpwatch -i eth0 -d                     # daemon mode, log changes
# Logs: new station, changed ethernet address, flip flop

# arptables — ARP-level firewall (like iptables for ARP)
arptables -A INPUT --src-mac ! 00:11:22:33:44:55 --src-ip 192.168.1.1 -j DROP

# Detect ARP spoofing with tcpdump
tcpdump -i eth0 -n arp | grep -i "reply"
# Look for: multiple MACs claiming same IP, or gateway MAC changing

# arpwatch alerts
# New station:           new device on network
# Changed ethernet addr: possible spoofing
# Flip flop:             two MACs fighting for same IP
```

## Monitoring & Debugging

```bash
# Watch ARP traffic in real time
tcpdump -i eth0 -n arp

# Verbose ARP decode
tcpdump -i eth0 -vvv -n arp

# Count ARP packets per second
tcpdump -i eth0 -n arp 2>/dev/null | pv -l -r > /dev/null

# Monitor ARP cache changes
watch -n 1 'ip neigh show'

# Check for ARP storms
tcpdump -i eth0 -c 1000 arp 2>/dev/null | wc -l
# More than a few hundred in seconds = likely ARP storm

# Kernel ARP statistics
cat /proc/net/arp
nstat -az | grep Arp
```

## Tips

- ARP only works within a broadcast domain (Layer 2 segment). Traffic to IPs outside your subnet goes to the default gateway's MAC, not the destination host's MAC. The router then ARPs on the remote segment.
- Linux ARP cache entries transition through REACHABLE -> STALE -> DELAY -> PROBE -> FAILED. The `base_reachable_time_ms` (default 30s) is randomized between 50-150% to prevent synchronization storms.
- Gratuitous ARP is critical for failover protocols (VRRP/HSRP/keepalived). When the VIP moves, a GARP updates all hosts' caches so traffic immediately flows to the new active node.
- On large flat Layer 2 networks (>500 hosts), ARP broadcast traffic becomes significant. Each host must process every ARP request even if it is not the target. Segment with VLANs.
- `arp_ignore=1` and `arp_announce=2` are essential for Linux load balancers using Direct Server Return (DSR). Without them, all backends reply to ARP for the VIP, breaking load distribution.
- The default `gc_thresh3=1024` ARP table limit can cause entry evictions on busy servers talking to thousands of peers. Increase to 4096 or higher for hypervisors and load balancers.
- ARP is completely unauthenticated. Any host can claim any IP-to-MAC mapping. Static entries for the default gateway and enabling DAI (Dynamic ARP Inspection) on managed switches are the primary defenses.
- IPv6 replaces ARP with NDP (Neighbor Discovery Protocol) using ICMPv6 messages (types 135/136). NDP has SEND (Secure Neighbor Discovery) for cryptographic protection, which ARP lacks.
- If `arping -D` detects a duplicate address, the interface should not use that IP. systemd-networkd and NetworkManager perform DAD automatically on address assignment.
- ARP entries for hosts behind a router all resolve to the router's MAC. If you see many IPs mapping to the same MAC, that MAC is likely a gateway, not a spoofing attack.

## See Also

- ethernet, ip, dhcp, vlan, tcpdump, iptables

## References

- [RFC 826 — An Ethernet Address Resolution Protocol](https://www.rfc-editor.org/rfc/rfc826)
- [RFC 5227 — IPv4 Address Conflict Detection](https://www.rfc-editor.org/rfc/rfc5227)
- [RFC 1027 — Using ARP to Implement Transparent Subnet Gateways (Proxy ARP)](https://www.rfc-editor.org/rfc/rfc1027)
- [Linux Kernel — Neighbor Subsystem](https://www.kernel.org/doc/html/latest/networking/ip-sysctl.html)
- [man arping](https://linux.die.net/man/8/arping)
- [man arpwatch](https://linux.die.net/man/8/arpwatch)
