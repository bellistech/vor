# UDP — ELI5 (Paper Airplanes Out the Window)

> UDP is throwing paper airplanes out the window. You write your message on the airplane, you fold it, you toss it, you hope it lands. No reply. No retry. No promise. Just a fast little airplane in the wind.

## Prerequisites

(none — but `cs ramp-up linux-kernel-eli5` and `cs ramp-up tcp-eli5` will help if you want to see what UDP is *not*)

This sheet is the very first stop for understanding UDP. You do not need to know networking. You do not need to know what an IP address is. You do not need to have ever heard the word "protocol" before. By the end of this sheet you will know all of those things, in plain English, and you will have typed real commands into a real terminal and watched real packets fly through the air.

If a word feels weird, look it up in the **Vocabulary** table near the bottom. Every weird word in this sheet is in that table with a one-line plain-English definition.

If you see a `$` at the start of a line in a code block, that means "type the rest of this line into your terminal." You do not type the `$`. The lines underneath that don't have a `$` are what your computer prints back at you. We call that "output."

If you have already read `tcp-eli5`, this sheet will feel like the opposite world. TCP is certified mail with tracking numbers and signature requirements and a guy in a uniform standing at the door. UDP is throwing paper airplanes out the window of a moving car. Both can deliver messages. Each is good for very different things. Once both pictures live in your head, the rest of networking starts to make sense.

## Plain English

### Imagine your computer wants to send a message

Picture two houses across the street from each other. You are in one house. Your friend is in the other house. You want to send your friend a message.

You could walk over and knock on the door. You could say "hi, are you home? hi, are you ready to talk? hi, here is my message. did you get it? say yes if you got it." That is one way. It is careful. It is slow. It uses a lot of words. Every step has to work or you start over.

Or, you could write a tiny note on a paper airplane, walk to your window, and throw it. Maybe it lands on your friend's lawn. Maybe a gust of wind takes it to the neighbor's bush. Maybe their dog grabs it. Maybe it lands perfectly on their porch and they pick it up and read it. You don't know. You did not wait. You did not ask. You did not even check that your friend was home.

The careful, slow, knock-on-the-door way is **TCP.** It is reliable. It is ordered. It is acknowledged. It is the boring grown-up way to talk.

The paper-airplane way is **UDP.** It is fast. It is light. It is "just send it." It does not promise the airplane will land. It does not promise your friend will read it. It does not even tell you if your friend was home.

This sheet is about that paper airplane.

### What does UDP stand for?

UDP stands for **User Datagram Protocol.** Three boring words. Let's pick them apart.

- **User** — UDP is for user-space programs. Anything you run, like your DNS lookup tool, your video call app, your game, your IoT sensor. They send UDP from user space.
- **Datagram** — a datagram is a single self-contained packet. A "telegram" of data. One package, one shot, complete in itself. A datagram is the paper airplane. The airplane is whole. It does not need to be combined with other airplanes to make sense. You read the airplane, you know what the message is.
- **Protocol** — a protocol is just an agreement. "If both sides agree to do X, Y, Z in order, then we can talk." UDP is the agreement: every paper airplane has a tiny header on it that says "this is from port A, going to port B, this many bytes long, here is a checksum so you know it didn't get smudged." That's it. That's the whole agreement.

So **UDP = a way of throwing self-contained packets between programs without any handshakes, without retransmission, without ordering, without flow control, without congestion control, without anything except the bare minimum to get the bits there if they happen to make it.**

### A factory floor with two ways to send orders

Picture a giant factory. Inside the factory, two managers are at opposite ends of the building. They need to send each other messages all day long.

Manager A could pick up the phone, dial Manager B, wait for "hello, this is Manager B," exchange pleasantries, slowly read the message, ask "did you get that," wait for "yes I got it," and hang up. That is the careful telephone way. That is TCP.

Or, Manager A could write a note on a piece of paper, fold it into a paper airplane, walk to the window of their office, and toss it across the factory floor. Maybe a worker catches it and walks it over. Maybe a forklift runs it over. Maybe it lands in a coffee cup. Either way, Manager A goes back to work. That is UDP.

Why would a smart manager ever pick the airplane? Because some messages are not worth the phone call. "It's two o'clock" — that's a one-shot fact. By the time the phone rings, three different rings, and a polite hello, two o'clock has already passed. Just throw the airplane. If it lands, great. If not, you'll hear the right time again in a minute anyway.

This is the heart of UDP. **Sometimes the conversation overhead is worse than the message itself.** Sometimes "hope it lands, throw another if not, no big deal" is the right answer.

### A radio DJ versus a phone call

Here is another picture. A radio DJ talks into a microphone. The radio waves go out in every direction. Anybody with a radio in range can hear. The DJ does not know who is listening. The DJ does not know if anybody is listening. The DJ does not stop and ask "did you hear that?" The DJ just keeps talking. If your radio is off, you miss it. If your radio crackles, you miss a word. The DJ moves on.

That is UDP. The packets go out. Maybe they arrive. Maybe they don't. The sender keeps going.

A phone call is the opposite. The line is open. Both sides have to be there. If the line breaks, the call drops, and you both notice. Each word goes back and forth. If the other person can't hear you, they say "what?" and you repeat yourself.

That is TCP. The connection has to exist. Both sides agree to it. If it breaks, both sides know. Every word gets a reply.

UDP is the radio DJ. TCP is the phone call.

### Why would anyone choose unreliable?

This is the question that confuses everybody at first. Why on earth would you want a protocol that doesn't even promise to deliver your message?

Three big reasons.

**Reason 1: Speed.** Every conversation takes time. The handshake at the start. The acknowledgments after every chunk. The teardown at the end. All of that is overhead. If you don't need it, skipping it is a huge speed win. A DNS query (looking up a website's address) is a tiny question with a tiny answer. If you do TCP, you spend more time setting up the handshake than actually asking and answering. UDP just throws the question, gets the answer back, and you're done. Way faster.

**Reason 2: Late is worse than lost.** This is the magic insight. For some kinds of data, a late packet is worse than a missing one. Think about a video call. You are watching your friend's face. The packet that has the picture for second 1.000 is supposed to be on your screen at second 1.000. If it gets stuck in traffic and arrives at second 1.500, do you want it? No! By the time it arrives, your screen is already showing second 1.500's picture. Showing the old picture would make the video go backwards. Just drop the late packet. Move on. The codec will smooth over the missing frame so smoothly you might not even notice.

**Reason 3: Many-to-many.** TCP is one-to-one. Two endpoints, one connection. UDP can be one-to-many. The radio DJ. A single packet can be sent to a special "many listeners" address (called a **multicast** address) and a hundred listeners get it without the sender knowing or caring how many listeners there are. TCP literally cannot do this. It is a private phone call. UDP is the radio broadcast.

So unreliable is sometimes a feature, not a bug. It is an opportunity to skip work the application does not need.

### What UDP does NOT do

Let's be very clear about everything UDP does not give you. This is important. People get burned when they assume UDP will do things it does not do.

UDP does not have **connections.** There is no "open" or "close." There is no "are you there?" There is no state shared between the two sides. Every packet is on its own.

UDP does not have **retransmission.** If a packet gets lost in the network, UDP does not notice and does not send another copy. The packet is just gone. Forever.

UDP does not have **acknowledgments.** The receiver does not say "I got it." The sender does not wait for any reply. UDP is fire and forget.

UDP does not have **ordering.** If you send packet A and then packet B, the receiver might get B first and A second. UDP does not put them back in order. It just hands them up to the application in whatever order they arrived.

UDP does not have **flow control.** If the receiver is slow and the sender is fast, the sender does not slow down to match. The receiver's buffer fills up and then packets get dropped. The sender does not know.

UDP does not have **congestion control.** If the network is busy, UDP does not slow down. It just keeps blasting at full speed. This can make congestion worse and cause real problems on shared networks.

UDP does not have **error correction.** It has a checksum so it can detect a corrupted packet. But all it does with a corrupt packet is throw it away. It does not try to fix it.

UDP does not have **session management.** No login, no session ID, no cookies, no nothing. Each packet is alone in the universe.

UDP does not have **encryption.** A UDP packet flies through the network in plain text unless the application layer adds encryption on top.

That is a lot of "does not." It feels like UDP is missing everything important. And, well, sometimes it is. But sometimes that is exactly what you want, because you can build only the parts you need on top.

### What UDP does give you

Despite all the "does not," UDP gives you a few important things.

UDP gives you **speed.** Sending a UDP packet is just "build header, build payload, hand it to the kernel, done." No handshake, no waiting. The packet hits the wire as fast as the kernel can build it.

UDP gives you **multiplexing.** Multiple applications on the same machine can listen on different UDP ports without bumping into each other. Port 53 is DNS. Port 123 is NTP. Port 5060 is SIP. The port number tells the kernel which application gets the packet. This is the same trick TCP uses, but for UDP it costs almost nothing.

UDP gives you **datagram boundaries.** If you `sendto()` a UDP packet of 100 bytes, the receiver does a `recvfrom()` and gets exactly 100 bytes. The boundary between packets is preserved. With TCP this is not true; TCP is a byte stream and you can read 50 bytes now and 50 bytes later from one 100-byte send. With UDP, one send equals one recv. This makes some things easier.

UDP gives you **support for one-to-many.** Multicast and broadcast both work with UDP. If you want to send a single packet to many receivers, UDP is your friend.

UDP gives you **a tiny header.** Just 8 bytes. Compare to TCP's 20-byte minimum (and often more with options). For tiny messages, the header overhead matters.

UDP gives you **freedom.** Build whatever reliability you actually need on top. Maybe you need order but not retries. Maybe you need retries but not order. Maybe you need encryption but no congestion control. With UDP you pick the menu. With TCP you get the whole prix fixe whether you wanted it or not.

### Why this matters: a real comparison

Imagine you're sending a single 100-byte message, and you have two choices.

**Choice A: TCP.** Your kernel does this:

1. Send a SYN packet. Wait for SYN-ACK from the other side. Send an ACK back. (One round trip just to open a connection.)
2. Send your 100-byte message in a data packet.
3. Wait for an ACK for that data.
4. (Your application now reads the response — assume that's another 100 bytes.)
5. Send an ACK for the response.
6. Send a FIN to close. Wait for FIN-ACK. Send the final ACK.

Total: at least 7 packets exchanged just to send one message and get one response. Plus a round-trip wait for the handshake before any data can move. If the round-trip time between you and the peer is 100 milliseconds, the TCP exchange takes at least 300 milliseconds (one for handshake, one for data and response, one for close). Your data was on the wire for one round trip; the protocol ate the other two.

**Choice B: UDP.** Your kernel does this:

1. Send your 100-byte message in a UDP packet.
2. Receive the 100-byte response.

Total: 2 packets. Latency: one round trip, 100 milliseconds. The protocol ate zero extra round trips.

For tiny exchanges, UDP is roughly three times faster than TCP just from skipping the handshake and teardown. This is why DNS uses UDP. Every web page load involves a DNS query before anything else can happen, and you really don't want to wait an extra 200 milliseconds just to ask "what's google.com's IP?"

### A picture of the difference

```
   TCP (one tiny request):                UDP (one tiny request):

   client ----- SYN ------> server        client --- DATA ------> server
   client <--- SYN-ACK ---- server        client <--- DATA ------ server
   client ----- ACK ------> server        (done — 2 packets)
   client ---- DATA ------> server
   client <--- ACK -------- server
   client <--- DATA ------- server
   client ---- ACK -------> server
   client ----- FIN ------> server
   client <--- FIN-ACK ---- server
   client ----- ACK ------> server
   (done — 10 packets)
```

If you only have 100 bytes to say, UDP is the obvious answer.

### Where UDP lives in the stack

Computer networks are built in layers. From bottom to top:

- **Physical layer** (cables, radio, fiber) — the actual wire or air.
- **Link layer** (Ethernet, Wi-Fi) — how packets move between two adjacent devices.
- **Network layer** (IP) — how packets move across many networks. IP says "this packet is going to this address."
- **Transport layer** (TCP, UDP, QUIC) — how programs on different machines talk. TCP and UDP both live here.
- **Application layer** (DNS, HTTP, RTP, etc.) — what the actual programs are saying.

UDP is layer 4, the transport layer, sitting right on top of IP. When a program sends a UDP packet, the kernel wraps it in an IP packet, the IP packet gets wrapped in an Ethernet frame (or Wi-Fi frame), and out it goes. On the other side, the layers get unwrapped in reverse, and the receiving program gets a UDP packet.

Picture:

```
   APPLICATION (DNS, video, game, IoT sensor, ...)
        |
        v
   UDP (8-byte header: src port, dst port, length, checksum)
        |
        v
   IP (source address, destination address, TTL, ...)
        |
        v
   ETHERNET / Wi-Fi / cellular (physical packet)
        |
        v
   THE WIRE (or air)
```

Each layer wraps the next layer's packet inside its own header. Kind of like nested envelopes. The post office reads the outermost envelope. When it arrives at your apartment building, the doorman reads the next envelope. When it arrives at your apartment, you read the innermost note. UDP is one of those middle envelopes.

## The UDP Header (8 Bytes Total!)

This is one of the most beautiful things in networking. The whole UDP header is 8 bytes. Eight. Bytes. That is the full agreement. The entire spec for UDP, RFC 768, is three pages long. The TCP spec, by contrast, is over a hundred pages.

Here is the UDP header, byte by byte.

```
   0                   1                   2                   3
   0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
  +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
  |          Source Port          |       Destination Port        |
  +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
  |             Length            |           Checksum            |
  +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
  |                        Payload (data)                         |
  +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

Let's go field by field.

**Source Port (2 bytes, 16 bits).** A number from 0 to 65535. This says "the program that sent this packet is listening on this port." When the receiver wants to reply, it knows what port to send back to. Sometimes the source port is "ephemeral" (a random high number the kernel picks just for this exchange) and sometimes it is a well-known port (like 53 for a DNS server replying to a query).

**Destination Port (2 bytes, 16 bits).** Another number from 0 to 65535. This says "deliver this packet to the program listening on this port." The kernel on the receiving side uses this number to find the right program. Port 53 means DNS. Port 123 means NTP. Port 5060 means SIP.

**Length (2 bytes, 16 bits).** Total length of the UDP packet, header plus payload, in bytes. Minimum is 8 (header only, empty payload). Maximum is 65535 (because 16 bits). In practice you almost never want a UDP packet that big; you want it to fit in one IP packet, which is usually limited by the path MTU (around 1500 bytes for Ethernet). Big UDP packets get fragmented, which is bad. But the field can technically hold up to 65535.

**Checksum (2 bytes, 16 bits).** A simple math thing the sender computes over the header and payload (and a "pseudo-header" that includes the IP addresses). The receiver re-computes it and checks. If they don't match, the packet got smudged in flight, and the receiver throws it away. The checksum is optional in IPv4 (a value of 0 means "I didn't bother computing it") but mandatory in IPv6.

**Payload.** The actual data. Whatever the application wanted to send. Could be a DNS query. Could be a video frame. Could be a game position update. Could be 1 byte. Could be 1400 bytes. Could be empty (just an 8-byte header with no payload, perfectly legal).

### How does UDP compare to TCP's header?

TCP has way more in its header. Here is the comparison:

```
              UDP (8 bytes)              vs              TCP (20+ bytes)

   +-------+-------+-------+-------+        +-------+-------+-------+-------+
   | src port      | dst port      |        | src port      | dst port      |
   +-------+-------+-------+-------+        +-------+-------+-------+-------+
   | length        | checksum      |        | sequence number               |
   +-------+-------+-------+-------+        +-------+-------+-------+-------+
   | payload...                    |        | acknowledgment number         |
   +-------+-------+-------+-------+        +-------+-------+-------+-------+
                                            | hdrlen|flags  | window size   |
                                            +-------+-------+-------+-------+
                                            | checksum      | urgent ptr    |
                                            +-------+-------+-------+-------+
                                            | options (variable, 0-40 bytes)|
                                            +-------+-------+-------+-------+
                                            | payload...                    |
                                            +-------+-------+-------+-------+
```

UDP has 4 fields. TCP has 10. UDP fits in two rows. TCP needs five (and that is before any options). Each TCP field is doing something: tracking sequence numbers, tracking acknowledgments, advertising window sizes, signaling SYN/ACK/FIN flags. All of that work is what TCP does to give you reliability and ordering.

UDP just has source port, destination port, length, and checksum. That is the whole airplane.

### Why is the header so small?

Because UDP does not need to track anything. There is no connection state. There are no sequence numbers (UDP has no sense of "this is packet number 5 of 10"). There are no acknowledgment numbers (UDP never acknowledges anything). There are no window sizes (UDP does not throttle itself). There is just enough info to deliver the packet to the right application on the right machine, and a checksum to detect corruption.

The smallness of the header is part of UDP's appeal for tiny messages. If you are sending a 4-byte sensor reading from a battery-powered IoT thermometer, you do not want 40 bytes of TCP header overhead. You want 8 bytes of UDP header so the actual ratio of useful data to overhead is reasonable.

### What about UDP-Lite?

There is a variant called **UDP-Lite** (RFC 3828) that lets you say "compute the checksum over only the first N bytes of the packet, ignore the rest." This is useful for things like video where you would rather have a slightly corrupted frame than no frame at all. UDP-Lite is rarely used in practice, but it exists and you might see it mentioned. It uses IP protocol number 136 instead of UDP's 17. For 99% of work, you will never touch it.

### Why is the checksum in a "pseudo-header"?

This is one of those weird design decisions that confuses everyone the first time. The UDP checksum is computed over the UDP header, the UDP payload, AND a "pseudo-header" that includes some IP-layer fields (source IP, destination IP, protocol, UDP length).

Why? Because the designers wanted the receiver to detect a packet that got delivered to the wrong machine. If the source or destination IP got corrupted in transit, but the UDP fields were still valid, you'd accept a packet you shouldn't. The pseudo-header in the checksum catches that.

This means the IP layer and UDP layer are not as cleanly separated as you'd like. If you're writing a NAT box that rewrites IP addresses, you also have to recompute the UDP checksum. Most NICs do this in hardware now (checksum offload).

### Why is the IPv4 checksum optional but IPv6's mandatory?

Back when UDP was designed, networks were noisy and CPUs were slow. Computing checksums on every packet was expensive. So the designers said "if you're confident the lower layers will catch corruption, you can skip the checksum by setting it to all zeros."

By the time IPv6 was designed, the philosophy had changed. IPv6 itself doesn't have a header checksum (TCP/UDP/etc. are responsible for catching corruption). So UDP-over-IPv6 must have a real checksum — there's no other line of defense. A zero checksum on IPv6 UDP is illegal and the receiver should drop the packet.

## When You Want UDP

Here is the menu of cases where UDP is the right choice. For each one, the same logic applies: the cost of waiting for reliability is worse than the cost of a lost message.

### DNS queries

When your computer wants to look up the IP address for `google.com`, it sends a tiny UDP packet to a DNS server (usually on port 53). The packet says "what is google.com's address?" The DNS server sends back another tiny UDP packet that says "172.217.x.x." The whole exchange is two packets, total maybe 200 bytes.

If this used TCP, you would need: 3-packet handshake to open the connection, 1 packet to ask, 1 packet ack, 1 packet for the response, 1 packet ack, 4-packet handshake to close. About 11 packets for the same work UDP does in 2.

DNS has tens of billions of queries per day worldwide. Multiply 11 packets versus 2 packets across that volume and the savings are enormous. Plus the latency. UDP DNS is one round trip. TCP DNS is at least three.

If a DNS query gets lost, the application notices the timeout (usually a few seconds), and asks again. The retry logic lives in the DNS resolver, not in the transport layer. UDP makes this easy.

There are exceptions: DNS over TCP exists (and is used for big responses or for zone transfers). DNS over TLS and DNS over HTTPS also exist (for privacy). But the default, the workhorse, the protocol that handles most DNS traffic, is UDP.

### Voice over IP and video conferencing

Your video call. Your phone call. Zoom. Teams. WebEx. Discord voice. All built on UDP. Why? Because lost packets are better than late packets. Audio and video are real-time streams. A late frame is useless. A missing frame is barely noticeable because the codec smooths it over.

The protocol almost always used here is **RTP** (Real-time Transport Protocol). RTP runs on UDP. RTP adds its own sequence numbers and timestamps so the receiver can reassemble frames in order, but if a packet is missing, RTP just leaves a gap, and the application's audio or video codec hides it. **RTCP** (RTP Control Protocol) runs alongside, on the next port up, sending control info and statistics. **SRTP** is RTP with encryption, used by basically every modern voice/video app.

### Online gaming

Multiplayer games (Counter-Strike, Fortnite, League of Legends, World of Warcraft) almost all use UDP for game state updates. The reason is the same as voice/video: latency matters more than reliability. If a packet about your character's position gets lost, the next packet (50 milliseconds later) will overwrite the missed one. By the time you would have gotten the lost packet retransmitted, the game has moved on.

Games also build their own reliability layer on top of UDP for things that absolutely must be delivered (you bought an item, you killed an enemy), while leaving routine state updates fire-and-forget. This selective reliability is impossible with TCP.

### DHCP

When your computer joins a Wi-Fi network, it does not yet have an IP address. It needs to ask "hey, anybody on this network want to give me an IP address?" That is broadcast. Broadcast cannot use TCP because TCP needs a known peer. UDP can broadcast just fine. So the DHCP exchange (Discover, Offer, Request, Acknowledge — the famous "DORA" sequence) is all UDP.

DHCP server listens on UDP port 67. DHCP client listens on UDP port 68. The first packet goes to the broadcast address (255.255.255.255 on IPv4) and any DHCP server in earshot can answer.

### NTP

The Network Time Protocol synchronizes clocks. Your computer asks a time server "what time is it?" and the time server replies with a timestamp. UDP is perfect because: timestamp packets are tiny, they are highly time-sensitive (a late timestamp is wrong), and a lost packet just means try again in a moment. NTP runs on UDP port 123.

NTP packets are 48 bytes. Plus 8 bytes UDP header. Plus 20 bytes IP header. Total about 76 bytes for a single time sync. Tiny. Frequent. Unreliable but easy to retry. Perfect UDP territory.

### SNMP

The Simple Network Management Protocol. Used by network admins to read stats from routers, switches, and servers. "Hey switch, how many bytes have come through interface 3?" "Hey router, what is your CPU usage?" Single short questions, single short answers. UDP port 161 for queries, UDP port 162 for "traps" (alerts the device sends to a manager).

If a query is lost, the admin's tool retries. Big deal. SNMP is fundamentally a poll-based protocol where missed polls just mean the next chart point is missing.

### Syslog

The classic Unix system logging protocol used UDP on port 514. A program on a machine sends a log line as a UDP packet to a central log server. Fire and forget. If a log line gets lost, eh, you missed one log line. Modern syslog can use TCP (port 6514) or TLS for reliability when needed, but the original is UDP.

### TFTP

Trivial File Transfer Protocol. A super-simple file transfer protocol that runs on UDP port 69. Used in network booting (PXE), embedded device firmware updates, and other places where you want a tiny client that can transfer a small file with no fuss. TFTP adds its own simple ack-and-retry scheme on top of UDP because the file does have to arrive correctly. It is a good example of "build the reliability you need on top of UDP."

### Modern QUIC and HTTP/3

This is the surprise plot twist of modern networking. The newest version of HTTP, **HTTP/3**, runs on **QUIC**, which runs on **UDP**. Why? Because QUIC needed to do all kinds of new things that TCP could not be made to do (faster handshakes, integrated TLS, no head-of-line blocking, connection migration across networks). Rather than try to change TCP (impossible on the open internet, where every router has opinions), the QUIC designers said "let's build it on UDP and do everything in user space." So the future of web traffic is, surprise, UDP underneath. We have a whole sheet on this — `cs ramp-up http3-quic-eli5`.

### IoT sensors

Battery-powered sensors send tiny readings over the network. A temperature reading. A door-open event. A motion detection. Each event is a few bytes. The sensor wants to send the packet and immediately go back to sleep to save battery. Spending several seconds and a dozen packets opening a TCP connection would be a battery disaster. UDP packets, sent and forgotten, fit the constraint perfectly. **CoAP** (Constrained Application Protocol) is a popular IoT protocol that runs on UDP.

### Multicast and broadcast

If you want to send the same data to many recipients at once, UDP is the only option. TCP cannot do this. A streaming video service inside a corporate network might send video to 200 conference rooms via UDP multicast: one packet from the server, copies arrive at all 200 receivers via the network multicast tree. Doing this with TCP would require 200 separate connections.

mDNS, SSDP, IGMP queriers, OSPF, RIPv2, and lots of network discovery protocols all use UDP multicast.

### Serverless networking and tunneling

WireGuard, GRE-over-IP-over-UDP, IKEv2 NAT-traversal mode, OpenVPN UDP mode, GTP-U (mobile networks): all of these use UDP because UDP is easy to tunnel through firewalls and NATs without all the connection overhead and tracking that TCP would require.

## When You DON'T Want UDP

Now the other side. Cases where UDP is wrong.

### File transfer

If you are downloading a 5 GB file, you need every byte. A missing byte ruins the file. A reordered byte ruins the file. You need TCP. Or you need a higher-level reliable protocol on top of UDP (like QUIC, which adds reliability), but you should not roll your own.

### HTTP/1.1 and HTTP/2

Both versions of HTTP before 3 use TCP because they need ordered, reliable byte streams. Web pages fail in weird and surprising ways if any byte gets lost or reordered. HTTP/3 changed that, but only by building the reliability inside QUIC.

### SSH

You absolutely need every byte and every byte in order. SSH uses TCP. Always.

### Database connections

PostgreSQL, MySQL, MongoDB, Redis: all use TCP. Database queries cannot tolerate lost or reordered bytes. The protocols are byte streams.

### Email (SMTP, IMAP, POP3)

Email content is text that has to arrive intact. SMTP is TCP. IMAP is TCP. POP3 is TCP.

### Anything where "kind of arrived" is worse than "didn't arrive"

If silently corrupted data is worse than missing data, do not use UDP without checks. UDP has a checksum but you need to make sure your application notices when packets are missing. With a file transfer, you cannot tolerate a missing chunk in the middle. With TCP, the protocol promises every byte. With UDP, your code has to manage that.

### Anything where "kind of arrived" is silent failure

Many bugs in distributed systems come from people using UDP and not noticing when packets are silently lost. If your monitoring system uses UDP for metrics (like statsd does), you might happily report that "the cluster is healthy" while in fact 30% of your metric packets are being dropped on a congested link. UDP is a great fit for high-volume metrics if you accept that some loss is OK and you have no illusions about it. If you cannot accept loss, do not use UDP without thinking very hard about how you will detect and respond to it.

### Cryptocurrency, blockchain, financial trading

Most blockchain peer-to-peer protocols use TCP because they need every byte. Most financial market data feeds use UDP multicast (because thousands of receivers need the same prices simultaneously, and a missed tick is OK because the next tick comes in milliseconds). Order entry into an exchange almost always uses TCP because losing your order is bad.

### Critical control systems

Industrial control protocols (SCADA, Modbus, DNP3) usually use TCP for command-and-control because you cannot have a missed "open valve" command. Some of them also have UDP variants for telemetry. The general rule: control = TCP, telemetry = UDP.

## Reliability On Top of UDP

Here is the move that takes UDP from "throw and pray" to "throw and retry exactly when I want." The pattern: applications add their own reliability features on top of UDP, picking exactly the amount of reliability they need.

### What you can layer on

You can add **sequence numbers** to your packets so the receiver can detect missing or reordered packets.

You can add **acknowledgments** so the sender knows what arrived.

You can add **retransmission** when the sender notices something is missing.

You can add **flow control** so a fast sender doesn't drown a slow receiver.

You can add **congestion control** so you back off when the network is busy.

You can add **encryption** with DTLS (we'll see DTLS in a moment).

You can add **session management** with cookies and session IDs.

You can pick any subset of these. That is the magic. With TCP you get all of it whether you want it or not. With UDP you get nothing and you opt in to what you need.

### Examples of reliable-on-UDP protocols

**QUIC** — gives you everything TCP gives you (reliable, ordered, congestion-controlled) plus more (faster handshake, multiple streams without head-of-line blocking, integrated TLS, connection migration). All built on UDP.

**RTP** — gives you sequence numbers and timestamps so the receiver can reassemble multimedia in order, but does not retry missing packets (that would be too slow for live media).

**DTLS** — gives you TLS encryption over UDP. We'll cover this in detail next.

**WireGuard** — a modern VPN. Uses UDP. Adds its own crypto, its own session keys, its own retransmission logic for the handshake but not for tunneled traffic.

**RUDP** (Reliable UDP) — a generic library you can use to add reliability to your own UDP traffic. Less common than QUIC.

**SRT** (Secure Reliable Transport) — used for video streaming over the internet. Adds enough reliability for live broadcast quality.

**Custom game protocols** — every multiplayer game has a custom UDP-based protocol with selective reliability for game events.

### A picture of the layering

```
  +-------------------------------+
  |   APPLICATION (game, voice)   |
  +-------------------------------+
  | RELIABILITY LAYER (custom)    |  <- you wrote this
  |   sequence numbers            |
  |   acks for important stuff    |
  |   retries for important stuff |
  +-------------------------------+
  |   UDP (8 bytes)               |  <- the kernel handles this
  +-------------------------------+
  |   IP                          |
  +-------------------------------+
  |   ETHERNET / Wi-Fi            |
  +-------------------------------+
```

The kernel only does UDP and below. Everything above is your code. That is the freedom.

### A real-world example: how a video call works

To make this concrete, let's walk through what happens during a video call (Zoom, Discord, Teams, WebEx — all roughly the same).

1. Your camera produces a stream of video frames at 30 frames per second.
2. The video codec (H.264 or VP8 or AV1) compresses each frame, often into multiple "slices" so a single damaged frame doesn't ruin everything.
3. Each slice goes into a UDP packet. Each packet has an RTP header with a sequence number and timestamp.
4. UDP sends the packets to the call's media server (or directly to the peer in some apps).
5. At the other end, the receiver picks up UDP packets, sorts them by RTP sequence number into a small "jitter buffer" (about 100ms long).
6. The jitter buffer hands frames to the codec in order, on time. If a packet is missing, the codec skips that slice; if a whole frame is missing, the codec uses the previous frame again. The viewer might see a brief artifact but mostly doesn't notice.
7. Audio is similar but more aggressive about hiding loss because audio gaps are more annoying than video gaps.

Total UDP packet rate per call: a few hundred per second per direction. That's why your Wi-Fi gets a workout during a video call: thousands of UDP packets per minute.

If this used TCP, every lost packet would stall the entire stream until the retransmission arrived. Even a 100-millisecond stall is a noticeable hiccup. UDP just skips the gap. That's why your video call mostly works even on a flaky cell connection.

## DTLS: TLS Over UDP

TLS is what makes HTTPS secure. TLS encrypts a TCP connection so nobody on the wire can read what you're sending. But TLS was designed for a reliable, ordered byte stream. TLS will not work over UDP, because UDP can lose, reorder, and duplicate packets, and TLS expects perfect order.

So the IETF made **DTLS** — Datagram Transport Layer Security. DTLS is "TLS but it handles UDP's unreliability." It adds:

- A "cookie exchange" during the handshake to defeat amplification attacks.
- Sequence numbers on every record.
- An "epoch" number so cipher state changes can be tracked across packet loss.
- Fragmentation handling so big TLS handshake messages fit in UDP packets.
- Logic to handle reordered packets gracefully.

DTLS gives you TLS-grade encryption and authentication on UDP traffic. Without DTLS, your UDP traffic is plaintext.

### Where DTLS is used

- **OpenVPN UDP mode** — uses DTLS-like encryption (actually OpenVPN's own protocol but inspired by DTLS).
- **WebRTC media streams** — the audio/video data in browser-based calls is encrypted with DTLS-SRTP (DTLS handshakes the keys, SRTP encrypts the media).
- **IKEv2** — IPsec's key exchange uses UDP (port 500 or 4500) and is its own encrypted protocol, similar in spirit to DTLS.
- **CoAPS** — secure CoAP, the IoT protocol, uses DTLS.
- **WireGuard** — does NOT use DTLS. WireGuard has its own encryption protocol designed from scratch, simpler than DTLS. But it serves the same role: encrypted UDP.

DTLS 1.2 is in RFC 6347 (2012). DTLS 1.3 is in RFC 9147 (2022). DTLS 1.3 is much faster (one round trip handshake) and tracks closely with TLS 1.3.

### A simple picture

```
   YOUR APP                 PEER APP
      |                        |
      v                        ^
    [DTLS]                   [DTLS]
      |                        ^
      v                        |
    [UDP]                    [UDP]
      |                        ^
      v                        |
       --------- network --------
            (encrypted bytes)
```

Without DTLS, the bytes between the two `[UDP]` blocks would be plaintext. Anybody listening could read everything. With DTLS, those bytes are encrypted, and only the two endpoints can decrypt them.

### How DTLS handles UDP's annoyances

DTLS has to deal with everything UDP throws at it that regular TLS never sees:

- **Packet loss in the handshake.** The TLS handshake is several big messages back and forth. If a packet is lost, DTLS uses timeouts and retransmits. The timeout starts at 1 second and doubles each retry.
- **Reordering during the handshake.** DTLS uses a sequence number on every record, plus a "message sequence" inside the handshake, so the receiver can put things back in order.
- **Big handshake messages.** Some TLS handshake messages (like the certificate chain) can be many kilobytes. UDP can't carry a 5 KB packet. So DTLS fragments the handshake message into multiple UDP packets, each labeled with its offset, and the receiver reassembles them.
- **Amplification attacks.** Without a defense, an attacker could send a small DTLS ClientHello to a server with a forged source IP, and the server would send a big response (like a certificate) to the victim. DTLS adds a "cookie exchange" so the server only commits to a real handshake after confirming the client can receive packets at the source IP they claim.

These are all accommodations for living on top of UDP. They're invisible to applications, but they're why DTLS exists at all.

## QUIC: The Future Built on UDP

Here is the plot twist. The brand-new internet protocol that handles HTTP/3 (the newest HTTP) is **QUIC**, and QUIC is built on UDP.

For decades, the assumption was: web traffic = TCP. HTTP, HTTPS, all of it, TCP. UDP was for DNS and games and VoIP.

Then around 2012, Google started experimenting with QUIC: a brand-new transport layer protocol that does everything TCP+TLS does but better and faster. The big innovations:

- **Faster handshake.** TCP+TLS needs 3 round trips to start sending data. QUIC needs 1 round trip (or 0 with session resumption).
- **No head-of-line blocking.** TCP delivers bytes in strict order. If one packet is lost, all later packets wait. QUIC has independent streams so a loss in one stream does not stall others.
- **Integrated TLS.** TLS is built into QUIC, not bolted on top.
- **Connection migration.** A QUIC connection can survive an IP address change. Useful when your phone hops from Wi-Fi to LTE.
- **Pluggable congestion control.** Each app can pick its own.

The catch: QUIC needs a way to ride on the internet without getting blocked or messed with by middleboxes (routers, firewalls) that "know" about TCP. QUIC's solution: ride on UDP. Routers and firewalls treat UDP as a generic packet pipe. They don't know what is inside the UDP payload. So QUIC encrypts everything inside UDP, including its own headers, and the network just sees opaque UDP traffic.

The result: a transport protocol with all of TCP's reliability and TLS's security, in user space (because UDP is just a thin kernel wrapper), evolvable (because no middlebox knows what's inside), and faster.

HTTP/3 = HTTP semantics on QUIC on UDP on IP. The newest web is built on UDP. There's a whole detailed sheet at `cs ramp-up http3-quic-eli5`.

### Why "UDP-shaped" matters for QUIC

A common question: "Why doesn't QUIC just become its own IP protocol like TCP?" Two reasons.

**Deployment.** A new IP protocol needs every router, firewall, NAT, and middlebox in the world to know about it and treat it correctly. That's never going to happen. UDP, on the other hand, is universally allowed.

**Iteration.** TCP is essentially frozen because middleboxes do "TCP optimization" or "deep packet inspection" that depends on TCP's exact wire format. Adding new TCP features risks breaking with these middleboxes. QUIC encrypts everything (even most of its own header) so middleboxes can't peek and can't break things. This means QUIC can keep evolving, even after deployment, without breaking the network.

The result is a transport protocol that is genuinely a moving target — each new version of QUIC can change wire format details safely, because the network can only see UDP and ciphertext.

## DNS: The Most Famous UDP User

DNS is the protocol that turns names like `google.com` into IP addresses like `142.250.80.46`. Every time you type a URL into your browser, your computer makes at least one DNS query. Often many more (one for each separate hostname referenced by the page).

Almost all DNS traffic is UDP on port 53.

### Walking through a DNS query

You type `google.com` into your browser. Your computer needs the IP address.

1. Your computer's resolver builds a tiny DNS query packet. It looks like: "Question: A record for google.com." That's about 40 bytes of payload.
2. The resolver picks a DNS server (usually the one your DHCP gave you, like `8.8.8.8` or `1.1.1.1` or your ISP's resolver). It picks an ephemeral UDP source port (a random number above 1024).
3. The resolver creates a UDP packet: source port = ephemeral, destination port = 53, length = 48 (or whatever), checksum = computed.
4. The kernel wraps that in an IP packet (source = your IP, destination = 8.8.8.8) and sends it.
5. The DNS server receives the packet. It looks up the answer. (Probably from its cache. If not, it asks other servers.)
6. The DNS server builds a response packet. Source port = 53, destination port = your ephemeral port. Payload includes your original question and the answer ("A record for google.com is 142.250.80.46").
7. The kernel on your machine receives the UDP packet, sees destination port = your ephemeral, hands it to your resolver.
8. The resolver gives the answer to your browser. Browser opens a TCP connection to 142.250.80.46.

Total: 2 UDP packets. The whole thing usually completes in under 50 milliseconds. Maybe 5 milliseconds if the resolver is on your network.

### The 512-byte limit (and EDNS0)

The original DNS spec said UDP DNS messages must be at most 512 bytes. If a response was bigger, the server would set a "TC" (truncated) flag and the resolver would retry over TCP.

Today, most DNS servers and resolvers support **EDNS0** (Extension Mechanisms for DNS, RFC 6891), which negotiates a larger UDP message size (often 4096 bytes). So big responses can usually fit in UDP. But if EDNS0 is not supported, or the response is even bigger, the fallback to TCP still works.

### Zone transfers always use TCP

There is one exception. Zone transfers (where one DNS server pulls a whole zone from another, called AXFR or IXFR) always use TCP. Why? Because zone transfers can be huge (megabytes of records), order matters, and you absolutely need every byte. TCP is the right tool there.

### DNSSEC, DoT, DoH

- **DNSSEC** is DNS with cryptographic signatures. Still UDP-based by default.
- **DNS over TLS (DoT)** runs DNS queries inside a TLS-encrypted TCP connection on port 853.
- **DNS over HTTPS (DoH)** runs DNS queries inside an HTTPS connection on port 443.
- **DNS over QUIC (DoQ)** runs DNS queries inside a QUIC connection on UDP port 853.

All of these add privacy. The default (plain UDP DNS on port 53) leaks every query you make to anybody on the wire and to your DNS resolver. The encrypted versions don't.

### A simple ASCII view of a DNS exchange

```
   YOUR APP                   YOUR RESOLVER              ROOT/AUTH SERVERS
   (e.g., browser)            (e.g., 1.1.1.1)
        |                         |                            |
        |--- "google.com?" ------>|                            |
        |   (UDP, port 53)        |                            |
        |                         |--- (cache miss?) --------->|
        |                         |    walks the tree:         |
        |                         |    "." -> "com." -> ...    |
        |                         |<------ "google.com is X"---|
        |<---- "google.com=X" ----|                            |
        |   (UDP, port 53)        |                            |
        |                         |                            |
   < your browser opens TCP to X to load the page >
```

Total UDP packets used to find one IP: typically 2 if your resolver had it cached, 4-12 if it had to walk the DNS tree. All of it on UDP except for zone transfers and oversized responses.

### How resolvers retry

If your resolver sends a UDP query and gets no response in (typically) 1 to 5 seconds, it just sends another UDP packet. Same query, same destination, possibly to a different server if it has multiple to choose from. After a couple of failed retries it gives up and tells your application "DNS lookup failed."

This retry logic lives in the resolver, not in UDP. UDP doesn't know anything about retries; it's the resolver's job to notice the timeout and try again. That's the pattern: UDP at the transport layer, application-level retry on top.

## Multicast and Broadcast

This is where UDP does something TCP literally cannot. UDP can send a single packet that gets delivered to many receivers.

### Broadcast

A broadcast packet goes to everybody on a local network segment. The destination IP is `255.255.255.255` (the "limited broadcast" address, IPv4). Every machine on the local subnet receives a copy.

Broadcast is used by DHCP (to find a DHCP server when you don't know its address), by ARP (to find which MAC address has which IP), and by some old service-discovery protocols.

Broadcast is loud. Every machine on the segment has to look at every broadcast packet. So broadcast is restricted to local segments only — routers do not forward broadcasts.

### Multicast

Multicast is "broadcast, but only to people who opted in." A multicast packet goes to a special "group address" (in IPv4, anything in `224.0.0.0/4`, which is `224.0.0.0` through `239.255.255.255`). Receivers join the group with the `IGMP` protocol, telling routers "hey, send group X traffic to me." Routers build a tree of who wants what and forward packets only along that tree.

Multicast can cross routers (unlike broadcast) but only to receivers that joined.

In IPv6, multicast is built into the foundation. Every IPv6 multicast address starts with `ff` (the high byte). The address `ff00::/8` is the IPv6 multicast range. IPv6 doesn't really have broadcast — it just has well-known multicast groups (like `ff02::1` for "all nodes on the link").

### Where multicast is used

- **mDNS** (Multicast DNS, RFC 6762): used by Bonjour, AirPlay, and lots of zero-config service discovery. Uses `224.0.0.251` (IPv4) or `ff02::fb` (IPv6) on UDP port 5353.
- **SSDP** (Simple Service Discovery Protocol): used by UPnP. Multicast on `239.255.255.250` UDP port 1900.
- **OSPF** (Open Shortest Path First, a routing protocol): uses multicast to distribute link-state updates.
- **PIM** (Protocol Independent Multicast): the protocol routers use to build multicast trees.
- **IPTV** in carrier networks: a single multicast stream serves a whole region's TV viewers.
- **RIPv2**: uses multicast to send routing updates.

### A picture of a multicast tree

```
             [SENDER]
                |
              [router R1]
              /         \
         [router R2]   [router R3]
          /     \        |
      [host A][host B]  [host C]    -- all joined group X --
```

The sender sends one packet. R1 makes a copy and sends to R2 and R3. R2 makes a copy and sends to A and B. R3 sends one to C. Total packets used to deliver to 3 receivers: 5 (1 from sender, 1 to R3, 1 from R2 to A, 1 from R2 to B, 1 from R3 to C). Without multicast, the sender would have to send 3 separate packets, each replicated all the way through the network. Multicast saves bandwidth.

### Why broadcast got phased out

In modern networks, broadcast is mostly a relic. It's still used for DHCP and ARP, but for almost everything else, multicast is preferred. Why?

- **Broadcast bothers everyone.** Every machine on the segment has to inspect every broadcast packet, even if they don't care about the protocol. Multicast only goes to opted-in receivers.
- **Broadcast doesn't cross routers.** This is by design (otherwise broadcast traffic would flood the internet), but it makes broadcast useless for anything beyond the local segment.
- **IPv6 doesn't have broadcast at all.** It just has well-known multicast groups.

So if you're designing a new protocol that needs one-to-many delivery, use multicast.

### A multicast group address quick guide

For IPv4 (the range `224.0.0.0/4`):

- `224.0.0.0/24` — well-known, link-local. Examples: `224.0.0.1` (all hosts), `224.0.0.2` (all routers), `224.0.0.5` (OSPF AllSPFRouters), `224.0.0.251` (mDNS).
- `224.0.1.0` to `238.255.255.255` — globally scoped. Used for things that span beyond a single link.
- `239.0.0.0/8` — administratively scoped (think "private use" for multicast). Like `192.168.0.0/16` is for unicast.

For IPv6 (everything starting with `ff`):

- `ff02::1` — all nodes on the link.
- `ff02::2` — all routers on the link.
- `ff02::fb` — mDNS.
- `ff02::1:ff00:0/104` — solicited-node multicast (used by IPv6 neighbor discovery instead of ARP).

You can join a group with `setsockopt(sock, IPPROTO_IP, IP_ADD_MEMBERSHIP, ...)` (IPv4) or `IPV6_JOIN_GROUP` (IPv6). The kernel sends an IGMP (or MLD on IPv6) message to inform routers that you want this group's traffic.

## UDP and the Linux Kernel

How does the Linux kernel actually handle UDP?

### `struct udp_sock`

When your program creates a UDP socket, the kernel allocates a `struct udp_sock`. This is much smaller and simpler than TCP's `struct tcp_sock`. There is no connection state to track. There is no congestion window, no slow-start variables, no retransmission queue, no out-of-order queue. There is just:

- the local port and address it's bound to
- the remote port and address (only if you called `connect()` on the socket)
- send and receive buffers
- a hash entry to find the socket from incoming packets

### UDP socket lookup

When a UDP packet arrives, the kernel needs to figure out which socket should get it. It does this by hashing on source IP, source port, destination IP, destination port. The hash table is `udp_table` (and `udp_hslot` for each bucket).

This is much simpler than TCP's lookup. UDP doesn't have any "is this connection still alive" or "is this packet in sequence" checks. It just finds the socket and queues the packet. Done.

### `SO_REUSEADDR` vs `SO_REUSEPORT`

If two programs both want to listen on UDP port 5000, who gets which packets? By default, only one program can bind to a port. Two socket options change this:

- **`SO_REUSEADDR`** (UDP): lets multiple sockets bind to the same address/port if all of them set this flag. The kernel picks one socket per packet (roughly the most-recently-bound).
- **`SO_REUSEPORT`** (UDP): introduced in Linux 3.9. Multiple sockets share the same port, and the kernel does a hash-based load distribution: each connection's 5-tuple (source IP, source port, destination IP, destination port, protocol) hashes to one socket consistently. This is the way to do multi-process UDP servers. Each worker process opens a `SO_REUSEPORT` socket on the same port, and the kernel spreads incoming packets evenly.

### UDP is fundamentally connectionless in the kernel

Even though you can call `connect()` on a UDP socket, that does not create a real connection. It just tells the kernel "from now on, only accept packets from this peer, and `send()` automatically sends to that peer." It is purely a kernel convenience. No packets are exchanged. No state is shared with the peer.

This is why a UDP server can serve thousands of clients with a single socket — there is no per-client state in the kernel.

### Buffers

Each UDP socket has a send buffer (`SO_SNDBUF`) and a receive buffer (`SO_RCVBUF`). The defaults are usually around 200 KB. You can read them with `sysctl net.core.rmem_default` and friends. If the receive buffer fills up (because the application is reading slowly), the kernel drops new incoming packets and bumps a counter (`UdpRcvbufErrors`). If you see a lot of those errors, your app is too slow or your buffer is too small.

### GSO, GRO, and offload

For high-throughput UDP (like VPN concentrators, video servers, QUIC servers), the kernel has tricks to bundle multiple UDP packets into a single segmentation operation, reducing per-packet overhead. **UDP-GSO** (Generic Segmentation Offload) lets the application send a giant UDP "super-packet" that the kernel splits into MTU-sized packets at the last moment. **GRO** is the receive-side counterpart. These are big wins for QUIC throughput, where modern Linux can hit tens of gigabits per second of UDP traffic.

### Per-CPU socket lookup

In modern Linux, the kernel uses per-CPU lookup paths for UDP to avoid lock contention when many cores are receiving packets. Each CPU pulls packets from its own queue without fighting other CPUs for a global lock. This is one of the reasons modern Linux can handle millions of UDP packets per second on a single machine.

### Early demux

There's a sysctl called `net.ipv4.udp_early_demux`. When set to 1 (the default), the kernel finds the destination socket as early as possible in the receive path, even before all the routing checks. This is a small optimization that saves CPU cycles per packet. It matters at high packet rates.

### XDP and AF_XDP

If you really, really need fast UDP — like millions of packets per second — modern Linux has XDP (eXpress Data Path). XDP runs an eBPF program at the lowest level of the network driver, before the kernel network stack. You can do per-packet decisions there: drop, redirect to another interface, modify, or pass to the regular stack. Combined with AF_XDP sockets, you can deliver packets straight to user space, skipping the kernel network stack entirely. This is how DDoS-mitigation systems and high-frequency-trading platforms get their UDP packets at line rate.

### conntrack and UDP "connections"

The Linux netfilter framework (the engine behind iptables and nftables) tracks "connections" even for UDP. When it sees an outgoing UDP packet from your machine to a server, it records a "pseudo-connection" in `/proc/net/nf_conntrack`. When the reply comes back, conntrack matches it to the pseudo-connection and lets it through your firewall.

UDP pseudo-connections expire fast — usually 30 seconds of inactivity by default. You can tune this with `sysctl net.netfilter.nf_conntrack_udp_timeout`. If you have very long-lived UDP flows (like a WireGuard tunnel) you may want to bump this up, otherwise the conntrack entry expires and your tunnel goes silent until traffic flows again.

## Common UDP Errors and Behaviors

Things that go wrong with UDP that beginners stub their toes on.

### "Connection refused"

UDP doesn't really have connections. So how can you get "connection refused"? Here is what happens. You send a UDP packet to a port nobody is listening on. The receiving machine's kernel sends back an ICMP "port unreachable" message. Your kernel sees the ICMP and translates it into an `ECONNREFUSED` error on your next read or write of the socket.

This works only if you `connect()`-ed your UDP socket. An unconnected UDP socket usually ignores ICMP errors because it has no idea which "connection" the error is associated with.

So "connection refused on a UDP socket" really means "the other side told us via ICMP that nobody was listening on that port, and we noticed."

### "Message too long"

If your UDP payload, plus headers, exceeds the path MTU, and the IP layer has the don't-fragment bit set, the IP layer will drop it. Your `sendto()` call returns `EMSGSIZE`. You need to send smaller messages.

The classic mistake is to assume you can send 64 KB UDP packets across the internet and have them work. They might fragment at IP, get dropped by a firewall that hates fragments, and disappear. The safe practical UDP payload size is around 1200 bytes (over IPv6, less for IPv4 with options). QUIC uses 1200 bytes as a sane default for exactly this reason.

### Silent packet loss

The killer issue with UDP. A packet leaves your machine. It does not arrive. You get no error. No notification. Nothing. It is just gone.

Your application has to either: (a) not care (it is OK to lose this packet), or (b) detect the loss itself (sequence numbers, timeouts, application-level acks).

If you do not handle this, your app will subtly malfunction in lossy networks and you will spend a long time debugging it. Welcome to UDP.

### "No buffer space available"

If your application sends UDP packets faster than the kernel can transmit them, the send buffer fills up and `sendto()` returns `ENOBUFS`. This is rare on modern systems but can happen with high-rate UDP blasting.

### Partial reads

With UDP, `recvfrom()` returns one whole datagram. If the datagram is bigger than the buffer you provided, the rest gets thrown away (not saved for the next read). This is different from TCP, where data is a stream and reads are chunks of whatever you ask for. Always provide a buffer big enough for the largest datagram you expect, or use `MSG_PEEK` to check size first.

### Packet reordering

The network can deliver UDP packets out of order. If you send `[A, B, C]`, the receiver might get `[B, A, C]`. UDP itself does nothing about this. Your application must either tolerate disorder or add sequence numbers.

### Packet duplication

The network can also deliver duplicates. If your application is not idempotent, you have a problem. Sequence numbers help here too.

### Spoofing and source-address authentication

UDP source addresses are easy to spoof. Anyone can craft a UDP packet that claims to come from any IP address. There's no handshake to verify the sender. This is why UDP-based protocols are often used in **amplification attacks**: send a small spoofed UDP query to a server (claiming to be from the victim), the server sends a much larger response to the victim. DNS, NTP, memcached, and SSDP have all been used for amplification attacks at terabit scale.

If you're building a UDP service, think about what an attacker can do with spoofed packets. Common defenses: rate-limit responses per source IP, use cookies (like DNS Cookies, RFC 7873), or require an initial handshake before sending big responses (this is what DTLS does).

## Hands-On

These commands let you actually see UDP traffic on your machine. None of these will break anything. Type, watch, learn.

### Command 1: List all UDP sockets

```
$ ss -uan
State       Recv-Q Send-Q       Local Address:Port            Peer Address:Port
UNCONN      0      0                  0.0.0.0:68                   0.0.0.0:*
UNCONN      0      0                127.0.0.53:53                  0.0.0.0:*
UNCONN      0      0                  0.0.0.0:5353                 0.0.0.0:*
UNCONN      0      0                  0.0.0.0:631                  0.0.0.0:*
UNCONN      0      0                     [::]:5353                    [::]:*
```

`ss -uan`: `-u` for UDP, `-a` for all, `-n` for no DNS resolution. UDP sockets appear with state `UNCONN` (unconnected) since UDP has no real connection state. Your output will differ.

### Command 2: List UDP listening sockets only

```
$ ss -ualn
State       Recv-Q Send-Q       Local Address:Port            Peer Address:Port
UNCONN      0      0                127.0.0.53:53                  0.0.0.0:*
UNCONN      0      0                  0.0.0.0:5353                 0.0.0.0:*
UNCONN      0      0                  0.0.0.0:631                  0.0.0.0:*
```

The `-l` filter shows only sockets in listen-like state. For UDP this means bound sockets.

### Command 3: Show the process owning each UDP socket

```
$ sudo ss -uanp
State  Recv-Q Send-Q  Local Address:Port  Peer Address:Port  Process
UNCONN 0      0       127.0.0.53:53       0.0.0.0:*          users:(("systemd-resolve",pid=812,fd=12))
UNCONN 0      0       0.0.0.0:5353        0.0.0.0:*          users:(("avahi-daemon",pid=781,fd=12))
UNCONN 0      0       0.0.0.0:631         0.0.0.0:*          users:(("cups-browsed",pid=1442,fd=7))
```

`-p` adds process info. Needs root to see all processes. Now you can see "DNS resolver is on port 53, mDNS is on port 5353, CUPS is on port 631."

### Command 4: UDP socket statistics

```
$ ss -us
Total: 1234 (kernel 0)
UDP:  9 (estab 0, closed 0, orphaned 0, synrecv 0, timewait 0/0), ports 0
Transport Total     IP        IPv6
*         0         -         -
UDP       9         5         4
```

`-s` shows summary statistics. UDP has no real "established" or "timewait" states (those are TCP concepts), so most fields are zero.

### Command 5: Read the kernel's UDP socket table

```
$ cat /proc/net/udp | head -5
   sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode ref pointer drops
    0: 00000000:8B53 00000000:0000 07 00000000:00000000 00:00000000 00000000  1000        0 32145 2 ffff8c... 0
    1: 0100007F:0035 00000000:0000 07 00000000:00000000 00:00000000 00000000   101        0 24512 2 ffff8c... 0
    2: 00000000:14E9 00000000:0000 07 00000000:00000000 00:00000000 00000000     0        0 18723 2 ffff8c... 0
```

This is the magic file. The kernel exposes its UDP socket list as a text file. Each row is one UDP socket. `local_address` is hex IP:port. `0100007F:0035` decodes to `127.0.0.1:53` (0x35 = 53, 0x7F000001 = 127.0.0.1). The `drops` column shows packets dropped by this socket due to buffer overflow.

### Command 6: System-wide UDP socket count

```
$ cat /proc/net/sockstat | grep UDP
UDP: inuse 9 mem 4
UDPLITE: inuse 0
```

`UDP: inuse N` is "currently allocated UDP sockets." `mem N` is memory used in pages.

### Command 7: Listen for UDP on a port

```
$ nc -u -l 5000
```

`nc` (netcat) with `-u` for UDP and `-l` for listen. This sits there waiting for UDP packets on port 5000. Press Ctrl-C to stop. Anything you type into a separate sender will appear here.

### Command 8: Send a UDP packet

In another terminal:

```
$ nc -u 127.0.0.1 5000
hello there
^C
```

Type `hello there` and press Enter. Switch back to the listener; you should see `hello there`. You just sent a UDP packet. Press Ctrl-C to leave.

### Command 9: UDP port probe

```
$ nc -uz -w 1 example.com 53
Connection to example.com 53 port [udp/domain] succeeded!
```

`-z` zero-I/O probe mode, `-w 1` 1-second timeout. The "succeeded" output for UDP is a bit of a lie — UDP has no handshake, so all this really means is "we sent a UDP packet and didn't get an ICMP unreachable back within 1 second." It cannot truly confirm anything. UDP port probing is fundamentally unreliable.

### Command 10: Real UDP query — DNS

```
$ dig +short google.com
142.250.80.46
```

`dig` is the swiss-army DNS tool. `+short` gives just the answer. Behind the scenes this was one UDP query to your resolver and one UDP response.

### Command 11: Force DNS over TCP

```
$ dig +tcp google.com | head -10
; <<>> DiG 9.18.18 <<>> +tcp google.com
;; global options: +cmd
;; Got answer:
;; ->>HEADER<<- opcode: QUERY, status: NOERROR, id: 12345
;; flags: qr rd ra; QUERY: 1, ANSWER: 1, AUTHORITY: 0, ADDITIONAL: 1

;; QUESTION SECTION:
;google.com.                    IN      A

;; ANSWER SECTION:
google.com.             295     IN      A       142.250.80.46
```

`+tcp` forces DNS over TCP. Useful for testing fallback. Look at the time: TCP DNS is noticeably slower than UDP DNS.

### Command 12: Capture DNS UDP traffic live

```
$ sudo tcpdump -i any -n -c 20 udp port 53
tcpdump: data link type LINUX_SLL2
13:42:11.123456 enp0s3 Out IP 192.168.1.10.41234 > 8.8.8.8.53: 1234+ A? google.com. (28)
13:42:11.165432 enp0s3 In  IP 8.8.8.8.53 > 192.168.1.10.41234: 1234 1/0/0 A 142.250.80.46 (44)
```

`-i any` listens on all interfaces. `-n` no name resolution. `-c 20` stops after 20 packets. Then run `dig google.com` in another terminal and watch packets fly. The `?` is a query, the absence is a response.

### Command 13: Capture a port range

```
$ sudo tcpdump -i any -n -c 20 udp portrange 5000-6000
```

Useful for capturing application-specific UDP traffic in a port range. Try running an `nc -u -l 5000` and a `nc -u 127.0.0.1 5000` in two other terminals.

### Command 14: UDP throughput test with iperf3

```
$ iperf3 -u -c iperf.example.com -b 100M -t 5
Connecting to host iperf.example.com, port 5201
[  5] local 192.168.1.10 port 41234 connected to 203.0.113.5 port 5201
[ ID] Interval           Transfer     Bitrate         Total Datagrams
[  5]   0.00-1.00   sec  11.9 MBytes  99.9 Mbits/sec  8497
[  5]   1.00-2.00   sec  11.9 MBytes  99.9 Mbits/sec  8505
...
[ ID] Interval           Transfer     Bitrate         Jitter    Lost/Total Datagrams
[  5]   0.00-5.00   sec  59.6 MBytes  100 Mbits/sec   0.123 ms  17/42532 (0.04%)
```

`-u` UDP mode. `-c host` connect as client. `-b 100M` send at 100 Mbps. `-t 5` for 5 seconds. The result shows throughput, jitter, and packet loss percentage.

### Command 15: Multi-client UDP listener with socat

```
$ socat - UDP-LISTEN:9999,fork
```

`socat` is a swiss-army I/O relay. `UDP-LISTEN:9999,fork` means "listen for UDP on port 9999 and fork a new instance for each peer." Connect from another terminal with `socat - UDP:127.0.0.1:9999` and type. This is the easy way to test a multi-client UDP server.

### Command 16: UDP port scan

```
$ sudo nmap -sU -p 53,123,161 example.com
Starting Nmap 7.94 ( https://nmap.org )
Nmap scan report for example.com (203.0.113.10)
Host is up (0.020s latency).
PORT    STATE         SERVICE
53/udp  open|filtered domain
123/udp open          ntp
161/udp closed        snmp
```

`-sU` UDP scan. The states for UDP are weird: `open|filtered` means we got no reply, which could mean the port is open or could mean a firewall ate the packet. UDP scanning is slow and often inconclusive.

### Command 17: Read kernel UDP buffer settings

```
$ cat /proc/sys/net/core/rmem_default
212992
$ cat /proc/sys/net/core/rmem_max
212992
$ cat /proc/sys/net/core/wmem_default
212992
$ cat /proc/sys/net/core/wmem_max
212992
```

These are the default and max receive (`rmem`) and send (`wmem`) buffer sizes for sockets in bytes. UDP sockets respect `SO_RCVBUF` and `SO_SNDBUF` set by applications, but capped by these system limits. About 212 KB by default on Linux.

### Command 18: All UDP-related sysctls

```
$ sysctl -a 2>/dev/null | grep ^net.ipv4.udp
net.ipv4.udp_early_demux = 1
net.ipv4.udp_l3mdev_accept = 0
net.ipv4.udp_mem = 758265 1011022 1516530
net.ipv4.udp_rmem_min = 4096
net.ipv4.udp_wmem_min = 4096
```

`udp_mem` controls UDP memory pressure thresholds. `udp_rmem_min` and `udp_wmem_min` are the minimum allowed buffer sizes. `udp_early_demux` is a performance tweak.

### Command 19: Look for UDP-related kernel messages

```
$ sudo dmesg | grep -i udp | tail -10
[  324.123456] UDP: short packet: From 192.0.2.1:1234 12/8 to 198.51.100.1:53
[ 1024.654321] nf_conntrack: table full, dropping packet
```

If the kernel is dropping UDP packets for any reason, it usually leaves a trace in dmesg. "Short packet" means a malformed UDP header. "Table full" usually means conntrack is maxed.

### Command 20: eBPF trace UDP sends

```
$ sudo bpftrace -e 'kprobe:udp_sendmsg { @[comm] = count(); }'
Attaching 1 probe...
^C
@[firefox]: 47
@[chrome]: 33
@[systemd-resolve]: 8
@[ntpd]: 3
@[Discord]: 152
```

`bpftrace` runs eBPF programs that hook into the kernel. `kprobe:udp_sendmsg` triggers every time the kernel's UDP send function runs. `@[comm]` counts by process name. Press Ctrl-C and you see who is sending UDP and how often. Discord is on top because voice/video. Firefox sends some too (probably QUIC). Try this for 30 seconds during a video call to see how chatty UDP can be.

### Command 21: Per-application UDP byte counts via ebpf

```
$ sudo bpftrace -e 'kprobe:udp_sendmsg { @bytes[comm] = sum(arg2); }'
Attaching 1 probe...
^C
@bytes[ntpd]: 168
@bytes[systemd-resolve]: 2310
@bytes[Discord]: 482910
@bytes[firefox]: 51234
```

Same idea but `arg2` is the message size in bytes. Now you see total UDP bytes per program. Discord cranks out half a megabyte in the time it takes to count.

### Command 22: Receive buffer drops per UDP socket

```
$ ss -ulm
State       Recv-Q Send-Q       Local Address:Port            Peer Address:Port
UNCONN      0      0                127.0.0.53:53                  0.0.0.0:*
   skmem:(r0,rb212992,t0,tb212992,f0,w0,o0,bl0,d0)
UNCONN      0      0                  0.0.0.0:5353                 0.0.0.0:*
   skmem:(r0,rb212992,t0,tb212992,f0,w0,o0,bl0,d0)
```

`-m` shows memory info. The interesting field is `d0` (or `dN`) at the end of `skmem`: that is the count of packets dropped from this socket due to lack of buffer space. If `dN` is climbing on a hot UDP socket, your application can't keep up.

### Command 23: System-wide UDP statistics

```
$ cat /proc/net/snmp | grep -A1 ^Udp:
Udp: InDatagrams NoPorts InErrors OutDatagrams RcvbufErrors SndbufErrors InCsumErrors IgnoredMulti
Udp: 124581 17 0 124612 0 0 0 4
```

This is the system's UDP statistics. `InDatagrams` is total received packets, `NoPorts` is packets that arrived for ports nobody was listening on (useful diagnostic), `InErrors` is malformed packets, `RcvbufErrors` and `SndbufErrors` are buffer overflows, `InCsumErrors` is checksum failures. These are cumulative since boot. Watch them tick up.

### Command 24: View UDP-related conntrack entries

```
$ sudo cat /proc/net/nf_conntrack | grep udp | head -5
ipv4 2 udp 17 28 src=192.168.1.10 dst=8.8.8.8 sport=53445 dport=53 src=8.8.8.8 dst=192.168.1.10 sport=53 dport=53445 use=2
ipv4 2 udp 17 158 src=192.168.1.10 dst=192.168.1.1 sport=51234 dport=53 src=192.168.1.1 dst=192.168.1.10 sport=53 dport=51234 use=2
```

Each row is a UDP "pseudo-flow" the kernel is tracking. The number after `udp 17` (17 is the IP protocol number for UDP) is the seconds remaining before this entry expires.

### Command 25: A live UDP flame, kind of

```
$ watch -n 1 "cat /proc/net/snmp | grep -A1 ^Udp:"
```

`watch` runs the command every second. Now you can see the UDP counters tick up in real time. Open a video call or run `dig` a bunch and watch `InDatagrams` climb.

## Common Confusions

These are the questions that come up over and over.

### "Is UDP unreliable for the wire or for the user?"

The wire never promises anything to anyone. IP itself is unreliable. The wire is the same wire whether you are running TCP or UDP. The difference is what TCP and UDP do on top.

TCP wraps the unreliable wire in a reliability blanket: retransmissions, ordering, acknowledgments. The user never sees the loss because TCP hides it.

UDP does not wrap anything. The user sees the wire as it is — unreliable. So when people say "UDP is unreliable," they mean "UDP shows the unreliability to the user; TCP hides it."

### "Why does DNS use UDP?"

Three reasons. First, the queries and answers are tiny — usually under 100 bytes. The TCP handshake overhead would dominate. Second, application-level retry is easy — if you don't get an answer in a couple seconds, ask again. Third, parallelism — a resolver can send dozens of UDP queries to different name servers in parallel without opening dozens of TCP connections.

### "Why doesn't UDP have 'connection refused'?"

It kind of does, indirectly. If you UDP-send to a port nobody is listening on, the kernel on the other side sends back an ICMP "port unreachable" message. If your kernel notices that ICMP and your socket is `connect()`ed, it surfaces it as `ECONNREFUSED` on your next read or write. If your socket is not connected, the ICMP gets ignored.

### "Can UDP saturate a link?"

Yes, easily. UDP has no congestion control. If you tell your machine to send UDP at 10 Gbps, it will try, and if the network is busy you will just push everybody else around. This is why UDP-blasting tools like `iperf3 -u` need to be used carefully and why production UDP applications usually implement their own congestion control (like QUIC does).

### "Why do video calls glitch?"

Because UDP packets are getting lost or arriving late, and the codec is hiding what it can. Each video frame is split into UDP packets. A lost packet means a damaged frame. The codec interpolates or uses a previous frame. If too many packets are lost, the picture freezes or pixelates. The audio codec does the same thing for sound — lost packets become little dropouts that the codec smooths over.

### "Should I just use UDP for everything fast?"

Only if you implement the reliability you actually need on top, and you understand the consequences. For most application work, TCP is the right choice. UDP is for: real-time media, query/response (DNS, NTP), broadcast/multicast, custom-tuned high-performance protocols (game, QUIC), and IoT.

### "Are UDP packets atomic?"

Yes, in a useful sense. One `sendto()` produces one IP datagram (possibly fragmented at IP if too large) and one `recvfrom()` consumes one whole datagram. The data is not split or merged like with TCP. This makes UDP application code simpler if your data fits in one datagram.

### "What is the maximum size of a UDP packet?"

In theory, 65535 bytes (because the length field is 16 bits). In practice, you should keep UDP payloads to about 1200 bytes or less to avoid IP fragmentation. Even 508 bytes is a safe lower bound (some old paths have very small MTUs). QUIC, which is finely tuned for the modern internet, uses 1200 bytes as a default.

### "What happens if I send a UDP packet bigger than the MTU?"

The IP layer fragments it (if the don't-fragment bit is not set). Each fragment travels independently. The receiver's IP layer reassembles them before delivering to UDP. If any fragment is lost, the whole datagram is lost. Many firewalls drop fragmented packets entirely. So big UDP packets are unreliable in a way small ones are not. Avoid.

### "Is UDP secure?"

UDP itself has no security. No encryption, no authentication. Anyone in the network path can read, modify, or spoof packets. If you need security, layer DTLS, QUIC, or your own crypto on top.

### "How is UDP connectionless if I can call connect()?"

`connect()` on a UDP socket is a kernel-side convenience. It does not exchange any packets. It just sets the default destination so you can use `send()` instead of `sendto()`, and it filters incoming packets so you only see traffic from that one peer. It does not establish a connection on the wire.

### "Why do I see UDP in firewalls if there are no connections?"

Stateful firewalls track UDP traffic by 5-tuple (source IP, source port, dest IP, dest port, protocol). When they see an outgoing UDP packet they create a "pseudo-flow" and allow return traffic for some timeout (often 30 seconds for UDP, much longer for TCP). After the timeout, the flow expires. This is why your home router lets DNS replies in even though you didn't open any port.

### "Does UDP have a Nagle algorithm or buffering like TCP?"

No. UDP sends immediately. Each `sendto()` results in (at least) one packet on the wire as soon as the kernel can send it. There is no batching. Some applications batch in user space if they want to.

### "Is UDP faster than TCP?"

For tiny exchanges, yes — much faster, because no handshake. For sustained high-throughput, modern TCP can be just as fast or faster in practice, because TCP has decades of congestion control optimizations that a naive UDP application won't match. QUIC closed that gap by adding sophisticated congestion control on top of UDP. So "faster" depends on what you're doing.

### "Why is QUIC on UDP and not its own protocol?"

Because middleboxes (NATs, firewalls, load balancers) all over the internet make assumptions about TCP and would either block or mangle a brand-new transport protocol. UDP is universally allowed. So QUIC rides on UDP to traverse the internet, then does everything else (encryption, reliability, streams) inside the UDP payload where middleboxes can't see it.

### "Why are there UDP errors at all if there are no connections?"

The kernel still has to route packets, fit them in MTU, find sockets, etc. So you get errors from those steps — `EMSGSIZE` (too big), `EHOSTUNREACH` (no route), `ECONNREFUSED` (got an ICMP unreachable for a connected socket), `ENOBUFS` (kernel out of buffers). These are not connection errors; they're kernel-level errors about handling the packet on this side.

### "Can I tell if my UDP packets are arriving?"

Not from UDP itself. You can see send/receive counts in `/proc/net/snmp` (the `Udp:` line shows `InDatagrams`, `OutDatagrams`, `InErrors`, `RcvbufErrors`, etc.) on each side. But these only tell you about packets the kernel saw. To know what's happening end-to-end, you have to add application-level acknowledgments or use tools like `tcpdump` on both sides.

### "Why do firewalls drop UDP after 30 seconds of silence?"

Because UDP has no "close" — there's no way to know when a flow is done. Firewalls have to guess based on a timeout. 30 seconds is a common default. If you have a long-lived UDP flow (like a VPN), you need to send keepalives every few seconds to keep the firewall's state alive, or your traffic will start getting dropped.

### "Should I use UDP or TCP for my new protocol?"

If you can use TCP and TCP gives you what you need, use TCP. It's simpler. If you need: low latency on small messages, multicast, or features TCP can't provide (like QUIC's no-head-of-line blocking), use UDP and build on top. If you need both reliability and UDP's flexibility, use QUIC if you can.

## Vocabulary

| Term | Plain English |
|------|---------------|
| **UDP** | User Datagram Protocol. A connectionless transport that throws self-contained packets between programs. |
| **Datagram** | A self-contained packet that has all the info needed to be understood on its own. The paper airplane. |
| **Header** | The little label on the front of a packet that says where it came from and where it's going. |
| **Source port** | A 16-bit number identifying the sending program's mailbox. |
| **Destination port** | A 16-bit number identifying the receiving program's mailbox. |
| **Length** | The 16-bit field in the UDP header giving the total UDP packet size in bytes (header + payload). |
| **Checksum** | A 16-bit math result used to detect packet corruption. The receiver throws away packets with bad checksums. |
| **Connectionless** | UDP doesn't open or close connections. Each packet stands alone. |
| **Stateless** | UDP keeps no per-conversation state in the kernel. |
| **MTU** | Maximum Transmission Unit. The largest packet size a network can carry without fragmentation. ~1500 on Ethernet. |
| **Fragmentation** | Splitting a too-big IP packet into smaller pieces that get reassembled at the receiver. |
| **DF bit** | Don't Fragment bit in the IP header. If set, oversized packets are dropped instead of fragmented. |
| **IPv4** | The old 32-bit-address version of IP. UDP runs on it. |
| **IPv6** | The new 128-bit-address version of IP. UDP also runs on it. |
| **Multicast** | Sending one packet to many subscribed receivers. UDP can do it, TCP cannot. |
| **Broadcast** | Sending one packet to everyone on a local network segment. UDP-only, IPv4-only. |
| **Anycast** | Routing a packet to whichever member of a group is "closest." Common for DNS root servers. |
| **mDNS** | Multicast DNS. Local-network service discovery on UDP port 5353. |
| **IGMP** | Internet Group Management Protocol. How IPv4 hosts join/leave multicast groups. |
| **MLD** | Multicast Listener Discovery. The IPv6 equivalent of IGMP. |
| **SSDP** | Simple Service Discovery Protocol. UPnP discovery on UDP port 1900. |
| **QUIC** | Modern reliable transport built on UDP. Powers HTTP/3. |
| **DTLS** | Datagram TLS. TLS adapted to work over the unreliable UDP. |
| **RTP** | Real-time Transport Protocol. The standard for audio/video media on UDP. |
| **RTCP** | RTP Control Protocol. Sends control and stats alongside RTP. |
| **SRTP** | Secure RTP. RTP with encryption baked in. |
| **ZRTP** | Key agreement protocol for SRTP that doesn't need a PKI. |
| **DNS** | Domain Name System. Resolves names to IPs. UDP port 53 by default. |
| **DHCP** | Dynamic Host Configuration Protocol. Hands out IP addresses. UDP ports 67/68. |
| **NTP** | Network Time Protocol. Syncs clocks. UDP port 123. |
| **SNMP** | Simple Network Management Protocol. Polls device stats. UDP port 161/162. |
| **Syslog** | The old Unix logging protocol. UDP port 514. |
| **TFTP** | Trivial File Transfer Protocol. Tiny file transfer. UDP port 69. |
| **NFS-over-UDP** | Network File System over UDP. Supported but mostly historical; modern NFS prefers TCP. |
| **Socket** | The kernel's handle for one end of a network conversation. |
| **sockaddr** | A C struct that holds an address and port for a socket. |
| **recvfrom** | The syscall a UDP receiver uses to pull one datagram out of the kernel. |
| **sendto** | The syscall a UDP sender uses to push one datagram into the kernel. |
| **SO_REUSEADDR** | Socket option that allows reusing a local address. |
| **SO_REUSEPORT** | Socket option that allows multiple sockets to share a port with kernel-level load distribution. |
| **SO_BROADCAST** | Socket option that allows sending to broadcast addresses. |
| **IP_MULTICAST_IF** | Socket option to choose which interface multicast packets go out. |
| **IP_MULTICAST_TTL** | Socket option that sets the TTL on outgoing multicast packets. |
| **IP_ADD_MEMBERSHIP** | Socket option to join a multicast group on IPv4. |
| **IPV6_JOIN_GROUP** | Socket option to join a multicast group on IPv6. |
| **ICMP unreachable** | An IPv4 error message saying "nobody listening at that destination." |
| **ICMPv6 unreachable** | The IPv6 equivalent. Error message saying the destination cannot be reached. |
| **ECONNREFUSED** | The errno your UDP socket returns when an ICMP unreachable came back. |
| **EMSGSIZE** | The errno when your packet is too big to send (usually due to path MTU). |
| **IP_PMTUDISC_DO** | Socket option to enable Path MTU Discovery. |
| **SO_RCVBUF** | Socket option to set the receive buffer size in bytes. |
| **SO_SNDBUF** | Socket option to set the send buffer size in bytes. |
| **sk_buff** | The Linux kernel's central data structure for representing a packet. |
| **udp_table** | The kernel's hash table of all UDP sockets. |
| **Hash** | A way of mapping inputs to bucket numbers for fast lookup. |
| **GSO** | Generic Segmentation Offload. Lets the kernel batch-build many UDP packets cheaply. |
| **GRO** | Generic Receive Offload. The receive-side counterpart of GSO. |
| **UFO** | UDP Fragmentation Offload. Older offload mechanism, deprecated in modern kernels. |
| **Checksum offload** | NIC hardware computing UDP/TCP checksums so the CPU doesn't have to. |
| **sendfile** | A zero-copy syscall for sending file data directly to a socket — TCP-only, no UDP equivalent. |
| **sendmmsg** | A syscall to send multiple UDP packets in one call, reducing syscall overhead. |
| **recvmmsg** | A syscall to receive multiple UDP packets in one call. |
| **MSG_DONTWAIT** | Flag to make a recv/send non-blocking. |
| **MSG_PEEK** | Flag to read a packet without removing it from the socket queue. |
| **MSG_CONFIRM** | Flag telling the kernel "this is a successful reply, refresh the ARP entry." |
| **MSG_ERRQUEUE** | Flag for receiving error-queue messages, like ICMP unreachables. |
| **Ephemeral port** | A short-lived random source port chosen by the kernel for outgoing connections. |
| **Well-known port** | Standard ports below 1024 like 53 (DNS) or 123 (NTP) reserved for known services. |
| **5-tuple** | The set of (source IP, source port, dest IP, dest port, protocol) that uniquely identifies a flow. |
| **NAT** | Network Address Translation. Routers rewriting source/dest addresses to share a single public IP. |
| **NAT traversal** | Tricks UDP-based protocols use to work through NATs (STUN, TURN, hole punching). |
| **STUN** | Session Traversal Utilities for NAT. Tells you what your public address is from outside. |
| **TURN** | Traversal Using Relays around NAT. Relays UDP through a server when NAT can't be punched. |
| **Hole punching** | A trick for two NATted peers to talk by simultaneously sending UDP at each other. |
| **Conntrack** | The Linux netfilter module that tracks per-flow state for stateful firewalling, including UDP "pseudo-flows." |
| **Netcat (nc)** | A general-purpose tool for moving bytes over a network. `-u` flag for UDP. |
| **socat** | A more capable netcat-alike, great for relays, multi-client UDP listeners, etc. |
| **iperf3** | A bandwidth and packet-loss measurement tool. `-u` flag for UDP. |
| **tcpdump** | The command-line packet capture tool, despite the name it captures UDP too. |
| **Wireshark** | The GUI version of tcpdump. Reads UDP just fine, with great dissectors. |
| **dig** | The DNS query tool. Most queries are UDP. |
| **bpftrace** | An eBPF tool for tracing kernel functions, including UDP send/receive. |
| **eBPF** | A safe in-kernel virtual machine that lets programs hook into kernel events. |
| **CoAP** | Constrained Application Protocol. An IoT protocol on UDP. |
| **CoAPS** | CoAP secured with DTLS. |
| **WireGuard** | A modern UDP-based VPN protocol with simple, fast cryptography. |
| **OpenVPN** | An older VPN that runs in either UDP or TCP mode. UDP mode is much faster. |
| **IKEv2** | The IPsec key exchange protocol. UDP port 500 (or 4500 for NAT-traversal). |

## Try This

These experiments are safe. Run them and watch what happens. There's nothing here that will break your computer.

### Experiment 1: Watch your computer's UDP DNS traffic

Open two terminals.

In terminal 1:

```
$ sudo tcpdump -i any -n udp port 53
```

In terminal 2:

```
$ dig google.com
$ dig anthropic.com
$ dig example.com
```

Watch the packets fly in terminal 1. You should see two packets per query (the question and the answer). Notice the source ports on the queries are random (ephemeral) and the destination is 53. The reply has source 53 and destination = your ephemeral port.

### Experiment 2: A toy UDP echo server

Run a simple UDP listener in one terminal:

```
$ socat - UDP-LISTEN:9999,fork
```

In a second terminal, send some packets:

```
$ echo "hello" | nc -u -w 1 127.0.0.1 9999
$ echo "world" | nc -u -w 1 127.0.0.1 9999
```

Watch them appear in the listener. You just used UDP to talk to localhost. Each message is a separate datagram.

### Experiment 3: Compare TCP and UDP packet captures side by side

In terminal 1:

```
$ sudo tcpdump -i lo -n -nn -c 30 port 12345
```

In terminal 2 — TCP:

```
$ nc -l 12345 &
$ echo "hi tcp" | nc -w 1 127.0.0.1 12345
```

Look at the captured packets. You'll see SYN, SYN-ACK, ACK (the handshake), then the data, then the FIN dance. Six or more packets.

Now in terminal 2 — UDP:

```
$ nc -u -l 12345 &
$ echo "hi udp" | nc -u -w 1 127.0.0.1 12345
```

You'll see ONE packet for the data. That's the whole exchange. The contrast is dramatic.

### Experiment 4: Saturate localhost with UDP

```
$ iperf3 -s -u &
$ iperf3 -u -c 127.0.0.1 -b 1G -t 3
```

This blasts a gigabit per second of UDP at the localhost loopback for 3 seconds. Watch the throughput report. You'll see real packet loss numbers, even on localhost, because the kernel buffer can fill up.

### Experiment 5: See DNS fall back to TCP

Some queries are too big for UDP. Find one:

```
$ dig +short any google.com | wc -l
$ dig +short txt google.com | head -5
```

Now compare:

```
$ dig +nocomment +nostats any google.com | grep -i tc
;; flags: qr rd ra tc; ...
```

If you see `tc` in the flags, the response was truncated and the resolver is supposed to retry over TCP. With `+tcp` you can see the full response. This is the UDP-to-TCP fallback in action.

### Experiment 6: Watch a video call's UDP traffic

If you have a video call app running, use `bpftrace` to see how chatty UDP is during a call:

```
$ sudo bpftrace -e 'kprobe:udp_sendmsg /comm == "Discord"/ { @ = count(); }'
^C
@: 31420
```

(Replace `Discord` with whatever your call app is.) That counted UDP send calls during the time the trace was running. Tens of thousands of UDP packets per minute is normal for a video call.

### Experiment 7: Multicast — see who's on your local network

```
$ avahi-browse -arpt 2>/dev/null | head -20
```

`avahi-browse` discovers services advertised on the local network via mDNS (multicast UDP on port 5353). You'll see printers, AirPlay devices, network shares, etc. Each one was found via UDP multicast.

### Experiment 8: Block UDP and watch DNS break

(Only do this if you understand undoing the change.)

```
$ sudo iptables -A OUTPUT -p udp --dport 53 -j DROP
$ dig +timeout=2 example.com
;; communications error to ...: timed out
```

DNS dies because UDP is blocked. Restore:

```
$ sudo iptables -D OUTPUT -p udp --dport 53 -j DROP
```

Now DNS works again. This shows just how dependent your system is on UDP for everyday things.

### Experiment 9: Time a UDP DNS query versus a TCP DNS query

```
$ time dig +short google.com
real    0m0.018s

$ time dig +tcp +short google.com
real    0m0.054s
```

UDP is roughly 3x faster for the same answer. Your numbers will vary, but UDP will always win for small queries because there's no handshake.

### Experiment 10: Build your own UDP packet by hand

```
$ python3 -c "
import socket
s = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
s.sendto(b'hi from python', ('127.0.0.1', 9999))
print('sent')
"
```

Run this with `socat - UDP-LISTEN:9999` in another terminal. Watch the line "hi from python" appear. You just sent a UDP packet from Python with three lines of code.

## Where to Go Next

When this sheet feels easy, the dense engineer-grade material is one command away. Stay in the terminal:

- **`cs networking udp`** — the dense reference. Real names for every option, every error, every kernel knob.
- **`cs detail networking/udp`** — the academic underpinning. Header math, GSO/GRO offload, the deep theory.
- **`cs networking tcp`** — UDP's reliable cousin. Read after this for the contrast.
- **`cs networking dns`** — the most famous UDP user.
- **`cs networking quic`** — the modern reliable transport on UDP.
- **`cs networking icmp`** — the protocol that delivers UDP's "connection refused" messages.
- **`cs ramp-up tcp-eli5`** — the paired ELI5 for the careful, slow, reliable side.
- **`cs ramp-up ip-eli5`** — what UDP and TCP both ride on top of.
- **`cs ramp-up icmp-eli5`** — the kid that delivers error messages.
- **`cs ramp-up http3-quic-eli5`** — the deep dive on QUIC, the future built on UDP.
- **`cs ramp-up linux-kernel-eli5`** — the kernel that actually moves your UDP packets around.
- **`cs kernel-tuning network-stack-tuning`** — when you want to tune UDP buffers and offloads for performance.

## See Also

- `networking/udp`
- `networking/tcp`
- `networking/ip`
- `networking/ipv4`
- `networking/ipv6`
- `networking/dns`
- `networking/dhcp`
- `networking/quic`
- `networking/icmp`
- `kernel-tuning/network-stack-tuning`
- `ramp-up/tcp-eli5`
- `ramp-up/ip-eli5`
- `ramp-up/icmp-eli5`
- `ramp-up/http3-quic-eli5`
- `ramp-up/linux-kernel-eli5`

## References

- **RFC 768** — User Datagram Protocol (1980). The whole spec is three pages. Read it once; it'll take you ten minutes.
- **RFC 8085** — UDP Usage Guidelines. The modern guide to building applications on UDP.
- **RFC 6347** — Datagram Transport Layer Security 1.2.
- **RFC 9147** — Datagram Transport Layer Security 1.3.
- **RFC 9000** — QUIC: A UDP-Based Multiplexed and Secure Transport.
- **RFC 9001** — Using TLS to Secure QUIC.
- **RFC 9002** — QUIC Loss Detection and Congestion Control.
- **RFC 3550** — RTP: A Transport Protocol for Real-Time Applications.
- **RFC 3711** — The Secure Real-time Transport Protocol (SRTP).
- **RFC 1035** — Domain Names — Implementation and Specification (DNS, including UDP usage).
- **RFC 6891** — Extension Mechanisms for DNS (EDNS0). How DNS got around the 512-byte UDP limit.
- **RFC 3828** — UDP-Lite, the partial-checksum variant.
- **`man 7 udp`** — the Linux UDP man page. Type `man 7 udp` in your terminal.
- **`man 2 recvfrom`** — the syscall for reading UDP datagrams.
- **`man 2 sendto`** — the syscall for sending UDP datagrams.
- **`man 7 socket`** — generic socket man page; covers many UDP-relevant options.
- **`man 7 ip`** — the IP layer man page; covers IP-level options that affect UDP.
- **"TCP/IP Illustrated, Volume 1: The Protocols"** by Stevens, Fall — the canonical book. Chapter on UDP is short, dense, perfect.
- **"The Linux Programming Interface"** by Kerrisk — exhaustive coverage of UDP socket programming.
- **`/proc/net/udp`** — kernel's live UDP socket table. Type `cat /proc/net/udp` to see it.
- **`/proc/net/snmp`** — UDP statistics counters maintained by the kernel.

Tip: every reference above can be read inside your terminal. RFCs can be read with `curl https://www.rfc-editor.org/rfc/rfc768.txt | less`. Man pages are right there. The book references can be downloaded as PDFs and read in a terminal-based viewer like `zathura`. You really do not need to leave the terminal.

— End of ELI5 —

When this sheet feels boring (and it will, faster than you think), graduate to `cs networking udp` — the engineer-grade reference. It uses real names for everything: socket options, kernel data structures, every flag, every counter. After that, `cs detail networking/udp` gives you the academic underpinning. By the time you've read both, you will be reading UDP packet captures and reasoning about congestion behavior without a flinch.

### One last thing before you go

Pick one command from the Hands-On section that you haven't run yet. Run it right now. Read the output. Try to figure out what each part means, using the Vocabulary table as your dictionary. Don't just trust this sheet — see for yourself. UDP is real. It is on your computer, throwing paper airplanes around the network, right now. The commands in this sheet let you peek at it.

Reading is good. Doing is better. Type the commands. Watch the airplanes fly.

The whole point of the North Star for the `cs` tool is: never leave the terminal to learn this stuff. Everything you need is here, or one `man` page away, or one RFC away. There is no Google search you need to do to start understanding UDP. You can sit at your terminal, type, watch, read, and learn forever.

Have fun. UDP is happy to be poked at. Nothing on this sheet will break anything. Try things. Type commands. Read what comes back. The more you do, the more it all clicks into place.

Throw the airplane. See where it lands. Throw another.

### A final picture to take with you

```
   TCP                                    UDP
   "certified mail"                       "paper airplane out the window"
   ----------------                       --------------------------------
   handshake first                        no handshake
   acknowledged                           hope it lands
   ordered                                arrives in any order
   retransmitted                          gone is gone
   flow-controlled                        senders go full speed
   congestion-aware                       blast away
   one-to-one                             one-to-many possible
   20+ byte header                        8 byte header
   stateful (kernel tracks)               stateless (kernel doesn't)
   reliable byte stream                   self-contained datagrams
   slow to start, smooth                  instant, jagged
   you trust the protocol                 you build what you need
   browser, ssh, email, files             dns, ntp, voice, video, games, quic
```

Both are useful. Both are everywhere. Both ride on top of the same IP packets, on the same wires, going to the same machines. The difference is what each protocol promises you. TCP promises a lot. UDP promises almost nothing. Pick the one whose promises match what you actually need.

Now go run a few commands. Read a UDP packet capture. Watch DNS go by. Hear a video call's UDP packets in your `bpftrace` output. Once you've seen UDP working, you'll never confuse it for TCP again. The paper airplanes are real. They're flying around your computer right now.
