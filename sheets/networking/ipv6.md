# IPv6 (Internet Protocol version 6 Reference)

A comprehensive reference for IPv6 addressing, configuration, neighbor discovery, and troubleshooting on Linux systems.

## Address Format

### 128-Bit Address Structure

```bash
# IPv6 addresses are 128 bits, written as 8 groups of 4 hex digits separated by colons
# Full form:
2001:0db8:0000:0000:0000:0000:0000:0001

# Leading zeros in each group can be omitted
2001:db8:0:0:0:0:0:1

# One consecutive run of all-zero groups can be replaced with ::
2001:db8::1

# :: can only appear ONCE in an address (ambiguity otherwise)
# WRONG: 2001::db8::1
# RIGHT: 2001:0:0:db8:0:0:0:1  or  2001::db8:0:0:0:1  or  2001:0:0:db8::1
```

### Prefix Notation (CIDR)

```bash
# Prefix length follows a slash, just like IPv4 CIDR
2001:db8:abcd::/48       # 48-bit network prefix, 80 bits for host
fe80::1%eth0/64          # link-local with scope ID and /64 prefix

# Common prefix lengths
# /32  — typical ISP allocation
# /48  — typical site allocation
# /56  — typical home/small office allocation
# /64  — single subnet (standard for SLAAC)
# /128 — single host (loopback, point-to-point)
```

## Address Types

### Link-Local (fe80::/10)

```bash
# Automatically assigned to every IPv6-enabled interface
# Not routable — only valid on the local link
# Always require a scope ID (%interface) when used with tools
ip -6 addr show dev eth0 | grep fe80

# Example: fe80::1a2b:3c4d:5e6f:7890%eth0
ping6 fe80::1%eth0
```

### Global Unicast (2000::/3)

```bash
# Routable on the public internet (equivalent to public IPv4)
# Range: 2000:: to 3fff:ffff:ffff:ffff:ffff:ffff:ffff:ffff
# Typically assigned via SLAAC or DHCPv6
ip -6 addr show scope global
```

### Unique-Local (fc00::/7)

```bash
# Private addresses, not routed on the internet (like RFC1918 in IPv4)
# fd00::/8 is the commonly used half (locally assigned)
# fc00::/8 is reserved for future use
# Example: fd12:3456:789a::1
ip -6 route show | grep "^fd"
```

### Multicast (ff00::/8)

```bash
# One-to-many delivery
# ff02::1   — all nodes on link-local
# ff02::2   — all routers on link-local
# ff02::fb  — mDNS
# ff02::101 — NTP
# ff05::2   — all routers on site-local
ip -6 maddr show dev eth0
```

### Loopback (::1)

```bash
# Equivalent to 127.0.0.1 in IPv4
ping6 ::1
# or
ping -6 ::1
```

### Special Addresses

```bash
# All-nodes multicast (link-local scope)
ff02::1

# All-routers multicast (link-local scope)
ff02::2

# Solicited-node multicast (ff02::1:ff00:0/104)
# Formed from last 24 bits of unicast address
# Used by NDP for efficient neighbor resolution
# For address 2001:db8::1a2b:3cff:fe4d:5678, solicited-node is:
# ff02::1:ff4d:5678

# Unspecified address (used during DAD, like 0.0.0.0 in IPv4)
::

# IPv4-mapped IPv6 address (for dual-stack sockets)
::ffff:192.168.1.1
```

## Address Assignment

### EUI-64 (Extended Unique Identifier)

```bash
# Derives interface ID from MAC address
# MAC: 00:1a:2b:3c:4d:5e
# Step 1: Insert ff:fe in the middle -> 00:1a:2b:ff:fe:3c:4d:5e
# Step 2: Flip the 7th bit (U/L bit)  -> 02:1a:2b:ff:fe:3c:4d:5e
# Result interface ID: 021a:2bff:fe3c:4d5e

# EUI-64 is predictable and exposes the MAC — privacy concern
# Check if interface is using EUI-64
ip -6 addr show dev eth0 | grep -i "ff:fe"
```

### Privacy Extensions (RFC 4941)

```bash
# Generate random temporary addresses to prevent tracking
# Check current setting
sysctl net.ipv6.conf.eth0.use_tempaddr
# 0 = disabled, 1 = generate but prefer public, 2 = prefer temporary

# Enable privacy extensions
sudo sysctl -w net.ipv6.conf.all.use_tempaddr=2
sudo sysctl -w net.ipv6.conf.default.use_tempaddr=2

# Persistent in /etc/sysctl.d/
echo "net.ipv6.conf.all.use_tempaddr = 2" | sudo tee /etc/sysctl.d/10-ipv6-privacy.conf
```

### SLAAC (Stateless Address Autoconfiguration)

```bash
# Host generates its own address from RA prefix + interface ID
# No server needed — router sends prefix via Router Advertisement
# Requires /64 prefix length

# Check if SLAAC is enabled
sysctl net.ipv6.conf.eth0.autoconf
# 1 = enabled (default)

# Disable SLAAC on an interface
sudo sysctl -w net.ipv6.conf.eth0.autoconf=0

# Accept Router Advertisements
sysctl net.ipv6.conf.eth0.accept_ra
# 0 = never, 1 = accept if not forwarding, 2 = always accept
```

### DHCPv6

```bash
# Stateful address assignment from a DHCPv6 server
# Can provide addresses, DNS servers, and other options
# Router Advertisement M and O flags control DHCPv6 behavior:
#   M (Managed) flag = use DHCPv6 for addresses
#   O (Other) flag   = use DHCPv6 for DNS/options only

# dhclient for DHCPv6
sudo dhclient -6 eth0

# Release DHCPv6 lease
sudo dhclient -6 -r eth0
```

## NDP (Neighbor Discovery Protocol)

### Neighbor Solicitation / Advertisement (NS/NA)

```bash
# NS/NA replace ARP from IPv4
# NS — "who has this IPv6 address?" (sent to solicited-node multicast)
# NA — "I have it, here is my MAC"

# View neighbor cache (equivalent to ARP table)
ip -6 neigh show

# Manually add a neighbor entry
sudo ip -6 neigh add 2001:db8::1 lladdr 00:1a:2b:3c:4d:5e dev eth0

# Delete a neighbor entry
sudo ip -6 neigh del 2001:db8::1 dev eth0

# Flush neighbor cache
sudo ip -6 neigh flush dev eth0
```

### Router Solicitation / Advertisement (RS/RA)

```bash
# RS — host asks for router information on boot
# RA — router announces prefix, default route, MTU, hop limit

# Send a Router Solicitation manually
rdisc6 eth0

# Monitor RAs with tcpdump
sudo tcpdump -i eth0 -n icmp6 and 'ip6[40] == 134'
# type 133 = RS, type 134 = RA
```

### Duplicate Address Detection (DAD)

```bash
# Before using an address, host sends NS for its own address
# If someone replies, address is duplicate — marked "dadfailed"

# Check for DAD failures
ip -6 addr show dadfailed

# Number of DAD probes to send (0 disables DAD)
sysctl net.ipv6.conf.eth0.dad_transmits

# Disable DAD (not recommended)
sudo sysctl -w net.ipv6.conf.eth0.dad_transmits=0
```

## Linux Configuration

### Address and Route Management

```bash
# Show all IPv6 addresses
ip -6 addr show

# Show IPv6 addresses on a specific interface
ip -6 addr show dev eth0

# Add a static IPv6 address
sudo ip -6 addr add 2001:db8::1/64 dev eth0

# Remove an IPv6 address
sudo ip -6 addr del 2001:db8::1/64 dev eth0

# Show IPv6 routing table
ip -6 route show

# Add a static route
sudo ip -6 route add 2001:db8:abcd::/48 via 2001:db8::ffff dev eth0

# Add default route
sudo ip -6 route add default via 2001:db8::1 dev eth0

# Delete a route
sudo ip -6 route del 2001:db8:abcd::/48
```

### Sysctl Tuning

```bash
# Disable IPv6 entirely on an interface
sudo sysctl -w net.ipv6.conf.eth0.disable_ipv6=1

# Disable IPv6 on all interfaces
sudo sysctl -w net.ipv6.conf.all.disable_ipv6=1

# Enable forwarding (make this host a router)
sudo sysctl -w net.ipv6.conf.all.forwarding=1

# Control accept_ra behavior when forwarding is enabled
sudo sysctl -w net.ipv6.conf.eth0.accept_ra=2

# Set hop limit for outgoing packets
sudo sysctl -w net.ipv6.conf.all.hop_limit=64
```

### Persistent Configuration (RHEL/CentOS)

```bash
# /etc/sysconfig/network-scripts/ifcfg-eth0
# IPV6INIT=yes
# IPV6ADDR=2001:db8::1/64
# IPV6_DEFAULTGW=2001:db8::ffff
# IPV6_AUTOCONF=no
# DHCPV6C=yes              # enable DHCPv6 client

# /etc/sysconfig/network
# NETWORKING_IPV6=yes
```

## Common Tools

### Connectivity Testing

```bash
# Ping an IPv6 address
ping6 2001:db8::1
# or on modern systems
ping -6 2001:db8::1

# Ping link-local (must include scope ID)
ping6 fe80::1%eth0

# Traceroute over IPv6
traceroute6 2001:db8::1
# or
traceroute -6 2001:db8::1

# IPv6-only socket listing
ss -6 -tuln
```

### Neighbor and Discovery Tools

```bash
# Show IPv6 neighbor table
ip -6 neigh show

# Discover neighbors on a link (ndisc6 package)
ndisc6 fe80::1 eth0

# Send Router Solicitation
rdisc6 eth0

# Resolve IPv6 address to name
host 2001:db8::1
dig -x 2001:db8::1
```

## Firewall

### ip6tables

```bash
# List IPv6 firewall rules
sudo ip6tables -L -n -v

# Allow ICMPv6 (critical — do not block all ICMPv6 in IPv6)
sudo ip6tables -A INPUT -p icmpv6 -j ACCEPT

# Allow established connections
sudo ip6tables -A INPUT -m conntrack --ctstate ESTABLISHED,RELATED -j ACCEPT

# Drop everything else
sudo ip6tables -A INPUT -j DROP
```

### nftables (inet family)

```bash
# nftables inet family handles both IPv4 and IPv6
sudo nft add table inet filter
sudo nft add chain inet filter input '{ type filter hook input priority 0; policy drop; }'

# Allow ICMPv6
sudo nft add rule inet filter input meta nfproto ipv6 icmpv6 type '{
  nd-neighbor-solicit, nd-neighbor-advert,
  nd-router-solicit, nd-router-advert,
  echo-request, echo-reply
}' accept

# Allow established
sudo nft add rule inet filter input ct state established,related accept
```

## DNS for IPv6

### AAAA Records and Reverse DNS

```bash
# Query AAAA record
dig AAAA example.com

# Reverse DNS for IPv6 uses ip6.arpa (nibble format)
# 2001:db8::1 -> 1.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.8.b.d.0.1.0.0.2.ip6.arpa.
dig -x 2001:db8::1

# Add AAAA record in zone file
# example.com.  IN  AAAA  2001:db8::1
```

## Dual-Stack Considerations

```bash
# Most modern systems run dual-stack (IPv4 + IPv6 simultaneously)
# Applications bind to both by default on :: (dual-stack socket)

# Check if a service is listening on both stacks
ss -tuln | grep -E '(\*|::):'

# Prefer IPv4 or IPv6 in /etc/gai.conf
# Uncomment to prefer IPv4:
# precedence ::ffff:0:0/96  100

# Test dual-stack connectivity
curl -4 https://example.com    # force IPv4
curl -6 https://example.com    # force IPv6
```

## Transition Mechanisms

### 6to4, NAT64, and 464XLAT

```bash
# 6to4 (deprecated, RFC 7526) — encapsulates IPv6 in IPv4
# Uses 2002::/16 prefix, largely abandoned due to reliability issues

# NAT64 — translates IPv6 packets to IPv4 (for IPv6-only networks)
# Uses well-known prefix 64:ff9b::/96
# Requires DNS64 to synthesize AAAA records for IPv4-only destinations

# 464XLAT — client-side CLAT (stateless NAT46) + server-side PLAT (stateful NAT64)
# Allows IPv4 apps to work on IPv6-only networks
# Common on mobile networks (Android, T-Mobile, etc.)

# Check for NAT64 prefix
dig AAAA ipv4only.arpa
# If you get a response, NAT64 is in use — the prefix is revealed
```

## Troubleshooting

```bash
# DAD failure — address shows "dadfailed" or "tentative"
ip -6 addr show dadfailed
# Fix: check for duplicate addresses on the link, or disable DAD temporarily

# RA not received — no global address assigned
rdisc6 eth0                            # manually request RA
sudo tcpdump -i eth0 icmp6             # watch for RA traffic
sysctl net.ipv6.conf.eth0.accept_ra    # must be 1 (or 2 if forwarding)

# Scope ID required for link-local
# WRONG: ping6 fe80::1
# RIGHT: ping6 fe80::1%eth0
# Find the correct scope ID
ip link show                           # interface names are the scope IDs

# Path MTU issues (common with tunnels)
ping6 -M do -s 1452 2001:db8::1        # test with do-not-fragment

# Verify IPv6 forwarding is enabled (for routers)
sysctl net.ipv6.conf.all.forwarding

# Check for rogue RAs (SLAAC attacks)
sudo tcpdump -i eth0 -n 'icmp6 and ip6[40] == 134'
```

## Tips

- Never block all ICMPv6 in firewall rules. NDP (neighbor discovery) and PMTUD depend on it. At minimum allow types 133-137.
- Link-local addresses always require a scope ID (`%eth0`) when used with CLI tools. Forgetting this is the most common IPv6 command error.
- SLAAC requires exactly a /64 prefix. Do not try to use /48 or /128 with SLAAC.
- Privacy extensions (use_tempaddr=2) should be enabled on client machines to prevent tracking via stable EUI-64 addresses.
- When troubleshooting "no IPv6 connectivity," check in order: (1) link-local present? (2) RA received? (3) global address assigned? (4) default route exists? (5) firewall rules?
- The `::` in an address can only appear once. If you need to represent multiple zero groups, only compress the longest run (or the leftmost if tied).
- Use `ip -6` commands instead of the deprecated `ifconfig` for IPv6 management.
- For production servers, prefer static addresses or DHCPv6 over SLAAC for predictable addressing.
- Test IPv6 reachability to the outside world with `ping6 2600::` (Sprint) or `curl -6 https://ipv6.google.com`.

## References

- [RFC 8200 — Internet Protocol, Version 6 (IPv6) Specification](https://www.rfc-editor.org/rfc/rfc8200)
- [RFC 4861 — Neighbor Discovery Protocol (NDP) for IPv6](https://www.rfc-editor.org/rfc/rfc4861)
- [RFC 4862 — IPv6 Stateless Address Autoconfiguration (SLAAC)](https://www.rfc-editor.org/rfc/rfc4862)
- [RFC 4193 — Unique Local IPv6 Unicast Addresses](https://www.rfc-editor.org/rfc/rfc4193)
- [RFC 5952 — A Recommendation for IPv6 Address Text Representation](https://www.rfc-editor.org/rfc/rfc5952)
- [RFC 6724 — Default Address Selection for IPv6](https://www.rfc-editor.org/rfc/rfc6724)
- [RFC 8415 — Dynamic Host Configuration Protocol for IPv6 (DHCPv6)](https://www.rfc-editor.org/rfc/rfc8415)
- [IANA IPv6 Special-Purpose Address Registry](https://www.iana.org/assignments/iana-ipv6-special-registry/iana-ipv6-special-registry.xhtml)
- [Linux Kernel — IPv6 Sysctl Documentation](https://www.kernel.org/doc/html/latest/networking/ip-sysctl.html)
- [RIPE — IPv6 Info Center](https://www.ripe.net/publications/ipv6-info-centre/)
