# Subnetting (IP Subnet Calculation Reference)

A practical reference for CIDR notation, subnet math, VLSM design, and IPv4/IPv6 subnetting with calculation tools.

## CIDR Notation

### Prefix-to-Netmask Table

```bash
# Prefix  Netmask              Addresses    Usable Hosts
# /8      255.0.0.0            16,777,216   16,777,214
# /9      255.128.0.0           8,388,608    8,388,606
# /10     255.192.0.0           4,194,304    4,194,302
# /11     255.224.0.0           2,097,152    2,097,150
# /12     255.240.0.0           1,048,576    1,048,574
# /13     255.248.0.0             524,288      524,286
# /14     255.252.0.0             262,144      262,142
# /15     255.254.0.0             131,072      131,070
# /16     255.255.0.0              65,536       65,534
# /17     255.255.128.0            32,768       32,766
# /18     255.255.192.0            16,384       16,382
# /19     255.255.224.0             8,192        8,190
# /20     255.255.240.0             4,096        4,094
# /21     255.255.248.0             2,048        2,046
# /22     255.255.252.0             1,024        1,022
# /23     255.255.254.0               512          510
# /24     255.255.255.0               256          254
# /25     255.255.255.128              128          126
# /26     255.255.255.192               64           62
# /27     255.255.255.224               32           30
# /28     255.255.255.240               16           14
# /29     255.255.255.248                8            6
# /30     255.255.255.252                4            2
# /31     255.255.255.254                2            2 (point-to-point, RFC 3021)
# /32     255.255.255.255                1            1 (host route)
```

## IPv4 Address Classes (Historical)

### Classful Addressing

```bash
# Class A:  1.0.0.0   – 126.255.255.255   /8   default mask
# Class B:  128.0.0.0 – 191.255.255.255   /16  default mask
# Class C:  192.0.0.0 – 223.255.255.255   /24  default mask
# Class D:  224.0.0.0 – 239.255.255.255   (multicast, no mask)
# Class E:  240.0.0.0 – 255.255.255.255   (reserved/experimental)

# Classful routing is DEAD — CIDR replaced it in 1993 (RFC 1519)
# Modern networks use CIDR exclusively
# Classful knowledge is still useful for understanding legacy configs
# and why private ranges are sized the way they are
```

## Private and Special Ranges

### RFC 1918 Private Ranges

```bash
# 10.0.0.0/8       — 16,777,216 addresses (Class A private)
# 172.16.0.0/12    — 1,048,576 addresses  (Class B private, 172.16–172.31)
# 192.168.0.0/16   — 65,536 addresses     (Class C private)

# These are not routed on the public internet
# Used behind NAT for internal networks
```

### Other Special Ranges

```bash
# 127.0.0.0/8       — loopback (localhost)
# 169.254.0.0/16    — link-local / APIPA (auto-assigned when DHCP fails)
# 224.0.0.0/4       — multicast (224.0.0.0 – 239.255.255.255)
# 0.0.0.0/8         — "this network" (0.0.0.0 = default route / unspecified)
# 100.64.0.0/10     — shared address space / CGN (RFC 6598)
# 192.0.2.0/24      — documentation (TEST-NET-1, RFC 5737)
# 198.51.100.0/24   — documentation (TEST-NET-2)
# 203.0.113.0/24    — documentation (TEST-NET-3)
# 255.255.255.255   — limited broadcast
```

## Subnet Math

### Calculating Network, Broadcast, and Usable Range

```bash
# Given: 192.168.10.75/26
#
# Step 1: Find the block size
# /26 = 256 - 192 = 64 addresses per subnet (or 2^(32-26) = 64)
#
# Step 2: Find the network address
# 75 / 64 = 1 (integer part) -> 1 * 64 = 64
# Network: 192.168.10.64
#
# Step 3: Find the broadcast address
# Network + block size - 1 = 64 + 64 - 1 = 127
# Broadcast: 192.168.10.127
#
# Step 4: Usable host range
# First usable: network + 1 = 192.168.10.65
# Last usable:  broadcast - 1 = 192.168.10.126
# Usable hosts: 64 - 2 = 62
```

### Key Formulas

```bash
# Total addresses in a subnet:
# 2^(32 - prefix_length)

# Usable hosts:
# 2^(32 - prefix_length) - 2
# (subtract network address and broadcast address)
# Exception: /31 has 2 usable hosts (point-to-point links, RFC 3021)
# Exception: /32 is a single host route

# Netmask from prefix:
# Netmask = 2^32 - 2^(32 - prefix)
# /24 = 2^32 - 2^8 = 4294967296 - 256 = 4294967040 = 255.255.255.0

# Number of subnets when splitting:
# Splitting a /N into /M gives 2^(M - N) subnets
# /24 into /26 = 2^(26-24) = 4 subnets
```

## Supernetting / Summarization

```bash
# Combine multiple contiguous subnets into one larger prefix
# Also called route aggregation or route summarization

# Example: summarize these four /24s:
# 10.1.0.0/24
# 10.1.1.0/24
# 10.1.2.0/24
# 10.1.3.0/24
# Summary: 10.1.0.0/22 (covers 10.1.0.0 – 10.1.3.255)

# Rules for summarization:
# 1. Subnets must be contiguous
# 2. The number of subnets must be a power of 2
# 3. The first subnet must be on a proper boundary
#    (network address must be divisible by the total size)

# Check: 10.1.0.0 / (4 * 256) = 10.1.0.0 / 1024 -> 0.0 is at boundary -> valid
```

## VLSM (Variable Length Subnet Masking)

### Allocating Different-Sized Subnets

```bash
# VLSM lets you use different prefix lengths within the same address space
# Allocate largest subnets first to avoid fragmentation

# Example: Subnet 192.168.1.0/24 for:
#   Dept A: 100 hosts -> needs /25 (128 addresses, 126 usable)
#   Dept B: 50 hosts  -> needs /26 (64 addresses, 62 usable)
#   Dept C: 25 hosts  -> needs /27 (32 addresses, 30 usable)
#   Link 1: 2 hosts   -> needs /30 (4 addresses, 2 usable)

# Allocation (largest first):
# Dept A: 192.168.1.0/25     (192.168.1.0   – 192.168.1.127)
# Dept B: 192.168.1.128/26   (192.168.1.128 – 192.168.1.191)
# Dept C: 192.168.1.192/27   (192.168.1.192 – 192.168.1.223)
# Link 1: 192.168.1.224/30   (192.168.1.224 – 192.168.1.227)
# Free:   192.168.1.228/30 through 192.168.1.252/30

# Always verify: no overlap, and each subnet starts on a valid boundary
```

## Common Subnetting Examples

### Splitting a /24

```bash
# Splitting 192.168.1.0/24 into /26s (4 subnets, 62 hosts each):
# Subnet 1: 192.168.1.0/26    (192.168.1.0   – 192.168.1.63)
# Subnet 2: 192.168.1.64/26   (192.168.1.64  – 192.168.1.127)
# Subnet 3: 192.168.1.128/26  (192.168.1.128 – 192.168.1.191)
# Subnet 4: 192.168.1.192/26  (192.168.1.192 – 192.168.1.255)

# Splitting 192.168.1.0/24 into /27s (8 subnets, 30 hosts each):
# Subnet 1: 192.168.1.0/27    (192.168.1.0   – 192.168.1.31)
# Subnet 2: 192.168.1.32/27   (192.168.1.32  – 192.168.1.63)
# Subnet 3: 192.168.1.64/27   (192.168.1.64  – 192.168.1.95)
# Subnet 4: 192.168.1.96/27   (192.168.1.96  – 192.168.1.127)
# Subnet 5: 192.168.1.128/27  (192.168.1.128 – 192.168.1.159)
# Subnet 6: 192.168.1.160/27  (192.168.1.160 – 192.168.1.191)
# Subnet 7: 192.168.1.192/27  (192.168.1.192 – 192.168.1.223)
# Subnet 8: 192.168.1.224/27  (192.168.1.224 – 192.168.1.255)

# Splitting 192.168.1.0/24 into /28s (16 subnets, 14 hosts each):
# Subnet 1:  192.168.1.0/28    (192.168.1.0   – 192.168.1.15)
# Subnet 2:  192.168.1.16/28   (192.168.1.16  – 192.168.1.31)
# Subnet 3:  192.168.1.32/28   (192.168.1.32  – 192.168.1.47)
# ...
# Subnet 16: 192.168.1.240/28  (192.168.1.240 – 192.168.1.255)
```

## IPv6 Subnetting

### Standard Prefix Lengths

```bash
# IPv6 subnetting is simpler — no broadcast, no "subtract 2" rule
# Standard allocation hierarchy:
#
# /32  — ISP allocation from RIR
# /48  — site allocation (65,536 /64 subnets)
# /56  — home/small office (256 /64 subnets)
# /64  — single subnet (standard for all LANs, required for SLAAC)
# /128 — single host (loopback, point-to-point)

# A /48 gives you 16 bits for subnetting:
# 2001:db8:abcd:0000::/64  through  2001:db8:abcd:ffff::/64
# = 65,536 subnets, each with 2^64 host addresses

# Nibble boundaries (multiples of 4) are easiest to manage:
# /48, /52, /56, /60, /64 — each hex digit = 4 bits
```

### IPv6 Subnetting Example

```bash
# Given: 2001:db8:cafe::/48 — allocate subnets for an organization

# Building A: 2001:db8:cafe:0100::/56  (256 /64 subnets: 0100–01ff)
# Building B: 2001:db8:cafe:0200::/56  (256 /64 subnets: 0200–02ff)
# Data Center: 2001:db8:cafe:1000::/52 (4096 /64 subnets: 1000–1fff)
# Management:  2001:db8:cafe:0001::/64 (single /64)
# Point-to-point links: 2001:db8:cafe:ff00::/56 (use /127 per link, RFC 6164)

# Unlike IPv4, no need to conserve addresses — use /64 for every subnet
# Never use anything longer than /64 for a regular LAN (breaks SLAAC)
```

## Calculation Tools

### ipcalc

```bash
# Install
sudo apt install ipcalc       # Debian/Ubuntu
sudo yum install ipcalc       # RHEL/CentOS

# Calculate subnet details
ipcalc 192.168.1.0/24
# Address:   192.168.1.0
# Netmask:   255.255.255.0 = 24
# Network:   192.168.1.0/24
# Broadcast: 192.168.1.255
# HostMin:   192.168.1.1
# HostMax:   192.168.1.254
# Hosts/Net: 254

# Split a network into subnets
ipcalc 192.168.1.0/24 -s 100 50 25
# Allocates subnets for 100, 50, and 25 hosts
```

### sipcalc

```bash
# Install
sudo apt install sipcalc       # Debian/Ubuntu

# IPv4 calculation
sipcalc 192.168.1.75/26
# Outputs: network, broadcast, usable range, netmask in multiple formats

# IPv6 calculation
sipcalc 2001:db8:abcd::/48
# Shows expanded address, prefix, network range

# Split a network
sipcalc 192.168.1.0/24 --split 26
# Shows all /26 subnets within the /24

# Wildcard mask (useful for ACLs)
sipcalc 192.168.1.0/24
# Includes wildcard: 0.0.0.255
```

### Quick CLI Calculations

```bash
# Convert prefix to netmask with Python one-liner
python3 -c "import ipaddress; print(ipaddress.IPv4Network('0.0.0.0/24', strict=False).netmask)"
# 255.255.255.0

# Get network details
python3 -c "
import ipaddress
n = ipaddress.IPv4Network('192.168.1.75/26', strict=False)
print(f'Network:   {n.network_address}')
print(f'Broadcast: {n.broadcast_address}')
print(f'Netmask:   {n.netmask}')
print(f'Hosts:     {n.num_addresses - 2}')
print(f'Range:     {list(n.hosts())[0]} – {list(n.hosts())[-1]}')
"

# Check if an IP is in a subnet
python3 -c "
import ipaddress
print(ipaddress.ip_address('192.168.1.75') in ipaddress.ip_network('192.168.1.64/26'))
"
# True
```

## Tips

- Always allocate largest subnets first when doing VLSM to avoid fragmentation and wasted address space.
- For point-to-point links, use /30 (IPv4) or /127 (IPv6, RFC 6164) to conserve addresses.
- The "magic number" shortcut: subtract the interesting octet of the mask from 256 to get the block size. For /26 (mask 192), block size = 256 - 192 = 64.
- In IPv6, always use /64 for LAN subnets. Using longer prefixes breaks SLAAC and violates RFC recommendations.
- Remember that /31 is valid for point-to-point links per RFC 3021 (no network or broadcast address needed).
- When planning subnets, document your allocation in a spreadsheet or IPAM tool. Address conflicts are the most common subnetting mistake.
- The 172.16.0.0/12 range spans 172.16.0.0 through 172.31.255.255 (not 172.16 through 172.32). This trips people up frequently.
- Use `ipcalc -s` to let the tool calculate optimal VLSM allocation for you when given host counts.
- For quick mental math: /24 = 256, /25 = 128, /26 = 64, /27 = 32, /28 = 16, /29 = 8, /30 = 4. Each step doubles or halves.

## References

- [RFC 4632 — Classless Inter-domain Routing (CIDR): The Internet Address Assignment and Aggregation Plan](https://www.rfc-editor.org/rfc/rfc4632)
- [RFC 1918 — Address Allocation for Private Internets](https://www.rfc-editor.org/rfc/rfc1918)
- [RFC 6890 — Special-Purpose IP Address Registries](https://www.rfc-editor.org/rfc/rfc6890)
- [RFC 950 — Internet Standard Subnetting Procedure](https://www.rfc-editor.org/rfc/rfc950)
- [RFC 1519 — CIDR: An Address Assignment and Aggregation Strategy](https://www.rfc-editor.org/rfc/rfc1519)
- [IANA IPv4 Special-Purpose Address Registry](https://www.iana.org/assignments/iana-ipv4-special-registry/iana-ipv4-special-registry.xhtml)
- [IANA IPv6 Special-Purpose Address Registry](https://www.iana.org/assignments/iana-ipv6-special-registry/iana-ipv6-special-registry.xhtml)
- [man ipcalc](https://man7.org/linux/man-pages/man1/ipcalc.1.html)
