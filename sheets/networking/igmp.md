# IGMP (Internet Group Management Protocol)

Layer 3 protocol (IP protocol number 2) that enables IPv4 hosts to report multicast group membership to neighboring routers, allowing the network to deliver multicast traffic only to segments with interested receivers. Defined in RFC 3376 (IGMPv3), it is the cornerstone of IP multicast on local segments, complemented by MLD for IPv6 and PIM for inter-router multicast routing.

---

## IGMP Message Format

```
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|  Type (8)     | Max Resp Code | Checksum (16)                 |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                    Group Address (32)                          |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+

IGMPv3 Membership Query additionally includes:
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
| Resv  |S|QRV  | QQIC          | Number of Sources (16)        |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                    Source Address [1]                          |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                    Source Address [N]                          |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+

Type Values:
  0x11 = Membership Query (v1/v2/v3)
  0x12 = v1 Membership Report
  0x16 = v2 Membership Report
  0x17 = v2 Leave Group
  0x22 = v3 Membership Report

# IGMPv3 reports are sent to 224.0.0.22 (all IGMPv3-capable routers)
# Queries are sent to 224.0.0.1 (all hosts) or the group address
# All IGMP packets use IP TTL=1 and Router Alert option
```

## Version Comparison

```
Feature                IGMPv1 (RFC 1112)   IGMPv2 (RFC 2236)   IGMPv3 (RFC 3376)
─────────────────────  ──────────────────   ──────────────────   ──────────────────
Leave mechanism        None (timeout)       Explicit Leave       Explicit Leave
Group-specific query   No                   Yes                  Yes
Source filtering       No                   No                   Yes (SSM support)
Max Resp Time field    Fixed (10s)          Variable             Variable (exp)
Querier election       Based on routing     Lowest IP wins       Lowest IP wins
Report suppression     Yes                  Yes                  No (v3 uses 224.0.0.22)
Leave message dest     N/A                  224.0.0.2            224.0.0.22

# IGMPv3 is backward compatible: v3 routers handle v1/v2 reports
# v3 introduced INCLUDE/EXCLUDE filter modes for SSM (source-specific multicast)
```

## Group Join and Leave Process

### IGMPv2 Join Flow

```
Host                              Router (Querier)
  |                                  |
  |  Unsolicited Membership Report   |
  |  Dst: 239.1.1.1 (group)         |
  |  Type: 0x16                      |
  |  ──────────────────────────────> |
  |                                  |  Router adds group to
  |                                  |  forwarding table for
  |                                  |  this interface
  |                                  |
  |  General Query (periodic)        |
  |  Dst: 224.0.0.1                  |
  |  <────────────────────────────── |
  |                                  |
  |  Membership Report (response)    |
  |  ──────────────────────────────> |
```

### IGMPv2 Leave Flow

```bash
# When last host in a group leaves:
# 1. Host sends Leave (0x17) to 224.0.0.2
# 2. Router sends Group-Specific Query to the group address
# 3. If no report within Last Member Query Interval (LMQI), group is pruned

# LMQI = Last Member Query Interval * Last Member Query Count
# Default: 1s * 2 = 2 seconds until group pruned after leave
```

### IGMPv3 Source Filtering

```bash
# IGMPv3 adds source-specific multicast (SSM) support
# Two filter modes per group membership:

# INCLUDE mode — only accept traffic from listed sources
# Record Type 1 (IS_IN): {239.1.1.1, INCLUDE, {10.0.0.1, 10.0.0.2}}

# EXCLUDE mode — accept from all except listed sources (ASM behavior)
# Record Type 2 (IS_EX): {239.1.1.1, EXCLUDE, {}}  # equivalent to v2 join

# State change records:
# Type 3 (TO_IN) — change to INCLUDE mode
# Type 4 (TO_EX) — change to EXCLUDE mode
# Type 5 (ALLOW)  — allow additional sources
# Type 6 (BLOCK)  — block listed sources
```

## IGMP Snooping

```bash
# IGMP snooping lets Layer 2 switches examine IGMP messages
# to restrict multicast forwarding to ports with interested receivers

# Check snooping status on Linux bridge
cat /sys/class/net/br0/bridge/multicast_snooping
# 1 = enabled (default)

# Disable snooping (floods multicast like broadcast)
echo 0 > /sys/class/net/br0/bridge/multicast_snooping

# View multicast group memberships per bridge port
bridge mdb show dev br0

# Add static MDB entry (pin multicast to specific port)
bridge mdb add dev br0 port eth1 grp 239.1.1.1 permanent

# Delete static MDB entry
bridge mdb del dev br0 port eth1 grp 239.1.1.1

# Bridge snooping timers
cat /sys/class/net/br0/bridge/multicast_query_interval
# Default: 12500 (centiseconds = 125 seconds)

cat /sys/class/net/br0/bridge/multicast_last_member_interval
# Default: 100 (centiseconds = 1 second)

# Enable IGMP querier on the bridge (if no external querier exists)
echo 1 > /sys/class/net/br0/bridge/multicast_querier
```

## Querier Election

```bash
# Only one IGMP querier per subnet — lowest IP address wins
# All routers start as querier and send General Queries
# When a router sees a query from a lower IP, it becomes non-querier
# Non-querier timer: Other Querier Present Interval (default ~255s)

# Check which router is the active querier
# On a Cisco switch:
# show ip igmp snooping querier

# On Linux bridge, the bridge itself can act as querier
echo 1 > /sys/class/net/br0/bridge/multicast_querier

# Query interval (how often the querier sends General Queries)
echo 12500 > /sys/class/net/br0/bridge/multicast_query_interval

# Startup query count (rapid queries at boot)
cat /sys/class/net/br0/bridge/multicast_startup_query_count
# Default: 2
```

## Multicast Addressing

```bash
# IPv4 multicast range: 224.0.0.0/4 (224.0.0.0 — 239.255.255.255)
# Mapped to Ethernet MAC: 01:00:5E:00:00:00 — 01:00:5E:7F:FF:FF

# MAC mapping formula (only low 23 bits of IP mapped):
# MAC[3] = IP[1] & 0x7F
# MAC[4] = IP[2]
# MAC[5] = IP[3]
# Example: 239.1.1.1 -> 01:00:5E:01:01:01
# WARNING: 239.129.1.1 also maps to 01:00:5E:01:01:01 (32:1 ambiguity)

# Well-known multicast addresses:
# 224.0.0.1   All Hosts (used for General Queries)
# 224.0.0.2   All Routers (used for v2 Leave messages)
# 224.0.0.5   OSPF All Routers
# 224.0.0.6   OSPF Designated Routers
# 224.0.0.9   RIPv2
# 224.0.0.13  PIM Routers
# 224.0.0.22  IGMPv3 Reports
# 224.0.0.251 mDNS
# 224.0.0.252 LLMNR
# 224.0.1.1   NTP

# IPv6 multicast: ff00::/8
# ff02::1     All nodes (link-local)
# ff02::2     All routers (link-local)
# ff02::16    MLDv2-capable routers

# TTL scoping for multicast:
# TTL=0   restricted to same host
# TTL=1   restricted to same subnet (link-local, 224.0.0.0/24)
# TTL=32  restricted to same site
# TTL=64  restricted to same region
# TTL=128 restricted to same continent
# TTL=255 unrestricted (global)

# Administrative scoping (preferred over TTL):
# 239.0.0.0/8 = administratively scoped (site-local)
```

## MLD — Multicast Listener Discovery (IPv6)

```bash
# MLD is the IPv6 equivalent of IGMP
# MLDv1 (RFC 2710) ≈ IGMPv2
# MLDv2 (RFC 3810) ≈ IGMPv3

# MLD uses ICMPv6 messages (not a separate protocol):
# Type 130: Multicast Listener Query
# Type 131: MLDv1 Multicast Listener Report
# Type 132: MLDv1 Multicast Listener Done
# Type 143: MLDv2 Multicast Listener Report

# View IPv6 multicast group memberships
ip -6 maddr show

# Check MLD snooping on a bridge
cat /sys/class/net/br0/bridge/multicast_mld_version
# 1 = MLDv1, 2 = MLDv2

# MLD messages are sent with IPv6 Hop Limit = 1
# and carry the Router Alert hop-by-hop extension header
```

## PIM Integration (Basics)

```bash
# PIM (Protocol Independent Multicast) works with IGMP:
# - IGMP: host-to-router (first hop)
# - PIM: router-to-router (multicast tree building)

# PIM-SM (Sparse Mode): explicit join, uses RP (Rendezvous Point)
# PIM-SSM (Source-Specific): no RP, uses IGMPv3 INCLUDE mode
# SSM range: 232.0.0.0/8 (IPv4), ff3x::/32 (IPv6)

# Install pimd (lightweight PIM-SM daemon)
apt install pimd

# pimd configuration (/etc/pimd.conf)
# phyint eth0 enable
# phyint eth1 enable
# bsr-candidate eth0 priority 5
# rp-candidate eth0 priority 20 group-prefix 239.0.0.0 masklen 8

# Check PIM neighbors
# Cisco: show ip pim neighbor
# FRR: show ip pim neighbor

# Install smcroute for static multicast routing
apt install smcroute

# Add static multicast route (source, group, outgoing interfaces)
smcroutectl add eth0 10.0.0.1 239.1.1.1 eth1 eth2

# Remove static route
smcroutectl remove eth0 10.0.0.1 239.1.1.1

# View multicast routing table (kernel)
ip mroute show
cat /proc/net/ip_mr_cache
```

## Linux Multicast Configuration

```bash
# Join a multicast group on an interface
ip maddr add 239.1.1.1 dev eth0

# Leave a multicast group
ip maddr del 239.1.1.1 dev eth0

# View all multicast group memberships
ip maddr show
ip maddr show dev eth0

# Enable multicast forwarding in kernel
sysctl -w net.ipv4.ip_forward=1
sysctl -w net.ipv4.conf.all.mc_forwarding=1

# Kernel multicast routing table
cat /proc/net/ip_mr_vif     # multicast virtual interfaces
cat /proc/net/ip_mr_cache   # multicast forwarding cache

# IGMP version control per interface
# Force IGMPv2 on eth0 (for compatibility with v2-only networks)
echo 2 > /proc/sys/net/ipv4/conf/eth0/force_igmp_version

# Reset to auto-negotiate
echo 0 > /proc/sys/net/ipv4/conf/eth0/force_igmp_version

# IGMP max memberships per socket (default 20)
sysctl net.ipv4.igmp_max_memberships
sysctl -w net.ipv4.igmp_max_memberships=256

# IGMP max source filter entries per socket
sysctl net.ipv4.igmp_max_msf

# Socket-level multicast join (used by applications)
# IP_ADD_MEMBERSHIP — join a group
# IP_ADD_SOURCE_MEMBERSHIP — join a (source, group) for SSM
# IP_DROP_MEMBERSHIP — leave a group
# IP_MULTICAST_TTL — set outgoing multicast TTL
# IP_MULTICAST_IF — set outgoing interface
# IP_MULTICAST_LOOP — enable/disable loopback of own multicast
```

## IGMP Proxy

```bash
# IGMP proxy allows a router to act as a proxy between
# an upstream multicast network and a downstream segment

# Install igmpproxy
apt install igmpproxy

# Configuration (/etc/igmpproxy.conf)
# quickleave mode on
#
# phyint eth0 upstream  ratelimit 0  threshold 1
#     altnet 0.0.0.0/0
#
# phyint eth1 downstream  ratelimit 0  threshold 1

# Start the proxy
igmpproxy /etc/igmpproxy.conf

# igmpproxy aggregates downstream IGMP reports and
# forwards them upstream, presenting downstream hosts
# as a single IGMP client to the upstream router
```

## Capturing and Debugging IGMP

```bash
# Capture IGMP packets with tcpdump
tcpdump -i eth0 -vv igmp

# Capture only IGMP queries
tcpdump -i eth0 -vv 'igmp[0] = 0x11'

# Capture only IGMPv3 reports
tcpdump -i eth0 -vv 'igmp[0] = 0x22'

# Capture only IGMPv2 leave messages
tcpdump -i eth0 -vv 'igmp[0] = 0x17'

# Capture all multicast traffic (Ethernet destination bit 0 set)
tcpdump -i eth0 'ether[0] & 1 != 0'

# Filter specific multicast group
tcpdump -i eth0 -vv 'host 239.1.1.1'

# Watch multicast join/leave in real time with timestamps
tcpdump -i eth0 -tttt -vv igmp

# Check socket multicast memberships
cat /proc/net/igmp
# Output columns: Idx, Device, Count, Querier, Group (hex), Timer, etc.

# Decode hex group address from /proc/net/igmp
# E10101EF = 239.1.1.1 (bytes are reversed: EF=239, 01=1, 01=1, E1=225 -- wrong)
printf '%d.%d.%d.%d\n' 0xEF 0x01 0x01 0x01
# 239.1.1.1

# Verify multicast route is installed
ip mroute show
# (Iif, Oifs) shows incoming and outgoing interfaces

# Test multicast reception
socat UDP4-RECVFROM:5000,ip-add-membership=239.1.1.1:eth0,fork -

# Test multicast send
echo "hello multicast" | socat - UDP4-DATAGRAM:239.1.1.1:5000,ip-multicast-if=eth0
```

---

## Tips

- Always verify IGMP snooping is enabled on your switches. Without it, multicast floods every port like broadcast, negating the bandwidth savings that multicast provides.
- When using IGMPv3 with SSM (source-specific multicast, 232.0.0.0/8), applications must use the `IP_ADD_SOURCE_MEMBERSHIP` socket option, not `IP_ADD_MEMBERSHIP`. The wrong option falls back to ASM behavior.
- The 32:1 MAC address ambiguity in multicast (23-bit mapping from 28-bit group ID) means switches may deliver unwanted groups to a host. IGMP snooping mitigates this at Layer 2.
- On Linux bridges, `multicast_querier` must be enabled if no external multicast router exists on the segment, otherwise group memberships time out and multicast stops flowing.
- Force IGMPv2 on interfaces connected to legacy networks by writing to `/proc/sys/net/ipv4/conf/<iface>/force_igmp_version`. Version mismatch causes silent group membership failures.
- IGMP messages use TTL=1 and the IP Router Alert option. Firewalls or ACLs that drop low-TTL packets or strip IP options will break multicast group management silently.
- Increase `igmp_max_memberships` (default 20) on hosts that need to join many groups simultaneously, such as IPTV set-top boxes or multicast monitoring tools.
- When debugging multicast, check all three layers: IGMP (host membership), PIM (router tree), and IGMP snooping (switch forwarding). A problem at any layer breaks delivery.
- The `quickleave` option in IGMP snooping immediately removes the port on Leave without waiting for the query response. Use it on segments with one host per port (like IPTV deployments).
- MLD snooping must also be enabled for IPv6 multicast environments. IPv6 relies heavily on multicast (neighbor discovery, router advertisements), so broken MLD snooping disrupts basic IPv6 connectivity.

---

## See Also

- ipv6, ndp, vlan, tc

## References

- [RFC 3376 — IGMPv3](https://www.rfc-editor.org/rfc/rfc3376)
- [RFC 2236 — IGMPv2](https://www.rfc-editor.org/rfc/rfc2236)
- [RFC 1112 — IGMPv1 / Host Extensions for IP Multicasting](https://www.rfc-editor.org/rfc/rfc1112)
- [RFC 3810 — MLDv2 for IPv6](https://www.rfc-editor.org/rfc/rfc3810)
- [RFC 4604 — Using IGMPv3 and MLDv2 for SSM](https://www.rfc-editor.org/rfc/rfc4604)
- [RFC 4541 — IGMP and MLD Snooping Switches](https://www.rfc-editor.org/rfc/rfc4541)
- [Linux Kernel — Multicast](https://www.kernel.org/doc/html/latest/networking/multicast.html)
- [man smcroute](https://github.com/troglobit/smcroute)
