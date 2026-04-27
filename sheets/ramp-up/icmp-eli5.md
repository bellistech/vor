# ICMP — ELI5 (The Internet's Walkie-Talkie)

> ICMP is the messenger protocol the network uses to talk to itself about how packets are doing — error messages, ping/pong, "are you there?" notes carried between routers and hosts, never used for your actual data.

## Prerequisites

(none)

## What Even Is ICMP?

Imagine the internet is a gigantic apartment building with thousands of
hallways and millions of rooms. Each room is a computer. Each hallway
is a network link. The mail in this building is what we call IP packets —
little envelopes with a destination written on the front, a sender written
on the back, and a payload tucked inside.

Most of the time, the mail moves around the building just fine. A web
page request walks from your room out into your hallway, takes the
elevator down, walks across the lobby, gets handed off to the mail truck
at the loading dock, drives across town, climbs up another building,
walks down another hallway, and gets dropped off at the right room.
Easy.

But sometimes things go wrong. The hallway is closed for maintenance.
The room number doesn't exist. The envelope is too thick to fit through
a mail slot. The mail carrier has been walking forever and the envelope
is now expired. When that happens, somebody has to tell somebody else
what went wrong, or the sender will sit there forever wondering what
happened to their letter.

That somebody is ICMP. The Internet Control Message Protocol. The
network's walkie-talkie. The internet's "hey, dude, your envelope didn't
make it" loudspeaker.

ICMP is not for sending your data. ICMP doesn't carry web pages or
emails or video streams. ICMP is the network talking to itself about
how packets are being delivered. Think of it as the radio chatter between
all the mail carriers, the elevator operators, the security guards, and
the lobby receptionists in our giant apartment building. They use ICMP
to coordinate, to report problems, to ask "are you still there?" and to
hear back "yep, still here."

ICMP runs directly on top of IP. There's no TCP or UDP underneath it.
The IP header has a field called "protocol" — when that field is set
to 1, the packet's payload is an ICMP message. When it's 6, that's TCP.
When it's 17, that's UDP. When it's 58, that's ICMPv6 (the IPv6 flavor
of ICMP, which is even more important than the IPv4 flavor — more on
that later).

Because ICMP rides on top of IP and not on top of TCP or UDP, ICMP
doesn't have ports. There's no "ICMP port 80" or "ICMP port 443." Ports
are a TCP/UDP idea — they let multiple programs on the same machine
share the same IP address by giving each program a numbered mailbox.
ICMP doesn't need that, because ICMP isn't for programs. ICMP is for
the network itself. The kernel handles every ICMP message. Most of the
time, your application never sees them at all (the one big exception
is `ping`, which is more or less the only normal program that talks
ICMP directly through a special raw socket).

Every ICMP message has the same little header at the start:

```
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|     Type      |     Code      |          Checksum             |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                  Rest of header (varies by Type)              |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                            Payload                            |
|                              ...                              |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

Three things in the header tell you everything you need to know:

- **Type** — what kind of message is this? Ping? Error? Redirect?
- **Code** — within that type, which specific flavor? "Network unreachable"
  vs. "Host unreachable" vs. "Port unreachable" vs. "Fragmentation needed"
  are all subtypes of one big "Destination Unreachable" type.
- **Checksum** — paranoia. Did anything get corrupted?

That's it. That's the whole protocol. Everything else is variations on
this theme — little extra fields after the type/code/checksum, plus
maybe a snippet of the original packet that caused the problem so the
sender can figure out which of their packets just failed.

ICMP messages are usually short — sometimes just 8 bytes plus 28 bytes
of "the original IP header and 8 bytes of payload" so the sender can
tell which connection the error is about. That's why ICMP is light and
fast — it's the network's text-message system, not a phone call.

## Ping: The Most Famous ICMP Use

If you've ever typed `ping` in a terminal, you've used ICMP. `ping` is
literally the canonical example of an ICMP-based tool. Here's how it
works.

`ping` sends a special ICMP message called **Echo Request** (Type 8 in
IPv4, Type 128 in IPv6). The Echo Request says, in effect: "Hey, computer
at this address, are you there? If you are, please send this exact
message right back to me." Inside the Echo Request is a tiny payload —
usually 56 bytes of patterned data on Linux, 32 bytes on Windows — and
two important little numbers: an **identifier** and a **sequence number**.

The identifier is so the receiving side knows which `ping` process the
reply belongs to. On Linux, by default, it's the process ID of the ping
program. The sequence number starts at 1 and goes up by one with each
ping, so you can tell which reply matches which request and you can
spot dropped pings (a missing sequence number means a packet got lost).

The receiving computer's kernel sees the ICMP Echo Request, looks at
the type, sees "oh, type 8, that's a ping," and immediately sends back
an **Echo Reply** (Type 0 in IPv4, Type 129 in IPv6) with the same
identifier, the same sequence number, and a copy of the same payload
the original Echo Request carried. The reply doesn't even involve any
user-space program — it's just the kernel saying "yep, I'm here, here's
your bytes back."

On the sender side, `ping` was busy sending one Echo Request per second.
For each Echo Request it sends, it stamps the local time into the
payload and remembers the sequence number. When an Echo Reply comes
back, `ping` looks at the time it stamped into the payload, subtracts
it from the current time, and out pops the **round-trip time**, or RTT.
RTT is the total time from "I asked" to "I got an answer back" — the
single most important number in network diagnostics.

A typical ping looks like this:

```
$ ping -c 4 8.8.8.8
PING 8.8.8.8 (8.8.8.8) 56(84) bytes of data.
64 bytes from 8.8.8.8: icmp_seq=1 ttl=117 time=12.3 ms
64 bytes from 8.8.8.8: icmp_seq=2 ttl=117 time=11.9 ms
64 bytes from 8.8.8.8: icmp_seq=3 ttl=117 time=12.4 ms
64 bytes from 8.8.8.8: icmp_seq=4 ttl=117 time=12.1 ms

--- 8.8.8.8 ping statistics ---
4 packets transmitted, 4 received, 0% packet loss, time 3004ms
rtt min/avg/max/mdev = 11.872/12.183/12.412/0.205 ms
```

Decode that line by line:

- `PING 8.8.8.8 (8.8.8.8) 56(84) bytes of data.` — we're pinging
  8.8.8.8 (Google's DNS), each ping is 56 bytes of payload (ICMP echo
  data) plus an 8-byte ICMP header plus a 20-byte IP header = 84 bytes
  total on the wire.
- `icmp_seq=1` — first ping. Started at one. Goes up.
- `ttl=117` — when Google sent the reply, they set TTL to whatever
  their default is (usually 64 or 128 or 255). It came back with 117
  remaining, meaning the reply traveled through (128 − 117) = 11 routers
  from Google to me, or (255 − 117) = 138 if Google starts at 255.
- `time=12.3 ms` — round-trip took 12.3 milliseconds.
- `0% packet loss` — every Echo Request got a reply. Healthy link.
- `rtt min/avg/max/mdev` — the spread of times. `mdev` is the mean
  deviation, a rough measure of jitter (how much the RTT bounces around).

Because ICMP doesn't have ports, ping doesn't aim at "8.8.8.8 port 80"
or anything like that — ping just aims at the IP address. The kernel
on the other side answers regardless of what's running. (Unless the
kernel has been told not to answer, which we'll cover later.)

Ping is nearly the entire ICMP user experience for most people, but
ICMP does a lot more than that.

## ICMP Types and Codes

There are dozens of ICMP types in the IANA registry, but the ones you'll
actually run into in the wild are about eight. Memorize these.

### Echo Request (Type 8) and Echo Reply (Type 0)

The ping/pong duo. Echo Request says "send this back," Echo Reply says
"here it is." This is the only ICMP exchange a normal user usually
triggers on purpose. Both use the same packet format with an identifier
and sequence number after the type/code/checksum.

In IPv6, these are Types 128 (Echo Request) and 129 (Echo Reply). The
behavior is identical, just different numbers.

### Destination Unreachable (Type 3)

This is the big one. When something can't deliver a packet to where it's
trying to go, somebody (usually the last router that tried, or sometimes
the destination host itself) sends back a Destination Unreachable
message. The **code** field tells you why. There are 16 codes (0–15)
defined in the original RFC 792 and a few more added later. The most
common ones:

- **Code 0 — Network Unreachable.** The router has no route to the
  destination network at all. "I don't know how to get to that whole
  zip code."
- **Code 1 — Host Unreachable.** The router has a route to the network
  but couldn't actually reach the specific host. "Yeah, that street
  exists, but nobody's home at that house."
- **Code 2 — Protocol Unreachable.** The destination doesn't speak the
  upper-layer protocol you're trying to use. Rare in practice.
- **Code 3 — Port Unreachable.** The destination host got the packet,
  but no program is listening on that TCP/UDP port. This is the most
  common ICMP error you'll see. Try `nc -u somehost.com 9999` and
  you'll trigger it.
- **Code 4 — Fragmentation Needed but DF Set.** Critical for Path MTU
  Discovery. We'll spend a whole section on this one.
- **Code 5 — Source Route Failed.** Source routing is basically dead;
  ignore this.
- **Code 6 — Destination Network Unknown.** Slightly different shade
  of code 0; usually means a routing entry is broken on purpose.
- **Code 7 — Destination Host Unknown.** Similar shade of code 1.
- **Code 8 — Source Host Isolated.** Old, rarely seen.
- **Code 9 — Network Administratively Prohibited.** A firewall said no
  to your whole network.
- **Code 10 — Host Administratively Prohibited.** A firewall said no
  to that one host.
- **Code 11 — Network Unreachable for ToS.** "I have a route, but not
  for the type of service you asked for." Rarely seen.
- **Code 12 — Host Unreachable for ToS.** Same idea, host-level.
- **Code 13 — Communication Administratively Prohibited.** Firewall
  block. The packet was filtered. Common.
- **Code 14 — Host Precedence Violation.** Almost never seen.
- **Code 15 — Precedence Cutoff.** Almost never seen.

### Time Exceeded (Type 11)

Two codes:

- **Code 0 — TTL Expired in Transit.** The TTL field in the IP header
  hit zero before the packet got to its destination. The router that
  decremented TTL to zero throws the packet away and sends back a Time
  Exceeded.
- **Code 1 — Fragment Reassembly Time Exceeded.** The destination got
  some fragments of a packet but the rest of the fragments never showed
  up before its reassembly timer expired.

Time Exceeded is what `traceroute` exploits. We'll come back to that.

### Redirect (Type 5)

A router noticed you're sending packets to it that would be better off
going to a different router on the same network. The Redirect message
says "hey, for that destination, talk to the other router instead."

Modern operating systems usually ignore ICMP Redirects by default for
security reasons (it's an easy way to attack you). On Linux, you can
disable acceptance of redirects with:

```
sysctl -w net.ipv4.conf.all.accept_redirects=0
sysctl -w net.ipv6.conf.all.accept_redirects=0
```

Codes:

- **Code 0 — Redirect for Network.**
- **Code 1 — Redirect for Host.**
- **Code 2 — Redirect for ToS and Network.**
- **Code 3 — Redirect for ToS and Host.**

### Source Quench (Type 4) — Deprecated

Used to mean "you're sending too fast, slow down." Officially deprecated
in RFC 6633 (2012). Don't generate it. Don't trust it. Modern flow control
happens at TCP, not at ICMP. The TCP/ECN/QoS stack handles congestion
much better than ICMP ever did.

### Router Advertisement (Type 9) and Router Solicitation (Type 10)

These are part of how hosts find their default router. In IPv4, almost
nobody uses them — DHCP does this job. In IPv6, the equivalents (Type
133 Router Solicitation, Type 134 Router Advertisement, in ICMPv6) are
critical and used constantly. They're how IPv6 stateless address
autoconfiguration (SLAAC) works.

### Parameter Problem (Type 12)

A router or destination host couldn't process the packet because some
field in the IP header was malformed. Codes:

- **Code 0 — Pointer indicates the error.** The "rest of header" field
  contains a pointer to the byte where the bad value is.
- **Code 1 — Missing a Required Option.**
- **Code 2 — Bad Length.**

You shouldn't see these often. If you do, somebody is generating malformed
packets — that's either a bug or an attack.

### Timestamp Request (Type 13) and Timestamp Reply (Type 14)

Old; rarely used. Lets two hosts compare clocks via ICMP. NTP is way
better and almost everyone uses NTP, so timestamp ICMP is mostly a
historical curiosity. Many firewalls block these because they can leak
information about a host.

### Address Mask Request (Type 17) and Address Mask Reply (Type 18)

Even more deprecated than timestamps. Almost nobody uses them. DHCP
does this job. Don't generate them.

That's the IPv4 cheat sheet. For ICMPv6, the same ideas with different
numbers, plus a whole new family of types (133–137) for Neighbor
Discovery, which we'll get to in the IPv6 section.

## Destination Unreachable Codes

Of all the ICMP types, Destination Unreachable (Type 3) is the one you
need to know cold. It tells you, in one byte (the code), exactly why
your packet died.

Quick reference at the ELI5 level:

```
Code  Name                                What it means
----  --------------------------------    -----------------------------
 0    Network Unreachable                 No route to that whole network
 1    Host Unreachable                    Have route but host is offline
 2    Protocol Unreachable                Host doesn't speak that proto
 3    Port Unreachable                    No program listening on port
 4    Fragmentation Needed, DF Set        Packet too big, DF blocked split
 5    Source Route Failed                 Old; ignore
 6    Destination Network Unknown         Variant of 0
 7    Destination Host Unknown            Variant of 1
 8    Source Host Isolated                Old; ignore
 9    Network Administratively Prohibited Firewall blocked the network
10    Host Administratively Prohibited    Firewall blocked the host
11    Network Unreachable for ToS         Old; ignore
12    Host Unreachable for ToS            Old; ignore
13    Communication Admin Prohibited      Firewall said no
14    Host Precedence Violation           Old; ignore
15    Precedence Cutoff                   Old; ignore
```

The codes you'll actually see in the wild:

### Code 0 — Network Unreachable

```
$ ping 192.0.2.99
PING 192.0.2.99 (192.0.2.99) 56(84) bytes of data.
From 10.0.0.1 icmp_seq=1 Destination Net Unreachable
```

The first router on your way out doesn't know how to get to that whole
network. Usually a routing problem upstream — your ISP forgot a route,
or BGP withdrew an announcement, or the network was decommissioned.

### Code 1 — Host Unreachable

```
$ ping 192.168.1.99
PING 192.168.1.99 (192.168.1.99) 56(84) bytes of data.
From 192.168.1.1 icmp_seq=1 Destination Host Unreachable
```

The router has a route to the LAN but can't ARP-resolve the host. The
host is probably offline, or its ARP entry timed out and the host isn't
answering.

### Code 3 — Port Unreachable

```
$ nc -u 8.8.8.8 9999
^C
```

Followed by tcpdump showing:

```
IP 8.8.8.8 > 10.0.0.5: ICMP 8.8.8.8 udp port 9999 unreachable, length 36
```

You sent a UDP packet to a port nobody's listening on. The destination
host's kernel says "nope, no program here, sorry" via ICMP. This is
the way DNS clients can tell when a DNS server is up but not listening
on the right port. It's also how `traceroute` knows it has reached the
destination (more on that below).

### Code 4 — Fragmentation Needed but DF Set

```
$ ping -c 1 -s 1500 -M do 8.8.8.8
PING 8.8.8.8 (8.8.8.8) 1500(1528) bytes of data.
From 10.0.0.1 icmp_seq=1 Frag needed and DF set (mtu = 1500)
```

This is the Path MTU Discovery message and it's the most operationally
critical ICMP message in the entire protocol. We'll spend a whole
section on this one because if you block it, the internet quietly
breaks.

### Code 13 — Communication Administratively Prohibited

```
$ ping 192.0.2.50
From 10.0.0.1 icmp_seq=1 Packet filtered
```

A firewall in the path explicitly said no. Some firewalls send this
politely (so the sender knows to give up); many just drop the packet
silently with no ICMP at all (a "stealth" firewall).

## Time Exceeded — How traceroute Works

Here's where ICMP gets clever and beautiful at the same time. The
`traceroute` tool figures out the entire path your packets take to a
destination — every router along the way — by deliberately sending out
packets it knows will fail, and then reading the failure messages.

### The TTL Trick

Every IP packet has an 8-bit field called **TTL** (Time To Live) in
IPv4 — or **Hop Limit** in IPv6. It starts at some value (usually 64
or 128 or 255 depending on the OS). Every time a router forwards the
packet, the router decrements TTL by 1. If a router decrements TTL and
the result is 0, the router throws the packet away — and sends an ICMP
Time Exceeded message back to the original sender.

This was originally a safety feature so packets that got into a routing
loop wouldn't circulate forever. But traceroute weaponizes it to map
the network.

### How Traceroute Uses It

Traceroute starts by sending a packet with TTL = 1.

```
[Me] ----TTL=1----> [R1]
   R1: "TTL hit zero, throw it away. Send Time Exceeded back to Me."
[Me] <----ICMP Time Exceeded from R1----
```

I just learned the IP address of router R1 — the first hop on my way
to the destination!

Now traceroute sends a packet with TTL = 2.

```
[Me] ----TTL=2----> [R1] ---TTL=1---> [R2]
   R2: "TTL hit zero, throw it away. Send Time Exceeded back to Me."
[Me] <----ICMP Time Exceeded from R2----
```

Now I know R2.

Traceroute keeps doing this — TTL=3, TTL=4, TTL=5 — and each time it
gets back a Time Exceeded from a different router. It builds up a list:
R1, R2, R3, R4, R5...

Eventually, the packet reaches the actual destination. The destination
doesn't decrement TTL the same way — it just tries to deliver the packet
to a program. If traceroute used a UDP packet aimed at a high port
(default), the destination sends back a Port Unreachable instead of a
Time Exceeded — and traceroute knows it's done.

Here's an ASCII diagram of the whole thing:

```
                                                    Destination
                                                         |
                                                         v
   Me      R1      R2      R3      R4      R5     [Server]
   |       |       |       |       |       |          |
   |--TTL=1->|     |       |       |       |          |
   |<-Time Exceeded                                    |
   |       |       |       |       |       |          |
   |--TTL=2-------->|      |       |       |          |
   |<-Time Exceeded                                    |
   |       |       |       |       |       |          |
   |--TTL=3---------------->|      |       |          |
   |<-Time Exceeded                                    |
   |       |       |       |       |       |          |
   |--TTL=4------------------------>|      |          |
   |<-Time Exceeded                                    |
   |       |       |       |       |       |          |
   |--TTL=5-------------------------------->|         |
   |<-Time Exceeded                                    |
   |       |       |       |       |       |          |
   |--TTL=6---------------------------------> reaches |
   |<-Port Unreachable from server -----------         |
                                                       |
              "Done! Found the whole path."
```

The output looks like this:

```
$ traceroute 8.8.8.8
traceroute to 8.8.8.8 (8.8.8.8), 30 hops max, 60 byte packets
 1  _gateway (10.0.0.1)            1.234 ms  1.110 ms  1.008 ms
 2  isp-rtr-1.example.net (1.2.3.4) 5.012 ms 4.998 ms 5.105 ms
 3  isp-core-1.example.net (1.2.3.5) 9.234 ms 9.112 ms 9.005 ms
 4  google-peer.example.net (1.2.3.6) 11.111 ms 11.005 ms 10.998 ms
 5  108.170.225.193  11.523 ms  11.612 ms  11.434 ms
 6  216.239.50.123  11.890 ms  11.745 ms  11.812 ms
 7  8.8.8.8  12.123 ms  12.045 ms  12.000 ms
```

Each line is one hop. The three numbers per line are three separate
probes (traceroute sends three by default, to spot variability).

### Why You See Stars

Sometimes you see a line like this:

```
 5  * * *
```

That hop didn't reply. The router silently dropped the probe instead
of sending a Time Exceeded. Possible reasons:

1. The router is configured not to generate ICMP Time Exceeded (rate-limited or disabled).
2. There's a firewall between you and the router that drops the ICMP.
3. The reverse-path filter on the router rejected the probe.
4. The router itself is overloaded and didn't get around to processing the probe.

Stars don't mean the path is broken. They mean ICMP is being suppressed
at that hop. Other hops will still answer.

### Probe Types

By default, Linux `traceroute` uses UDP packets aimed at high ports
(33434 and up). Macs and Windows use ICMP Echo Requests by default.
Many firewalls in cloud environments block ICMP entirely, in which case
you can use TCP-based probes:

```
traceroute -T -p 443 google.com
```

This sends TCP SYN packets with increasing TTL. Each router still
decrements TTL and sends back ICMP Time Exceeded. The destination sends
back a TCP RST or SYN-ACK depending on whether port 443 is open.
Beautifully reliable.

The `mtr` tool (My Traceroute) is traceroute + ping in a continuous
loop — it shows packet loss per hop in real time and is enormously
useful for diagnosing intermittent path problems.

## Path MTU Discovery — When ICMP Is Critical

Here's the section that explains why network engineers cry when they
see a firewall rule that says "DROP ICMP."

### The MTU Problem

Every link on the internet has a **Maximum Transmission Unit** (MTU).
That's the largest packet the link can carry without splitting. Ethernet
classically has an MTU of 1500 bytes. PPPoE-over-ADSL has an MTU of
1492. Some VPN tunnels run at 1400 or smaller. Some data center networks
run at 9000 ("jumbo frames").

When a sender wants to send a 1500-byte packet over a link with an MTU
of 1400, something has to give. Two options:

1. **Fragment the packet.** Split it into pieces small enough to fit.
   IPv4 routers can do this. IPv6 routers cannot.
2. **Tell the sender to send smaller packets.** Send back an ICMP
   "Fragmentation Needed but DF Set" message (Type 3, Code 4) and let
   the sender adjust.

Modern operating systems set the **Don't Fragment (DF)** bit on most
TCP packets. The DF bit tells routers "do NOT fragment this packet —
if it doesn't fit, drop it and tell me." This is how Path MTU Discovery
(PMTUD) works.

### How PMTUD Works

1. Sender sends a 1500-byte packet with DF=1.
2. Packet goes through Router A (MTU 1500), Router B (MTU 1500), Router
   C (MTU 1400), Router D...
3. Router C sees: "1500 > 1400, but DF is set, so I can't split it. I'll
   drop the packet and send back an ICMP Type 3 Code 4 Fragmentation
   Needed, with the next-hop MTU (1400) in the header."
4. Sender receives the ICMP Frag Needed message. Linux kernel updates
   its routing cache: "for this destination, max packet size is 1400."
5. Sender retransmits the data in 1400-byte chunks. Through the path
   they go, with no fragmentation needed.

Here's the flow as a diagram:

```
   Sender          R-A           R-B           R-C           Receiver
   (MTU 1500)    (MTU 1500)    (MTU 1500)    (MTU 1400)    (MTU 1500)
      |              |             |              |              |
      |-1500, DF=1->|              |              |              |
      |              |--1500,DF=1->|              |              |
      |              |              |--1500,DF=1-->|             |
      |              |              |     R-C: "Too big!"        |
      |              |              |  ICMP Type 3, Code 4        |
      |              |              | "Frag Needed, MTU=1400"    |
      |<------- ICMP message -----------------------|             |
      |   "Oh, MTU is 1400. Cache that."                          |
      |                                                            |
      |-1400, DF=1->|              |              |              |
      |              |--1400,DF=1->|              |              |
      |              |              |--1400,DF=1-->|             |
      |              |              |              |--1400,DF=1->|
      |                                                            |
      |              <-- delivered successfully! -->              |
```

### Why It Breaks: The ICMP Black Hole

If somebody, somewhere on the path, blocks ICMP — and specifically blocks
ICMP Type 3 Code 4 — then the sender never gets the message. Step 4
just doesn't happen. The sender keeps sending 1500-byte packets, they
keep hitting Router C, Router C keeps dropping them, and the sender
never knows why.

This is called an **ICMP black hole**. The connection just hangs.
Sometimes web pages load partially. Sometimes downloads stall at exactly
some random byte count and never finish. The TCP retransmission timer
keeps firing, packets keep getting dropped, and there is no error
message anywhere.

The fix is one of:

1. Don't block ICMP Type 3 Code 4 at firewalls. Let PMTUD work.
2. Lower your MTU manually so you never trigger the problem (e.g. set
   the local MTU to 1400 for VPN tunnels).
3. Use **TCP MSS clamping** on a router along the path — the router
   reaches into the TCP SYN packet and rewrites the MSS option to a
   smaller value, so the TCP handshake negotiates a lower segment size
   from the start.
4. Use **PLPMTUD** (Packetization Layer Path MTU Discovery, RFC 4821) —
   a probing-based method that doesn't require ICMP at all. Linux supports
   this and uses it by default for IPv6.

### Test for the Black Hole

```
$ ping -c 1 -s 1500 -M do 8.8.8.8
```

`-s 1500` makes the ping payload 1500 bytes. With an 8-byte ICMP header
and 20-byte IP header, that's 1528 bytes on the wire — too big for a
normal Ethernet link. `-M do` sets the DF bit ("don't fragment").

If PMTUD is working, you'll get back a "Frag needed and DF set" message,
or your kernel will silently retry with a smaller packet. If you're in
a black hole, the ping will just time out forever.

## ICMPv6 Is Different

ICMPv6 (RFC 4443, which obsoletes the older RFC 2463) is way more
important than ICMPv4. It does everything ICMPv4 does, plus:

- **Neighbor Discovery (NDP)** — replaces ARP. RFC 4861.
- **Router Advertisement (RA)** — replaces DHCP for stateless config (SLAAC). RFC 4862.
- **Multicast Listener Discovery (MLD)** — replaces IGMP for IPv6 multicast.
- **Duplicate Address Detection (DAD)** — checks if anyone else is using your IP.
- **Path MTU Discovery** — RFC 8201 (which obsoletes RFC 1981) — uses
  ICMPv6 Type 2 (Packet Too Big), since IPv6 routers don't fragment at all.

In IPv6, blocking ICMP doesn't just break PMTUD — it breaks everything.
Your computer can't even find its router without ICMPv6. RFC 4890
(Recommendations for Filtering ICMPv6) lays out what you can safely
block and what you absolutely must not.

### ICMPv6 Type Numbers

The numbers got renumbered in IPv6. Quick map:

```
Type   Name                              Replaces in IPv4
----   ------------------------------    -----------------------
   1   Destination Unreachable           ICMPv4 Type 3
   2   Packet Too Big                    ICMPv4 Type 3 Code 4
   3   Time Exceeded                     ICMPv4 Type 11
   4   Parameter Problem                 ICMPv4 Type 12
 128   Echo Request                      ICMPv4 Type 8
 129   Echo Reply                        ICMPv4 Type 0
 130   Multicast Listener Query          IGMP query
 131   Multicast Listener Report         IGMP report
 132   Multicast Listener Done           IGMP leave
 133   Router Solicitation               (new)
 134   Router Advertisement              (new)
 135   Neighbor Solicitation             ARP request
 136   Neighbor Advertisement            ARP reply
 137   Redirect                          ICMPv4 Type 5
```

### Neighbor Discovery — How ICMPv6 Replaces ARP

In IPv4, when your computer wants to send a packet to another machine
on the same LAN, it has to find that machine's MAC address. It does
this with ARP — a layer-2 broadcast: "Who has 192.168.1.5? Tell
192.168.1.10." The owner of that IP replies with its MAC.

In IPv6, ARP is gone. Replaced by ICMPv6 Neighbor Discovery (NDP).
Same job, different mechanism:

```
   Host A (fe80::a)                       Host B (fe80::b)
        |                                       |
        |--- ICMPv6 Type 135 ----->             |
        |    Neighbor Solicitation              |
        |    "Who has fe80::b? My MAC is..."    |
        |    (sent to solicited-node multicast) |
        |                                       |
        |     <----- ICMPv6 Type 136 -----------|
        |           Neighbor Advertisement       |
        |           "I'm fe80::b, my MAC is..."  |
```

The key difference from ARP: NDP runs over IPv6 (which means it has IP
headers, can be authenticated, can be filtered) and it uses multicast
instead of broadcast (more efficient on big networks). It also handles
several extra functions:

- **Solicited-Node Multicast Address.** Each IPv6 address has a derived
  multicast address (ff02::1:ff00:0/104 + last 24 bits of the unicast
  address). Hosts subscribe to this address. Neighbor Solicitations are
  sent to it, so only the relevant host has to process them — much
  better than ARP broadcasts, which every host on the LAN has to look at.
- **Duplicate Address Detection (DAD).** Before claiming an address, a
  host sends a Neighbor Solicitation for it. If anyone replies, the
  address is taken. RFC 4862.
- **Router Solicitation/Advertisement.** Type 133 (RS) asks "are there
  any routers here?" Type 134 (RA) advertises "yes, I'm a router, here's
  your default route, here are the prefixes you can use, here are some
  DNS servers." This is how SLAAC works — your IPv6 address is built
  from the prefix in the RA + your interface ID (often EUI-64-derived
  or a randomly generated stable identifier).

You don't need DHCP for IPv6 if you don't want it. RAs alone can give
you everything you need, courtesy of ICMPv6.

### Why You Cannot Block ICMPv6

If a firewall blocks ICMPv6 entirely:

- Your computer can't find its router (no Router Advertisements, no
  Router Solicitation replies).
- Your computer can't find other hosts on the LAN (no Neighbor Discovery).
- Path MTU Discovery is completely broken (because IPv6 routers don't
  fragment, the only signaling channel is ICMPv6 Type 2 Packet Too Big).
- DAD breaks, so duplicate IPv6 addresses can quietly happen.
- Multicast (e.g. mDNS, link-local services) breaks.

RFC 4890 says: **drop these specific types only**, and **must allow these
other types**. The basic safe set to allow:

- Type 1 (Destination Unreachable)
- Type 2 (Packet Too Big) — must allow
- Type 3 (Time Exceeded)
- Type 4 (Parameter Problem)
- Type 128, 129 (Echo Request/Reply, optional)
- Types 130–132 (Multicast Listener) on link-local
- Types 133–137 (NDP) on link-local

Don't block ICMPv6. Just don't.

## Security: Smurf, Ping of Death, Ping Floods

ICMP has been around since 1981 (RFC 792). It's old. It's been abused
in many creative ways. Modern operating systems and firewalls have
largely closed the holes, but knowing the history helps you understand
why some defaults are the way they are.

### Smurf Attack

Pre-2000s classic. The attacker spoofs the source address of an ICMP
Echo Request to be the victim's IP. Then the attacker sends that ICMP
Echo Request to the broadcast address of a large network.

```
                                       BIG NETWORK (10.0.0.0/8)
                                            ________
                                           |        |
   Attacker --- spoofed echo --->          | host 1 |--+
   src=victim, dst=10.255.255.255          | host 2 |--+--> all reply
                                           | host 3 |--+    to victim
                                           |  ...   |--+
                                           | host N |--+
                                           |________|         |
                                                              v
                                                          [Victim]
                                                       drowning in
                                                       Echo Replies
```

Every host on the network responds to the broadcast Echo Request. All
their replies go to the spoofed source — the victim. One small attack
packet becomes thousands of reply packets. Amplification attack.

Modern defenses:

1. Most operating systems no longer respond to broadcast pings. Linux:

   ```
   net.ipv4.icmp_echo_ignore_broadcasts = 1   # default since forever
   ```

2. Routers no longer forward IP broadcasts by default ("ip directed-broadcast"
   on Cisco is disabled).

3. ISPs do ingress filtering to drop spoofed source addresses.

The Smurf attack is essentially extinct today, but it taught a generation
of engineers about amplification.

### Ping of Death

Even older. From the era when IP fragmentation was poorly implemented.
The attacker sends a pile of IP fragments that, when reassembled, would
form a packet bigger than 65535 bytes (the maximum legal IP packet size).
Some old kernels would crash on overflow. Modern kernels reject the
malformed reassembly cleanly.

Effectively dead, but the term lives on as a warning about input
validation in low-level networking code.

### Ping Flood

The simplest attack. Just send a ton of pings as fast as possible. If
you have more bandwidth than the victim's link, you can saturate them.
This isn't a clever protocol exploit; it's just brute force.

Defenses:

1. Rate-limit ICMP at the kernel level:

   ```
   net.ipv4.icmp_ratelimit = 1000   # max ICMP per second (token bucket interval in ms)
   net.ipv4.icmp_ratemask = 0x1818  # which types are rate-limited
   ```

   The default ratemask covers Destination Unreachable, Source Quench,
   Time Exceeded, Parameter Problem.

2. Rate-limit ICMP at firewalls:

   ```
   iptables -A INPUT -p icmp --icmp-type echo-request \
       -m limit --limit 1/s -j ACCEPT
   iptables -A INPUT -p icmp --icmp-type echo-request -j DROP
   ```

3. Block ICMP at the network edge if you really need to. (But remember
   to allow Type 3 Code 4 for PMTUD.)

### ICMP Tunneling

Bad guys can hide data inside ICMP Echo Request payloads. The payload
of an Echo Request can be anything — and the receiver echoes it back
unchanged. Two endpoints with a tool like `icmptunnel` or `ptunnel` can
exchange data inside Echo Requests/Replies, evading firewalls that
only block TCP/UDP.

Defenses:

- Deep packet inspection on ICMP payloads.
- Limit ICMP payload size (legitimate pings are usually 56-byte payloads;
  drop pings with massive payloads).
- Anomaly detection: lots of ICMP between two hosts is unusual.

### ICMP Redirect Hijacking

A malicious host on your LAN sends you ICMP Redirects telling you that
the best route to the gateway is through them. Your kernel updates its
routing table. Now your traffic goes through the attacker.

Modern Linux defaults:

```
net.ipv4.conf.all.accept_redirects = 0
net.ipv4.conf.all.send_redirects = 0
net.ipv6.conf.all.accept_redirects = 0
```

Don't accept redirects from anyone; don't send them either (unless
you're a router, in which case do the opposite).

### What to Allow at the Border

Modern best practice for IPv4 firewall ICMP:

- **Allow** Type 3 (Destination Unreachable), all codes — especially Code 4.
- **Allow** Type 11 (Time Exceeded) — for traceroute to work.
- **Allow** Type 12 (Parameter Problem) — for diagnostics.
- **Allow** Type 8 (Echo Request) inbound from trusted sources only —
  it's nice to be ping-able for monitoring, but you don't have to be.
- **Allow** Type 0 (Echo Reply) inbound — because YOU may have pinged
  out and need the reply.
- **Rate-limit** all of the above.
- **Drop everything else** — type 4 (deprecated source quench), type
  5 (redirect at the border is sus), types 17/18 (address mask), etc.

For IPv6: read RFC 4890 carefully. Generally allow ICMPv6 broadly with
specific drops, not the other way around.

## Hands-On

Every command below is paste-and-runnable on a modern Linux box. Output
shown is representative — your numbers will differ. Where the command
needs root, the prompt is `#`; for unprivileged it's `$`.

### Basic Ping

```
$ ping -c 4 8.8.8.8
PING 8.8.8.8 (8.8.8.8) 56(84) bytes of data.
64 bytes from 8.8.8.8: icmp_seq=1 ttl=117 time=12.3 ms
64 bytes from 8.8.8.8: icmp_seq=2 ttl=117 time=11.9 ms
64 bytes from 8.8.8.8: icmp_seq=3 ttl=117 time=12.4 ms
64 bytes from 8.8.8.8: icmp_seq=4 ttl=117 time=12.1 ms

--- 8.8.8.8 ping statistics ---
4 packets transmitted, 4 received, 0% packet loss, time 3004ms
rtt min/avg/max/mdev = 11.872/12.183/12.412/0.205 ms
```

`-c 4` sends exactly 4 pings. `8.8.8.8` is Google Public DNS.

### Faster Ping (interval 0.2s)

```
$ ping -c 4 -i 0.2 8.8.8.8
PING 8.8.8.8 (8.8.8.8) 56(84) bytes of data.
64 bytes from 8.8.8.8: icmp_seq=1 ttl=117 time=11.8 ms
64 bytes from 8.8.8.8: icmp_seq=2 ttl=117 time=11.9 ms
64 bytes from 8.8.8.8: icmp_seq=3 ttl=117 time=12.0 ms
64 bytes from 8.8.8.8: icmp_seq=4 ttl=117 time=12.1 ms

--- 8.8.8.8 ping statistics ---
4 packets transmitted, 4 received, 0% packet loss, time 605ms
rtt min/avg/max/mdev = 11.812/11.952/12.121/0.108 ms
```

`-i 0.2` sets interval to 200ms. Sub-second intervals require root for
unprivileged users in many distros.

### Path MTU Probe (DF set, big packet)

```
$ ping -c 1 -s 1500 -M do 8.8.8.8
PING 8.8.8.8 (8.8.8.8) 1500(1528) bytes of data.
From 192.168.1.1 icmp_seq=1 Frag needed and DF set (mtu = 1492)
ping: local error: message too long, mtu=1492

--- 8.8.8.8 ping statistics ---
1 packets transmitted, 0 received, +1 errors, 100% packet loss
```

`-s 1500` sets payload to 1500 bytes (becomes 1528 on wire with IP+ICMP
headers). `-M do` sets DF bit. The router reports back the actual MTU
of the next hop (here 1492 — typical PPPoE link).

### Jumbo-Frame Probe

```
$ ping -c 1 -s 9000 -M do 8.8.8.8
PING 8.8.8.8 (8.8.8.8) 9000(9028) bytes of data.
ping: local error: message too long, mtu=1500

--- 8.8.8.8 ping statistics ---
1 packets transmitted, 0 received, +1 errors, 100% packet loss
```

Local MTU is 1500, so the kernel rejects this before it even leaves
the box. To test jumbo frames you need an end-to-end 9000-byte path,
which is rare outside data centers.

### IPv6 Ping

```
$ ping -c 4 2606:4700:4700::1111
PING 2606:4700:4700::1111(2606:4700:4700::1111) 56 data bytes
64 bytes from 2606:4700:4700::1111: icmp_seq=1 ttl=58 time=8.91 ms
64 bytes from 2606:4700:4700::1111: icmp_seq=2 ttl=58 time=9.10 ms
64 bytes from 2606:4700:4700::1111: icmp_seq=3 ttl=58 time=8.85 ms
64 bytes from 2606:4700:4700::1111: icmp_seq=4 ttl=58 time=9.02 ms

--- 2606:4700:4700::1111 ping statistics ---
4 packets transmitted, 4 received, 0% packet loss, time 3005ms
rtt min/avg/max/mdev = 8.851/8.973/9.105/0.115 ms
```

That's Cloudflare DNS over IPv6. On modern Linux, `ping` works for both
v4 and v6; older systems use `ping6`. Note the field is now `ttl` (it
should really say `hop limit` in IPv6 but the tool kept the v4 label).

### Default Traceroute (UDP probes)

```
$ traceroute 8.8.8.8
traceroute to 8.8.8.8 (8.8.8.8), 30 hops max, 60 byte packets
 1  _gateway (192.168.1.1)         1.234 ms  1.110 ms  1.008 ms
 2  10.0.0.1 (10.0.0.1)            5.012 ms  4.998 ms  5.105 ms
 3  72.14.215.85 (72.14.215.85)    9.234 ms  9.112 ms  9.005 ms
 4  108.170.225.193                11.111 ms 11.005 ms 10.998 ms
 5  216.239.50.123                 11.523 ms 11.612 ms 11.434 ms
 6  8.8.8.8                        12.123 ms 12.045 ms 12.000 ms
```

Linux default uses UDP packets at high ports. Each line is one hop.
Three probes per hop.

### ICMP-Probe Traceroute

```
$ traceroute -I 8.8.8.8
traceroute to 8.8.8.8 (8.8.8.8), 30 hops max, 60 byte packets
 1  _gateway (192.168.1.1)         1.110 ms  1.005 ms  0.998 ms
 2  10.0.0.1 (10.0.0.1)            5.005 ms  4.991 ms  5.087 ms
 3  72.14.215.85 (72.14.215.85)    9.220 ms  9.108 ms  8.999 ms
 4  108.170.225.193                11.115 ms 11.001 ms 10.985 ms
 5  216.239.50.123                 11.500 ms 11.580 ms 11.420 ms
 6  8.8.8.8                        12.050 ms 12.005 ms 11.985 ms
```

`-I` makes traceroute use ICMP Echo Request probes instead of UDP. Often
needed when UDP probes are filtered.

### TCP-Probe Traceroute (when ICMP is blocked)

```
$ traceroute -T -p 443 google.com
traceroute to google.com (142.250.69.78), 30 hops max, 60 byte packets
 1  _gateway (192.168.1.1)         1.234 ms  1.180 ms  1.105 ms
 2  10.0.0.1                       5.012 ms  4.998 ms  5.105 ms
 3  72.14.215.85                   9.234 ms  9.112 ms  9.005 ms
 4  108.170.225.193                11.111 ms 11.005 ms 10.998 ms
 5  216.239.50.123                 12.500 ms 12.612 ms 12.434 ms
 6  142.250.69.78                  13.123 ms 13.045 ms 13.000 ms
```

`-T -p 443` sends TCP SYN to port 443. Each hop still returns ICMP Time
Exceeded (because TTL expiration is layer 3), but the destination
responds with TCP, which can't be confused with anything else.

### MTR — Continuous Ping+Traceroute

```
$ mtr -rwbz 8.8.8.8
Start: 2026-04-27T12:30:00+0000
HOST: my-host                              Loss%   Snt   Last   Avg  Best  Wrst StDev
  1. AS???    192.168.1.1                   0.0%    10    1.2   1.3   1.0   1.5   0.1
  2. AS12345  10.0.0.1                      0.0%    10    5.0   5.1   4.9   5.5   0.2
  3. AS12345  72.14.215.85                  0.0%    10    9.2   9.2   9.0   9.5   0.2
  4. AS15169  108.170.225.193               0.0%    10   11.1  11.1  11.0  11.4   0.1
  5. AS15169  216.239.50.123                0.0%    10   11.5  11.6  11.4  12.0   0.2
  6. AS15169  8.8.8.8                       0.0%    10   12.1  12.1  12.0  12.4   0.1
```

`-r` report mode (don't run forever), `-w` wide names, `-b` show IPs
plus hostnames, `-z` show ASN. mtr is the gold standard for diagnosing
intermittent packet loss because it shows you which hop is dropping
packets in real time.

### Tracepath — No-Root MTU Discovery

```
$ tracepath 8.8.8.8
 1?: [LOCALHOST]                                 pmtu 1500
 1:  _gateway                                    1.234ms
 1:  _gateway                                    1.110ms
 2:  10.0.0.1                                    5.012ms
 3:  72.14.215.85                                9.234ms asymm 4
 4:  108.170.225.193                             11.111ms
 5:  216.239.50.123                              11.523ms
 6:  8.8.8.8                                     12.123ms reached
     Resume: pmtu 1500 hops 6 back 6
```

`tracepath` is like traceroute but it also probes the path MTU at each
hop. It doesn't need root. The final "Resume" line tells you the
end-to-end MTU. "asymm 4" means the return path has a different number
of hops than the forward path.

### IPv6 Neighbor Table

```
$ ip -6 neigh show
fe80::1 dev eth0 lladdr 00:11:22:33:44:55 REACHABLE
2001:db8::1 dev eth0 lladdr 00:11:22:33:44:55 router REACHABLE
fe80::a00:27ff:feab:cdef dev eth0 lladdr 0a:00:27:ab:cd:ef STALE
```

This is the IPv6 equivalent of the ARP table. It's populated by ICMPv6
Neighbor Discovery. STALE means the entry is old and might need
re-verifying.

### IPv4 ARP Table

```
$ ip neigh show
192.168.1.1 dev eth0 lladdr 00:11:22:33:44:55 REACHABLE
192.168.1.5 dev eth0 lladdr 0a:00:27:ab:cd:ef STALE
192.168.1.9 dev eth0  FAILED
```

Old `arp -a` still works on most systems too. FAILED means we tried to
reach this host and the ARP request never got a reply.

### Capture ICMP with tcpdump

```
# tcpdump -i any -n icmp
tcpdump: data link type LINUX_SLL2
tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on any, link-type LINUX_SLL2 (Linux cooked v2), snapshot length 262144 bytes
12:30:00.123456 eth0  Out IP 192.168.1.10 > 8.8.8.8: ICMP echo request, id 12345, seq 1, length 64
12:30:00.135678 eth0  In  IP 8.8.8.8 > 192.168.1.10: ICMP echo reply, id 12345, seq 1, length 64
12:30:01.123456 eth0  Out IP 192.168.1.10 > 8.8.8.8: ICMP echo request, id 12345, seq 2, length 64
12:30:01.135678 eth0  In  IP 8.8.8.8 > 192.168.1.10: ICMP echo reply, id 12345, seq 2, length 64
```

`-i any` listens on all interfaces. `-n` skips DNS resolution (fast).
`icmp` is the BPF filter — only ICMPv4 (use `icmp6` for v6). Needs root
or CAP_NET_RAW.

### Capture ICMPv6

```
# tcpdump -i any -n icmp6
12:30:00.123456 eth0  Out IP6 2001:db8::10 > 2606:4700:4700::1111: ICMP6, echo request, seq 1, length 64
12:30:00.131234 eth0  In  IP6 2606:4700:4700::1111 > 2001:db8::10: ICMP6, echo reply, seq 1, length 64
12:30:05.012345 eth0  Out IP6 fe80::a > ff02::1:ff00:1: ICMP6, neighbor solicitation, who has 2001:db8::1, length 32
12:30:05.013456 eth0  In  IP6 fe80::1 > fe80::a: ICMP6, neighbor advertisement, tgt is 2001:db8::1, length 32
```

You'll see Echo Request/Reply along with Neighbor Solicitation/Advertisement
chatter. Real networks are noisy.

### Sysctl: Ignore All Pings

```
$ cat /proc/sys/net/ipv4/icmp_echo_ignore_all
0
```

Default 0 = answer pings. Set to 1 to silently ignore all pings. Set
with `sysctl -w net.ipv4.icmp_echo_ignore_all=1`. Often used on servers
that don't want to be pinged for stealth reasons. Not actually a great
security move, since it gives you no benefit beyond mild obscurity.

### Sysctl: Ignore Broadcast Pings

```
$ cat /proc/sys/net/ipv4/icmp_echo_ignore_broadcasts
1
```

Default 1 = ignore broadcast pings. This is the Smurf attack defense.
Don't change it.

### Sysctl: Rate Limit Window

```
$ cat /proc/sys/net/ipv4/icmp_ratelimit
1000
```

Token bucket interval in milliseconds. Defines the minimum spacing
between rate-limited ICMP messages. Default is 1000 (= 1 message/second
for each rate-limited type).

### Sysctl: Rate Limit Mask

```
$ cat /proc/sys/net/ipv4/icmp_ratemask
6168
```

Bitmap of which ICMP types get rate-limited. Default 6168 (= 0x1818)
covers Destination Unreachable (3), Source Quench (4), Time Exceeded
(11), and Parameter Problem (12). Echo replies are NOT rate-limited by
default — you want pings to keep working.

### iptables: List Current ICMP Rules

```
# iptables -L INPUT -v -n | grep -i icmp
   42  4200 ACCEPT     icmp --  *      *       0.0.0.0/0    0.0.0.0/0    icmptype 3
    8   672 ACCEPT     icmp --  *      *       0.0.0.0/0    0.0.0.0/0    icmptype 8 limit: avg 2/sec burst 5
   15  1260 ACCEPT     icmp --  *      *       0.0.0.0/0    0.0.0.0/0    icmptype 11
```

Shows which ICMP types are allowed in. Above: Destination Unreachable
(3) unconditional, Echo Request (8) rate-limited, Time Exceeded (11)
unconditional. Note `-v -n` for verbose numeric output.

### nftables: List ICMP Rules

```
# nft list table inet filter | grep -iE 'icmp|udp'
        icmp type echo-request limit rate 2/second burst 5 packets accept
        icmp type { destination-unreachable, time-exceeded, parameter-problem } accept
        icmpv6 type { nd-router-advert, nd-neighbor-solicit, nd-neighbor-advert } accept
        icmpv6 type echo-request limit rate 2/second burst 5 packets accept
```

nftables (modern replacement for iptables) makes ICMP rules cleaner.
You can list types by name instead of number.

### hping3 — Custom ICMP Crafting

```
# hping3 -c 1 --icmp 8.8.8.8
HPING 8.8.8.8 (eth0 8.8.8.8): icmp mode set, 28 headers + 0 data bytes
len=46 ip=8.8.8.8 ttl=117 id=0 icmp_seq=0 rtt=12.3 ms

--- 8.8.8.8 hping statistic ---
1 packets transmitted, 1 packets received, 0% packet loss
round-trip min/avg/max = 12.3/12.3/12.3 ms
```

`hping3` is a low-level packet crafter. `--icmp` mode sends echo requests
but you can construct arbitrary ICMP packets with `--icmptype` and
`--icmpcode`. Needs root. Useful for testing firewall rules.

### nmap Host Discovery (Ping Scan)

```
$ nmap -sn 192.168.1.0/24
Starting Nmap 7.94 ( https://nmap.org )
Nmap scan report for _gateway (192.168.1.1)
Host is up (0.0012s latency).
Nmap scan report for laptop (192.168.1.10)
Host is up (0.0035s latency).
Nmap scan report for printer (192.168.1.50)
Host is up (0.012s latency).
Nmap done: 256 IP addresses (3 hosts up) in 2.45 seconds
```

`-sn` is "ping scan, no port scan." nmap pings all 256 addresses in the
/24 (using a mix of ICMP Echo Request, ICMP Timestamp Request, ARP for
local subnets, and TCP SYN to port 443) to find live hosts.

### Socket Summary

```
$ ss -s | head -10
Total: 1234
TCP:   456 (estab 123, closed 234, orphaned 1, timewait 33)

Transport Total     IP        IPv6
RAW       2         1         1
UDP       45        20        25
TCP       222       150       72
INET      269       171       98
FRAG      0         0         0
```

`ss -s` shows socket counts. ICMP doesn't show up here directly because
the kernel handles ICMP without sockets (except for raw sockets used
by ping).

### Interface ICMP Counters

```
# ip -s -s link show eth0 | grep -iE 'icmp|errors'
    RX errors: 0  dropped 0  overruns 0  frame 0
    TX errors: 0  dropped 0  carrier 0  collsns 0
```

The `link show` doesn't break out ICMP separately, but you can see
overall errors. For ICMP-specific counters, use `/proc/net/snmp`.

### /proc/net/snmp ICMP Stats

```
$ cat /proc/net/snmp | grep ^Icmp:
Icmp: InMsgs InErrors InCsumErrors InDestUnreachs InTimeExcds InParmProbs InSrcQuenchs InRedirects InEchos InEchoReps InTimestamps InTimestampReps InAddrMasks InAddrMaskReps OutMsgs OutErrors OutDestUnreachs OutTimeExcds OutParmProbs OutSrcQuenchs OutRedirects OutEchos OutEchoReps OutTimestamps OutTimestampReps OutAddrMasks OutAddrMaskReps
Icmp: 156 0 0 4 2 0 0 0 142 8 0 0 0 0 152 0 0 0 0 0 0 144 8 0 0 0 0
```

Two-line format: header line names the counters, data line has the
values. Above: 156 ICMP messages received total, 4 of which were
Destination Unreachable, 142 were Echo Requests, 8 were Echo Replies.
On the outbound side: 144 Echo Requests sent (this is `ping` activity),
8 Echo Replies sent.

### IPv6 ICMPv6 Stats

```
$ cat /proc/net/snmp6 | grep -E 'Icmp6'
Icmp6InMsgs                     	250
Icmp6InErrors                   	0
Icmp6OutMsgs                    	248
Icmp6OutErrors                  	0
Icmp6InEchos                    	2
Icmp6InEchoReplies              	35
Icmp6InNeighborSolicits         	100
Icmp6InNeighborAdvertisements   	50
Icmp6OutNeighborSolicits        	48
Icmp6OutNeighborAdvertisements  	100
Icmp6OutEchos                   	35
Icmp6OutEchoReplies             	2
```

Notice the volume of Neighbor Solicits/Advertisements on a busy IPv6
box. This is normal traffic — it's how the LAN works.

### conntrack — See ICMP "Connections"

```
# conntrack -L -p icmp
icmp     1 27 src=192.168.1.10 dst=8.8.8.8 type=8 code=0 id=12345 src=8.8.8.8 dst=192.168.1.10 type=0 code=0 id=12345 mark=0 use=1
```

Linux conntrack treats ICMP echo request/reply as a "connection" so
the firewall can match the reply to the original request. This is how
stateful firewall rules like `-m state --state ESTABLISHED,RELATED`
let echo replies in.

## Common Confusions

### "Should I block ICMP at my firewall?"

**No** — at least not all of it. You can block Echo Request (Type 8)
inbound if you want to be invisible to pings, but you must allow:

- Type 3 Code 4 (Fragmentation Needed) — or PMTUD breaks.
- Type 3 (other codes) — or you'll never know why connections fail.
- Type 11 (Time Exceeded) — or traceroute won't work, and TCP can lose
  the reverse path information it uses.
- Type 0 (Echo Reply) — for outbound pings to come back.

For IPv6: leave ICMPv6 mostly open per RFC 4890. Blocking ICMPv6 breaks
the network at layer 3.

### "Is ping guaranteed to work?"

**No.** Many networks rate-limit, drop, or otherwise filter ICMP Echo
Requests and Replies. Some servers have `icmp_echo_ignore_all=1`. Some
firewalls drop pings at the edge. So a "host doesn't respond to ping"
is **not** the same as "host is offline." Try TCP probes (e.g.
`nmap -Pn -p 443 host`) to confirm.

### "Why does my traceroute show stars?"

A hop where every probe gets dropped. The hop is up (because later hops
respond), but the router at that hop either rate-limits ICMP, blocks
ICMP, or just doesn't generate Time Exceeded. You can also see stars
when the reverse path is broken — the probe got there but the reply
couldn't get back. Stars are not a fatal sign.

### "Is ICMP a layer-3 or layer-4 protocol?"

**Layer 3.** ICMP runs over IP (protocol number 1 for ICMPv4, 58 for
ICMPv6) but it's part of the IP suite — it has no transport semantics.
The OSI model isn't a perfect fit here, but ICMP is closer to "the
control plane of IP" than to "an application protocol." Some textbooks
say "ICMP is layer 3.5" or "ICMP runs at layer 3 over IP." Both fine.

### "Why doesn't ICMP have ports?"

Because ICMP isn't for transport. Ports are how multiple programs share
an IP address. ICMP isn't a program — it's the network kernel itself
talking to other network kernels. There's nothing to multiplex. (The
ping identifier serves a similar function for matching replies to requests,
but it's not a port.)

### "Why is ARP not ICMP?"

ARP (RFC 826) is layer 2 (Ethernet). It uses an EtherType (0x0806),
not an IP protocol number. So ARP can't be ICMP because ARP doesn't
have an IP header to put ICMP in. In IPv6, the equivalent function is
moved into ICMPv6 (Neighbor Discovery), but in IPv4 ARP stays separate
and lower in the stack. Different design decisions, same job.

### "Does TCP use ICMP?"

**Indirectly, yes.** TCP doesn't generate ICMP itself, but TCP relies
on ICMP for Path MTU Discovery and for "fast fail" (when a TCP SYN
hits a closed port and gets back an ICMP Port Unreachable, the kernel
delivers that to the TCP stack, which immediately fails the connection
attempt instead of waiting for retransmissions to time out).

### "Can I ping a hostname or just an IP?"

`ping example.com` works because `ping` does DNS resolution first, then
pings the resulting IP. The actual ICMP is to the IP. If the DNS lookup
fails, you'll get an "unknown host" error before any ICMP is sent.

### "Why does my ping show different TTLs?"

Different OSes ship with different default TTLs:

- Linux/macOS: 64
- Windows: 128
- Some routers/Cisco: 255

If you ping `8.8.8.8` and see TTL=117 in the reply, you can guess Google
sent it with TTL=128 (or 255) and it traversed (128 − 117 = 11) or
(255 − 117 = 138) hops on the way back. The forward and reverse paths
might differ in hop count.

### "Should I see ICMP for every TCP connection?"

**No, not normally.** ICMP only shows up when something goes wrong, or
for PMTUD when MTU mismatches occur, or for diagnostics like ping. A
healthy TCP connection over a path that doesn't mismatch MTUs may
generate zero ICMP packets. ICMP traffic is a small fraction of normal
network traffic.

### "What's the difference between TTL and hop limit?"

Same concept, different name. IPv4 calls it TTL ("Time To Live") because
originally it was supposed to be measured in seconds. In practice it's
always been a hop counter, so IPv6 renamed it "Hop Limit" to be honest.
Both are 8-bit fields; both decrement at each router.

### "Why is sequence number sometimes huge in `ping` output?"

Sequence numbers are 16 bits and start at 1 (or some implementations
start at 0). Each ping bumps the counter. After about 65535 pings, it
wraps. If you start a long-running ping, eventually you'll see seq=65534
followed by seq=1.

### "What's the deal with Echo Reply having Type 0?"

Historical accident. Type 0 just happens to be assigned to Echo Reply.
Type 8 to Echo Request. Don't read meaning into the numbers; they were
just allocated in the order RFC 792 happened to list them.

### "Can ICMP travel across NAT?"

Yes, but it's tricky. NAT routers track ICMP echo identifier values
the same way they track TCP/UDP port numbers, so multiple machines
behind a NAT can ping the same destination simultaneously and the NAT
sorts the replies. Other ICMP types (Destination Unreachable, Time
Exceeded) are even more fun — the NAT has to look at the embedded copy
of the original packet inside the ICMP message and rewrite the inner
addresses too. Most modern NATs do this correctly. CGNAT (carrier-grade
NAT) gets harder because identifier collisions become more likely with
millions of customers behind one IP.

### "Why does `ping -f` (flood) need root?"

Flood ping sends as fast as possible (no delay). Without root, the
kernel rate-limits unprivileged ICMP sends to prevent abuse. With root,
you can spam ICMP at line rate. Don't use this on production networks.

## Vocabulary

- **ICMP** — Internet Control Message Protocol (RFC 792). The network's
  walkie-talkie. IP protocol number 1.
- **ICMPv6** — IPv6 version (RFC 4443). Obsoletes RFC 2463. IP protocol
  number 58. Includes Neighbor Discovery.
- **IP** — Internet Protocol. ICMP rides on top of IP.
- **layer 3** — Network layer in the OSI model. ICMP is layer 3.
- **header** — The fixed-size metadata at the front of every ICMP message:
  type, code, checksum.
- **type** — 8-bit field indicating the broad class of ICMP message
  (Echo, Destination Unreachable, etc.).
- **code** — 8-bit subtype within a type. E.g., Type 3 / Code 4 means
  "Destination Unreachable, Fragmentation Needed."
- **checksum** — 16-bit one's-complement sum used to detect transmission
  corruption. Computed over the whole ICMP message.
- **sequence** — 16-bit counter inside Echo messages; lets `ping` match
  replies to requests and detect drops.
- **identifier** — 16-bit field inside Echo messages; lets multiple
  `ping` processes coexist (often the process ID).
- **payload** — The data field inside an ICMP message. For Echo, it's
  arbitrary bytes that the receiver echoes back unchanged.
- **echo** — ICMP "are you there?" message (Type 8 v4, Type 128 v6).
  AKA Echo Request.
- **echo reply** — Response to an Echo Request (Type 0 v4, Type 129 v6).
- **ping** — User-space tool that sends Echo Requests and times the
  Echo Replies.
- **pong** — Slang for the Echo Reply.
- **RTT** — Round-Trip Time. The total time from sending an Echo Request
  to receiving the corresponding Echo Reply.
- **jitter** — Variation in RTT over time. High jitter is bad for
  latency-sensitive apps.
- **TTL** — Time To Live. 8-bit field in IPv4 header decremented at each
  router; when it hits 0 the packet is dropped.
- **hop limit** — IPv6 equivalent of TTL.
- **traceroute** — Tool that maps the network path by sending packets
  with increasing TTL and reading Time Exceeded replies.
- **tracepath** — Like traceroute but probes path MTU; doesn't need root.
- **MTR** — My Traceroute. Continuous combined ping+traceroute showing
  per-hop loss in real time.
- **fragment** — A piece of an IP packet that was split because it was
  too big for a link.
- **MTU** — Maximum Transmission Unit. The biggest packet a link can
  carry without splitting.
- **PMTU** — Path MTU. The smallest MTU along an end-to-end path.
- **PMTUD** — Path MTU Discovery (RFC 1191 v4, RFC 8201 v6). Mechanism
  for the sender to learn the PMTU using ICMP Type 3 Code 4 (or ICMPv6
  Type 2).
- **DF** — Don't Fragment bit in IPv4 header. When set, routers must
  drop oversized packets and send Frag Needed.
- **MF** — More Fragments bit in IPv4 header. Set on every fragment except
  the last, to tell the receiver more pieces are coming.
- **Path MTU Discovery** — see PMTUD.
- **ICMP black hole** — Pathological condition where ICMP messages
  needed for PMTUD are silently dropped, breaking large-packet
  transmission with no error message.
- **redirect** — ICMP Type 5 message: "use a different next-hop router."
  Modern hosts ignore by default.
- **source quench** — Deprecated ICMP Type 4 ("slow down"). RFC 6633
  obsoletes its generation.
- **parameter problem** — ICMP Type 12. Header field was malformed.
- **router solicitation** — ICMPv6 Type 133. "Are there any routers
  here?"
- **router advertisement** — ICMPv6 Type 134. "I'm a router, here's
  your default route, here are prefixes."
- **neighbor solicitation** — ICMPv6 Type 135. "Who has this IPv6
  address?" Replaces ARP request.
- **neighbor advertisement** — ICMPv6 Type 136. "I have this IPv6
  address, here's my MAC." Replaces ARP reply.
- **multicast listener report** — ICMPv6 Types 130–132 / MLD. Equivalent
  to IGMP for IPv4 multicast group management.
- **ARP** — Address Resolution Protocol (RFC 826). Layer-2 IPv4 mechanism
  to find a host's MAC from its IP.
- **NDP** — Neighbor Discovery Protocol (RFC 4861). ICMPv6-based
  replacement for ARP, plus router discovery and SLAAC support.
- **DAD** — Duplicate Address Detection. ICMPv6 mechanism to verify
  no other host claims a candidate IPv6 address before using it.
- **SLAAC** — StateLess Address AutoConfiguration (RFC 4862). Hosts
  build their own IPv6 address from a router-advertised prefix and an
  interface ID.
- **EUI-64** — Modified 64-bit interface identifier derived from a MAC
  address. Used by SLAAC unless privacy extensions override it.
- **link-local** — IPv6 address starting `fe80::/10`. Valid only on the
  local link, used heavily by NDP and routing protocols.
- **multicast** — One-to-many addressing. IPv4 224.0.0.0/4, IPv6 ff00::/8.
- **unicast** — One-to-one addressing. The normal kind.
- **anycast** — One address shared by many hosts; routing delivers to
  the closest. Used by 8.8.8.8, root DNS, etc.
- **time exceeded** — ICMP Type 11. TTL hit zero (Code 0) or fragment
  reassembly timed out (Code 1).
- **destination unreachable** — ICMP Type 3. Various codes for various
  reasons.
- **fragmentation needed** — ICMP Type 3 Code 4. PMTUD signal.
- **port unreachable** — ICMP Type 3 Code 3. No program listening on
  the destination's UDP/TCP port.
- **host unreachable** — ICMP Type 3 Code 1. ARP/ND failed for the
  destination on its LAN.
- **administratively prohibited** — ICMP Type 3 Code 13 (or Code 9/10).
  A firewall blocked it.
- **smurf attack** — Broadcast-amplification attack using spoofed echo
  requests. Mostly extinct.
- **ping of death** — Old crash bug from oversized fragmented ICMP
  packets. Mostly historical.
- **ping flood** — Brute-force bandwidth-saturation via fast pings.
- **fragmentation attack** — Various exploits using overlapping or
  malformed IP fragments.
- **hping3** — Low-level packet crafting tool. Sends arbitrary ICMP/TCP/UDP.
- **nmap** — Network mapping tool. Uses ICMP among other techniques for
  host discovery.
- **mtr** — My Traceroute. See above.
- **tracepath** — see above.
- **tcpdump** — Packet capture CLI. Use `tcpdump -n icmp` to see ICMP
  traffic in real time.
- **wireshark** — GUI packet analyzer. Same job as tcpdump with prettier
  output.
- **nft** — nftables CLI; modern Linux firewall front-end.
- **iptables** — Older Linux firewall front-end. Still widely used.
- **conntrack** — Linux connection tracking; treats ICMP echo as a
  pseudo-connection.
- **icmp_echo_ignore_all** — Linux sysctl to silently drop all incoming
  ICMP Echo Requests.
- **icmp_ratelimit** — Linux sysctl controlling minimum spacing between
  rate-limited ICMP messages.
- **RFC 792** — Original ICMP spec, 1981.
- **RFC 1191** — Path MTU Discovery for IPv4, 1990.
- **RFC 4443** — ICMPv6 spec; obsoletes RFC 2463.
- **RFC 4861** — Neighbor Discovery for IPv6.
- **RFC 4862** — IPv6 Stateless Address Autoconfiguration.
- **RFC 4890** — Recommendations for filtering ICMPv6 in firewalls.
- **RFC 6633** — Deprecation of ICMP Source Quench, 2012.
- **RFC 8201** — Path MTU Discovery for IPv6; obsoletes RFC 1981.
- **TTL expired in transit** — Time Exceeded Code 0 message text.
- **fragment reassembly time exceeded** — Time Exceeded Code 1 message.
- **proto** — IP header field naming the upper-layer protocol; 1=ICMP,
  6=TCP, 17=UDP, 58=ICMPv6.
- **EtherType** — Ethernet frame field naming the layer-3 protocol;
  0x0800=IPv4, 0x0806=ARP, 0x86DD=IPv6.
- **rate limit** — Cap on outgoing message frequency. Common ICMP
  defense.
- **DPI** — Deep Packet Inspection. Looking inside packet payloads,
  including ICMP payloads, to detect anomalies.
- **CGNAT** — Carrier-Grade NAT. Multiple layers of NAT used by ISPs
  to extend IPv4. Stresses ICMP identifier handling.

## Try This

Run these on your own box. Each one is safe and reversible.

### 1. Find your link's MTU

Start small and ramp up:

```
$ ping -c 1 -s 100 -M do 8.8.8.8
$ ping -c 1 -s 1400 -M do 8.8.8.8
$ ping -c 1 -s 1450 -M do 8.8.8.8
$ ping -c 1 -s 1472 -M do 8.8.8.8
$ ping -c 1 -s 1473 -M do 8.8.8.8
```

Where it stops succeeding is your local link's MTU minus 28 bytes (IP
+ ICMP headers). On standard Ethernet that's 1472 bytes payload, 1500
total.

### 2. Watch ICMP in real time as you traceroute

In one terminal:

```
# tcpdump -i any -n icmp
```

In another:

```
$ traceroute -I 8.8.8.8
```

You'll see Echo Requests going out at increasing TTLs and Time Exceeded
replies coming back from each hop.

### 3. Confirm port unreachable

In one terminal:

```
# tcpdump -i any -n icmp
```

In another:

```
$ nc -u 8.8.8.8 9999
hello
^C
```

(Hit Ctrl-C after a couple of seconds.) You'll see the UDP packet go
out and an ICMP Port Unreachable come back.

### 4. Trigger a Frag Needed message

```
$ ping -c 1 -s 9000 -M do 1.1.1.1
```

If you have any path with a smaller MTU than 9000, the kernel either
intercepts (showing "message too long") or a router on the way returns
Frag Needed.

### 5. Look at the IPv6 RAs on your network

```
# tcpdump -i any -n icmp6 and 'ip6[40] = 134'
```

Wait. If you have IPv6 on your LAN, you'll see periodic Router
Advertisements (Type 134). Inside is your default gateway, prefixes,
and DNS servers.

### 6. Disable redirects and verify

```
# sysctl -w net.ipv4.conf.all.accept_redirects=0
# sysctl net.ipv4.conf.all.accept_redirects
net.ipv4.conf.all.accept_redirects = 0
```

(Likely already 0 on modern Linux.)

### 7. Compare UDP traceroute and ICMP traceroute

```
$ traceroute 8.8.8.8       # UDP probes
$ sudo traceroute -I 8.8.8.8   # ICMP probes
$ sudo traceroute -T -p 443 8.8.8.8   # TCP probes
```

Note any differences. Some routers respond to one probe type but not
others.

### 8. Inspect /proc/net/snmp before and after

```
$ cat /proc/net/snmp | grep ^Icmp:
$ ping -c 100 8.8.8.8 > /dev/null
$ cat /proc/net/snmp | grep ^Icmp:
```

You should see InEchoReps and OutEchos increase by 100 each.

### 9. Test if your firewall drops ICMP

From an outside source, ping you. Or use a public service:

```
$ curl -s 'https://api.ipify.org'   # learn your public IP
```

Then have a friend ping it (or use online tools). If they can't ping
you, your ISP/firewall is dropping echo requests inbound.

### 10. Send a malformed ICMP and watch it fail

(Don't do this on a production network!) Use hping3:

```
# hping3 --icmp --icmptype 99 -c 1 8.8.8.8
```

Type 99 is unassigned. Most receivers will silently drop. Watch with
tcpdump on your end to see it leave; the absence of any reply tells
you the receiver discarded it.

## Where to Go Next

- `cs networking icmp` — dense reference of types, codes, packet formats
- `cs detail networking/icmp` — formal layer-3 message-format math,
  checksum proofs, IANA tables
- `cs networking ipv6` — where ICMPv6 lives and breathes
- `cs networking arp` — what ICMPv6 ND replaces; layer 2 background
- `cs networking tcpdump` — capture techniques for ICMP and beyond
- `cs networking mtr` — continuous traceroute / loss localization
- `cs ramp-up bgp-eli5` — how the routing happens that ICMP helps
- `cs ramp-up linux-kernel-eli5` — the kernel that's actually sending
  these messages
- `cs network-tools nmap` — host discovery via mixed ICMP/TCP probes
- `cs security firewall-design` — what to allow and what to drop

## See Also

- `networking/icmp`
- `networking/ipv6`
- `networking/arp`
- `networking/tcp`
- `networking/udp`
- `networking/dns`
- `networking/tcpdump`
- `networking/mtr`
- `networking/nmap`
- `security/firewall-design`
- `ramp-up/bgp-eli5`
- `ramp-up/linux-kernel-eli5`

## References

- RFC 792 — Internet Control Message Protocol (1981)
- RFC 1191 — Path MTU Discovery
- RFC 4443 — ICMPv6 (the IPv6 ICMP), obsoletes RFC 2463
- RFC 4861 — Neighbor Discovery for IPv6
- RFC 4862 — IPv6 Stateless Address Autoconfiguration
- RFC 4890 — Recommendations for Filtering ICMPv6 Messages in Firewalls
- RFC 6633 — Deprecation of ICMP Source Quench Messages
- RFC 8201 — Path MTU Discovery for IP version 6, obsoletes RFC 1981
- RFC 826 — An Ethernet Address Resolution Protocol (the ARP RFC,
  context for ICMPv6 ND)
- man ping, man ping6, man traceroute, man mtr, man tracepath
- man tcpdump, man nft, man iptables, man hping3, man nmap
- man 7 icmp (Linux ICMP socket interface)
- iana.org/assignments/icmp-parameters
- iana.org/assignments/icmpv6-parameters
