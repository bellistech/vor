# IPv4 (Internet Protocol version 4)

Layer 3 protocol providing logical addressing and best-effort packet delivery across interconnected networks.

## Header Format

```
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|Version|  IHL  |    DSCP   |ECN|         Total Length          |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|         Identification        |Flags|    Fragment Offset      |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|  Time to Live |    Protocol   |        Header Checksum        |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                       Source Address                          |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                    Destination Address                        |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                    Options (if IHL > 5)                       |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

- **Version**: Always 4 for IPv4
- **IHL**: Internet Header Length in 32-bit words (min 5 = 20 bytes, max 15 = 60 bytes)
- **DSCP**: Differentiated Services Code Point (6 bits, QoS marking)
- **ECN**: Explicit Congestion Notification (2 bits)
- **Total Length**: Entire packet size in bytes (header + data, max 65535)
- **Identification**: Unique fragment group identifier
- **Flags**: 3 bits (bit 0 reserved, bit 1 DF, bit 2 MF)
- **Fragment Offset**: Position of fragment in original datagram (units of 8 bytes)
- **TTL**: Hop limit, decremented by each router
- **Protocol**: Upper layer protocol number
- **Header Checksum**: Ones complement checksum of header only (recomputed at each hop)

## Addressing

### Address Classes (Classful — Historical)

```
Class A:   0.0.0.0   – 127.255.255.255   /8    (leading bit 0)
Class B:   128.0.0.0 – 191.255.255.255   /16   (leading bits 10)
Class C:   192.0.0.0 – 223.255.255.255   /24   (leading bits 110)
Class D:   224.0.0.0 – 239.255.255.255          (multicast, leading bits 1110)
Class E:   240.0.0.0 – 255.255.255.255          (reserved/experimental, leading bits 1111)
```

### CIDR Notation

```
# Network/prefix notation replaces classful addressing
192.168.1.0/24      # 256 addresses (254 usable hosts)
10.0.0.0/8          # 16,777,216 addresses
172.16.0.0/12       # 1,048,576 addresses

# Subnet mask to prefix length
255.255.255.0   = /24   (256 hosts)
255.255.255.128 = /25   (128 hosts)
255.255.255.192 = /26   (64 hosts)
255.255.255.224 = /27   (32 hosts)
255.255.255.240 = /28   (16 hosts)
255.255.255.248 = /29   (8 hosts)
255.255.255.252 = /30   (4 hosts, 2 usable — point-to-point links)
255.255.255.254 = /31   (2 hosts, point-to-point per RFC 3021)
255.255.255.255 = /32   (single host)
```

### Private Ranges (RFC 1918)

```
10.0.0.0/8          # 10.0.0.0 – 10.255.255.255      (Class A block)
172.16.0.0/12       # 172.16.0.0 – 172.31.255.255     (Class B block)
192.168.0.0/16      # 192.168.0.0 – 192.168.255.255   (Class C block)
```

### Special Address Ranges

```
127.0.0.0/8         # Loopback (typically 127.0.0.1)
169.254.0.0/16      # Link-local / APIPA (auto-assigned when no DHCP)
0.0.0.0/8           # "This network" (0.0.0.0 = default route / unspecified)
224.0.0.0/4         # Multicast (224.0.0.0 – 239.255.255.255)
240.0.0.0/4         # Reserved / future use (240.0.0.0 – 255.255.255.254)
255.255.255.255     # Limited broadcast (never forwarded by routers)
100.64.0.0/10       # Shared address space / CGN (RFC 6598)
192.0.2.0/24        # Documentation — TEST-NET-1 (RFC 5737)
198.51.100.0/24     # Documentation — TEST-NET-2 (RFC 5737)
203.0.113.0/24      # Documentation — TEST-NET-3 (RFC 5737)
```

## Protocol Numbers

```
 1  ICMP    — Internet Control Message Protocol
 2  IGMP    — Internet Group Management Protocol
 6  TCP     — Transmission Control Protocol
17  UDP     — User Datagram Protocol
47  GRE     — Generic Routing Encapsulation
50  ESP     — Encapsulating Security Payload (IPsec)
51  AH      — Authentication Header (IPsec)
58  ICMPv6  — ICMP for IPv6
89  OSPF    — Open Shortest Path First
132 SCTP    — Stream Control Transmission Protocol
```

## Fragmentation

### Key Concepts

```
MTU                 # Maximum Transmission Unit — largest packet a link can carry
                    # Ethernet default: 1500 bytes
DF bit (Don't Fragment)  # If set, router drops packet and sends ICMP Type 3 Code 4
MF bit (More Fragments)  # Set on all fragments except the last
Fragment Offset     # Position in original datagram, in units of 8 bytes
```

### Path MTU Discovery

```bash
# Path MTU discovery sends packets with DF bit set
# Routers return ICMP "fragmentation needed" (Type 3, Code 4) with next-hop MTU
# Linux enables PMTUD by default

# Check current PMTU to a destination
ip route get 8.8.8.8               # shows "mtu" in output if cached

# Manually probe path MTU
ping -M do -s 1472 8.8.8.8        # -M do sets DF bit, -s 1472 + 28 byte header = 1500

# Disable/enable PMTUD
sysctl net.ipv4.ip_no_pmtu_disc=0  # 0=enabled (default), 1=disabled
```

## DSCP / ToS

```
# DSCP values (6-bit field, decimal)
 0  CS0  / Best Effort (default)
 8  CS1  / Scavenger
10  AF11 — Assured Forwarding (low drop)
12  AF12 — Assured Forwarding (medium drop)
14  AF13 — Assured Forwarding (high drop)
16  CS2
18  AF21
20  AF22
22  AF23
24  CS3
26  AF31
28  AF32
30  AF33
32  CS4
34  AF41
36  AF42
38  AF43
40  CS5  / Signaling
46  EF   / Expedited Forwarding (voice/real-time)
48  CS6  / Network control
56  CS7  / Reserved
```

## Linux Configuration

```bash
# Show IP addresses
ip addr show                        # all interfaces
ip -4 addr show eth0                # IPv4 only on eth0

# Add/remove addresses
ip addr add 192.168.1.10/24 dev eth0
ip addr del 192.168.1.10/24 dev eth0

# Routing
ip route show                       # display routing table
ip route add 10.0.0.0/8 via 192.168.1.1 dev eth0
ip route add default via 192.168.1.1
ip route del 10.0.0.0/8

# Enable IP forwarding (router mode)
sysctl -w net.ipv4.ip_forward=1               # temporary
echo "net.ipv4.ip_forward = 1" >> /etc/sysctl.conf  # persistent

# Key /proc/sys/net/ipv4/ tunables
cat /proc/sys/net/ipv4/ip_forward              # forwarding status
cat /proc/sys/net/ipv4/ip_default_ttl          # default TTL (64)
cat /proc/sys/net/ipv4/ip_local_port_range     # ephemeral port range
cat /proc/sys/net/ipv4/icmp_echo_ignore_all    # ignore pings (0=no, 1=yes)
cat /proc/sys/net/ipv4/conf/all/rp_filter      # reverse path filtering
cat /proc/sys/net/ipv4/conf/all/accept_redirects
```

## ICMP

### Common Types

```
Type  Code  Description
─────────────────────────────────────────────────
  0    0    Echo Reply (ping response)
  3    0    Destination Unreachable — network unreachable
  3    1    Destination Unreachable — host unreachable
  3    3    Destination Unreachable — port unreachable
  3    4    Destination Unreachable — fragmentation needed (PMTUD)
  3   13    Destination Unreachable — administratively prohibited
  5    0    Redirect — redirect for network
  5    1    Redirect — redirect for host
  8    0    Echo Request (ping)
 11    0    Time Exceeded — TTL expired in transit (traceroute)
 11    1    Time Exceeded — fragment reassembly time exceeded
```

### ICMP Tools

```bash
ping -c 4 192.168.1.1              # send 4 echo requests
ping -I eth0 192.168.1.1           # specify source interface
ping -W 2 192.168.1.1              # 2 second timeout

traceroute 8.8.8.8                 # UDP-based by default on Linux
traceroute -I 8.8.8.8              # ICMP-based traceroute
traceroute -T 8.8.8.8              # TCP-based traceroute (port 80)
mtr 8.8.8.8                        # real-time combined ping + traceroute
```

## ARP (Address Resolution Protocol)

```bash
# View ARP cache
arp -a                              # traditional command
ip neigh show                       # modern iproute2 equivalent

# Add/delete static ARP entries
ip neigh add 192.168.1.1 lladdr aa:bb:cc:dd:ee:ff dev eth0
ip neigh del 192.168.1.1 dev eth0

# Flush ARP cache
ip neigh flush dev eth0

# Gratuitous ARP — announces IP-to-MAC mapping to all hosts
#   Used for: failover, duplicate IP detection, updating caches after MAC change
arping -U -I eth0 192.168.1.10     # unsolicited ARP reply (gratuitous)
arping -A -I eth0 192.168.1.10     # ARP announcement

# Proxy ARP — router answers ARP on behalf of another subnet
sysctl -w net.ipv4.conf.eth0.proxy_arp=1
```

## Troubleshooting

```bash
# Basic connectivity
ping -c 3 192.168.1.1              # test reachability
ping -c 3 8.8.8.8                  # test internet connectivity
ping -c 3 google.com               # test DNS + connectivity

# Path analysis
traceroute -n 8.8.8.8              # numeric output, no DNS lookups
mtr --report 8.8.8.8               # statistical report after 10 probes

# MTU issues — symptoms: small packets work, large fail; SSH works, SCP hangs
ping -M do -s 1472 192.168.1.1    # test with 1500 byte packets (1472 + 28 header)
ip route add 10.0.0.0/8 via 192.168.1.1 mtu 1400  # force lower MTU for a route

# Fragmentation analysis
tcpdump -i eth0 'ip[6:2] & 0x3fff != 0'  # capture fragmented packets

# ARP issues — symptoms: "no route to host" on local subnet
ip neigh show                       # check for FAILED or INCOMPLETE entries
arping -c 3 192.168.1.1            # test ARP resolution directly

# Check for duplicate IPs
arping -D -I eth0 192.168.1.10     # DAD (Duplicate Address Detection)
```

## Tips

- The minimum IPv4 header is 20 bytes (IHL=5). Options can extend it to 60 bytes, but options are rarely used in practice and some routers slow-path packets with options.
- TTL of 64 is the Linux default, 128 is Windows, 255 is network gear. You can fingerprint OS by observed TTL in ping replies.
- When PMTUD blackholes occur (ICMP blocked by firewalls), TCP connections hang on large transfers. Fix by either allowing ICMP Type 3 Code 4 or clamping MSS: `iptables -t mangle -A FORWARD -p tcp --tcp-flags SYN,RST SYN -j TCPMSS --clamp-mss-to-pmtu`.
- The DF bit is set by default on Linux for TCP. UDP does not set DF by default, so large UDP datagrams get fragmented.
- Gratuitous ARP has no authentication, making ARP spoofing trivial on local networks. Use Dynamic ARP Inspection (DAI) on managed switches or static ARP entries for critical hosts.
- RFC 6598 (100.64.0.0/10) is for carrier-grade NAT. Do not use it for private addressing in your infrastructure; use RFC 1918 ranges instead.
- `/proc/sys/net/ipv4/conf/all/rp_filter` set to 1 (strict mode) drops packets arriving on unexpected interfaces, preventing IP spoofing but breaking asymmetric routing. Set to 2 (loose mode) if you have multiple paths.
- When debugging MTU issues, remember that ICMP adds 8 bytes of header and IP adds 20, so `ping -s 1472` tests a 1500-byte packet (1472 + 8 ICMP + 20 IP).

## See Also

- ipv6, subnetting, ipsec, ip, iptables

## References

- [RFC 791 — Internet Protocol (IPv4)](https://www.rfc-editor.org/rfc/rfc791)
- [RFC 1918 — Address Allocation for Private Internets](https://www.rfc-editor.org/rfc/rfc1918)
- [RFC 4632 — Classless Inter-domain Routing (CIDR)](https://www.rfc-editor.org/rfc/rfc4632)
- [RFC 6890 — Special-Purpose IP Address Registries](https://www.rfc-editor.org/rfc/rfc6890)
- [RFC 1122 — Requirements for Internet Hosts: Communication Layers](https://www.rfc-editor.org/rfc/rfc1122)
- [RFC 792 — Internet Control Message Protocol (ICMP)](https://www.rfc-editor.org/rfc/rfc792)
- [IANA IPv4 Special-Purpose Address Registry](https://www.iana.org/assignments/iana-ipv4-special-registry/iana-ipv4-special-registry.xhtml)
- [Linux Kernel — IP Sysctl Documentation](https://www.kernel.org/doc/html/latest/networking/ip-sysctl.html)
- [man ip-address](https://man7.org/linux/man-pages/man8/ip-address.8.html)
