# UDP (User Datagram Protocol)

Minimal, connectionless transport protocol providing fast, low-overhead datagram delivery with no guarantees on order, delivery, or duplication.

## Header Format

```
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|          Source Port          |       Destination Port        |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|            Length             |           Checksum            |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

- **Source Port**: Optional (set to 0 if unused), 16-bit
- **Destination Port**: Required, 16-bit
- **Length**: Total datagram length (header + data) in bytes, minimum 8
- **Checksum**: Covers pseudo-header (src/dst IP, protocol, length) + header + data. Optional in IPv4 (set to 0 to skip), mandatory in IPv6.

Total header size: 8 bytes (compare to TCP's 20-60 bytes).

## Characteristics

```
Connectionless      — No handshake, no state, no teardown
No ordering         — Datagrams may arrive out of order
No retransmission   — Lost packets are simply gone
No flow control     — Sender can overwhelm the receiver
No congestion ctrl  — Sender can overwhelm the network
Low overhead        — 8-byte header, no connection state
Message-oriented    — Preserves message boundaries (unlike TCP byte stream)
```

## Common UDP Services

```
Port    Service         Description
─────────────────────────────────────────────────────
  53    DNS             Domain Name System (queries and responses)
  67    DHCP server     Dynamic Host Configuration Protocol
  68    DHCP client     Client-side DHCP
  69    TFTP            Trivial File Transfer Protocol
 123    NTP             Network Time Protocol
 161    SNMP            Simple Network Management Protocol (queries)
 162    SNMP Trap       SNMP notifications
 443    QUIC/HTTP3      QUIC transport for HTTP/3
 514    Syslog          BSD syslog protocol
1194    OpenVPN         OpenVPN default (can also use TCP)
4789    VXLAN           Virtual Extensible LAN overlay
5353    mDNS            Multicast DNS (Bonjour, Avahi)
5355    LLMNR           Link-Local Multicast Name Resolution
51820   WireGuard       WireGuard VPN
```

```
# Protocols using UDP as transport
RTP/RTCP        — Real-time audio/video streaming
SIP             — Session Initiation Protocol (VoIP signaling)
QUIC            — Multiplexed encrypted transport (HTTP/3)
Game protocols  — Most real-time multiplayer games
```

## Linux Configuration

```bash
# UDP buffer sizes — controls kernel buffer for all UDP sockets
sysctl -w net.core.rmem_max=26214400       # max receive buffer (bytes)
sysctl -w net.core.wmem_max=26214400       # max send buffer (bytes)
sysctl -w net.core.rmem_default=1048576    # default receive buffer
sysctl -w net.core.wmem_default=1048576    # default send buffer

# UDP memory limits (pages): min, pressure, max
sysctl -w net.ipv4.udp_mem="1048576 2097152 4194304"

# Per-socket buffer (set in application code)
# setsockopt(fd, SOL_SOCKET, SO_RCVBUF, &size, sizeof(size));
# setsockopt(fd, SOL_SOCKET, SO_SNDBUF, &size, sizeof(size));
# SO_RCVBUF capped at rmem_max (doubled internally by kernel for bookkeeping)
```

## Monitoring

```bash
# Socket statistics
ss -u                              # all UDP sockets
ss -uln                            # listening UDP sockets, numeric
ss -ua                             # all UDP including non-listening
ss -unp                            # UDP sockets with process info

# Legacy netstat
netstat -uan                       # all UDP sockets, numeric

# Kernel UDP stats
cat /proc/net/udp                  # raw UDP socket table
cat /proc/net/snmp | grep Udp     # InDatagrams, NoPorts, InErrors, OutDatagrams, RcvbufErrors, SndbufErrors

# Watch for receive buffer overflows
nstat -az UdpRcvbufErrors          # packets dropped due to full receive buffer
nstat -az UdpInErrors              # total UDP input errors

# Packet capture
tcpdump -i eth0 udp                # all UDP traffic
tcpdump -i eth0 udp port 53       # DNS traffic only
tcpdump -i eth0 udp portrange 5000-5100  # port range
```

## Broadcast and Multicast

```bash
# Broadcast — send to all hosts on a subnet
# Destination: 255.255.255.255 (limited) or subnet broadcast (e.g., 192.168.1.255)
# Socket must set SO_BROADCAST option

# Multicast — send to a group of interested receivers
# Range: 224.0.0.0/4 (224.0.0.0 – 239.255.255.255)
# Application joins multicast group via IP_ADD_MEMBERSHIP

# Common multicast addresses
# 224.0.0.1   — All hosts on this subnet
# 224.0.0.2   — All routers on this subnet
# 224.0.0.251 — mDNS
# 239.x.x.x   — Administratively scoped (private use)

# Join a multicast group on an interface
ip maddr show                      # show multicast group memberships
socat UDP4-RECVFROM:5000,reuseaddr,ip-add-membership=239.1.1.1:eth0 -

# Enable/disable multicast routing
sysctl -w net.ipv4.conf.eth0.mc_forwarding=1
```

## UDP-Lite (RFC 3828)

```
# UDP-Lite allows partial checksum coverage
# Useful for codecs that tolerate bit errors (audio/video)
# Protocol number 136 (not 17)
# Replaces Length field with Checksum Coverage field
# Checksum coverage = 0 means checksum covers entire datagram
# Checksum coverage = 8 means checksum covers only the header
# Available in Linux via IPPROTO_UDPLITE (136)
```

## Reliability Over UDP

```
# Applications needing reliability on UDP implement it themselves:

# Sequence numbers    — Detect reordering and duplicates
# Acknowledgments     — Confirm receipt of specific datagrams
# Retransmission      — Resend unacknowledged datagrams after timeout
# Flow control        — Application-level windowing

# Examples:
# QUIC         — Full reliable stream multiplexing over UDP
# TFTP         — Simple stop-and-wait ARQ (1 packet at a time)
# Game netcode — Selective reliability (reliable for state, unreliable for position)
# RTP/RTCP     — RTCP provides feedback, RTP carries sequence numbers
```

## Testing

```bash
# Netcat (nc) — quick UDP send/receive
nc -ul 5000                        # listen on UDP port 5000
nc -u 192.168.1.1 5000             # send to UDP port 5000
echo "test" | nc -u -w1 192.168.1.1 5000  # send one-shot, 1s timeout

# Socat — more versatile
socat UDP-LISTEN:5000,reuseaddr -              # listen
socat - UDP:192.168.1.1:5000                   # send
socat UDP-LISTEN:5000,fork UDP:10.0.0.1:5000   # UDP relay/proxy

# Ncat (from nmap)
ncat --udp -l 5000                 # listen on UDP 5000
ncat --udp 192.168.1.1 5000        # connect to UDP 5000

# iperf3 — UDP throughput testing
iperf3 -s                          # server
iperf3 -c 192.168.1.1 -u -b 100M  # client: UDP mode, 100 Mbps target
iperf3 -c 192.168.1.1 -u -b 0     # unlimited bandwidth test (careful!)
```

## Common Issues

### Packet Loss

```bash
# Symptoms: missing data, gaps in sequence numbers, application timeouts
# Check kernel drop stats
nstat -az UdpRcvbufErrors          # receive buffer overflows (most common cause)
nstat -az UdpInErrors              # checksum errors, header errors

# Increase receive buffer
sysctl -w net.core.rmem_max=26214400
# Application must also set SO_RCVBUF to match
```

### Buffer Overflow

```bash
# Symptoms: UdpRcvbufErrors incrementing, no errors on wire
# Cause: application not reading fast enough, burst traffic

# Check current socket buffer size
ss -ulnm                           # shows Recv-Q and buffer limits

# Mitigations
# 1. Increase buffer: SO_RCVBUF + rmem_max
# 2. Use recvmmsg() for batch reads
# 3. Multiple receiver threads
# 4. Consider SO_REUSEPORT for kernel-level load balancing across sockets
```

### MTU and Fragmentation

```bash
# UDP datagrams > MTU get fragmented at IP layer (DF bit not set by default)
# Max UDP payload without fragmentation on Ethernet: 1472 bytes (1500 - 20 IP - 8 UDP)
# Fragmented UDP is reassembled at destination — if ANY fragment is lost, entire datagram is dropped

# Avoid fragmentation for reliability-sensitive applications
# Set DF bit via IP_DONTFRAG socket option, keep datagrams < path MTU
```

### Firewall Blocking

```bash
# UDP has no connection state — stateful firewalls track by timeout
# Default UDP timeout in conntrack: 30 seconds (vs 5 days for established TCP)

# Check conntrack settings
sysctl net.netfilter.nf_conntrack_udp_timeout           # unreplied
sysctl net.netfilter.nf_conntrack_udp_timeout_stream     # bidirectional

# Long-lived UDP flows (VPN, gaming) may need higher timeouts
sysctl -w net.netfilter.nf_conntrack_udp_timeout_stream=180
```

## Tips

- UDP checksum is optional in IPv4 but mandatory in IPv6. Always enable it; the performance savings of disabling it are negligible on modern hardware.
- `UdpRcvbufErrors` in `/proc/net/snmp` is the single most important counter for diagnosing UDP reliability issues. If it is incrementing, you are dropping packets in the kernel before your application ever sees them.
- `SO_REUSEPORT` (Linux 3.9+) allows multiple sockets to bind to the same port with kernel-level load balancing. This is the best way to scale UDP receive performance across CPU cores.
- Unlike TCP, UDP preserves message boundaries. A single `sendto()` of 1000 bytes results in a single `recvfrom()` of 1000 bytes (if it arrives). There is no partial read or stream reassembly.
- When testing with `nc -u`, remember that UDP is connectionless: the "connection" only exists as long as you keep the nc process running, and there is no indication if the remote side is not listening.
- For high-throughput UDP, use `sendmmsg()`/`recvmmsg()` to batch system calls. Also consider `SO_ZEROCOPY` (Linux 4.18+) for large sends to avoid kernel copies.
- The theoretical max UDP payload is 65507 bytes (65535 IP max - 20 IP header - 8 UDP header), but anything above ~1472 bytes on Ethernet will be fragmented. Stick to under the path MTU for reliability.

## See Also

- tcp, quic, dns, snmp, tcpdump

## References

- [RFC 768 — User Datagram Protocol](https://www.rfc-editor.org/rfc/rfc768)
- [RFC 8085 — UDP Usage Guidelines](https://www.rfc-editor.org/rfc/rfc8085)
- [RFC 6936 — Applicability Statement for the Use of IPv6 UDP Datagrams with Zero Checksums](https://www.rfc-editor.org/rfc/rfc6936)
- [man udp](https://man7.org/linux/man-pages/man7/udp.7.html)
- [Linux Kernel — UDP Sysctl Documentation](https://www.kernel.org/doc/html/latest/networking/ip-sysctl.html)
- [man socket](https://man7.org/linux/man-pages/man7/socket.7.html)
- [Cloudflare Blog — Everything You Ever Wanted to Know About UDP Sockets](https://blog.cloudflare.com/everything-you-ever-wanted-to-know-about-udp-sockets-but-were-afraid-to-ask-part-1/)
