# How Networking Works (From Tin Cans to TCP/IP)

A tiered guide to computer networking fundamentals.

## ELI5

### Sending Letters

Computers talk to each other like pen pals sending letters through the mail.

Every computer has an **address** (like a home address). When your computer wants
to talk to another computer, it writes a message, puts it in a **digital envelope**
called a **packet**, writes the address on the outside, and drops it in the
mailbox.

The postal system (the **network**) reads the address and delivers the envelope.
The receiving computer opens it and reads the message inside.

### Breaking Up Big Messages

You cannot send a whole book in one envelope. It would be too big. Instead, you
tear the book into small pieces, put each piece in its own envelope, number them
("page 1 of 12," "page 2 of 12"), and send them all separately.

When your computer sends a web page, a video, or an email, it does the same
thing. It breaks the message into small pieces and puts each piece in its own
packet. The other computer collects all the packets and puts the pieces back
together in order.

### What Is on the Envelope

Just like a real envelope has a "From" address and a "To" address, every packet
has information written on the outside:

```
+------------------------------------------+
|         THE OUTSIDE (HEADER)             |
|                                          |
|  From: 192.168.1.5  (your computer)     |
|  To:   142.250.80.4 (Google's computer) |
|  Type: TCP  (reliable delivery)          |
|  Port: 443  (the "mailbox slot")         |
|                                          |
+------------------------------------------+
```

The network does not care what is inside the envelope. It just reads the address
on the outside and delivers it -- like the post office delivering a birthday card
or a tax return without opening either one.

### Post Offices Along the Way

Your letter almost never goes directly from your house to the destination. It
stops at several post offices along the way. Each post office reads the address
and decides which direction to send it next.

```
Your          Post        Post        Post        Google's
Computer  --> Office  --> Office  --> Office  --> Computer
              (Home)      (City)      (Google)
```

In networking, these post offices are called **routers**. The packet might pass
through 10 or 15 routers before it arrives.

### Two Kinds of Mail

There are two ways to send packets:

- **TCP** -- like registered mail. You get a confirmation that it was delivered.
  If it gets lost, it gets re-sent. Slower but reliable.
- **UDP** -- like dropping a postcard in a mailbox. You hope it arrives, but you
  do not get a confirmation. Faster but less reliable.

---

## Middle School

### IP Addresses -- Phone Numbers for Computers

Every device on a network has an **IP address**. There are two versions:

**IPv4** -- four numbers (0-255) separated by dots:
```
192.168.1.100
```

**IPv6** -- eight groups of hexadecimal digits separated by colons:
```
fd00:3f:75:1::1
```

IPv4 supports about 4.3 billion addresses (we ran out). IPv6 supports 340
undecillion (a 34 followed by 37 zeros).

### Private vs. Public Addresses

Some address ranges are reserved for internal networks:

```
Private IPv4: 10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16
Private IPv6: fd00::/8
```

Your home router has one public IP from your ISP. All your devices share it
using NAT (Network Address Translation).

### Ports -- Apartment Numbers

If an IP address is a street address, a **port** is an apartment number. One
computer runs many programs, and each listens on a different port (0-65535):

| Port | Service | What It Does |
|------|---------|-------------|
| 22 | SSH | Remote terminal access |
| 53 | DNS | Translates domain names to IP addresses |
| 80 | HTTP | Unencrypted web traffic |
| 443 | HTTPS | Encrypted web traffic |
| 5432 | PostgreSQL | Database queries |

### Routers -- Mail Sorters

A **router** connects different networks. It reads the destination IP address on
each packet and decides where to send it next, using a **routing table** (a list
of "if the destination is in network X, send it through interface Y").

### WiFi vs. Ethernet

- **Ethernet** (wired): faster, more reliable, lower latency
- **WiFi** (wireless): convenient, shared medium, signal degrades with distance

Both deliver packets the same way. The difference is the physical medium.

### DNS -- The Phone Book

**DNS** (Domain Name System) translates human-readable names to IP addresses:

```
You type:    google.com
DNS returns: 142.250.80.4
Browser connects to: 142.250.80.4:443
```

Without DNS you would have to memorize numeric addresses for every website.

### What Happens When You Visit a Website

```
1. You type "example.com" in your browser
2. Browser asks DNS: "What is example.com's IP?"    --> 93.184.216.34
3. Browser opens a TCP connection to 93.184.216.34:443 (three-way handshake)
4. Browser sends: "GET / HTTP/1.1"
5. Server responds: "200 OK" + the HTML page
6. Browser renders the page
7. Connection closes
```

---

## High School

### The OSI Model (7 Layers)

The OSI model describes networking in layers. Each layer handles one concern:

| Layer | Name | What It Does | Example |
|:---:|:---|:---|:---|
| 7 | Application | User-facing protocols | HTTP, SMTP, DNS |
| 6 | Presentation | Data format, encryption | TLS, JPEG, UTF-8 |
| 5 | Session | Connection management | RPC sessions, NetBIOS |
| 4 | Transport | Reliable delivery, flow control | TCP, UDP |
| 3 | Network | Routing between networks | IP, ICMP |
| 2 | Data Link | Local delivery on a segment | Ethernet, WiFi (802.11) |
| 1 | Physical | Bits on the wire | Copper, fiber, radio |

Data flows down the stack on the sender (each layer adds a header), across the
wire, and back up the stack on the receiver (each layer strips its header).

### TCP vs. UDP in Detail

**TCP** (Transmission Control Protocol):
```
Client                    Server
  |--- SYN ----------------->|   "I want to connect"
  |<-- SYN+ACK -------------|   "OK, I am ready"
  |--- ACK ----------------->|   "Connection open"
  |=== DATA ================|   Reliable, ordered delivery
  |--- FIN ----------------->|   "I am done"
  |<-- ACK -----------------|   "Acknowledged"
```

TCP guarantees: ordered delivery, retransmission of lost packets, flow control,
congestion control.

**UDP** (User Datagram Protocol): no handshake, no ordering, no retransmission.
Used when speed > reliability (DNS, video, gaming).

### MAC Addresses and ARP

Every network interface has a **MAC address** (48-bit hardware address):
```
a4:5e:60:b8:12:3f
```

**ARP** (Address Resolution Protocol) maps IP addresses to MAC addresses on the
local network:
```
"Who has 192.168.1.1? Tell 192.168.1.5"
"192.168.1.1 is at a4:5e:60:b8:12:3f"
```

### Subnets and CIDR

**CIDR notation** defines which bits of an IP address identify the network vs.
the host:

```
192.168.1.0/24
^^^^^^^^^ ^^^
network    24 bits are the network portion
           remaining 8 bits = 256 addresses (254 usable)
```

| CIDR | Subnet Mask | Hosts |
|:---|:---|:---:|
| /8 | 255.0.0.0 | 16,777,214 |
| /16 | 255.255.0.0 | 65,534 |
| /24 | 255.255.255.0 | 254 |
| /28 | 255.255.255.240 | 14 |
| /32 | 255.255.255.255 | 1 |

### NAT (Network Address Translation)

NAT allows many devices to share one public IP. Your home router keeps a
translation table:

```
Internal: 192.168.1.5:54321  <-->  External: 203.0.113.1:40001
Internal: 192.168.1.6:54322  <-->  External: 203.0.113.1:40002
```

### DHCP

**DHCP** (Dynamic Host Configuration Protocol) automatically assigns IP
addresses to devices joining a network:

```
Device:  "I need an IP" (DHCPDISCOVER)
Server:  "How about 192.168.1.50?" (DHCPOFFER)
Device:  "I will take it" (DHCPREQUEST)
Server:  "It is yours for 24 hours" (DHCPACK)
```

### HTTP Request/Response Cycle

```
GET /api/users HTTP/1.1          <-- Method, Path, Version
Host: api.example.com            <-- Headers
Accept: application/json
Authorization: Bearer token123

---

HTTP/1.1 200 OK                  <-- Status Code
Content-Type: application/json   <-- Response Headers
Content-Length: 42

{"users": [{"id": 1, "name": "Alice"}]}   <-- Body
```

### Traceroute

Shows every router hop between you and a destination:

```bash
traceroute example.com
# 1  192.168.1.1      1.2 ms    (your router)
# 2  10.0.0.1         5.4 ms    (ISP)
# 3  72.14.233.105    12.1 ms   (backbone)
# 4  93.184.216.34    18.3 ms   (destination)
```

---

## College

### TCP State Machine

TCP connections move through well-defined states:

```
CLOSED --> LISTEN --> SYN_RECEIVED --> ESTABLISHED
CLOSED --> SYN_SENT --> ESTABLISHED
ESTABLISHED --> FIN_WAIT_1 --> FIN_WAIT_2 --> TIME_WAIT --> CLOSED
ESTABLISHED --> CLOSE_WAIT --> LAST_ACK --> CLOSED
```

**TIME_WAIT** lasts 2 x MSL (Maximum Segment Lifetime, typically 60 seconds).
This prevents delayed packets from a previous connection being misinterpreted.

### Congestion Control

TCP throttles its send rate to avoid overwhelming the network:

**Slow Start**: start with a small congestion window (cwnd), double it each RTT
until packet loss occurs.

**AIMD** (Additive Increase, Multiplicative Decrease): after slow start, increase
cwnd by 1 MSS per RTT (additive). On loss, halve cwnd (multiplicative).

```
cwnd
 ^
 |        /\      /\
 |       /  \    /  \
 |      /    \  /    \
 |     /      \/      \
 |    /                 \
 |   / (slow start)
 |  /
 | /
 +--------------------------> time
   loss    loss
```

Modern variants: **CUBIC** (Linux default -- cubic function instead of linear),
**BBR** (Google -- models bottleneck bandwidth and RTT).

### BGP and AS Numbers

**BGP** (Border Gateway Protocol) is the routing protocol of the internet. It
connects **Autonomous Systems** (AS) -- independently operated networks (ISPs,
cloud providers, enterprises).

Each AS has a unique **ASN** (e.g., AS13335 = Cloudflare, AS15169 = Google).

BGP routers exchange **path vectors** -- lists of AS numbers to reach a
destination prefix. Policy decides which path to prefer.

### MPLS

**MPLS** (Multiprotocol Label Switching) inserts a 32-bit label between L2 and
L3 headers. Routers forward based on the label (fast table lookup) instead of
the full IP address (longest prefix match).

```
[Ethernet] [MPLS Label: 42] [IP Header] [TCP] [Payload]
```

Label operations: **push** (add label), **swap** (change label), **pop** (remove
label at egress).

### VXLAN Overlays

**VXLAN** (Virtual Extensible LAN) encapsulates L2 Ethernet frames inside UDP
packets, creating virtual L2 networks over L3 infrastructure:

```
[Outer Ethernet] [Outer IP] [UDP:4789] [VXLAN Header (VNI)] [Inner Ethernet] [Inner IP] [Payload]
```

24-bit VNI (VXLAN Network Identifier) supports ~16 million virtual networks
(vs. 4,096 VLANs).

### SDN (Software-Defined Networking)

Separates the **control plane** (deciding where traffic goes) from the **data
plane** (actually forwarding packets). A centralized controller programs flow
rules into switches via protocols like OpenFlow.

### Network Namespaces

Linux kernel feature that gives processes isolated network stacks (interfaces,
routes, iptables rules). Foundation of container networking:

```bash
ip netns add red
ip netns exec red ip addr add 10.0.0.1/24 dev veth0
ip netns exec red ip link set veth0 up
```

### Socket Programming

```c
// TCP Server (C)
int sockfd = socket(AF_INET, SOCK_STREAM, 0);
bind(sockfd, &addr, sizeof(addr));
listen(sockfd, SOMAXCONN);
int client = accept(sockfd, NULL, NULL);
read(client, buf, sizeof(buf));
write(client, response, strlen(response));
close(client);
```

```go
// TCP Server (Go)
ln, _ := net.Listen("tcp", ":8080")
conn, _ := ln.Accept()
io.Copy(conn, conn)  // echo server
```

### High-Performance Network I/O

**epoll** (Linux): event-driven I/O multiplexing for thousands of concurrent
connections. Register file descriptors, get notified when they are ready.

**io_uring** (Linux 5.1+): submission queue + completion queue in shared memory.
Zero-copy, zero-syscall batched I/O. Successor to epoll for high-throughput
network servers.

```
User Space         Kernel
+------+          +------+
| SQ   | -------> | CQ   |
| push |          | poll |
+------+          +------+
   Submission        Completion
```

---

## Tips

- Use `ping` to test basic connectivity, `traceroute` to diagnose routing issues
- `ss -tlnp` shows listening TCP ports (faster than `netstat`)
- `tcpdump -i any -nn port 443` captures HTTPS traffic for debugging
- `dig +short example.com` for quick DNS lookups
- Start learning with the TCP/IP model (4 layers) before the OSI model (7 layers)
- Every protocol adds headers -- understand encapsulation and you understand networking
- When debugging, work bottom-up: physical link, then IP, then transport, then application

## See Also

- tcp
- udp
- dns
- http
- subnetting
- arp
- bgp
- nat
- dhcp
- traceroute
- network-namespaces
- vxlan
- mpls
- io-uring
- ipv4
- ipv6

## References

- RFC 791 (IPv4), RFC 8200 (IPv6)
- RFC 793 (TCP), RFC 768 (UDP)
- RFC 1034/1035 (DNS)
- RFC 2616 (HTTP/1.1), RFC 9110 (HTTP Semantics)
- RFC 4271 (BGP-4)
- RFC 7348 (VXLAN)
- Tanenbaum, "Computer Networks" (6th ed.)
- Stevens, "TCP/IP Illustrated, Volume 1"
- Kurose & Ross, "Computer Networking: A Top-Down Approach"
