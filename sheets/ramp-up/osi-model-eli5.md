# OSI Model — ELI5

> The OSI Model is a 7-floor office tower that turns "send a letter to grandma" into "drop a glowing pulse onto a glass fiber halfway across the planet."

## Prerequisites

(none — start here)

You do not need to know what a "network" is. You do not need to have ever written a line of code. You do not need to know what TCP, IP, Ethernet, or WiFi mean. By the end of this sheet you will know all seven layers of the OSI model in plain English, you will have typed real commands into a real terminal that show you each layer working, and you will know which layer to blame when your video call freezes.

If a word feels weird, look it up in the **Vocabulary** section near the bottom. Every weird word in this sheet is in that table with a one-line plain-English definition.

If you see a `$` at the start of a line in a code block, that means "type the rest of this line into your terminal." You do not type the `$`. The lines underneath that don't have a `$` are what your computer prints back at you. We call that **output**.

This sheet talks about networks, but it never assumes you know how networks work. It just says, "imagine the network is an office tower," and then we walk through the office tower one floor at a time.

## What Even Is a Layered Model

### The big idea

Imagine you are an office worker. Your job is to send a birthday card to your friend who lives in another country. You want to get the card to her by tomorrow. You sit down at your desk and write the card. Now what?

You do not personally fly the card across the ocean. You do not personally drive it to the airport. You do not personally know which roads to take. You do not personally know the exact street and house number on the other end. You just want the card to get there.

So what do you do? You walk down the hall and hand the card to the **mail clerk**. The mail clerk puts the card in an envelope, writes the address on the envelope, sticks a stamp on it, and puts it in a big bag. The mail clerk does not know how planes work either. The mail clerk hands the bag to a **shipping company**. The shipping company drives the bag to the airport. At the airport, the bag goes onto a plane. The plane lands in your friend's country. A truck picks up the bag. A local mail carrier walks down your friend's street and puts the card in her mailbox. She opens the card and reads it.

Look what happened. Each person in that chain did **one job** and then handed the work to the next person. You wrote the card. The mail clerk addressed it. The shipping company moved it. The airport handled the plane. Each one trusted the next one to do their part. None of them needed to know how the whole system worked. They just did their layer of the work and passed it on.

**That is what a layered model is.**

A layered model is just a way of breaking a big complicated job into smaller jobs that each layer is responsible for. The bottom layer does one thing. The next layer up does another thing. They each only have to talk to the layer above and below them. They do not have to know about the layers further away. This is called **separation of concerns**, and it is one of the most important ideas in all of computing.

### Why we use seven layers for networks

Sending data across the internet is just like sending that birthday card, except instead of paper, you are sending little electric pulses. Instead of a postman, you have routers and switches. Instead of an envelope, you have **headers**. And instead of one or two people in the chain, you have *seven*.

The seven layers are called the **OSI model**, which stands for **Open Systems Interconnection**. It was made by a group called the **International Organization for Standardization** (everyone calls it ISO) in **1984**. The official name is **ISO/IEC 7498**, but nobody calls it that. They just call it "the OSI model" or "the seven layers."

Here is the picture we are going to use the whole rest of this sheet:

```
+----------------------------------------------------+
|  Floor 7  |  APPLICATION   |  the user types here  |
+-----------+----------------+-----------------------+
|  Floor 6  |  PRESENTATION  |  language translator  |
+-----------+----------------+-----------------------+
|  Floor 5  |  SESSION       |  conversation desk    |
+-----------+----------------+-----------------------+
|  Floor 4  |  TRANSPORT     |  packaging & tracking |
+-----------+----------------+-----------------------+
|  Floor 3  |  NETWORK       |  postal route maps    |
+-----------+----------------+-----------------------+
|  Floor 2  |  DATA LINK     |  loading the truck    |
+-----------+----------------+-----------------------+
|  Floor 1  |  PHYSICAL      |  the actual road      |
+-----------+----------------+-----------------------+
```

Each floor only talks to the floor right above it and the floor right below it. Floor 7 hands its work to Floor 6, which hands it to Floor 5, all the way down to Floor 1. Floor 1 sends the bits across the wire (or the radio waves, or the fiber). On the other end, the bits go back **up** the floors, one at a time, until they reach Floor 7 on the other computer.

That trip down the floors and back up the other side is the entire story of how data moves on a network. We are going to look at each floor, one by one, in plain English.

### A vocabulary head start

A few quick words you will see over and over.

A **protocol** is a set of rules that two computers agree to follow. Think of it like a language. If two people both speak French, they can have a conversation. If one speaks French and the other speaks Korean, they cannot. Computers need protocols to talk.

A **PDU** is a "Protocol Data Unit." That is a fancy way of saying "the chunk of stuff that this layer is sending." Every layer has its own name for its chunk of stuff. We will list them all.

A **header** is a little label stuck on the front of a chunk of data that says "this chunk is for layer X." A **trailer** is a label stuck on the back. The actual data the user wants to send is called the **payload**. So a chunk of network data looks like this:

```
[ HEADER ][ PAYLOAD ][ TRAILER ]
```

Some layers add only a header. Some add a header and a trailer. Each layer wraps the chunk from the layer above. By the time your data goes out on the wire, it has been wrapped seven times like a Russian doll.

That wrapping is called **encapsulation**. The unwrapping on the other end is called **decapsulation**. We will walk through both.

## The Seven OSI Layers

We are going to go through every floor of the tower from the bottom up. For each layer we will say:

- What kind of chunk it sends (its **PDU**).
- What it does in plain English.
- What runs there (the protocols that live on this floor).
- A quick example you can see on your own computer.

### L1 Physical: bits on a wire (or fiber, or radio)

The very bottom floor.

**PDU name:** **bit** (a single 1 or 0).

**Plain English:** Floor 1 is the actual road. It is the physical thing that carries the data — a copper wire, a glass fiber, a radio wave through the air. Nothing else. It does not know what the data means. It does not know who sent it. It does not know where it is going. It just turns 1s and 0s into a real-world signal: a voltage on a wire, a flash of light in a fiber, a wiggle in a radio wave.

If Floor 1 is broken, nothing else works. If you pull the cable out of your laptop, every other floor falls apart, because there is no road for them to drive on.

**What runs there:**

- **Ethernet PHY** — the chip on your network card that turns 1s and 0s into electrical signals on the cable. PHY is short for "physical."
- **100BASE-TX** — old "Fast Ethernet," 100 megabits per second, runs on copper twisted pair (Category 5 cable). Still very common.
- **1000BASE-T** — Gigabit Ethernet over copper, 1 gigabit per second. The most common home/office cable today (Cat5e or Cat6).
- **10GBASE-SR** — 10 gigabit Ethernet over short-range multimode fiber. "SR" means "short reach."
- **10GBASE-LR** — 10 gigabit Ethernet over single-mode fiber, longer distances.
- **40GBASE-SR4** / **100GBASE-SR4** — 40/100 gigabit Ethernet over four parallel fiber lanes.
- **WiFi PHY** — radio waves at 2.4 GHz or 5 GHz or 6 GHz. The physical part of 802.11.
- **Bluetooth PHY** — radio waves at 2.4 GHz, much shorter range and lower power.
- **Cellular PHY** — radio waves on licensed cellular bands (LTE, 5G NR).
- **DOCSIS** — the cable modem standard. Physical layer that runs on coax cable.
- **DSL** — digital subscriber line, physical layer over old phone lines.
- **GPON** — gigabit passive optical network, fiber to the home.

**Fiber types:**

- **Multimode fiber (MMF)** — fatter core (50 or 62.5 microns), uses cheap LED or VCSEL lasers, light bounces around inside, good for short distances (up to ~400 m at 10 Gbps), uses orange or aqua jacket cables (OM3, OM4, OM5).
- **Single-mode fiber (SMF)** — thin core (9 microns), uses expensive precise lasers, light goes in a straight line, good for long distances (kilometers to thousands of kilometers), yellow jacket cables (OS1, OS2).

**Signal encoding:** Floor 1 cannot just send raw 1s and 0s — long runs of all 1s or all 0s are hard to recover. So we use **line coding** schemes that mix transitions in. Examples: **Manchester encoding** (old 10BASE-T, every bit has a transition), **4B/5B** (100BASE-TX maps every 4 data bits to 5 line bits), **8B/10B** (gigabit fiber and many serial buses), **64B/66B** (10GbE and faster), **PAM4** (4-level pulse amplitude modulation for 100GbE+ lanes).

**Cabling:**

- **Twisted pair** — eight thin copper wires twisted in pairs, terminated in an **RJ45** connector. The twisting cancels electrical interference. Categories: Cat3 (old), Cat5 (100 Mbit), Cat5e (gig), Cat6 (gig with better margin), Cat6a (10 gig), Cat7/8 (rare, datacenter).
- **Coax** — central copper conductor surrounded by shielding. Used by old 10BASE2, cable modems (DOCSIS), and **BNC** connectors on test gear.
- **Fiber** — glass strand carrying laser light. Connectors: **LC** (small, common), **SC** (older square), **ST** (round bayonet), **MTP/MPO** (multi-fiber for parallel optics).
- **DAC (Direct Attach Cable)** — a fixed cable with SFP+/QSFP transceivers built into both ends, copper inside, used for short links inside a rack (1-7 m).
- **AOC (Active Optical Cable)** — same idea but the cable is fiber and the transceivers convert to/from light at each end, longer range than DAC.
- **SFP / SFP+ / SFP28 / QSFP+ / QSFP28 / QSFP-DD** — pluggable transceiver modules in switch ports. SFP = 1G, SFP+ = 10G, SFP28 = 25G, QSFP+ = 40G, QSFP28 = 100G, QSFP-DD = 400G.

**Try this:** see your link speed.

```
$ ip link show
```

```
1: lo: <LOOPBACK,UP,LOWER_UP> mtu 65536 qdisc noqueue state UNKNOWN
    link/loopback 00:00:00:00:00:00 brd 00:00:00:00:00:00
2: enp0s31f6: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 qdisc fq_codel state UP
    link/ether 14:dd:a9:7c:21:0e brd ff:ff:ff:ff:ff:ff
```

The line `state UP` and the words `LOWER_UP` mean Floor 1 is alive — there is a real signal on the cable.

```
$ ethtool enp0s31f6
```

```
Settings for enp0s31f6:
        Supported ports: [ TP ]
        Supported link modes:   10baseT/Half 10baseT/Full
                                100baseT/Half 100baseT/Full
                                1000baseT/Full
        Speed: 1000Mb/s
        Duplex: Full
        Port: Twisted Pair
        Auto-negotiation: on
        Link detected: yes
```

`Speed: 1000Mb/s` means you have a gigabit link. `Link detected: yes` means Floor 1 is working.

### L2 Data Link: frames and MAC addresses

Floor 2.

**PDU name:** **frame**.

**Plain English:** Floor 2 is the loading dock. It takes the bunch of bits from Floor 1 and groups them into chunks called **frames**. Every frame has a little label on the front that says "this frame goes to the machine with hardware address XX:XX:XX:XX:XX:XX," and another label on the back that says "and here is a tiny checksum so the receiver can tell if any of the bits got scrambled along the way."

Floor 2 only cares about getting a frame from one machine to the next machine on the same local network — same WiFi, same Ethernet cable, same switch. It does not know about the wider internet. The router (Floor 3) handles that.

**Hardware addresses (MAC addresses):**

A **MAC address** (sometimes called a **hardware address**, **physical address**, or **link layer address**) is a 48-bit number burned into every network card, written like `14:dd:a9:7c:21:0e` (six hexadecimal pairs separated by colons). The first three pairs (`14:dd:a9`) are the **OUI** — Organizationally Unique Identifier — assigned to the manufacturer (in this case, Intel). The last three pairs are the unique serial number for that card.

MAC addresses come in two main forms: **EUI-48** (48-bit, the classic) and **EUI-64** (64-bit, used in IPv6 stateless address autoconfiguration). Special MACs:

- `ff:ff:ff:ff:ff:ff` — the **broadcast** address. "Send this to every machine on the local network."
- `01:00:5e:xx:xx:xx` — IPv4 multicast.
- `33:33:xx:xx:xx:xx` — IPv6 multicast.
- A MAC where the second-least-significant bit of the first byte is set (like `02:...`) is **locally administered** instead of factory-burned.

**What runs there:**

- **Ethernet (IEEE 802.3)** — the king of wired LANs. The frame format starts with a preamble, has source and destination MACs, an EtherType, the payload, and a 32-bit Frame Check Sequence (CRC) at the end.
- **WiFi (IEEE 802.11)** — same idea but over radio. Adds extra fields for security and access control.
- **PPP (Point-to-Point Protocol)** — used on serial links, dial-up modems, some carrier circuits. Just two endpoints, no MAC needed.
- **HDLC** — high-level data link control, an older serial protocol.
- **ARP (Address Resolution Protocol)** — used on IPv4 networks to map an IP address to a MAC address. It says "who has 192.168.1.1, tell me." (Some textbooks put ARP at L2.5 since it sits between layers.)
- **STP (Spanning Tree Protocol)** — keeps switched networks loop-free. Without it, frames could circle forever in a loop and melt the network.
- **RSTP, MSTP** — faster and multi-instance versions of STP.
- **VLAN tagging (IEEE 802.1Q)** — adds a 4-byte tag to the frame to mark which "virtual LAN" it belongs to. Lets one cable carry many isolated networks.
- **QinQ (802.1ad)** — stacks two VLAN tags, used by service providers.
- **802.1X** — port-based authentication. The switch port stays closed until the device proves who it is.
- **LACP / 802.3ad** — bundles multiple physical links into one logical link (a LAG, Link Aggregation Group) for more speed and redundancy.
- **PoE (Power over Ethernet)** — sends power and data on the same cable. Used for IP phones, cameras, wireless access points.

**Framing:** Floor 2 is also responsible for telling where one frame ends and the next begins. Ethernet uses the preamble plus the start-of-frame delimiter; serial links like PPP use special escape characters; WiFi uses radio-level boundaries. Without framing, the receiver would just see an endless river of bits.

**Error detection:** Every Ethernet frame ends with a 32-bit **CRC** (Cyclic Redundancy Check). If even one bit got flipped on the wire, the CRC will not match and the receiver throws the frame away. Floor 2 does not retransmit — it just drops bad frames and lets a higher floor (usually Floor 4, TCP) figure out it is missing.

**Try this:** see your MAC address and the MACs of nearby machines.

```
$ ip link show enp0s31f6
```

```
2: enp0s31f6: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 qdisc fq_codel state UP
    link/ether 14:dd:a9:7c:21:0e brd ff:ff:ff:ff:ff:ff
```

That `link/ether` line is your MAC. The `brd` is the broadcast MAC.

```
$ ip neigh show
```

```
192.168.1.1 dev enp0s31f6 lladdr 9c:5c:8e:11:22:33 REACHABLE
192.168.1.42 dev enp0s31f6 lladdr fc:aa:14:55:66:77 STALE
192.168.1.50 dev enp0s31f6 lladdr 00:0e:c6:88:99:aa REACHABLE
```

That is your computer's **ARP cache** — every L2 neighbor it has talked to recently, mapping IPv4 (Floor 3) to MAC (Floor 2).

### L3 Network: packets and IP addresses

Floor 3.

**PDU name:** **packet**.

**Plain English:** Floor 3 is the postal route map. It takes a frame's payload (which Floor 2 carried across one local hop) and decides where to send it next on a planet-spanning network. Every packet has a source IP address and a destination IP address. Floor 3 looks up the destination in a routing table and says "the next hop is router R."

Floor 2 is local. Floor 3 is global. Floor 2 knows "this WiFi network." Floor 3 knows "the entire internet."

**Addresses:**

- **IPv4 address** — 32 bits, written as four numbers 0-255 separated by dots, like `192.168.1.42` or `8.8.8.8`. There are about 4.3 billion possible IPv4 addresses, which sounded like a lot in 1981 and is comically not enough today.
- **IPv6 address** — 128 bits, written as eight groups of four hex digits separated by colons, like `2001:0db8:85a3:0000:0000:8a2e:0370:7334`. Long runs of zeros can be shortened with `::`. There are 340 undecillion IPv6 addresses (3.4 × 10^38), which will probably last a while.

**What runs there:**

- **IPv4 (RFC 791)** — the original Internet Protocol from 1981. Header is 20 bytes minimum, has source/dest IP, TTL (time to live), protocol number, header checksum.
- **IPv6 (RFC 8200)** — the modern replacement. Header is 40 bytes fixed, no checksum (rely on lower and upper layers), has a hop limit instead of TTL, and uses **extension headers** chained together (Hop-by-Hop, Routing, Fragment, Authentication, ESP, Destination Options).
- **ICMP (Internet Control Message Protocol)** — the helper protocol for IPv4. Sends "destination unreachable," "time exceeded," and the famous **echo request / echo reply** that `ping` uses. RFC 792.
- **ICMPv6** — same idea for IPv6 but bigger; absorbs neighbor discovery (replacing ARP), router solicitation, multicast listener discovery.
- **IGMP / IGMPv2 / IGMPv3** — IPv4 multicast group membership. "I want to join multicast group 224.0.1.1, please."
- **MLD** — IPv6 equivalent of IGMP, riding on ICMPv6.
- **IPsec (AH and ESP)** — encryption and authentication that lives at L3. Used to build VPNs.
- **OSPF (Open Shortest Path First)** — interior routing protocol. Routers tell each other "I see these networks, here is the cost." Uses Dijkstra's algorithm.
- **IS-IS** — another interior routing protocol, popular with ISPs.
- **EIGRP** — Cisco's hybrid distance-vector routing protocol.
- **BGP (Border Gateway Protocol)** — the routing protocol that holds the entire internet together. Speaks between Autonomous Systems. RFC 4271.
- **RIP / RIPv2 / RIPng** — old simple distance-vector routing protocols.
- **GRE, IP-in-IP** — tunnel protocols, wrap a packet inside another packet.

**Try this:** see your IP and your routing table.

```
$ ip addr show enp0s31f6
```

```
2: enp0s31f6: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 qdisc fq_codel state UP
    link/ether 14:dd:a9:7c:21:0e brd ff:ff:ff:ff:ff:ff
    inet 192.168.1.42/24 brd 192.168.1.255 scope global dynamic enp0s31f6
    inet6 fe80::16dd:a9ff:fe7c:210e/64 scope link
```

`inet 192.168.1.42` is your IPv4 (Floor 3). `inet6 fe80::...` is your IPv6 link-local (also Floor 3). `link/ether` is Floor 2.

```
$ ip route show
```

```
default via 192.168.1.1 dev enp0s31f6 proto dhcp metric 100
192.168.1.0/24 dev enp0s31f6 proto kernel scope link src 192.168.1.42
```

That is your routing table. The `default via 192.168.1.1` line says "for any IP not on my local network, send the packet to my router at 192.168.1.1."

### L4 Transport: end-to-end delivery

Floor 4.

**PDU name:** **segment** (TCP) or **datagram** (UDP). People say "TCP segment" and "UDP datagram."

**Plain English:** Floor 4 is the packaging desk. It takes the data the application wants to send and breaks it into chunks, slaps a label on each chunk that says which **port** it is going to, and (for TCP) tracks every chunk so it can re-send any that get lost. Floor 4 cares about the conversation between two end machines, not the hops in between.

**Ports:** A port is a 16-bit number (0-65535) that identifies a specific service on a machine. Your web browser talks to port 443 on a web server. Your SSH client talks to port 22. The port is how the server knows "this connection is for the web server, not the SSH server." Three ranges:

- **Well-known** (0-1023) — reserved for standard services. 22 SSH, 25 SMTP, 53 DNS, 80 HTTP, 110 POP3, 143 IMAP, 443 HTTPS, 631 IPP printing.
- **Registered** (1024-49151) — assigned by IANA for specific applications. 3306 MySQL, 5432 PostgreSQL, 6379 Redis, 8080 alt HTTP.
- **Ephemeral** (49152-65535) — temporary ports that clients pick at random when opening an outbound connection.

A **socket** is the combination of an IP and a port: `192.168.1.42:51234`. A **connection** is identified by a 4-tuple: source IP, source port, destination IP, destination port.

**What runs there:**

- **TCP (Transmission Control Protocol)** — RFC 793, 1981. Connection-oriented, reliable, in-order, congestion-controlled. The most-used L4 protocol on Earth. If you are reading a web page, watching a YouTube video buffered, or sending email, it probably runs on TCP. TCP does a 3-way handshake to open a connection (SYN, SYN-ACK, ACK) and a 4-way handshake to close it (FIN, ACK, FIN, ACK).
- **UDP (User Datagram Protocol)** — RFC 768, 1980. Connectionless, unreliable, no congestion control. You send a datagram and hope it arrives. Used by DNS, NTP, video calls, online games, and anything that cares more about speed than perfection.
- **QUIC** — RFC 9000, 2021. Built on top of UDP but provides reliability, encryption, and multiplexing inside. HTTP/3 runs on QUIC. We will talk more about QUIC's layer-blurring later.
- **SCTP (Stream Control Transmission Protocol)** — RFC 4960. Originally designed for telephony signaling. Multi-streaming, multi-homing. Used in cellular core networks (Diameter, S1AP).
- **DCCP** — Datagram Congestion Control Protocol. Niche.

**Try this:** see what is talking on what port.

```
$ ss -tnp
```

```
State    Recv-Q  Send-Q  Local Address:Port    Peer Address:Port  Process
ESTAB    0       0       192.168.1.42:51234    140.82.114.21:443  users:(("firefox",pid=4321,fd=80))
ESTAB    0       0       192.168.1.42:48922    34.117.65.55:443   users:(("chromium",pid=5678,fd=42))
LISTEN   0       128     0.0.0.0:22            0.0.0.0:*          users:(("sshd",pid=789,fd=3))
```

Each `ESTAB` line is an L4 TCP connection. `LISTEN` means a server process is waiting for connections.

### L5 Session: opening and closing the conversation

Floor 5.

**PDU name:** **data** (formally "session PDU," but no specific name in modern stacks).

**Plain English:** Floor 5 is the conversation desk. Its job is to start, manage, and end a "session" — a series of related back-and-forth messages. Think of it like a phone call: you pick up, you talk, you say goodbye, you hang up. The session layer handles "pick up" and "hang up," and remembers things like "we are in the middle of a call right now."

In the real world, **almost nothing lives at L5 anymore.** Most of L5's old jobs have been absorbed into other layers — TCP handles connection setup at L4, and applications handle their own session state at L7. So when you read about L5 in modern networks, the answer is usually "this layer mostly does not exist as a discrete thing anymore."

**What runs there (mostly historical):**

- **NetBIOS (Network Basic Input/Output System)** — old Microsoft session API. Lives on top of TCP/IP via NetBT. Mostly replaced by SMB over TCP.
- **RPC (Remote Procedure Call)** — Sun RPC, DCE/RPC, Microsoft RPC. Lets a program call a function on another machine. Some call this L5, some call it L7.
- **SOCKS (4 / 4a / 5)** — a proxy protocol that opens a TCP session to a target through a proxy server. Listed at L5 historically.
- **PPTP** — point-to-point tunneling protocol, mostly dead now.
- **L2TP** — layer 2 tunneling protocol, often paired with IPsec.
- **NetBEUI** — old NetBIOS extended user interface. Truly gone.
- **ZIP** — Zone Information Protocol from old AppleTalk.
- **ASP** — AppleTalk Session Protocol.

**Reality check:** When you write a modern application, you probably never think about Floor 5. TCP gives you a reliable byte stream; HTTP gives you request-response semantics; you keep your own session state in cookies or tokens at L7. Most curricula teach L5 mostly so students can answer the OSI question on a certification exam.

### L6 Presentation: encoding and encryption

Floor 6.

**PDU name:** **data** (no specific PDU name).

**Plain English:** Floor 6 is the language translator and the safe-deposit box. It takes the data from Floor 7 (which is in whatever native format the application uses) and turns it into a standard wire format. It also handles encryption and compression. Then on the other side it does the reverse.

**What runs there:**

- **Character encodings** — ASCII, UTF-8, UTF-16, ISO-8859-1. Turn letters into bytes.
- **Image and video formats** — JPEG, PNG, GIF, MPEG, H.264, H.265, AV1. Compress visual data into smaller bytes.
- **Audio formats** — MP3, AAC, Opus, FLAC.
- **ASN.1 (Abstract Syntax Notation One)** — a way to define data structures abstractly and serialize them. Used in LDAP, SNMP, X.509 certificates, GSM/3GPP signaling.
- **BER, DER, PER, XER** — encoding rules for ASN.1 (Basic, Distinguished, Packed, XML).
- **JSON, XML, YAML, Protocol Buffers, MessagePack, CBOR, Avro** — modern serialization formats. Some textbooks put these at L6, some say "applications handle this themselves at L7."
- **TLS (Transport Layer Security)** — sometimes drawn at L6, sometimes drawn between L4 and L7. We will untangle this later. RFC 8446 defines TLS 1.3.
- **DTLS** — Datagram Transport Layer Security, TLS but for UDP. RFC 9147.
- **SSL (Secure Sockets Layer)** — the deprecated old name for TLS. SSL 3.0 was killed by POODLE. Nobody uses SSL anymore but the name persists in old habits ("the SSL cert").

**Reality check:** Like L5, L6 is mostly a polite fiction. TLS does not fit cleanly anywhere. Character encoding happens way up in the application. The 7-layer model was designed before any of this existed in modern form.

### L7 Application: what users see

Floor 7. The top floor.

**PDU name:** **data** (or "message" depending on protocol).

**Plain English:** Floor 7 is where you, the user, actually live. Every application you use that talks to the network is talking to Floor 7. Your web browser. Your email client. Your text messages. Your video call. Your `git pull`. Your `apt install`. All Floor 7.

**What runs there:**

- **HTTP / HTTP/1.1 / HTTP/2 / HTTP/3** — HyperText Transfer Protocol. Web pages, REST APIs, APIs, file downloads. RFC 9110 (semantics), 9112 (HTTP/1.1), 9113 (HTTP/2), 9114 (HTTP/3).
- **HTTPS** — HTTP wrapped in TLS. Same protocol, encrypted transport.
- **DNS** — Domain Name System. RFC 1034, 1035. Turns names like `www.example.com` into IPs.
- **mDNS** — Multicast DNS. Lets devices on a LAN find each other without a DNS server. RFC 6762.
- **DHCP / DHCPv6** — Dynamic Host Configuration Protocol. Hands out IP addresses, gateway, DNS server. RFC 2131. (DHCP itself is L7 even though what it configures is L3.)
- **BOOTP** — old, deprecated, kind of survives in DHCPv6.
- **SMTP** — Simple Mail Transfer Protocol. Sending email between mail servers. RFC 5321.
- **IMAP** — Internet Message Access Protocol. Reading email from a server, leaving it on the server. RFC 9051.
- **POP3** — Post Office Protocol. Reading email by downloading and removing it from the server. RFC 1939.
- **SSH** — Secure Shell. Encrypted remote login, file transfer, port forwarding. RFC 4253.
- **Telnet** — old, dead, plaintext remote login. RFC 854. Use SSH instead.
- **FTP** — File Transfer Protocol. RFC 959. Largely dead, replaced by SFTP, FTPS, HTTPS, rsync.
- **SFTP** — SSH File Transfer Protocol. Runs inside an SSH session.
- **NTP** — Network Time Protocol. Keeps your clock accurate. RFC 5905.
- **SNMP** — Simple Network Management Protocol. Network device monitoring. RFC 3411-3418.
- **LDAP** — Lightweight Directory Access Protocol. Directory queries. RFC 4511.
- **Kerberos** — Network authentication. RFC 4120.
- **TFTP** — trivial file transfer, used by network device firmware loaders.
- **WebSocket** — full-duplex over HTTP upgrade. RFC 6455.
- **gRPC** — Google's RPC framework over HTTP/2.
- **MQTT** — message queue telemetry, IoT pub/sub.
- **AMQP** — message queueing.
- **IRC, XMPP, Matrix** — chat protocols.
- **BitTorrent** — peer-to-peer file sharing.

This is the layer with the most stuff because it is the layer humans actually use.

**Try this:** make a Floor 7 request and see all the lower layers in action.

```
$ curl -v https://example.com 2>&1 | head -30
```

```
*   Trying 93.184.216.34:443...
* Connected to example.com (93.184.216.34) port 443 (#0)
* ALPN, offering h2
* ALPN, offering http/1.1
* TLSv1.3 (OUT), TLS handshake, Client hello (1):
* TLSv1.3 (IN), TLS handshake, Server hello (1):
* TLSv1.3 (IN), TLS handshake, Encrypted Extensions (1):
* TLSv1.3 (IN), TLS handshake, Certificate (1):
* TLSv1.3 (IN), TLS handshake, CERT verify (1):
* TLSv1.3 (IN), TLS handshake, Finished (1):
* TLSv1.3 (OUT), TLS handshake, Finished (1):
* SSL connection using TLSv1.3 / TLS_AES_256_GCM_SHA384
> GET / HTTP/2
> Host: example.com
> User-Agent: curl/7.81.0
> Accept: */*
>
< HTTP/2 200
< content-type: text/html; charset=UTF-8
```

In that one command you saw L3 (`Trying 93.184.216.34:443`), L4 (`port 443`), L6/L4-ish (`TLS handshake`), and L7 (`GET / HTTP/2`).

## The DoD / TCP-IP Model

The OSI model is a teaching model. The actual internet runs on a different model called the **TCP/IP model**, sometimes called the **DoD model** (because the U.S. Department of Defense funded it through DARPA in the 1970s). It has only **four** layers, and it predates OSI by **three years** — RFC 791 (IP) and RFC 793 (TCP) were published in **1981**. OSI's ISO/IEC 7498 came out in **1984**.

```
+---------------------------------+   +-----------------+
|  OSI 7-LAYER MODEL              |   |  DoD / TCP-IP   |
+---------------------------------+   +-----------------+
|  L7  Application                |   |                 |
+---------------------------------+   |   Application   |
|  L6  Presentation               |   |    (L5 + L6     |
+---------------------------------+   |    + L7 fused)  |
|  L5  Session                    |   |                 |
+---------------------------------+   +-----------------+
|  L4  Transport                  |   |   Transport     |
+---------------------------------+   +-----------------+
|  L3  Network                    |   |   Internet      |
+---------------------------------+   +-----------------+
|  L2  Data Link                  |   |   Link          |
+---------------------------------+   |   (L1 + L2      |
|  L1  Physical                   |   |    fused)       |
+---------------------------------+   +-----------------+
```

The four DoD layers:

1. **Link layer** — combines OSI L1 + L2. Ethernet, WiFi, the cable, the frame.
2. **Internet layer** — same as OSI L3. IPv4, IPv6, ICMP. Routes packets across networks.
3. **Transport layer** — same as OSI L4. TCP, UDP, QUIC.
4. **Application layer** — combines OSI L5 + L6 + L7. HTTP, DNS, SSH, every application protocol.

The DoD model is honest about what really happens in practice. There is no clean L5/L6/L7 separation in real protocols; an HTTP server just opens a TCP socket and sends bytes that contain everything from the request line to the JSON body. The OSI model says "those are three layers"; the DoD model says "no, that's one application doing one job."

The official "Host Requirements" RFCs that all internet hosts must obey are **RFC 1122** (link, internet, transport) and **RFC 1123** (application support). They use the four-layer DoD model.

**Some textbooks teach a "5-layer hybrid" model** that splits the DoD link layer back into OSI's L1 physical and L2 data link, while keeping L5-L7 fused into "application." This 5-layer model is what Kurose & Ross use in their popular university textbook. So you may see all three:

- 7 layers (OSI, ISO/IEC 7498)
- 5 layers (Kurose & Ross hybrid)
- 4 layers (DoD / TCP-IP, RFC 1122)

They are all describing the same packets going across the same wires.

## The RINA Model

There is a third model you should at least know exists, even if it is not in widespread deployment. It is called **RINA** — **Recursive InterNetwork Architecture**. It comes from a body of work by **John Day**, especially his book *Patterns in Network Architecture: A Return to Fundamentals* (2008). Active research happens at Boston University and a handful of European universities.

The RINA argument is, roughly: we got it wrong.

Day's claim is that the OSI and TCP/IP layering both started from the same mistake — they treated layers as different *functions* (physical, link, network, transport, etc.) when they should have treated them as *the same function repeated at different scopes*. The single function he calls **IPC** — **Inter-Process Communication**.

In RINA, every layer is just IPC running between processes. The only thing that changes between layers is the *scope* (how far the IPC reaches) and the *policy* (what reliability, addressing, and congestion control rules apply). It is the same machinery — just **recursing**.

```
+----------------------------------------------+
|   RINA: one function, repeated at scope      |
+----------------------------------------------+
|     [IPC at largest scope = whole internet]  |
|       [IPC at provider scope = your ISP]     |
|         [IPC at site scope = your company]   |
|           [IPC at LAN scope = your office]   |
|             [IPC at host scope = one box]    |
+----------------------------------------------+
        Same primitives at every scope.
```

Each scope is a "DIF" (Distributed IPC Facility), with its own naming, addressing, and policies, but the same underlying primitives. A higher DIF uses a lower DIF as a transport, just like a process uses an OS pipe.

The benefit RINA proponents claim is that NAT, multihoming, mobility, security, and QoS all become natural instead of bolt-ons. The claim is debatable; what matters for a learner is that RINA exists as a "third model" that says **layering by function** is wrong and **layering by scope** is right.

You will not configure anything in RINA on a normal Linux box today. But knowing the model exists makes it easier to understand why TCP/IP feels messy — it really is messy, because it grew organically rather than being designed clean.

## "Layer 8" Joke

There is a running joke in network engineering: **the Layer 8 problem.**

```
+---------------+
|  L9   Money   |   (sometimes added as L9)
+---------------+
|  L8   People  |
+---------------+
|  L7   App     |
|  ...          |
|  L1   Phys    |
+---------------+
```

Officially, the OSI model stops at L7. But every network engineer has worked on a problem where the cables were fine, the routing was fine, the firewall was fine — and the actual problem was a person. Maybe a manager who would not approve a change. Maybe a user who refused to read the email about the maintenance window. Maybe a vendor that promised something and did not deliver. Maybe a regulation that prevented a deployment.

So engineers joke about **Layer 8** — the **user / political / organizational layer**. Some go further and joke about **Layer 9** for **money / budget**, **Layer 10** for **government / regulatory**, etc.

The joke matters because the lesson is real: most production network outages turn out to be Layer 8 problems wearing Layer 1-7 costumes. "The link is down" is often "the network engineer cleaning the rack tripped on the cable." "The route is missing" is often "the change request was rejected." "The certificate expired" is often "no one was paid to renew it."

When troubleshooting, do not skip Layer 8.

## Encapsulation Walkthrough — A Single HTTP GET

Let's watch what actually happens when you type `https://example.com` in your browser.

### Down the floors at the sender

Step 1. **L7 Application.** Your browser builds an HTTP/2 request like this:

```
HEADERS frame:
  :method = GET
  :scheme = https
  :authority = example.com
  :path = /
  user-agent = Mozilla/5.0 ...
  accept = text/html
```

That is the application's data. We will call this chunk "DATA."

Step 2. **L6 Presentation.** TLS encrypts the HTTP/2 request. Now we have ciphertext. Call this chunk "ENC(DATA)."

Step 3. **L5 Session.** In modern stacks, L5 does basically nothing visible. The TCP connection is the session, and TCP lives at L4. So we just hand ENC(DATA) down.

Step 4. **L4 Transport.** TCP wraps the encrypted bytes into a segment with a 20-byte header containing source port (random ephemeral, like 51234), destination port (443 for HTTPS), sequence number, ack number, flags, window size, and a checksum.

```
[TCP HEADER 20B][ENC(DATA)]
```

Step 5. **L3 Network.** IPv4 wraps the TCP segment into a packet with a 20-byte header containing source IP (192.168.1.42), destination IP (93.184.216.34), TTL, protocol = 6 (TCP), and a header checksum.

```
[IP HEADER 20B][TCP HEADER 20B][ENC(DATA)]
```

Step 6. **L2 Data Link.** Ethernet wraps the IP packet into a frame with a 14-byte header (destination MAC of the next hop router, source MAC of your network card, EtherType = 0x0800 for IPv4) and a 4-byte trailer (CRC).

```
[ETH HEADER 14B][IP HEADER 20B][TCP HEADER 20B][ENC(DATA)][ETH CRC 4B]
```

Step 7. **L1 Physical.** The whole frame becomes voltage pulses on the copper or photons in the fiber or radio waves in the air.

That is **encapsulation**. Each layer adds its header (and sometimes a trailer) and passes the whole bundle to the next layer down.

### The Russian-doll picture

```
+-----------------------------------------------+
| L1   bits on a wire / fiber / radio           |
| +-------------------------------------------+ |
| | L2 Eth hdr [ payload ] CRC                | |
| | +---------------------------------------+ | |
| | | L3 IP hdr [ payload ]                 | | |
| | | +-----------------------------------+ | | |
| | | | L4 TCP hdr [ payload ]            | | | |
| | | | +-------------------------------+ | | | |
| | | | | L5/6/7 HTTP+TLS payload bytes | | | | |
| | | | +-------------------------------+ | | | |
| | | +-----------------------------------+ | | |
| | +---------------------------------------+ | |
| +-------------------------------------------+ |
+-----------------------------------------------+
```

Each outer layer wraps the inner. By the time the bits leave your network card, your tiny `GET /` request is buried under four to seven layers of headers.

### Up the floors at the receiver

The web server's network card receives the bits.

Step 1. **L1 Physical** turns voltage pulses back into bits and hands them to L2.

Step 2. **L2 Data Link** finds the start of the frame, checks the CRC. If the CRC fails, drop the frame. If it passes, strip the Ethernet header and trailer and look at the EtherType. EtherType 0x0800 means "the payload is IPv4," so hand it up to L3 IPv4.

Step 3. **L3 Network** (IPv4) checks the header checksum, looks at the destination IP. If the destination is "me," strip the IP header and look at the protocol field. Protocol 6 means "the payload is TCP," so hand it up to L4 TCP.

Step 4. **L4 Transport** (TCP) checks the TCP checksum, finds the right connection by 4-tuple (src ip, src port, dst ip, dst port), reassembles in-order bytes, and hands the byte stream up to the application listening on port 443.

Step 5. **L6 Presentation** (TLS) decrypts the bytes back into plaintext.

Step 6. **L7 Application** (HTTP/2) parses the request, sees `:method GET, :path /`, hands it to the web server logic, which generates a response.

Then the response goes back down the floors on the server side, across the wire, and back up the floors on your side. All of this happens in **milliseconds**.

## Where Each Protocol Actually Lives

The OSI model is a *teaching* tool. Real protocols do not always sit at exactly one layer. Here is the messy truth.

### TLS

TLS is the most common offender. Officially, TLS provides **encryption and authentication**, which sounds like Floor 6 Presentation. But TLS also opens a connection (Floor 5 Session) and sits on top of TCP (which is Floor 4) and below HTTP (which is Floor 7).

A common shorthand: **TLS sits between L4 and L7.** People draw it as "L6.5" or "between L5 and L6" or just "the TLS layer." The OSI model has no good home for it because the OSI designers did not anticipate a generic security layer that wraps any application protocol.

```
+----------------------------------+
|   L7   HTTP                      |
+----------------------------------+
|        TLS  <-- here, no clean L |
+----------------------------------+
|   L4   TCP                       |
+----------------------------------+
```

### QUIC

QUIC is even more disruptive. QUIC takes the jobs of L4 (transport, with reliability and congestion control), L5 (session establishment), L6 (encryption — TLS 1.3 is built in), and pieces of L7 (multiplexed streams) and **collapses them into a single protocol** that sits inside UDP.

```
+----------------------------------+
|   L7   HTTP/3                    |
+----------------------------------+
|        QUIC (L4 + L5 + L6 + L7?) |
+----------------------------------+
|   L4   UDP                       |
+----------------------------------+
|   L3   IP                        |
+----------------------------------+
```

QUIC's designers explicitly chose to ignore the OSI layer boundaries because following them was making the protocol slower and worse. QUIC is in some sense the IETF saying "we are done pretending the OSI separation is a real thing."

### MPLS

MPLS (Multiprotocol Label Switching) is sometimes called **Layer 2.5** because it sits between Ethernet (L2) and IP (L3) — it adds a 4-byte label header that switches use to forward packets without looking at the IP header.

```
+----------------------------------+
|   L3   IP                        |
+----------------------------------+
|        MPLS  <-- "L2.5"          |
+----------------------------------+
|   L2   Ethernet                  |
+----------------------------------+
```

### NAT

Network Address Translation (NAT) sits at L3 (rewriting source IPs) and L4 (rewriting source ports). It is a **layer violation** — the L3 router cannot really do its job correctly without inspecting and rewriting L4 fields. NAT is one of the reasons people say "layered models are aspirational, not literal."

### ARP, ICMP

ARP sits between L2 and L3. ICMP rides on top of IP (L3) but also generates messages about IP itself, which is recursive. Both are usually drawn at the same layer they ride on, with a footnote.

## L2 / L3 / L4 / L7 Switches

In networking marketing, you will hear devices described by the layer they operate at. This is mostly truthful but partly hype.

**L1 hub** — old, dumb, just repeats every signal out every port. Effectively dead.

**L2 switch** — looks at the destination MAC in each frame and forwards to the right port. Builds a MAC address table by learning. This is what most "switches" are.

**L3 switch** — also a router. Looks at the destination IP and forwards based on a routing table. The line between "L3 switch" and "router" is mostly market segmentation; both do the same job, with L3 switches optimized for high-port-density LAN/datacenter use and routers optimized for WAN with more features.

**Router** — same as L3 switch in terms of forwarding logic; usually has more software features (BGP, complex policy, NAT, VPN).

**L4 switch** — really a load balancer that looks at TCP/UDP ports and source/destination IPs to distribute connections across a pool of servers. "L4 switch" is mostly a marketing term for "low-level load balancer."

**L7 switch** — a load balancer that inspects the application payload (HTTP headers, TLS SNI, etc.) and routes based on URL path, hostname, cookies, etc. A.k.a. "application delivery controller," "L7 load balancer," "ADC," "reverse proxy." nginx, HAProxy, Envoy, F5 BIG-IP, AWS ALB, Cloudflare are all examples.

**Firewall** — a device that filters traffic based on policy. Stateless firewalls match packets at L3/L4. Stateful firewalls track connections (still L3/L4 but smarter). Application-aware firewalls (next-gen firewalls, NGFW) inspect L7. The marketing keeps adding numbers.

**Switch vs router vs gateway vs firewall** — these are **roles**, not strict layers. One physical box can do all four. A modern home WiFi router is doing L1 (radio), L2 (switch), L3 (routing), L4 (NAT), L7 (DHCP server, DNS forwarder), and security (firewall) all at once.

## MTU and Fragmentation

**MTU** stands for **Maximum Transmission Unit**. It is the largest payload (usually in bytes) that a given link can carry in one frame.

The default Ethernet MTU is **1500 bytes**. That is the maximum size of an IP packet that can fit inside one Ethernet frame. The L2 frame itself is bigger because of the Ethernet header (14 bytes) and CRC (4 bytes), but the L3 payload (the IP packet) is 1500 bytes max.

Some networks support **jumbo frames** with MTUs of 9000 bytes or more. This is common in datacenters because it reduces per-frame overhead. WAN links often have smaller MTUs (PPPoE is 1492; tunnels can be 1400 or less).

If an L3 packet is too big to fit in an L2 frame, two things can happen:

1. **Fragmentation (IPv4 only by default).** The router that hits the small MTU breaks the packet into smaller pieces, sets the More Fragments flag and a fragment offset in each piece, and sends them. The receiving host reassembles. Fragmentation is slow, breaks NAT in funny ways, and makes troubleshooting hard.
2. **Path MTU Discovery (PMTUD).** The sender sets the **DF (Don't Fragment)** bit on each packet. If a packet is too big, the router drops it and sends back an ICMP "fragmentation needed" message saying "the next-hop MTU is N." The sender then sends smaller packets. This is the modern correct way. IPv6 *requires* PMTUD because IPv6 routers do not fragment at all — only the end host can.

**MSS (Maximum Segment Size)** is the L4 cousin of MTU. It is the largest TCP payload (no headers) that fits in a single segment. The standard formula:

```
MSS = MTU - IP header - TCP header
    = 1500 - 20 - 20
    = 1460 bytes
```

For IPv6: `MSS = 1500 - 40 - 20 = 1440` bytes.

For PPPoE: `MSS = 1492 - 20 - 20 = 1452` bytes.

When you see weird "everything works except large file uploads through a VPN," the answer is almost always MTU/MSS mismatch. The fix is "MSS clamping" or lowering the MTU on the tunnel.

```
$ cat /sys/class/net/enp0s31f6/mtu
1500
```

```
$ ip link show enp0s31f6 | head -1
2: enp0s31f6: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 qdisc fq_codel state UP
```

```
$ ping -M do -s 1472 -c 2 1.1.1.1
PING 1.1.1.1 (1.1.1.1) 1472(1500) bytes of data.
1480 bytes from 1.1.1.1: icmp_seq=1 ttl=58 time=8.21 ms
1480 bytes from 1.1.1.1: icmp_seq=2 ttl=58 time=8.05 ms

--- 1.1.1.1 ping statistics ---
2 packets transmitted, 2 received, 0% packet loss, time 1001ms
rtt min/avg/max/mdev = 8.054/8.131/8.208/0.077 ms
```

`-M do` sets the DF bit. `-s 1472` is 1500 minus 20 (IP header) minus 8 (ICMP header). If your path MTU is 1500, this works. If it is smaller, you get "Frag needed and DF set."

## Why Layers Sometimes Leak

The OSI model wants each layer to be totally independent. In reality, the protocols we actually use have **leaks** — places where one layer cares about something in another layer's header.

Here are some of the most common leaks.

### IP DF and ECN

The IP header (L3) has a **Don't Fragment (DF)** bit and **Explicit Congestion Notification (ECN)** bits. TCP (L4) cares about both. TCP wants the DF bit set so PMTUD works. TCP wants the ECN bits readable so it can react to congestion signals. So L4 reaches down into L3's header. This is a leak.

### DSCP / QoS

The IP header has a 6-bit **DSCP** (Differentiated Services Code Point) field that tells L3 routers how to prioritize a packet. But the *application* (L7) often wants to mark its own traffic — "this is voice, prioritize it" — so applications set socket options that propagate down to the IP header. Another leak.

### TLS SNI

TLS (L6-ish) puts the destination hostname in the **SNI** extension in the TLS ClientHello, in plaintext. This is for L7 routing (the load balancer needs to know which website to route to). But it is L6 data leaking up to be used by L4-L7 load balancers and sometimes by middleboxes that censor based on hostname. This is why **Encrypted Client Hello (ECH)** is being deployed.

### NAT

NAT rewrites L3 addresses and L4 ports. To do this correctly with FTP, SIP, IRC DCC, and other "control connections that mention IPs in the payload," NAT has to peek into L7 payloads and rewrite them. These are called **Application Layer Gateways (ALGs)**. ALGs are the worst layer violation in normal use.

### Conntrack

The Linux kernel has a **conntrack** subsystem that tracks every L4 connection through the box. This is needed for stateful firewalling and NAT. But it has to inspect L3 + L4 + sometimes L7 to do its job, and has hooks at every layer. It is not a layer violation in OSI terms exactly, but it is an example of cross-layer state.

### MTU, MSS, PMTUD

The L4 MSS depends on the L3 MTU minus L3+L4 headers. The MSS option is in the TCP header (L4) but its value is computed from the IP MTU (L3) which depends on the L2 MTU. Three layers of cross-talk.

The lesson: **the OSI model is a useful map. The territory is messier.**

## Common Errors

Verbatim error messages, the layer they implicate, and the canonical fix.

### `Network is unreachable` — L3

Your machine has no route to the destination network.

```
$ ping 10.99.99.1
ping: connect: Network is unreachable
```

Fix: check `ip route show`. You probably do not have a default route or the network you are trying to reach is not in your routing table. Typical fix is to fix the default gateway: `sudo ip route add default via 192.168.1.1`.

### `No route to host` — L3

You have a route, but the next hop says the destination is dead. Often this means an ICMP "host unreachable" or a stale ARP entry.

```
$ ssh 192.168.1.250
ssh: connect to host 192.168.1.250 port 22: No route to host
```

Fix: ping the gateway, check `ip neigh show`, look at the firewall on the destination.

### `Destination Host Unreachable` — L3

ICMP type 3 code 1 reply.

```
$ ping 192.168.1.250
PING 192.168.1.250 (192.168.1.250) 56(84) bytes of data.
From 192.168.1.42 icmp_seq=1 Destination Host Unreachable
From 192.168.1.42 icmp_seq=2 Destination Host Unreachable
```

Fix: the host is offline, or has a firewall dropping ARP, or a different VLAN. Check the host directly.

### `Connection refused` — L4

The destination IP is reachable and answers, but no process is listening on that port. Often a TCP RST.

```
$ curl http://192.168.1.42:8080
curl: (7) Failed to connect to 192.168.1.42 port 8080 after 1 ms: Connection refused
```

Fix: start the server, or check the actual listening port with `ss -tnlp`.

### `Connection timed out` — L3 or L4

Packets are going out and nothing is coming back. Could be silent firewall drop, wrong IP, or destination overloaded.

```
$ curl http://10.0.0.1
curl: (28) Failed to connect to 10.0.0.1 port 80 after 134003 ms: Connection timed out
```

Fix: traceroute to find where the path dies; check firewalls between you and the destination.

### `TCP RST` — L4

Tcpdump shows a RST flag. The remote end actively rejected your connection (often after accepting). Common with idle timeouts, app crashes, or stateful firewalls killing flows.

```
$ tcpdump -ni any 'tcp[tcpflags] & tcp-rst != 0'
14:32:11.123 IP 10.0.0.5.443 > 192.168.1.42.51234: Flags [R.], seq 1, ack 1, win 0, length 0
```

Fix: check application logs on the remote end; if it is a firewall, look for "TCP reset out of state."

### `ARP timeout` / `Neighbor State FAILED` — L2

You sent an ARP for a local IP and got no answer.

```
$ ip neigh show
192.168.1.250 dev enp0s31f6  FAILED
```

Fix: the host is offline, or on a different VLAN, or has port-security / 802.1X blocking. Check the L2 switch.

### `SSL_ERROR_BAD_CERT_DOMAIN` / `x509: certificate is valid for X, not Y` — L6/L7

The TLS certificate's Subject Alternative Names do not match the hostname you connected to.

```
$ curl https://wrong.host.example.com
curl: (60) SSL: no alternative certificate subject name matches target host name 'wrong.host.example.com'
```

Fix: connect to the correct hostname, or fix the cert SAN list, or (carefully, in dev only) use `-k` to bypass.

### `502 Bad Gateway` — L7

A reverse proxy (nginx, AWS ALB, Cloudflare) tried to talk to your backend and got nothing usable.

```
$ curl https://api.example.com/health
HTTP/2 502
```

Fix: check the backend is running, listening on the right port, and reachable from the proxy.

### `503 Service Unavailable` — L7

The server is up but is refusing the request right now (overloaded, maintenance, no healthy backend).

```
$ curl https://api.example.com/
HTTP/2 503
retry-after: 60
```

### `504 Gateway Timeout` — L7

A reverse proxy gave up waiting for the backend.

### `Permission denied` — L7

Authorization denied at the application layer.

```
$ ssh user@host
user@host: Permission denied (publickey).
```

Fix: provide a valid key, password, or whatever the server expects.

### `No buffer space available` — L4 (kernel)

The kernel ran out of socket buffers, or you hit a limit.

```
$ curl http://example.com
curl: (7) Failed to connect: No buffer space available
```

Fix: usually means a process leak, or a `net.ipv4.tcp_*` sysctl is too tight, or you ran out of ephemeral ports.

### `Network is down` — L1 / L2

The interface is administratively down or the cable is unplugged.

```
$ ping 192.168.1.1
ping: connect: Network is down
```

Fix: `ip link set enp0s31f6 up`, plug the cable in, restart the WiFi.

### `Address already in use` — L4

You tried to bind to a port that is already taken.

```
$ python3 -m http.server 8080
OSError: [Errno 98] Address already in use
```

Fix: pick a different port or kill the existing listener.

### `Cannot assign requested address` — L3

You tried to bind to an IP that is not configured on this machine.

```
$ python3 -m http.server --bind 10.99.99.99
OSError: [Errno 99] Cannot assign requested address
```

Fix: configure the IP first, or bind to one you actually have.

### `Operation not permitted` — L7 / kernel

Often you are trying to bind to a privileged port (under 1024) without root or `CAP_NET_BIND_SERVICE`.

```
$ python3 -m http.server 80
PermissionError: [Errno 13] Permission denied
```

Fix: use a port over 1023, or run with `sudo`, or grant the capability.

## Hands-On

Here is a long parade of commands that show each layer in action. Run them. Watch the output. The point is to see the abstract floors of the tower as concrete things on your computer.

### L1 Physical commands

```
$ ip link show
$ ethtool enp0s31f6
$ ethtool -S enp0s31f6
$ mii-tool enp0s31f6
$ cat /sys/class/net/enp0s31f6/speed
$ cat /sys/class/net/enp0s31f6/duplex
$ cat /sys/class/net/enp0s31f6/carrier
$ dmesg | grep -i eth
$ journalctl -k | grep -i link
$ iw dev
$ iwconfig
$ iw dev wlp3s0 link
```

Example output of `ethtool -S`:

```
NIC statistics:
     rx_packets: 814231
     tx_packets: 612344
     rx_bytes: 911232123
     tx_bytes: 71823821
     rx_crc_errors: 0
     rx_missed_errors: 0
     tx_aborted_errors: 0
```

`rx_crc_errors` going up is a Floor 1 problem — the cable, the connector, the SFP. Replace the patch cord first.

### L2 Data Link commands

```
$ ip neigh show
$ ip -s link show enp0s31f6
$ arping -c 1 -I enp0s31f6 192.168.1.1
$ bridge fdb show
$ bridge link show
$ brctl show
$ ip link show type vlan
$ ip link show type bond
$ tcpdump -ni any -e arp
```

Example `arping`:

```
$ arping -c 1 -I enp0s31f6 192.168.1.1
ARPING 192.168.1.1 from 192.168.1.42 enp0s31f6
Unicast reply from 192.168.1.1 [9C:5C:8E:11:22:33]  0.781ms
Sent 1 probes (1 broadcast(s))
Received 1 response(s)
```

That tells you Floor 2 works between you and your gateway. Compare to:

```
$ arping -c 1 -I enp0s31f6 192.168.1.250
ARPING 192.168.1.250 from 192.168.1.42 enp0s31f6
Sent 1 probes (1 broadcast(s))
Received 0 response(s)
```

Zero responses means the host is not on the L2 segment.

### L3 Network commands

```
$ ip addr show
$ ip route show
$ ip route show table all
$ ip rule show
$ ip -6 addr show
$ ip -6 route show
$ ping -c 4 1.1.1.1
$ ping6 -c 4 2606:4700:4700::1111
$ traceroute -n 1.1.1.1
$ tracepath 1.1.1.1
$ mtr -n 1.1.1.1
$ ip route get 8.8.8.8
$ tcpdump -ni any 'icmp or icmp6'
```

Example:

```
$ traceroute -n 1.1.1.1
traceroute to 1.1.1.1 (1.1.1.1), 30 hops max, 60 byte packets
 1  192.168.1.1   0.421 ms  0.402 ms  0.391 ms
 2  10.10.0.1     2.314 ms  2.302 ms  2.291 ms
 3  172.16.5.1    5.881 ms  5.870 ms  5.862 ms
 4  1.1.1.1       8.122 ms  8.110 ms  8.099 ms
```

Each line is a Floor 3 router decrementing the TTL. That is L3 hop-by-hop.

### L4 Transport commands

```
$ ss -tnp
$ ss -tnlp
$ ss -unp
$ ss -unlp
$ ss -s
$ ss -o state established
$ ss -tipemo state established
$ netstat -tnp
$ netstat -unp
$ netstat -s
$ tcpdump -ni any 'tcp port 443'
$ tcpdump -ni any 'udp port 53'
```

Example `ss -s`:

```
$ ss -s
Total: 412 (kernel 0)
TCP:   89 (estab 14, closed 65, orphaned 0, synrecv 0, timewait 65/0), ports 0

Transport Total     IP        IPv6
*         0         -         -
RAW       1         0         1
UDP       18        12        6
TCP       24        18        6
INET      43        30        13
FRAG      0         0         0
```

That gives you a quick L4 summary.

### L5/L6/L7 Application commands

```
$ dig example.com
$ dig +short A example.com
$ dig +short AAAA example.com
$ dig +short MX example.com
$ host example.com
$ nslookup example.com
$ getent hosts example.com
$ openssl s_client -connect example.com:443 -servername example.com </dev/null
$ curl -v https://example.com
$ curl -I https://example.com
$ curl --resolve example.com:443:93.184.216.34 https://example.com
$ ssh -v user@host
$ nc -v example.com 443
$ telnet example.com 80
$ httpie GET https://example.com
$ wget -v https://example.com
```

Example `dig`:

```
$ dig +short A example.com
93.184.216.34
```

Example `openssl s_client`:

```
$ openssl s_client -connect example.com:443 -servername example.com </dev/null 2>&1 | head -20
CONNECTED(00000003)
depth=2 C = US, O = Internet Security Research Group, CN = ISRG Root X1
verify return:1
depth=1 C = US, O = Let's Encrypt, CN = R3
verify return:1
depth=0 CN = example.com
verify return:1
---
Certificate chain
 0 s:CN = example.com
   i:C = US, O = Let's Encrypt, CN = R3
---
Server certificate
-----BEGIN CERTIFICATE-----
MIIFazCCBFOgAwIBAgISA...
```

That output shows L4 (`CONNECTED`), L6 (TLS handshake and certificate), and lets you see exactly what cert the server is presenting.

### Multi-layer tools

```
$ tcpdump -ni any -vv -X
$ tshark -ni any -V
$ wireshark
$ netcat -lvnp 4444
$ socat - TCP:example.com:80
$ iftop -ni enp0s31f6
$ nload enp0s31f6
$ bmon
$ iotop
$ nethogs
```

A pcap captures Floors 2 through 7 in one file. You can open it in Wireshark and click between layers — Wireshark has a "layer" pane that shows exactly which OSI floor each header belongs to.

## Common Confusions

Things that trip up almost every learner.

### "TLS is L6"

**Wrong.** TLS does not have a clean home in the OSI model. It encrypts (L6), establishes a session (L5), runs on top of TCP (L4), and applications (L7) plug it in directly. People call it "L6" because there is no better answer. **Fixed:** TLS is its own thing. Drawing it as "between L4 and L7" or "L4.5" is more honest.

### "OSI is the protocol stack of the internet"

**Wrong.** The OSI model is a reference model, not a deployed protocol stack. There was a real OSI protocol stack — IS-IS, X.400, X.500, CMIP, CLNP — and it lost to TCP/IP in the 1980s and 1990s. The internet runs on TCP/IP, which uses the 4-layer DoD model. **Fixed:** OSI is a teaching tool. TCP/IP is what is on the wire.

### "There are exactly 7 layers"

**Wrong.** OSI has 7. DoD/TCP-IP has 4. The 5-layer hybrid model (Kurose & Ross) is widely taught. RINA has infinite recursive layers. The number depends on the model. **Fixed:** "7 layers" is one specific model.

### "The 5-layer model is the same as OSI"

**Wrong.** The 5-layer hybrid model splits the DoD link layer back into L1 and L2 (matching OSI), but fuses L5/L6/L7 into one "application" layer (matching DoD). It is its own thing. **Fixed:** 7-layer OSI, 5-layer hybrid, and 4-layer DoD are three distinct models.

### "ARP is L2"

**Mostly right but not totally clean.** ARP frames have an EtherType (0x0806) and ride directly on Ethernet, so they look L2. But ARP's purpose is to discover an L3-to-L2 mapping, so it sits across the boundary. Some textbooks call it L2.5. **Fixed:** ARP is "between L2 and L3." It is a glue protocol.

### "IPv6 still uses ARP"

**Wrong.** IPv6 replaced ARP with **Neighbor Discovery Protocol (NDP)**, which rides on **ICMPv6**. So in IPv6, the L2-to-L3 mapping job moves up from a standalone L2.5 protocol to a sub-protocol of L3 ICMPv6. **Fixed:** IPv6 uses NDP / ICMPv6 for what ARP did in IPv4.

### "L4-L7 traffic" in load balancers means specific layers

**Mostly right.** "L4 traffic" means the load balancer routes by IP and port without inspecting payload. "L7 traffic" means the load balancer inspects HTTP headers, paths, cookies, TLS SNI. "L4-L7" just means "we do both depending on policy." **Fixed:** It is marketing speak for "we look at this much of each packet."

### "Switches are L2, routers are L3"

**Mostly right.** Classic switches forward by MAC (L2). Classic routers forward by IP (L3). But "L3 switch" is a thing (a router optimized for LAN), and "L2 router" usually means "bridge." **Fixed:** Modern boxes blur the line; specs matter more than the noun.

### "X.25 / ATM / Frame Relay are TCP/IP"

**Wrong.** X.25 was an early packet-switched WAN technology, mostly in the 1970s-1990s. ATM (Asynchronous Transfer Mode) used 53-byte cells and was thought to be the future of telecom and computing in the 1990s; it survives in some carrier networks. Frame Relay was the link-layer for many leased lines until killed by MPLS and Ethernet WAN. None of these are TCP/IP. **Fixed:** They are alternative L2/L3 technologies that mostly lost to Ethernet + IP.

### "What is Layer 0?"

**Not standard, but useful.** Some engineers call the dark fiber, wavelength assignments, and physical infrastructure "Layer 0." A 100 Gbps wavelength on a DWDM system, the leased dark fiber pair between two data centers, the ocean cable system in the Atlantic — these are below L1. **Fixed:** "L0" is a slang term for the physical infrastructure that L1 PHYs ride on.

### "Encapsulation overhead is small"

**Depends.** For a full-MTU TCP/IPv4 over Ethernet packet: 14 (Ethernet) + 20 (IP) + 20 (TCP) + 4 (CRC) = 58 bytes of headers around 1460 bytes of payload, about 4% overhead. For a tiny TCP segment carrying one keystroke (1 byte), the overhead is 5800%. **Fixed:** Overhead matters more for small packets and tunneled traffic.

### "Headers are always 20 bytes"

**Wrong.** IPv4 header is 20-60 bytes (5-15 32-bit words; options bring it past 20). IPv6 header is exactly 40 bytes, but extension headers stack on top. TCP header is 20-60 bytes (5-15 32-bit words; options like timestamps, SACK, MSS, window scaling). UDP header is exactly 8 bytes. Ethernet header is 14 bytes (without VLAN tag) or 18 bytes (with one 802.1Q tag). **Fixed:** Header sizes vary; only UDP is truly fixed.

### "Switches do not change packets"

**Wrong (but mostly right).** A pure L2 switch forwards frames untouched. A managed switch can rewrite VLAN tags, prioritize by 802.1p bits, count frames, and apply ACLs. An L3 switch decrements TTL and rewrites MACs (because it routes). **Fixed:** Even L2 switches sometimes rewrite frames; L3+ definitely do.

### "Firewalls block traffic. That's it."

**Wrong.** Firewalls do many things: stateless packet filter (L3/L4), stateful connection tracking (L3/L4 with state), NAT (L3/L4 rewriting), proxy / ALG (L7 inspection), IDS/IPS (deep packet inspection at all layers), VPN endpoint (L3 IPsec or L4 OpenVPN). **Fixed:** "Firewall" is a category with many capabilities.

### "MPLS is a routing protocol"

**Wrong.** MPLS is a forwarding mechanism that uses pre-computed labels. The labels are distributed by routing protocols (LDP, RSVP-TE, BGP labeled-unicast). MPLS is sometimes called "L2.5" because it sits between L2 and L3. **Fixed:** MPLS forwards. Routing protocols decide what to forward.

### "QUIC replaces TCP"

**Partly.** QUIC is a competing transport that delivers reliable streams over UDP, with built-in TLS 1.3, multiplexing, 0-RTT, and connection migration. HTTP/3 runs over QUIC. But TCP is not going anywhere — most servers still talk TCP, lots of protocols live on TCP, and many networks block UDP. **Fixed:** QUIC is rising, TCP is durable, both will coexist for decades.

## Vocabulary

A long table of words that show up in every networking conversation. One line each, in plain English.

- **layer** — one floor of the model. Each layer has one job.
- **protocol** — a set of rules two computers agree on so they can talk.
- **PDU** — Protocol Data Unit. The chunk of data this layer sends. Each layer has a different name (frame, packet, segment, etc.).
- **SDU** — Service Data Unit. The chunk *handed down* from the layer above before this layer wraps it.
- **encapsulation** — wrapping the chunk from the layer above with this layer's header (and maybe trailer).
- **decapsulation** — the receiver unwrapping each layer in turn.
- **header** — a small label at the front of a chunk telling the receiver what to do with it.
- **trailer** — a small label at the back of a chunk, often a checksum.
- **payload** — the actual data the application cared about, before headers were added.
- **frame** — the L2 chunk. Has source MAC, destination MAC, payload, CRC.
- **packet** — the L3 chunk. Has source IP, destination IP, payload.
- **segment** — the L4 chunk in TCP. Has source port, dest port, sequence number, etc.
- **datagram** — the L4 chunk in UDP. Has source port, dest port, length, checksum.
- **ATM cell** — a 53-byte fixed-size unit used by old ATM networks. Now niche.
- **MAC address** — 48-bit hardware address of a network card. Local-only identifier.
- **OUI** — Organizationally Unique Identifier. The first 3 bytes of a MAC, identifies the vendor.
- **EUI-48** — 48-bit Extended Unique Identifier. Same thing as a classic MAC.
- **EUI-64** — 64-bit version, used in IPv6 SLAAC interface IDs.
- **IPv4 address** — 32-bit network identifier, like 192.168.1.1.
- **IPv6 address** — 128-bit network identifier, like 2001:db8::1.
- **port** — a 16-bit number identifying a service on a host. 0-65535.
- **well-known port** — 0-1023. Reserved for standard services like HTTP (80) and SSH (22).
- **registered port** — 1024-49151. Assigned by IANA to specific applications.
- **ephemeral port** — 49152-65535. Picked at random by clients for outgoing connections.
- **socket** — the combination of an IP and a port, like 192.168.1.42:51234.
- **BSD socket API** — the standard API in C and many other languages for opening network connections.
- **TLS** — Transport Layer Security. Encryption and authentication for TCP-based protocols.
- **DTLS** — Datagram TLS. Same idea for UDP.
- **SSL** — Secure Sockets Layer, the deprecated old name for TLS.
- **HTTP** — HyperText Transfer Protocol. Web pages and APIs.
- **HTTP/1.1** — text-based version, one request per connection (with keepalive). RFC 9112.
- **HTTP/2** — binary, multiplexed over one TCP connection. RFC 9113.
- **HTTP/3** — HTTP over QUIC over UDP. RFC 9114.
- **HTTPS** — HTTP wrapped in TLS.
- **FTP** — File Transfer Protocol. Old. Mostly replaced.
- **SMTP** — Simple Mail Transfer Protocol. Sending mail.
- **IMAP** — Internet Message Access Protocol. Reading mail server-side.
- **POP3** — Post Office Protocol. Downloading mail and deleting from server.
- **SSH** — Secure Shell. Encrypted remote login.
- **Telnet** — old plaintext remote login. Do not use.
- **DNS** — Domain Name System. Names to IPs.
- **mDNS** — Multicast DNS. Local-network name discovery without a server.
- **DHCP** — Dynamic Host Configuration Protocol. Hands out IPs.
- **BOOTP** — old DHCP precursor.
- **NTP** — Network Time Protocol. Clock sync.
- **SNMP** — Simple Network Management Protocol. Device monitoring.
- **IGMP** — Internet Group Management Protocol. IPv4 multicast group joins.
- **ARP** — Address Resolution Protocol. IPv4 address to MAC mapping.
- **RARP** — Reverse ARP. Old, deprecated.
- **NDP** — Neighbor Discovery Protocol. The IPv6 replacement for ARP, runs on ICMPv6.
- **ICMP** — Internet Control Message Protocol. Diagnostic and error messages for IPv4.
- **ICMPv6** — same idea for IPv6, and also carries NDP and MLD.
- **MLD** — Multicast Listener Discovery. IPv6 multicast group joins.
- **IP** — Internet Protocol. The L3 protocol.
- **IPv4** — 32-bit IP. RFC 791. 1981.
- **IPv6** — 128-bit IP. RFC 8200.
- **TCP** — Transmission Control Protocol. Reliable byte stream. RFC 793. 1981.
- **UDP** — User Datagram Protocol. Unreliable datagrams. RFC 768. 1980.
- **QUIC** — Encrypted reliable transport on top of UDP. RFC 9000. 2021.
- **SCTP** — Stream Control Transmission Protocol. Multistream transport. RFC 4960.
- **DCCP** — Datagram Congestion Control Protocol. RFC 4340. Niche.
- **RDP** — Reliable Datagram Protocol. RFC 908. Historic.
- **Ethernet** — the dominant L1+L2 wired LAN tech. IEEE 802.3.
- **IEEE 802.3** — Ethernet standards.
- **802.1Q VLAN tag** — 4-byte tag adding a VLAN ID to Ethernet frames.
- **802.1ad QinQ** — stacks two 802.1Q tags. For service providers.
- **802.1X** — port-based authentication.
- **802.11** — WiFi family.
- **802.15** — Bluetooth, Zigbee, low-rate wireless.
- **802.16** — WiMax. Mostly gone.
- **STP** — Spanning Tree Protocol. Stops L2 loops. IEEE 802.1D.
- **RSTP** — Rapid Spanning Tree. Faster STP. IEEE 802.1w.
- **MSTP** — Multiple Spanning Tree. Per-VLAN topology. IEEE 802.1s.
- **LACP** — Link Aggregation Control Protocol. Bundles multiple links. IEEE 802.3ad.
- **LAG** — Link Aggregation Group. Bundle of physical links acting as one logical link.
- **ECMP** — Equal-Cost Multi-Path. L3 load-spreading across equal-cost routes.
- **PoE** — Power over Ethernet. Sends power on the data cable.
- **fiber** — glass cable carrying laser light.
- **copper** — metal cable carrying electrical signal.
- **twisted pair** — two wires twisted to cancel interference.
- **coax** — center conductor + shielding. Used in cable modems and some test gear.
- **BNC** — bayonet connector for coax.
- **RJ45** — 8-pin connector for twisted pair Ethernet.
- **SFP** — Small Form-factor Pluggable transceiver. 1G.
- **SFP+** — 10G version.
- **SFP28** — 25G version.
- **QSFP+** — 4-lane 10G = 40G.
- **QSFP28** — 4-lane 25G = 100G.
- **QSFP-DD** — 8-lane = 400G.
- **DAC** — Direct Attach Cable. Copper cable with built-in transceivers.
- **AOC** — Active Optical Cable. Fiber cable with built-in transceivers.
- **MTU** — Maximum Transmission Unit. Largest L3 payload that fits in one L2 frame.
- **fragmentation** — splitting a packet into smaller pieces because it does not fit.
- **MSS** — Maximum Segment Size. Largest TCP payload (no headers). MTU - IP - TCP.
- **TCP RTT** — round-trip time, measured in milliseconds.
- **OSI model** — the 7-layer reference model. ISO/IEC 7498. 1984.
- **ITU-T X.200** — the ITU's name for the same OSI reference model.
- **DoD model** — the 4-layer TCP/IP model used by the actual internet.
- **RFC 1122** — Host Requirements: communication layers.
- **RFC 1123** — Host Requirements: application and support.
- **RFC 791** — Internet Protocol (IPv4). 1981.
- **RFC 793** — Transmission Control Protocol. 1981.
- **RFC 768** — User Datagram Protocol. 1980.
- **RFC 826** — Address Resolution Protocol. 1982.
- **RINA** — Recursive InterNetwork Architecture. Day's alternative model.
- **IPC** — Inter-Process Communication. RINA's universal abstraction.
- **recursive** — a function that calls itself; in RINA, layering the same function at different scopes.
- **reachability** — whether a destination can be reached at all.
- **flow** — a stream of related packets between two endpoints.
- **scope** — the reach of a network — a LAN, a site, a continent, the internet.
- **layer 8** — joke layer for "the user / political." Where many real outages live.
- **encapsulation overhead** — the bytes added by all the headers, as a percent of total.
- **header chain** — IPv6's stack of extension headers between the main IPv6 header and the upper layer.
- **Hop-by-Hop options** — IPv6 extension processed by every router.
- **Routing header** — IPv6 extension specifying intermediate hops.
- **Fragment header** — IPv6 extension marking pieces of a fragmented packet.
- **Authentication Header (AH)** — IPsec authentication header.
- **ESP** — Encapsulating Security Payload. IPsec encryption header.
- **Destination Options** — IPv6 extension processed only by the destination.
- **end-to-end principle** — RFC concept that intelligence belongs at the edges, not the middle.
- **dumb network smart hosts** — restatement of end-to-end. The network just forwards; hosts do the work.
- **NAT** — Network Address Translation. Rewrites IP and port to share one public IP among many private hosts.
- **CGNAT** — Carrier-Grade NAT. Multiple layers of NAT inside an ISP.
- **port forwarding** — telling NAT "incoming traffic to port X should go to internal host Y."
- **conntrack** — Linux's connection tracking subsystem.
- **stateful firewall** — tracks connections and allows replies automatically.
- **stateless filter** — matches per-packet, no state.
- **ALG** — Application Layer Gateway. NAT helper that peeks at L7 (FTP, SIP).
- **DPI** — Deep Packet Inspection. Looking inside the L7 payload.
- **IP/MPLS** — IP traffic carried over MPLS labels in carrier networks.
- **Layer 2.5** — informal name for MPLS, sitting between L2 and L3.
- **bridge** — a device that forwards L2 frames between segments. Old word for L2 switch.
- **gateway** — a device at the boundary of a network. Often the default route.
- **default route** — `0.0.0.0/0` (IPv4) or `::/0` (IPv6). Where to send packets that have no specific match.
- **routing table** — the list of known networks and which next hop to use.
- **next hop** — the immediate L3 neighbor a packet should be sent to.
- **link-local address** — IPv6 `fe80::/10` or IPv4 `169.254.0.0/16`. Only valid on the same link.
- **loopback** — `127.0.0.1` or `::1`. Talks to yourself.
- **broadcast** — IPv4 only. Send to every host on the local network.
- **multicast** — send to a group of subscribers.
- **unicast** — send to one specific host.
- **anycast** — send to whichever member of a group is "nearest."
- **subnet** — a chunk of the IP address space.
- **CIDR** — Classless Inter-Domain Routing. The slash notation `192.168.1.0/24`.
- **prefix length** — the number after the slash.
- **netmask** — older way to express the same idea, `255.255.255.0`.
- **gateway address** — the IP of the router on your local subnet.
- **AS** — Autonomous System. A network under one administrative control. BGP speaks between ASes.
- **ASN** — AS Number. 16-bit (legacy) or 32-bit (modern). Identifies an AS.
- **PMTUD** — Path MTU Discovery. Find the smallest MTU on the path.
- **DF bit** — Don't Fragment bit in IPv4 header. Routers must not fragment if set.
- **ECN** — Explicit Congestion Notification. 2 bits in IP header signaling congestion.
- **DSCP** — Differentiated Services Code Point. 6 bits in IP header for QoS classes.
- **QoS** — Quality of Service. Network prioritization.
- **VPN** — Virtual Private Network. A secure tunnel across an untrusted network.
- **IPsec** — IP-layer encryption and authentication. RFC 4301.
- **WireGuard** — modern simple VPN protocol over UDP.
- **OpenVPN** — older popular VPN over TCP or UDP.
- **GRE** — Generic Routing Encapsulation. Tunnel protocol. RFC 2784.
- **VXLAN** — Virtual Extensible LAN. L2 over L3 tunnel for datacenter overlays.
- **firewall** — a device or process that filters traffic.
- **proxy** — a device that intermediates connections, often at L7.
- **reverse proxy** — a proxy in front of a server, accepting connections on its behalf.
- **load balancer** — a device that spreads traffic across many backends.
- **NIC** — Network Interface Card. The chip and port that connects a host to a network.
- **PHY** — Physical layer chip. Turns digital bits into analog signals.
- **MAC layer** — the L2 layer in IEEE terminology.
- **PHY layer** — the L1 layer in IEEE terminology.
- **carrier sense** — the technique by which Ethernet stations check if the wire is busy.
- **CSMA/CD** — Carrier Sense Multiple Access with Collision Detection. Old half-duplex Ethernet.
- **CSMA/CA** — Carrier Sense Multiple Access with Collision Avoidance. WiFi.
- **half-duplex** — only one direction at a time. Old Ethernet over a hub.
- **full-duplex** — both directions at once. Modern switched Ethernet.
- **autonegotiation** — the L1 dance where two ends pick a common speed and duplex.
- **link** — a physical connection between two endpoints.
- **lossy link** — a link with significant packet loss.
- **lossless link** — a link with negligible loss.

That is a deep starter vocabulary. Each entry should be enough to follow a conversation; the deep dives live in their own sheets.

## Try This

A short list of experiments that drive each layer's lesson home.

```
$ ping -c 1 8.8.8.8
$ ping -M do -s 1472 -c 1 8.8.8.8
$ traceroute -n 8.8.8.8
$ mtr -n -c 5 8.8.8.8
$ dig +short example.com
$ curl -v https://example.com 2>&1 | grep -E '^[<>*]'
$ ss -tnlp
$ ip -s link show
$ ip neigh show
$ openssl s_client -connect example.com:443 -servername example.com </dev/null 2>&1 | grep -E 'subject|issuer'
```

Then, for each command, ask yourself: which OSI layer is this exercising? Most exercise more than one. `curl` exercises every layer from 1 through 7.

## Where to Go Next

After you understand the layered model, you should learn the actual protocols at each layer.

- L1/L2 deep dive: `ramp-up/linux-kernel-eli5`, `networking/ethernet`.
- L3 deep dive: `ramp-up/ip-eli5`, `networking/ip`, `networking/ipv4`, `networking/ipv6`, `networking/icmp`, `ramp-up/icmp-eli5`.
- L4 deep dive: `ramp-up/tcp-eli5`, `networking/tcp`, `ramp-up/udp-eli5`, `networking/udp`.
- L6/L7 deep dive: `ramp-up/tls-eli5`, `ramp-up/dns-eli5`, `networking/dns`.
- The big picture: `fundamentals/how-networking-works`, `fundamentals/how-the-internet-works`.

Read those sheets in the order above. By the end you will have a complete mental model of how a packet travels from your laptop to the other side of the world and back.

## See Also

- `networking/ip`
- `networking/ipv4`
- `networking/ipv6`
- `networking/tcp`
- `networking/udp`
- `networking/icmp`
- `networking/dns`
- `networking/ethernet`
- `networking/arp`
- `networking/mpls`
- `fundamentals/how-networking-works`
- `fundamentals/how-the-internet-works`
- `ramp-up/ip-eli5`
- `ramp-up/tcp-eli5`
- `ramp-up/udp-eli5`
- `ramp-up/icmp-eli5`
- `ramp-up/dns-eli5`
- `ramp-up/tls-eli5`
- `ramp-up/linux-kernel-eli5`

## References

- ISO/IEC 7498-1:1994 — *Information technology — Open Systems Interconnection — Basic Reference Model: The Basic Model* (the official OSI reference).
- ITU-T Recommendation X.200 — the ITU's identical text of the OSI reference model.
- RFC 1122 — *Requirements for Internet Hosts -- Communication Layers*.
- RFC 1123 — *Requirements for Internet Hosts -- Application and Support*.
- RFC 791 — *Internet Protocol* (IPv4). 1981.
- RFC 793 — *Transmission Control Protocol*. 1981.
- RFC 768 — *User Datagram Protocol*. 1980.
- RFC 826 — *An Ethernet Address Resolution Protocol*. 1982.
- RFC 8200 — *Internet Protocol, Version 6 (IPv6) Specification*.
- RFC 8446 — *The Transport Layer Security (TLS) Protocol Version 1.3*.
- RFC 9000 — *QUIC: A UDP-Based Multiplexed and Secure Transport*. 2021.
- RFC 9110 — *HTTP Semantics*.
- RFC 9112 — *HTTP/1.1*.
- RFC 9113 — *HTTP/2*.
- RFC 9114 — *HTTP/3*.
- W. Richard Stevens, *TCP/IP Illustrated, Volume 1: The Protocols* (2nd ed., Fall et al., 2011) — the canonical packet-by-packet walkthrough.
- Russ White and Ethan Banks, *Computer Networking Problems and Solutions* (Addison-Wesley, 2017) — modern survey organized by problem rather than protocol.
- John Day, *Patterns in Network Architecture: A Return to Fundamentals* (Prentice Hall, 2008) — the RINA book.
- IEEE 802.3 — Ethernet standards.
- IEEE 802.11 — WiFi standards.
- IEEE 802.1Q — VLAN tagging.
- `ip(8)`, `ss(8)`, `tcpdump(8)`, `ethtool(8)`, `traceroute(8)`, `mtr(8)`, `dig(1)`, `curl(1)`, `openssl(1ssl)` — Linux man pages for the commands in this sheet.
