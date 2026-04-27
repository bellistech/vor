# Wireshark — ELI5

> Wireshark is an X-ray machine for the wires. Every little envelope that flies past your computer's network door gets caught, opened, photographed, labelled, and shown to you, byte by byte, with every part of it explained in plain words.

## Prerequisites

It helps if you have read **ramp-up/tcp-eli5**, **ramp-up/dns-eli5**, **ramp-up/icmp-eli5**, and **ramp-up/tls-eli5** first. You do not need to be an expert in any of those — even a fuzzy memory of "TCP is the reliable conversation, UDP is the shouted note, DNS is the phone book, TLS is the locked envelope" is enough. We will refer back to those ideas a lot, and we will explain again whenever a new word shows up. If you have never read any of those, that is fine too — keep this sheet open in one window and the **Vocabulary** section open in another, and look up each weird word as it appears.

You also need to know what a **terminal** is (a black window where you type commands) and what a **command** is (a line of text you press Enter on to make the computer do a thing). If you know how to type `ls` and see a list of files come back, you are ready.

If you see a `$` at the start of a line in a code block, that means "type the rest of this line into your terminal." You do not type the `$`. The lines underneath that don't have a `$` are what your computer prints back at you. We call that **output.**

## What Even Is Wireshark

### The picture: an X-ray machine for envelopes

Picture a giant mail sorting facility. Every second, millions of envelopes fly past on conveyor belts. Each envelope has an address on the front, an address on the back, a little stamp showing what country it came from, a customs declaration on the side, and inside the envelope is a folded letter. Some of the letters are sealed in another smaller envelope inside (locked with a wax seal). Some of the envelopes have other envelopes nested inside them, like Russian dolls.

You are an investigator. Your job is to figure out why a particular package was lost, or why two packages were trying to deliver to the same address at the same time, or why an envelope kept being sent back, or why somebody opened an envelope they weren't supposed to.

To do your job you have an X-ray machine. You point the X-ray at the conveyor belt, and the machine instantly photographs every envelope, opens it (without breaking the seal), reads the addresses, reads the customs forms, reads the contents, looks at any envelopes nested inside, peels those open too, and prints out a perfect labelled diagram of every layer.

**That X-ray machine is Wireshark.**

The conveyor belt is your **network card.** The envelopes are **packets.** The addresses on the front and back are **source and destination IP addresses.** The customs declaration is the **TCP or UDP header.** The folded letter inside is the **payload** — the actual thing the program was trying to send (a webpage, a search query, a pixel of a video, a piece of a song). The sealed envelope inside the envelope is **TLS** — the encrypted layer that hides what is being said. The Russian-doll nesting is **encapsulation** — every protocol on the network sits inside another protocol, like layers of an onion.

Wireshark sees through all of it. Every packet. Every header. Every flag. Every byte. And it shows you each one in three panes: a list of packets at the top, a tree of every layer in the middle, and the raw bytes at the bottom.

### Why does anyone need this?

Networks are invisible. When you click a link in your browser and a webpage shows up, you have no idea what just happened. Some envelopes flew through the air, some envelopes flew through wires, some of them got lost, some of them came back twice, some of them were resent, some of them were encrypted, some of them got compressed, some of them went to a server in another country and back. All of this is invisible to you. You just see the page show up.

Most of the time, that's fine. The page shows up, you read it, you move on with your life. But sometimes the page **does not** show up. Or it shows up slowly. Or it shows up wrong. Or your video call freezes. Or your remote login times out. Or your phone says "no internet" but the Wi-Fi bars are full.

When that happens, you need an X-ray machine. You need to see what envelopes actually flew, what addresses they had, what got lost, what got resent, what came back with a "DENIED" stamp on it. **Wireshark is that machine.**

There is a second reason: **learning.** If you want to actually understand TCP, or DNS, or HTTP, or TLS, the very best way is not to read about it — it is to capture real traffic and look at the actual envelopes flying around your computer right now. Every textbook in the world is less useful than ten minutes of staring at real packets. Wireshark is the textbook that writes itself based on what your computer is actually doing.

### Wireshark, tshark, tcpdump — three siblings

There are three programs people will mention in the same breath:

- **Wireshark** — the big graphical X-ray machine with menus, columns, mouse clicks, color-coded packets, charts, and pop-up windows.
- **tshark** — the same X-ray machine but in your terminal. No mouse, no pictures of packets, just text. Same dissectors, same filters. If you live on a server with no graphical desktop, this is the one you use.
- **tcpdump** — an older, smaller cousin. It can capture packets and print one-line summaries, but it cannot dissect HTTP/2 or follow streams or read pcapng files as nicely. It is everywhere though — every Linux box ships it, and it is the easiest first capture tool to reach for. See `networking/tcpdump`.

For this sheet we will use **Wireshark** for clicky things and **tshark** for terminal things. They share the same brain — a library called **libwireshark** that contains over 3,000 protocol dissectors. A "dissector" is a tiny program that knows how to open one type of envelope. There is a TCP dissector, an HTTP dissector, a DNS dissector, a Bluetooth dissector, a Modbus dissector, a USB dissector. If a protocol exists, somebody has probably written a Wireshark dissector for it.

### What Wireshark cannot do

Wireshark is not a firewall. It does not block packets. It does not change packets. It only **reads** them. Think of it as a scientist with binoculars, not a referee with a whistle. If you want to block traffic, you need iptables, nftables, or a firewall. If you want to redirect traffic, you need iproute2 or a load balancer. Wireshark just watches.

Wireshark is also not magic. It cannot decrypt traffic by itself. If somebody is sending TLS-encrypted data, Wireshark will see the encrypted bytes but cannot tell you what they say — unless you give it the **keys.** We'll cover that in the **Decryption** section.

Wireshark also cannot capture traffic that never reaches your network card. If two computers on a switch are talking to each other and your computer is plugged into a different port on that switch, the switch will not send those packets to you. To see them you would need to be on a span port, or a tap, or you would need to be one of the two endpoints. The X-ray machine only sees what passes by its window.

### The path of a packet from wire to your screen

Here is a picture worth memorizing. Every packet that you eventually see in Wireshark went through this path:

```
   [physical wire / radio waves]
              |
              v
       +-------------+
       |    NIC      |   <-- the network card hardware
       +-------------+
              |
              v
   kernel ring buffer (PACKET_MMAP / TPACKET_V3)
              |
              v
       +-------------+
       |    BPF      |   <-- capture filter runs here in-kernel
       +-------------+
              |
              v
        kept packets
              |
              v
       +-------------+
       |   libpcap   |   <-- user-space capture library
       +-------------+
              |
              v
       +-------------+
       |   dumpcap   |   <-- writes to pcap file
       +-------------+
              |
              v
        pcapng file
              |
              v
       +-------------+
       |   tshark/   |   <-- reads file, dissects every layer
       |  Wireshark  |
       +-------------+
              |
              v
        display filter
              |
              v
        your eyeballs
```

Every step matters. The BPF filter at the top can drop 99% of traffic before it ever touches disk. The dissector at the bottom can pull apart any of 3,000+ protocols. The display filter at the end hides what you're not looking at right now.

Two things are interesting:

1. **Wireshark itself is just a viewer.** All the privilege-needing stuff happens up at dumpcap. Once the pcap exists, anyone can dissect it.
2. **The capture filter has to be cheap because it runs on every packet.** The display filter can be expensive because it only runs on the packets you've already saved.

This split is what makes Wireshark fast. If you tried to dissect every HTTP request in real time during a 10 Gbps capture, you'd drop packets. Instead, capture wide with simple BPF, save to disk, and let dissection happen at your leisure.

## Capture vs Display Filters

Wireshark has **two filter languages** and they look different. This trips up everyone. Once you understand why, it stops being confusing.

### Two languages, two jobs

```
                  +--------------------+
   wire ---->     |   capture filter   |   ---> only matching packets reach disk
                  |   (BPF, kernel)    |
                  +--------------------+
                            |
                            v
                  +--------------------+
                  |   pcap file        |
                  +--------------------+
                            |
                            v
                  +--------------------+
                  |   display filter   |   ---> only matching packets visible
                  |  (Wireshark dialect)|
                  +--------------------+
                            |
                            v
                       your eyeballs
```

A **capture filter** runs in the kernel, before any packet is even saved. Its job is to throw packets away as fast as possible so we don't waste disk space recording 50 GB of traffic when we only care about 10 MB of it. The kernel needs the filter to be ridiculously fast, so it has to be written in a tiny language called **BPF** (Berkeley Packet Filter). BPF can only do simple things: match a port, match an IP, match a protocol, combine those with `and`/`or`/`not`. It cannot read inside HTTP. It cannot match a hostname. It cannot follow a stream.

A **display filter** runs after the packets have been captured. Wireshark has already opened every layer, named every field, and built a giant tree. The display filter just hides packets that don't match. It can be incredibly precise: "show me HTTP requests with method GET going to host example.com that returned a 404 status code more than 2 seconds after the previous packet." The display filter language is its own thing, invented by Wireshark, and it is much richer than BPF.

Mixing them up is the #1 mistake new users make. If you type `tcp.port == 443` into the **capture filter** field, Wireshark will reject it with `Capture filter syntax error`. If you type `port 443` into the **display filter** field, Wireshark will reject it with `Display filter syntax error`. They look similar but they are not the same dialect.

### The cheat sheet for both

| Want                            | Capture filter (BPF)                      | Display filter (Wireshark)             |
|---------------------------------|-------------------------------------------|----------------------------------------|
| Port 443                        | `port 443`                                | `tcp.port == 443`                      |
| Host 10.0.0.1                   | `host 10.0.0.1`                           | `ip.addr == 10.0.0.1`                  |
| Source host                     | `src host 10.0.0.1`                       | `ip.src == 10.0.0.1`                   |
| Dest host                       | `dst host 10.0.0.1`                       | `ip.dst == 10.0.0.1`                   |
| TCP only                        | `tcp`                                     | `tcp`                                  |
| UDP only                        | `udp`                                     | `udp`                                  |
| Subnet                          | `net 10.0.0.0/24`                         | `ip.addr == 10.0.0.0/24`               |
| Not broadcast                   | `not broadcast and not multicast`         | `not eth.dst == ff:ff:ff:ff:ff:ff`     |
| HTTP GET                        | (impossible — too deep)                   | `http.request.method == "GET"`         |
| DNS query for example.com       | (impossible)                              | `dns.qry.name contains "example.com"`  |
| TLS Client Hello                | (impossible)                              | `tls.handshake.type == 1`              |
| Combined (and)                  | `host 10.0.0.1 and port 443`              | `ip.addr == 10.0.0.1 and tcp.port == 443` |
| Combined (or)                   | `port 443 or port 80`                     | `tcp.port == 443 or tcp.port == 80`    |

The rule of thumb: if you know **before** you start capturing exactly which subset you want, write a capture filter. If you want everything and then drill in afterwards, capture wide and use display filters. Both are useful. Both are normal.

### Why BPF is so primitive

BPF runs in the kernel. The kernel processes millions of packets per second. Every cycle counts. So BPF is intentionally tiny: a few dozen instructions, no loops you can't bound, no string matching, no hash tables. The compiler will turn `host 10.0.0.1 and port 443` into about ten kernel-level checks: load this byte, compare to that byte, branch if equal, load the next byte, and so on.

This means BPF is **fast**, but also that BPF is **stupid**. It cannot say "show me packets where the user-agent header contains Firefox" because that would require scanning a string of arbitrary length, and that's not a thing BPF can do safely.

Modern Linux has a much bigger version called **eBPF** (extended BPF) that can do more, but Wireshark and tcpdump still use the older, simpler "classic" BPF for capture filters because it is supported everywhere. See `ramp-up/ebpf-eli5` for what eBPF can do that classic BPF cannot.

## Capture Mode

### Live interfaces

A **network interface** is a place packets enter and leave. On a normal laptop you might have:

- `en0` (or `wlan0` on Linux) — your Wi-Fi card.
- `eth0` (or `enp3s0`) — your wired Ethernet card.
- `lo` (or `lo0`) — the loopback interface, where packets you send to your own machine go.
- `any` (Linux only) — a fake interface that captures from all real interfaces at once.

To see all of them:

```bash
$ tshark -D
1. en0
2. lo0 (Loopback)
3. utun0
4. utun1
5. p2p0
6. awdl0
```

The numbers can be passed instead of names: `tshark -i 1`. Names also work: `tshark -i en0`. On Linux, `tshark -i any` is a special shortcut that captures from every interface simultaneously, which is great for "I don't know which interface this traffic is on, just give me everything."

To capture forever and write to a file:

```bash
$ sudo tshark -i en0 -w /tmp/cap.pcapng
```

(Press Ctrl-C to stop.) The output file is **pcapng** — the modern packet capture format. The older format is **pcap**, which still works but doesn't support nanosecond timestamps, multiple interfaces in one file, or per-packet comments. Wireshark has defaulted to pcapng since version 1.10.

### File rotation: the ring buffer

Networks can produce **a lot** of packets. A busy server might do 100 MB of traffic per second. If you start `tshark -i any -w /tmp/cap.pcapng` and walk away, you will fill up your disk in a few minutes.

The trick is **file rotation.** Tell Wireshark to start a new file every so often, and only keep the most recent N files. This is called a **ring buffer.**

```bash
# rotate every 100 MB, keep last 10 files (1 GB total max)
$ sudo tshark -i any \
    -w /tmp/cap.pcapng \
    -b filesize:100000 \
    -b files:10
```

The `-b filesize:100000` means "rotate when this file hits 100,000 KB" (which is 100 MB). The `-b files:10` means "keep the last 10 files." When the 11th file would be written, the 1st gets deleted. This way you can run a capture forever and never fill your disk.

You can also rotate by time:

```bash
# rotate every 60 seconds, keep last 60 files (1 hour of capture)
$ sudo tshark -i any -w /tmp/cap.pcapng -b duration:60 -b files:60
```

Or by packet count:

```bash
# rotate every 1,000,000 packets, keep last 5 files
$ sudo tshark -i any -w /tmp/cap.pcapng -b packets:1000000 -b files:5
```

This is incredibly useful for long-running troubleshooting. Start the capture in the morning, do your normal day's work, and if a problem shows up at 2pm, the last hour of traffic is sitting in those files.

### Snap length

By default Wireshark captures the entire packet. If you only care about the headers (and not the payload), you can tell it to truncate every packet to the first N bytes. This is called the **snap length** or **snaplen.** Smaller snaplen = much smaller capture files.

```bash
$ sudo tshark -i en0 -s 96 -w /tmp/headers-only.pcapng
```

`-s 96` means "only capture the first 96 bytes of each packet." That's enough for Ethernet (14 bytes) + IPv4 (20 bytes) + TCP (20 to 60 bytes) plus a little extra, so you'll see all the metadata but none of the actual payload. This is great for "I want to know who is talking to whom and how often, but I don't care what they're saying."

To snap the full packet, use `-s 0` (Linux/macOS) or just leave the flag off.

### Promiscuous mode

A network card normally only hands the kernel packets that are addressed **to that card.** Everything else gets dropped at the hardware level. This makes sense — your laptop doesn't need to look at packets meant for the printer in the next room.

But if you are doing packet captures on a network and want to see **everything that flies past your wire,** you want **promiscuous mode.** This tells the card "hey, give us every packet you see, even if it's not for us." Wireshark turns this on by default when you start a capture. You can turn it off with `--no-promiscuous-mode`.

On Wi-Fi, promiscuous mode is more complicated, because Wi-Fi cards can be in three different modes: regular (only sees your own traffic), promiscuous (sees other traffic on the same network you're joined to), or **monitor** mode (sees raw 802.11 frames including traffic for other networks). Monitor mode is rare and requires special drivers; most Wi-Fi cards on most laptops will refuse. We'll cover this more under Common Confusions.

## Filter Examples

Display filters are where Wireshark really shines. Here is a catalog of the most useful ones, by topic.

### IP and basic networking

```
ip.addr == 10.0.0.1                # source OR dest is 10.0.0.1
ip.src == 10.0.0.1                 # source only
ip.dst == 10.0.0.1                 # dest only
ip.addr == 10.0.0.0/24             # subnet match
ip.proto == 6                      # TCP (6), UDP (17), ICMP (1)
ipv6.addr == 2001:db8::1
ip.ttl < 5                         # packets close to expiring
not ip                             # non-IP traffic only (ARP, STP, LLDP)
arp                                # all ARP packets
icmp                               # all ICMPv4
icmpv6                             # all ICMPv6
icmp.type == 8                     # ICMP echo request (ping)
```

### TCP

```
tcp                                # any TCP
tcp.port == 443                    # source OR dest port 443
tcp.srcport == 22                  # source port only
tcp.dstport == 80
tcp.flags.syn == 1 and tcp.flags.ack == 0   # initial SYN (handshake start)
tcp.flags.reset == 1               # connection resets
tcp.analysis.retransmission        # retransmits (Wireshark-detected)
tcp.analysis.duplicate_ack         # duplicate ACKs
tcp.analysis.zero_window           # receiver advertised window 0
tcp.window_size < 1000             # small windows (often a problem)
tcp.len > 0                        # data-bearing packets only (no pure ACKs)
tcp.stream eq 5                    # all packets in stream #5
```

### UDP

```
udp
udp.port == 53                     # all DNS
udp.length > 1400                  # big UDP packets (often fragmented)
```

### DNS

```
dns                                # all DNS
dns.flags.response == 0            # queries only
dns.flags.response == 1            # responses only
dns.qry.name contains "example"    # questions about anything matching example
dns.qry.name == "www.example.com"  # exact match
dns.qry.type == 1                  # A queries
dns.qry.type == 28                 # AAAA queries
dns.flags.rcode != 0               # error responses (NXDOMAIN, SERVFAIL, etc.)
dns.flags.rcode == 3               # NXDOMAIN specifically
```

### HTTP

```
http                               # all HTTP/1.x
http.request                       # only requests
http.response                      # only responses
http.request.method == "GET"
http.request.method == "POST"
http.host == "example.com"
http.user_agent contains "Firefox"
http.response.code == 404
http.response.code >= 500          # any 5xx
http.content_type contains "json"
```

### TLS

```
tls                                # all TLS records
tls.handshake.type == 1            # ClientHello (start of handshake)
tls.handshake.type == 2            # ServerHello
tls.handshake.type == 11           # Certificate
tls.handshake.type == 16           # ClientKeyExchange
tls.handshake.extensions_server_name == "example.com"   # SNI match
tls.alert_message                  # any TLS alert (problem signal)
tls.handshake.version == 0x0303    # TLS 1.2 specifically
```

### QUIC and HTTP/3

```
quic                               # all QUIC
quic.long.packet_type == 0         # initial packet
quic.short.dcid                    # filter by destination connection ID
http3
```

### Layer 2

```
eth.addr == aa:bb:cc:dd:ee:ff      # MAC source or dest
eth.type == 0x0806                 # ARP
vlan.id == 100                     # specific VLAN
stp                                # spanning tree
lldp
cdp
```

### Combining filters

Use `and`, `or`, `not`, parentheses:

```
ip.addr == 10.0.0.1 and tcp.port == 443
(http.request.method == "GET" or http.request.method == "POST") and http.host contains "api"
not (ip.addr == 192.168.1.1) and tcp
```

### Operators

```
==     equal
!=     not equal
>      greater than
<      less than
>=     greater or equal
<=     less or equal
contains   substring (strings only)
matches    regex
&&     same as "and"
||     same as "or"
!      same as "not"
```

### Filter recipes for real situations

You're rarely going to type a filter from scratch. Here are recipes for things people actually need.

**"Show me only the slow stuff."**

```
tcp.analysis.ack_rtt > 0.5
```

Packets where the round-trip time was more than half a second. If your captures have many of these, the network or remote end is slow.

**"Show me everything that went wrong."**

```
tcp.analysis.flags or icmp.type == 3 or dns.flags.rcode != 0 or http.response.code >= 400
```

A megafilter for "anything that smells like an error." TCP analysis flags (retransmits, zero windows, etc.), ICMP unreachable, DNS error responses, HTTP 4xx/5xx. Always a good starting view.

**"Show me only data, hide the bookkeeping."**

```
tcp.len > 0
```

Pure ACKs have no payload (length 0). Filtering those out leaves only the actual data segments, which is much easier to read.

**"Find the moment a connection started."**

```
tcp.flags.syn == 1 and tcp.flags.ack == 0
```

The very first packet of any TCP connection is a SYN with no ACK. Every TCP conversation in your capture starts with one of these.

**"Find the moment a connection died ungracefully."**

```
tcp.flags.reset == 1
```

A TCP RST means somebody slammed the door. Either the receiver had no socket open, or some middlebox decided to kill the connection.

**"Show me only the queries my browser sent, not what came back."**

```
dns.flags.response == 0
```

**"Only the answers."**

```
dns.flags.response == 1
```

**"Find malformed packets."**

```
_ws.malformed
```

Wireshark's special meta-field for "the dissector got confused." If you have a lot of these, either you snapped too short or your traffic isn't what Wireshark thinks it is.

**"Only my traffic, not other people's."**

```
ip.src == 10.0.0.5 or ip.dst == 10.0.0.5
```

Replace 10.0.0.5 with your IP. (Hint: `ip addr` shows it.) Useful on a busy multi-host capture.

**"Hide the noise."**

```
not (mdns or ssdp or dhcp or icmpv6.type == 135 or icmpv6.type == 136)
```

Removes mDNS chatter, SSDP service discovery, DHCP lease renewal, and ICMPv6 neighbor solicitations/advertisements. Suddenly your capture is 90% smaller and easier to read.

## Following Streams

When you click a TCP packet and pick **Follow → TCP Stream** (or hit Ctrl-Alt-Shift-T), Wireshark gathers every packet in that conversation, reorders them by sequence number, removes duplicates, splices the bytes back together, and shows you the stream as if it were one continuous thing. Like watching a Netflix movie instead of seeing each frame as a separate JPEG.

```
                packets seen on wire:
                +---+ +---+ +---+ +---+ +---+
                | 4 | | 1 | | 3 | | 2 | | 5 |   (out of order!)
                +---+ +---+ +---+ +---+ +---+

                follow stream reassembles:
                +-------------------------------+
                | 1 -> 2 -> 3 -> 4 -> 5         |
                +-------------------------------+
                              |
                              v
                       readable text
```

There are several flavours of follow:

- **Follow TCP Stream** — concatenates raw bytes in both directions, color-coded (one direction red, the other blue).
- **Follow UDP Stream** — same idea but for UDP datagrams.
- **Follow HTTP Stream** — like TCP but understands chunked encoding and gzip, so you see the decoded text body, not the compressed bytes.
- **Follow HTTP/2 Stream** — picks one HTTP/2 stream out of a multiplexed connection. Multiple streams share the same TCP connection, and Follow HTTP/2 separates them.
- **Follow TLS Stream** — only useful if Wireshark has decryption keys (otherwise you just see encrypted gibberish). With keys, this shows the decrypted plaintext.
- **Follow QUIC Stream** — for HTTP/3 sessions.

In tshark:

```bash
$ tshark -r capture.pcapng -q -z follow,tcp,ascii,5
```

`-z follow,tcp,ascii,5` means "follow TCP stream number 5 and show it as ASCII text." Stream numbers come from `-z conv,tcp` (see Statistics).

### Reassembly under the hood

Reassembly is one of those features you take for granted until something goes wrong with it. The picture:

```
Wire (4 segments of an HTTP response):

  +---------------+   +---------------+   +---------------+   +---------------+
  | seq=1, len=200|   | seq=201,len=200|  | seq=401,len=200|  | seq=601,len=23 |
  +---------------+   +---------------+   +---------------+   +---------------+
       packet 5            packet 7            packet 9            packet 11

Wireshark glues them together:

  +-----------------------------------------------------+
  | full HTTP response (623 bytes)                      |
  | dissected by HTTP dissector, then JSON dissector    |
  +-----------------------------------------------------+
                       (shown on packet 11)
```

The dissector can only run after enough segments have arrived to form a complete message. Wireshark waits, gathers, and then runs the application-layer dissector on the assembled bytes. That's why you sometimes see "Reassembled TCP segments (623 bytes): #5(200), #7(200), #9(200), #11(23)" in the dissector tree.

If a segment is missing (truncated capture, packet loss, snaplen too short), reassembly fails and you get "TCP segment of a reassembled PDU" or "Malformed packet."

## Decryption

### TLS via SSLKEYLOGFILE

When two computers talk over TLS, they negotiate a shared secret in the first few packets. After that, everything is encrypted with keys derived from that secret. Wireshark captures the encrypted bytes but cannot derive the key — that requires the private key of the server, **or** the per-session keys themselves.

The clever trick: most browsers and many programs (curl, Firefox, Chrome, Node.js with NODE_OPTIONS, OpenSSL since 1.1.1) will **write the per-session keys to a file** if you set the environment variable `SSLKEYLOGFILE`. Wireshark can then read that file and decrypt the captured traffic on the fly.

```bash
$ export SSLKEYLOGFILE=/tmp/sslkeys.log
$ firefox     # or chrome, or curl https://example.com
```

Now `/tmp/sslkeys.log` will contain lines like:

```
CLIENT_RANDOM 0123abcd...  4567ef89...
SERVER_HANDSHAKE_TRAFFIC_SECRET 0123abcd...  9876fedc...
EXPORTER_SECRET 0123abcd...  1111aaaa...
```

This is the **NSS keylog format** — a text file where each line is one secret. Wireshark loads it via **Preferences → Protocols → TLS → (Pre)-Master-Secret log filename.**

```
                      browser
                         |
                  TLS handshake
                         |
                         v
                  derives keys
                  /          \
                 v            v
             encrypts      writes to
             traffic     SSLKEYLOGFILE
                 |            |
                 v            |
             over wire        |
                 |            |
                 v            v
            +--- Wireshark ---+
            | reads both     |
            | combines them  |
            | shows plaintext|
            +-----------------+
```

After loading the file, encrypted streams turn into readable text right inside Wireshark. **Follow TLS Stream** now works.

This has been the standard approach since OpenSSL 1.1.1 (released 2018) and curl 7.61, and Firefox/Chrome have supported it since 2015.

### Why some TLS sessions still don't decrypt

Even with the key file:

- **Wrong key file** — the keys must be from the same session you captured. Capturing yesterday and using today's keys won't work.
- **Session resumption** — if the client and server use a session ticket from a previous session, the new session's keys won't be in the file.
- **TLS 1.3 0-RTT** — early data is encrypted with a different key that may not be logged.
- **Pre-shared keys (PSK)** — TLS-PSK uses keys that aren't part of the handshake.

If decryption fails, Wireshark shows `Decoding TLS: no SSL keys available` in the packet detail.

### IPsec via key file

For IPsec ESP traffic, you can give Wireshark a file of SAs (security associations). **Preferences → Protocols → ESP → ESP SAs** lets you list source IP, destination IP, SPI, encryption algorithm, key, authentication algorithm, and key. Each captured packet will be matched and decrypted.

This only works for **IKEv1/IKEv2 with manual keying.** Most modern IPsec uses dynamic keying via IKE, which is harder to extract. You can have **strongSwan** dump the SAs with `swanctl --list-sas` and feed them into Wireshark by hand.

### WPA2 via PSK

For Wi-Fi captures (in monitor mode) of WPA2 traffic, Wireshark can decrypt the air if you provide the network's password and Wireshark sees the 4-way handshake. **Preferences → Protocols → IEEE 802.11 → Decryption keys → wpa-pwd** with format `password:SSID`.

Without seeing the 4-way handshake, decryption fails — the handshake is what derives the per-client key. You may need to deauthenticate the client briefly to force a re-handshake.

WPA3 (SAE) is much harder to decrypt because it uses Diffie-Hellman per-client and there is no shared key to reverse-engineer.

### What decryption looks like, end to end

Let me draw the full picture of how SSLKEYLOGFILE decryption works, because this is where many people get lost.

```
Step 1: launch the client with the env var set
  $ export SSLKEYLOGFILE=/tmp/sslkeys.log
  $ curl https://example.com

Step 2: simultaneously start the capture
  $ sudo tshark -i any -f 'tcp port 443' -w /tmp/cap.pcapng

Step 3: curl handshakes with example.com
  +----------+                              +-----------+
  |  curl    |  ClientHello (random=R1) --->|  server   |
  |          |<--ServerHello (random=R2)----|           |
  |          |    ... key exchange ...      |           |
  +----------+                              +-----------+
        |
        | derives shared key K from R1, R2, exchange
        |
        v
  +-------------------------+
  | writes to keylog file:  |
  | CLIENT_RANDOM <R1> <K>  |
  +-------------------------+

Step 4: traffic flows encrypted with K
  +----------+   ENC(GET /, K)              +-----------+
  |  curl    |----------------------------->|  server   |
  |          |<-- ENC(200 OK + body, K)-----|           |
  +----------+                              +-----------+
        |                                         |
        +------------> wire <---------------------+
                       captured
                          |
                          v
                  /tmp/cap.pcapng
                  (encrypted bytes)

Step 5: tell Wireshark about the keylog
  Preferences > Protocols > TLS > (Pre)-Master-Secret log filename
                       = /tmp/sslkeys.log

Step 6: Wireshark matches R1 in the file with R1 in the capture,
        derives K, decrypts the bytes, dissects HTTP.

  +----------------------------------------+
  | Hypertext Transfer Protocol            |
  |   GET / HTTP/1.1\r\n                   |
  |   Host: example.com\r\n                |
  |   ...                                  |
  +----------------------------------------+
```

This works for **TLS 1.2 and 1.3.** For 1.2, the line is `CLIENT_RANDOM`. For 1.3, you'll see four lines: `CLIENT_HANDSHAKE_TRAFFIC_SECRET`, `SERVER_HANDSHAKE_TRAFFIC_SECRET`, `CLIENT_TRAFFIC_SECRET_0`, `SERVER_TRAFFIC_SECRET_0`. Wireshark handles both transparently.

The file is append-only. Every new connection adds new lines. Don't delete it between captures or you'll lose the keys for the captured traffic.

## Statistics

Wireshark has a whole menu of analysis tools that turn raw packets into summaries.

### Conversations

**Statistics → Conversations** groups packets by source/destination pair. For each pair you see: address A, address B, packets A->B, packets B->A, bytes A->B, bytes B->A, total packets, total bytes, duration, and bits/sec. This is the fastest way to find the noisy talker on your network.

In tshark:

```bash
$ tshark -r capture.pcapng -q -z conv,tcp
================================================================================
TCP Conversations
                                               |       <-      | |       ->      |
                                               | Frames  Bytes | | Frames  Bytes |
10.0.0.5:54321  <-> 93.184.216.34:443             45    35,201    50    4,812
10.0.0.5:54322  <-> 142.250.80.110:443            120  201,883   115   13,944
================================================================================
```

You can do `conv,udp`, `conv,ip`, `conv,eth` too.

### Endpoints

**Statistics → Endpoints** is similar but groups by single host instead of pairs. This is "who is talking the most overall."

```bash
$ tshark -r capture.pcapng -q -z endpoints,ip
```

### IO Graphs

**Statistics → I/O Graphs** plots packet rate or byte rate over time. You can layer multiple lines (one per filter), so you can see "all HTTP traffic" alongside "TCP retransmits" alongside "DNS queries" on the same graph. Spikes that line up tell you correlated events.

In tshark you get a text version:

```bash
$ tshark -r capture.pcapng -q -z io,stat,1
===================================================================================
| IO Statistics                                                                   |
|                                                                                 |
| Duration: 12.452 secs                                                           |
| Interval:  1.000 secs                                                           |
|                                                                                 |
| Col 1: Frames and bytes                                                         |
|---------------------------------------------------------------------------------|
|              |1                 |                                               |
| Interval     | Frames |   Bytes |                                               |
|---------------------------------------------------------------------------------|
|  0.0 <>  1.0 |     45 |   12345 |                                               |
|  1.0 <>  2.0 |     67 |   18900 |                                               |
|  2.0 <>  3.0 |    102 |   31200 |                                               |
===================================================================================
```

### Expert Info

**Analyze → Expert Info** lists Wireshark's automatically detected anomalies: retransmissions, duplicate ACKs, zero windows, out-of-order packets, malformed packets, ICMP errors. Each entry is colored by severity (Chat, Note, Warning, Error). Click an entry and Wireshark jumps to the packet.

This is the first place to look when troubleshooting "the network feels slow." High retransmission rates, lots of zero windows, or many duplicate ACKs are immediate red flags.

### Protocol Hierarchy

**Statistics → Protocol Hierarchy** shows a tree breakdown of how many packets used each protocol. Useful for "what's actually on this network?" — sometimes the answer is "70% mDNS noise from printers" which you can immediately filter out.

## Profiles

When you open Wireshark for the first time, it shows you the default columns: number, time, source, destination, protocol, length, info. That is fine for general use, but as you specialize, you want different columns for different jobs.

A **profile** is a saved bundle of: column layout, color rules, capture filters, display filters, preferences, and a few other settings. Switching profiles re-skins your entire Wireshark.

Examples:
- A **VoIP profile** with columns showing RTP sequence number and SSRC, plus color rules that paint RTP green and SIP blue.
- A **TLS handshake profile** with columns showing SNI and cipher suite, plus color rules highlighting alerts in red.
- A **DHCP profile** with columns showing client MAC, requested IP, and option fields.

Profiles live in `~/.config/wireshark/profiles/<name>/` on Linux and macOS. Each subdirectory contains files like `colorfilters`, `dfilters`, `cfilters`, `preferences`, `recent`. To copy a profile between machines, just copy the folder.

To create a profile: **Edit → Configuration Profiles → Plus → name it.** Click the new profile in the list and click OK to switch to it. Now any column changes, color rules, or preference tweaks happen in that profile only.

The profile name is visible in the bottom-right corner of the Wireshark status bar.

### Color rules: making important packets jump out

When you stare at thousands of packets, your eyes glaze over. Color rules paint each packet a background color based on a display filter, so important things visually leap off the screen.

The default rule list (top-down, first match wins):

1. Bad TCP — red — for retransmits, zero windows, etc.
2. HSRP State Change — yellow — first hop redundancy events
3. Spanning Tree Topology Change — yellow
4. OSPF State Change — yellow
5. ICMP errors — red
6. ARP — pale yellow
7. UDP — pale blue
8. TCP — pale gray
9. HTTP — pale green
10. ... and many more.

You can add your own. Right-click a packet → **Colorize Conversation → New Coloring Rule**. Or **View → Coloring Rules**.

Common custom rules:

```
Filter: tls.alert_message              Color: bright red
Filter: dns.flags.rcode == 3            Color: orange     (NXDOMAIN)
Filter: tcp.analysis.zero_window        Color: dark red
Filter: http.response.code >= 500       Color: red
Filter: tcp.flags.syn == 1 and tcp.flags.ack == 0   Color: bright green   (new connections)
```

Now in any capture, retransmits glow red, NXDOMAINs glow orange, new connections glow green. Pattern-matching on color is much faster than reading every row.

Color rules live in the active profile, so you can have one set of rules for VoIP captures and another for web debugging.

## Lua Dissectors

Wireshark already understands 3,000+ protocols, but if you have a custom protocol — say your company invented their own message format on top of UDP port 9999 — you can teach Wireshark to dissect it without recompiling. Wireshark embeds a **Lua interpreter,** and any `.lua` file in `~/.config/wireshark/plugins/` (or `~/.local/lib/wireshark/plugins/` on Linux) gets loaded at startup.

A minimal Lua dissector:

```lua
local p_my = Proto("myproto", "My Custom Protocol")

local f_id   = ProtoField.uint16("myproto.id",   "Message ID")
local f_len  = ProtoField.uint16("myproto.len",  "Length")
local f_data = ProtoField.string("myproto.data", "Payload")

p_my.fields = { f_id, f_len, f_data }

function p_my.dissector(buf, pinfo, tree)
  pinfo.cols.protocol = "MYPROTO"
  local subtree = tree:add(p_my, buf(0))
  subtree:add(f_id,   buf(0, 2))
  subtree:add(f_len,  buf(2, 2))
  subtree:add(f_data, buf(4))
end

DissectorTable.get("udp.port"):add(9999, p_my)
```

Drop that into `~/.config/wireshark/plugins/myproto.lua`, restart Wireshark, and any UDP packet on port 9999 will now be dissected as your protocol with named fields you can filter on (`myproto.id == 42`).

This is the fast path for proprietary protocols. The slow path is writing a C dissector and contributing it upstream.

## Capture for Analysis Later

Sometimes you can't be in front of the screen when the problem happens. The pattern is: start a long capture, let it run, and when the problem hits, grab the recent file and analyze it offline.

### The classic "leave it running" recipe

```bash
$ sudo dumpcap -i any \
    -f 'not port 22' \
    -w /var/log/captures/cap.pcapng \
    -b filesize:100000 \
    -b files:48
```

Translation: capture from all interfaces, ignoring SSH (so you don't capture your own session), write 100 MB files, keep the last 48. That's about 4.8 GB max, covering many hours of typical server traffic.

When a problem happens at 3:47 PM, list the files:

```bash
$ ls -lh /var/log/captures/
-rw-r--r-- 1 root root 100M Apr 27 14:12 cap_00001_20260427141200.pcapng
-rw-r--r-- 1 root root 100M Apr 27 14:35 cap_00002_20260427143501.pcapng
-rw-r--r-- 1 root root 100M Apr 27 15:02 cap_00003_20260427150200.pcapng
-rw-r--r-- 1 root root  78M Apr 27 15:48 cap_00004_20260427154800.pcapng
```

The filenames carry timestamps. Pick the one covering 3:47 and open it.

### Time-based slicing

If you only want a window around the problem:

```bash
$ editcap -A "2026-04-27 15:30:00" -B "2026-04-27 15:50:00" \
    cap_00004.pcapng problem-window.pcapng
```

`-A` is "after" (start time), `-B` is "before" (end time). The output is a smaller file with only the packets in that window.

### Splitting

```bash
# split into chunks of 1000 packets
$ editcap -c 1000 big.pcapng small.pcapng
```

You get `small_00000_*.pcapng`, `small_00001_*.pcapng`, etc. Useful for emailing a snippet to a vendor.

### Merging

```bash
$ mergecap -w merged.pcapng cap_00001.pcapng cap_00002.pcapng cap_00003.pcapng
```

Merges multiple captures into one, sorted by timestamp.

### Markers and time references

**Marker** (Ctrl-M): toggles a black bar next to a packet. Use markers to highlight "this is the request that started the bug" and "this is when the page finally loaded." Markers persist in the saved pcapng but only across Wireshark.

**Time reference** (Ctrl-T): sets the displayed time of a chosen packet to 0:00:00. All other times become deltas relative to it. Best used at the start of a transaction to count milliseconds from there: "the page took 247ms from request to response, and the SQL query took 31ms of that."

You can have multiple time references in one capture. Each one resets the clock; subsequent packets show time since the most recent reference.

## tshark Power Tools

tshark is a Swiss army knife for terminal-based analysis. The big idea: you can pipe its output into other tools.

### Field extraction

```bash
$ tshark -r cap.pcapng -T fields \
    -e frame.number -e ip.src -e ip.dst -e tcp.srcport -e tcp.dstport
1   10.0.0.5  93.184.216.34  54321  443
2   93.184.216.34  10.0.0.5  443    54321
3   10.0.0.5  93.184.216.34  54321  443
```

`-T fields` says "tab-separated columns." `-e <field>` says "include this field." Pipe to `awk`, `cut`, `sort`, `uniq -c`. Wireshark dissectors give you names for **every** field — there are over 200,000.

To list them:

```bash
$ tshark -G fields | grep tls.handshake
tls.handshake          TLS Handshake Protocol  ...
tls.handshake.type     Handshake Type          ...
tls.handshake.version  Version                 ...
...
```

### JSON output

```bash
$ tshark -r cap.pcapng -T json | jq '.[].layers.ip.dst' | sort -u
```

JSON is the safest output for piping into another tool because it preserves all structure. Combine with `jq` for queries.

### One-shot stats from the command line

```bash
# top talkers by IP
$ tshark -r cap.pcapng -q -z conv,ip

# DNS query counts
$ tshark -r cap.pcapng -Y dns.flags.response==0 -T fields -e dns.qry.name | sort | uniq -c | sort -rn | head

# count HTTP response codes
$ tshark -r cap.pcapng -Y http.response -T fields -e http.response.code | sort | uniq -c
```

`-Y <filter>` is a display filter applied during reading. `-q` says "be quiet, just print the requested stats." `-z <stat>` selects the statistic.

### Verbose mode

```bash
$ tshark -r cap.pcapng -V | less
```

`-V` prints every field of every packet — the same dissector tree you'd see in Wireshark's middle pane, just as text. Often the fastest way to dump a single packet for a Stack Overflow question.

## dumpcap

Here's a fact that surprises everyone: **Wireshark does not capture packets.** It only displays them. The actual capture is done by a separate tiny program called **dumpcap.** When you click "start capture" in Wireshark, it spawns dumpcap, which reads from the kernel and writes to a file, and then Wireshark reads that file as it grows.

Why split it? **Privilege.** Capturing packets requires elevated permissions (root, basically). Displaying them does not. By keeping the privileged code in a tiny standalone binary, only `dumpcap` needs to run as root, and Wireshark itself runs as your normal user. If a malicious pcap exploits a Wireshark dissector bug, the attacker has your user's privileges, not root's.

You can use dumpcap directly:

```bash
$ sudo dumpcap -i any -w /tmp/cap.pcapng
```

Same flags as tshark for the most part: `-i`, `-f`, `-w`, `-b`, `-s`. dumpcap is simpler and faster — it only captures, it doesn't dissect. For long-running captures on busy networks, prefer dumpcap over tshark.

## Capture Privileges

To capture packets you need to read raw network frames from the kernel. That requires either:

1. Run the program as **root** (with `sudo`).
2. Give the program the **capabilities** `CAP_NET_RAW` (read raw sockets) and `CAP_NET_ADMIN` (configure the network).
3. Make the program **setuid root** (it briefly becomes root, captures, then drops back).

Approach 1 is the sledgehammer: `sudo wireshark`. Works but runs the whole UI as root, which is a security smell.

Approach 2 is the modern way on Linux:

```bash
$ sudo setcap cap_net_raw,cap_net_admin=eip /usr/bin/dumpcap
```

This gives `dumpcap` exactly the permissions it needs and nothing else. Now any user can run dumpcap (and through it, Wireshark) without sudo. Most distro packages do this automatically when you install Wireshark.

Approach 3 is the BSD/macOS way: `dumpcap` is setuid root. When you run it as a normal user, the kernel briefly gives it root, it opens the capture socket, then it drops back to your user before reading any data. macOS Wireshark uses this via the `ChmodBPF` package.

To check your dumpcap's capabilities on Linux:

```bash
$ getcap /usr/bin/dumpcap
/usr/bin/dumpcap = cap_net_admin,cap_net_raw+eip
```

If this is empty, dumpcap can't capture without sudo.

### Adding yourself to the wireshark group

On Debian/Ubuntu after installing Wireshark, you'll be asked "Should non-superusers be able to capture packets?" If you say yes, a group called `wireshark` is created and dumpcap is set group-readable to that group with the right capabilities. You then add yourself:

```bash
$ sudo usermod -aG wireshark $USER
$ newgrp wireshark
```

Log out and back in, and Wireshark works without sudo.

## RPCAP / Remote Capture

Sometimes you want to capture packets on machine A while looking at them on machine B. **RPCAP** (Remote Packet Capture) is the old protocol for this, supported by Wireshark via the `rpcap://` interface URI. The remote machine runs an `rpcapd` daemon, and Wireshark connects over TCP and pulls packets across.

```bash
# on the remote machine:
$ sudo rpcapd -n -p 2002

# in Wireshark on your laptop:
# Capture > Manage Interfaces > Remote Interfaces > Add
# Host: remote.example.com  Port: 2002  Auth: Null
```

In practice, RPCAP is rarely the right tool today. It is unencrypted by default, requires opening a port, and is not packaged on most distros. The modern alternative is just SSH:

```bash
# stream tcpdump output from remote into Wireshark over SSH
$ ssh remote 'sudo tcpdump -U -s0 -w - "not port 22"' | wireshark -k -i -
```

`-w -` writes the pcap to stdout, `-U` flushes per-packet, `wireshark -k -i -` reads from stdin and starts capture immediately. This gives you a live remote capture over SSH with no extra daemon.

### A worked example: capture, filter, follow, decrypt

Let's walk through a complete real-world session. You're trying to figure out why a particular curl request is misbehaving.

```bash
# step 1: prepare the keylog file
$ export SSLKEYLOGFILE=/tmp/sslkeys.log
$ : > /tmp/sslkeys.log    # truncate any old keys

# step 2: start the capture in another terminal
$ sudo dumpcap -i any -f 'host api.example.com' -w /tmp/debug.pcapng

# step 3: reproduce the bug
$ curl -v https://api.example.com/v1/users/42

# step 4: stop the capture (Ctrl-C in the dumpcap terminal)

# step 5: decrypt and follow the stream
$ tshark -r /tmp/debug.pcapng \
    -o tls.keylog_file:/tmp/sslkeys.log \
    -q -z follow,tls,ascii,0
```

You should see the cleartext HTTP request and response. Read it. Compare to what curl printed. Often the bug is "the API returned a 500 with a body that curl truncated, but Wireshark sees the full body."

If you want to inspect the request headers as fields:

```bash
$ tshark -r /tmp/debug.pcapng \
    -o tls.keylog_file:/tmp/sslkeys.log \
    -Y http \
    -T fields -e http.request.method -e http.request.uri \
    -e http.user_agent -e http.host
```

Or as a verbose dump:

```bash
$ tshark -r /tmp/debug.pcapng \
    -o tls.keylog_file:/tmp/sslkeys.log \
    -Y http -V | less
```

## Common Errors

Real error messages and what they mean.

**"There are no interfaces on which a capture can be done"**

The user running Wireshark doesn't have permission to read raw packets. Either run with sudo, give dumpcap capabilities (see above), or add yourself to the `wireshark` group.

**"You don't have permission to capture on that device"**

Same root cause as above, expressed differently. Often appears on macOS the first time you run Wireshark, until you install ChmodBPF.

**"Capture: Couldn't run /usr/sbin/dumpcap"**

Wireshark can't find or execute dumpcap. Check `which dumpcap`. On macOS check that you installed Wireshark with the official .dmg (Homebrew formula sometimes misses dumpcap).

**"Decoding TLS: no SSL keys available"**

You're trying to decrypt TLS but Wireshark has no key file. Set `SSLKEYLOGFILE` before launching the client, then point Wireshark at the file in **Preferences → Protocols → TLS.**

**"Capture file appears to be damaged or corrupt"**

The pcap or pcapng file got truncated or corrupted. This often happens if dumpcap was killed mid-write. Try `pcapfix capture.pcapng` to recover what's possible. If that fails, the data after the corruption is gone.

**"Display filter syntax error"**

You typed BPF (capture filter) syntax into the display filter box. Use Wireshark's dialect: `tcp.port == 443` not `port 443`. The error message usually tells you the exact column where it choked.

**"Selected interface 'X' is not running"**

The interface is down (`ip link set X up` to bring it up), or you typo'd the name. Check `tshark -D` for the actual list.

**"BPF: too many simultaneous capture sessions"**

The kernel has a per-user or system-wide limit on how many BPF capture sessions can be open at once. Close other captures (tcpdump, other Wireshark windows). On Linux, raise `/proc/sys/net/core/bpf_jit_limit` if you're doing something extreme.

**"Couldn't load module"**

A Lua plugin failed to parse. Check `~/.config/wireshark/plugins/` for syntax errors. Wireshark prints the line number on startup.

## Hands-On

A long catalog of commands you can run right now to see real packets.

```bash
# capture from your default interface for 10 seconds, write to file
$ sudo tshark -i en0 -a duration:10 -w /tmp/cap.pcapng

# read the file back and print summary lines
$ tshark -r /tmp/cap.pcapng | head

# capture from any interface, only TCP port 443
$ sudo tshark -i any -f 'tcp port 443'

# verbose mode - print every dissected field of every packet
$ tshark -r /tmp/cap.pcapng -V | less

# extract specific fields as tab-separated columns
$ tshark -r /tmp/cap.pcapng -T fields \
    -e frame.number -e ip.src -e ip.dst -e tcp.port

# JSON output
$ tshark -r /tmp/cap.pcapng -T json | jq '.[0]' | head -20

# only packets matching a display filter while reading
$ tshark -r /tmp/cap.pcapng -Y 'http.response.code == 404'

# IO statistics, 1 second buckets
$ tshark -r /tmp/cap.pcapng -q -z io,stat,1

# TCP conversations
$ tshark -r /tmp/cap.pcapng -q -z conv,tcp

# follow TCP stream number 0
$ tshark -r /tmp/cap.pcapng -q -z follow,tcp,ascii,0

# dumpcap with ring buffer
$ sudo dumpcap -i any -f 'host 10.0.0.1' \
    -w /tmp/cap.pcapng -b filesize:100000 -b files:10

# editcap: deduplicate packets
$ editcap -d input.pcapng deduped.pcapng

# editcap: shift all timestamps by +5 seconds (rare, but useful for clock skew)
$ editcap -t 5 input.pcapng shifted.pcapng

# editcap: split into 1000-packet chunks
$ editcap -c 1000 input.pcapng chunk.pcapng

# editcap: keep only packets between two times
$ editcap -A "2026-04-27 15:00:00" -B "2026-04-27 15:30:00" \
    input.pcapng window.pcapng

# mergecap: combine two captures
$ mergecap -w merged.pcapng a.pcapng b.pcapng

# capinfos: print summary stats about a file
$ capinfos /tmp/cap.pcapng

# text2pcap: turn a hex dump back into a pcap (useful for paper logs)
$ text2pcap input.txt output.pcap

# pcap-filter syntax demo
$ man pcap-filter

# tcpdump-style one-line summaries
$ tshark -r /tmp/cap.pcapng -q

# bring an interface up before capture
$ sudo ip link set eth0 up

# explicitly enable promiscuous mode
$ sudo ip link set eth0 promisc on

# disable hardware checksum offload to get correct checksums in capture
$ sudo ethtool -K eth0 tx-checksumming off
$ sudo ethtool -K eth0 rx-checksumming off

# show driver and hardware features that affect captures
$ ethtool -k eth0

# check Wireshark and tshark versions
$ wireshark --version
$ tshark --version

# list every TLS-related field name
$ tshark -G fields | grep '^F\s*tls'

# list field types
$ tshark -G ftypes

# dump every protocol Wireshark knows about
$ tshark -G protocols | head

# generate synthetic traffic for testing
$ randpkt -t tcp -c 100 /tmp/synthetic.pcap

# get capture-time fields available
$ tshark -r /tmp/cap.pcapng -T fields -e frame.time -e frame.len | head

# count DNS queries by name
$ tshark -r /tmp/cap.pcapng -Y 'dns.flags.response==0' \
    -T fields -e dns.qry.name | sort | uniq -c | sort -rn | head

# show all TLS Client Hellos with SNI
$ tshark -r /tmp/cap.pcapng -Y 'tls.handshake.type == 1' \
    -T fields -e ip.dst -e tls.handshake.extensions_server_name

# show HTTP requests with method and host
$ tshark -r /tmp/cap.pcapng -Y http.request \
    -T fields -e http.request.method -e http.host -e http.request.uri

# decrypt TLS using a key file
$ tshark -r /tmp/cap.pcapng -o tls.keylog_file:/tmp/sslkeys.log \
    -Y 'tls' -T fields -e http.request.uri

# capture with a key log file at the same time
$ SSLKEYLOGFILE=/tmp/sslkeys.log curl -s https://example.com >/dev/null
$ sudo tshark -i en0 -f 'host example.com' -a duration:5 \
    -o tls.keylog_file:/tmp/sslkeys.log -V

# disable name resolution (often makes captures faster and cleaner)
$ tshark -r /tmp/cap.pcapng -n

# show only packet count
$ capinfos -c /tmp/cap.pcapng

# print absolute timestamps
$ tshark -r /tmp/cap.pcapng -t ad

# print delta times between displayed packets
$ tshark -r /tmp/cap.pcapng -t d
```

That is well over 30 commands. Run each one. Watch what comes back. Wireshark only becomes intuitive once you have run captures on real traffic and stared at the output.

## Common Confusions

**Capture filter vs display filter syntax.** They look similar but they are different languages. Capture filters use BPF (`port 443`, `host 10.0.0.1`); display filters use Wireshark dialect (`tcp.port == 443`, `ip.addr == 10.0.0.1`). If the capture filter box rejects your filter, you typed Wireshark dialect; switch dialects.

**Checksum errors due to offload.** Modern NICs compute TCP/UDP/IP checksums in hardware on transmit, and Wireshark captures the packet before that hardware step runs. So every outgoing packet shows a "checksum incorrect" warning. **The checksum is fine on the wire** — it just doesn't get filled in until later. Disable with `ethtool -K eth0 tx-checksumming off` or just tell Wireshark to ignore the warning in **Preferences → Protocols → IPv4/TCP/UDP → Validate the checksum if possible.**

**Promiscuous mode vs monitor mode.** Promiscuous mode tells your NIC to hand the kernel every packet that flies past, even ones not addressed to you. Monitor mode (Wi-Fi only) tells the wireless card to capture raw 802.11 frames including beacons, probes, and frames for other networks. Most laptops can do promiscuous easily; monitor mode requires specific cards and driver support.

**Why your VM doesn't see traffic on the host.** A VM with a NAT'd network card only sees its own NAT'd traffic, not the host's traffic. To capture host traffic from inside a VM, you need bridge networking, or you need to capture on the host and share the file. Promiscuous mode does not cross hypervisor boundaries.

**How SSLKEYLOGFILE actually works.** It is **not** a master TLS key. It is a per-session log. The browser writes one or more lines per TLS session, recording the random and the secret used by that session. Wireshark reads the file, matches the random against captured packets, and derives the per-session keys. Decryption only works for sessions whose lines are in the file. If you start curl, it writes lines for that session; the next curl writes more lines for a different session.

**Why some TLS sessions don't decrypt even with the key log.** Session resumption can reuse keys from a session you didn't capture. TLS 1.3 0-RTT data uses keys derived earlier. Some libraries don't support keylog (BoringSSL, some Go versions) so they don't write the file. And of course, Java's standard TLS library does not write SSLKEYLOGFILE without a third-party agent.

**DNS over HTTPS hiding from your capture.** If a browser uses DNS-over-HTTPS (DoH), the DNS query is wrapped inside a TLS connection to a resolver like `1.1.1.1` or `8.8.8.8`. Your capture sees TLS traffic but no DNS. Disable DoH in the browser if you want plain DNS to capture, or capture inside the resolver. Same for DNS-over-TLS (DoT, port 853) and DNS-over-QUIC (DoQ).

**Follow TCP Stream vs Follow HTTP Stream.** Follow TCP shows you the raw bytes of the connection, including chunked transfer encoding markers and any gzip-compressed body. Follow HTTP undoes both — you see the cleartext HTML/JSON. For a single GET request the difference is small; for a streaming response with chunked encoding, the HTTP follow is much more readable.

**BPF tcpdump expression vs ip filters in display.** `tcpdump -i any host 10.0.0.1` uses BPF. `tshark -Y 'ip.addr == 10.0.0.1'` uses display filter. They look similar but pass through different machinery. tcpdump's filter is compiled to BPF and runs in the kernel; tshark's `-Y` runs after dissection in user space. For the same logical match, the BPF version is faster.

**What "Reassembled TCP segments" really means.** TCP delivers byte streams, and a single application-level message may span many TCP segments. Wireshark glues them back together so the dissector for the application protocol (HTTP, TLS, etc.) sees the whole message. The "Reassembled" annotation in the packet detail tells you which packets contributed to the reassembled message. Don't be confused — the wire still saw N separate segments; reassembly happens only inside Wireshark.

**HTTP/2 vs HTTP/1.1 framing.** HTTP/1.1 is one request followed by one response, both as text. HTTP/2 is binary frames multiplexed over a single TCP connection — many requests and responses interleave. In Wireshark, HTTP/2 packets show a `Stream ID` field and you can use **Follow HTTP/2 Stream** to extract one request/response pair from the multiplexed soup.

**Why some packets show as Malformed.** Either the packet really is malformed (truncated, corrupted, sender bug) or Wireshark's dissector got out of sync. Sometimes a missing earlier packet means later packets can't be reassembled. Sometimes you captured with `-s 96` and the dissector hit the truncation. Sometimes a custom protocol on a standard port confuses the dissector. Inspect the packet bytes; the truth is in the hex.

**Wireshark says "no captures" but tcpdump works.** You need `cap_net_raw` and `cap_net_admin` on dumpcap, or membership in the `wireshark` group. tcpdump runs setuid in many distros; dumpcap only does after install scripts run.

**Capture files keep growing forever.** You forgot the ring buffer flags. Always use `-b filesize:N -b files:M` for long captures.

**Packet timestamps look wrong.** Either your system clock is wrong (check `chronyc tracking` or `timedatectl`), or you captured on one machine and analyzed on another with different time zone. Use `-t ad` to force absolute timestamps with date and zone displayed.

## Vocabulary

- **Wireshark** — the graphical packet analyzer.
- **tshark** — the terminal version of Wireshark.
- **dumpcap** — the actual capture engine; everything else just calls dumpcap.
- **editcap** — pcap file editor (split, merge, convert, time-shift, dedupe).
- **mergecap** — merges multiple pcaps into one in timestamp order.
- **capinfos** — prints summary stats about a pcap file.
- **text2pcap** — turns a hex dump (from a log) back into a pcap.
- **randpkt** — generates synthetic packets for testing.
- **sharkd** — daemon that exposes Wireshark's dissector engine over a JSON-RPC-ish socket.
- **libpcap** — the C library that opens a capture socket on the OS.
- **libwiretap** — the file-format library inside Wireshark (knows pcap, pcapng, snoop, etc.).
- **libwireshark** — the dissector library.
- **npcap** — the modern Windows packet capture driver. Replaced WinPcap (deprecated since 2013, no updates after 2018). npcap ships with current Wireshark on Windows.
- **WinPcap** — old Windows capture driver, deprecated.
- **pcap** — the original capture file format (microsecond timestamps, single interface).
- **pcapng** — the newer file format with nanosecond timestamps, multiple interfaces, comments. Default since Wireshark 1.10 (2013).
- **packet** — a unit of data on the network. Generic term.
- **frame** — a unit of data at layer 2 (Ethernet frame, Wi-Fi frame).
- **datagram** — a unit of data in a connectionless protocol like UDP or IP.
- **segment** — a unit of data at the TCP layer.
- **capture filter (BPF)** — filter that runs in the kernel before saving.
- **display filter** — filter that runs after dissection, hides packets from view.
- **dissector** — code that knows how to parse one protocol.
- **expert info** — Wireshark's auto-detected anomalies (retransmits, malformed, etc.).
- **color rule** — a rule that paints matching packets a specific color in the list.
- **profile** — saved bundle of columns, color rules, and preferences.
- **marker** — a per-packet bookmark you set with Ctrl-M.
- **time reference** — a per-packet "set time = 0 here" anchor for relative-time display.
- **IO graph** — a chart of packet/byte rate over time, optionally per filter.
- **conversation** — all packets between one source/destination pair.
- **endpoint** — all packets to/from a single host.
- **follow stream** — reassemble and display one stream's content.
- **reassembly** — gluing TCP segments back into application messages.
- **defragmentation** — gluing IP fragments back into a single packet.
- **decryption** — turning encrypted bytes back into plaintext using keys.
- **SSLKEYLOGFILE** — environment variable; tells supported clients to write per-session TLS secrets to a file Wireshark can read.
- **NSS keylog format** — the text format used by SSLKEYLOGFILE (lines like `CLIENT_RANDOM <hex> <hex>`).
- **premaster secret** — TLS 1.2 intermediate value from which keys are derived.
- **server-random** — random bytes the server picks at the start of TLS handshake.
- **client-random** — random bytes the client picks at the start of TLS handshake.
- **RSA key file** — the server's private key, can decrypt TLS 1.2 sessions that used RSA key exchange (rare today).
- **DH key** — Diffie-Hellman key exchange. Forward-secret. Cannot be decrypted from server private key alone.
- **ECDHE** — Elliptic Curve Diffie-Hellman Ephemeral. Modern default. Forward-secret.
- **IPsec ESP key** — encryption key for an IPsec security association.
- **WPA2 PSK** — pre-shared key for WPA2 Wi-Fi. With handshake captured, lets Wireshark decrypt.
- **WPA3 SAE** — Simultaneous Authentication of Equals; replaces PSK exchange in WPA3.
- **monitor mode** — Wi-Fi card mode that captures raw 802.11 including frames not for you.
- **promiscuous mode** — NIC mode that delivers all frames to the kernel, not just ones for you.
- **RPCAP** — remote packet capture protocol. Old; use SSH instead today.
- **remote capture** — capturing on one machine, displaying on another.
- **capture privileges** — the rights needed to read raw packets.
- **CAP_NET_RAW** — Linux capability for raw socket access.
- **CAP_NET_ADMIN** — Linux capability for network configuration.
- **setcap** — command that sets file capabilities on a binary.
- **dumpcap setuid** — alternative where dumpcap is setuid root.
- **--enable-cap-ng-support** — Wireshark build flag that uses libcap-ng to drop privileges.
- **ring buffer** — circular file rotation: keep last N files, drop the oldest.
- **file rotation** — splitting a long capture into multiple files.
- **multi-file output** — same idea; produces files like `cap_00001.pcapng`, `cap_00002.pcapng`.
- **snap length / snaplen** — max bytes captured per packet. `0` means full packet.
- **BPF JIT** — kernel feature that compiles BPF filters to native machine code for speed.
- **libpcap-mmap** — memory-mapped variant of libpcap that's faster on Linux.
- **PACKET_MMAP** — Linux kernel mechanism for fast packet capture via shared memory ring.
- **AF_PACKET** — Linux raw socket family for layer 2 packet I/O.
- **TPACKET_V3** — version 3 of AF_PACKET's mmap protocol; used by modern dumpcap.
- **ETHTOOL_GFEATURES** — ethtool ioctl to read NIC offload settings.
- **RX-checksum offload** — NIC computes/verifies receive checksums in hardware.
- **TX-checksum offload** — NIC computes transmit checksums in hardware (causes "incorrect checksum" warnings in Wireshark for outgoing packets).
- **TSO** — TCP Segmentation Offload. NIC splits big segments into MTU-sized ones in hardware. Wireshark sees the giant pre-split frame.
- **GRO** — Generic Receive Offload. Kernel coalesces small segments before delivery. Wireshark sees the merged frame, not the wire frames.
- **GSO** — Generic Segmentation Offload. Software equivalent of TSO.
- **LRO** — Large Receive Offload. Hardware version of GRO.
- **packet dissectors** — there are 3,000+ built into Wireshark.
- **frame.number** — packet number in the capture.
- **frame.time** — capture timestamp.
- **frame.len** — frame size in bytes.
- **eth.src / eth.dst / eth.type** — Ethernet source MAC, destination MAC, EtherType.
- **ip.src / ip.dst / ip.version / ip.ttl / ip.proto** — IPv4 fields.
- **ipv6.src / ipv6.dst / ipv6.hlim / ipv6.nxt** — IPv6 fields.
- **tcp.srcport / tcp.dstport / tcp.seq / tcp.ack / tcp.flags / tcp.window** — TCP fields.
- **udp.srcport / udp.dstport / udp.length** — UDP fields.
- **dns.qry.name / dns.flags.response** — DNS query/response.
- **http.request.method / http.host / http.user_agent / http.response.code** — HTTP fields.
- **tls.handshake.type / tls.handshake.ciphersuite / tls.handshake.version** — TLS handshake fields.
- **quic.long.packet_type / quic.short.dcid / quic.frame_type** — QUIC fields.
- **ICMP echo-request / echo-reply** — the ping pair (types 8 and 0).
- **ICMPv6 ND** — IPv6 Neighbor Discovery (replaces ARP).
- **ARP** — Address Resolution Protocol (IPv4 MAC↔IP).
- **DHCP** — Dynamic Host Configuration Protocol; assigns IPs.
- **DNS** — Domain Name System; names ↔ IPs.
- **mDNS** — Multicast DNS; for `.local` names without a server.
- **LLDP** — Link Layer Discovery Protocol; switches advertise themselves.
- **CDP** — Cisco Discovery Protocol; Cisco's older equivalent.
- **STP** — Spanning Tree Protocol; prevents loops on layer 2.
- **RSTP** — Rapid STP.
- **MSTP** — Multiple STP.
- **OSPF** — link-state routing protocol.
- **BGP** — Border Gateway Protocol; the routing protocol of the internet.
- **IS-IS** — link-state routing protocol; common in service provider networks.
- **RADIUS** — authentication protocol for network access.
- **TACACS+** — Cisco's auth/authz protocol.
- **SNMP** — Simple Network Management Protocol; reading switch/router stats.
- **stream index** — Wireshark assigns each TCP/UDP/QUIC conversation an integer stream index.
- **packet bytes pane** — bottom pane showing raw hex.
- **packet details pane** — middle pane showing the dissector tree.
- **packet list pane** — top pane showing one row per packet.
- **filter expression button** — toolbar shortcut you can save for one-click filters.
- **coloring rules** — list of colors applied top-down; first match wins.
- **packet diagram** — Wireshark 4.x added a graphical view of packet structure.
- **statistics** — the menu with Conversations, Endpoints, IO Graph, Expert Info, etc.
- **export packet bytes** — save raw bytes of a selected packet to a file.
- **export PDU** — save a higher-layer message (decrypted plaintext, reassembled HTTP) to a file.
- **dissector table** — internal lookup that maps "TCP port 443" to "TLS dissector," "UDP port 53" to "DNS dissector," etc.
- **heuristic dissector** — a dissector that examines payload content rather than port to decide if it should claim the packet.
- **decode as** — manual override telling Wireshark "treat this port as that protocol."
- **preferences** — Wireshark's giant config tree.
- **Lua plugin** — a `.lua` file in the plugins directory; loaded at startup.
- **C plugin** — a compiled dissector loaded as a shared library.
- **wireshark-gtk** — old GTK-based UI, removed in 4.x.
- **wireshark-qt** — current Qt-based UI; default since 2.0 (2015).
- **TUI** — text user interface; tshark is the closest thing Wireshark has.
- **packet generator** — tool for crafting custom packets (Scapy, hping3, ostinato).
- **pcap-filter** — the BPF dialect spec (`man pcap-filter`).
- **--no-promiscuous-mode** — disable promiscuous mode for capture.
- **--no-name-resolution** — don't resolve IPs to hostnames; faster and less spammy.
- **Wi-Fi PHY headers** — the radio-layer info (signal strength, channel) added in monitor-mode captures.
- **radiotap** — encapsulation that adds Wi-Fi PHY metadata to monitor-mode pcaps.
- **PPI** — Per-Packet Information; another Wi-Fi metadata format.
- **Bluetooth HCI** — Host-Controller Interface log dissectors.
- **USBPcap** — capture USB traffic on Windows.
- **usbmon** — Linux USB packet capture (Wireshark dissects).
- **D-Bus dissector** — Wireshark can dissect D-Bus messages.
- **sFlow** — packet sampling protocol; Wireshark dissects sFlow datagrams.
- **NetFlow** — Cisco's flow export; Wireshark dissects v5/v9/IPFIX.
- **IPFIX** — successor to NetFlow v9; standardized.
- **ERSPAN** — Cisco's encapsulated remote SPAN.
- **GRE** — Generic Routing Encapsulation; Wireshark unwraps GRE-encapsulated traffic.
- **VXLAN** — Virtual Extensible LAN; tunnel protocol Wireshark dissects.
- **GENEVE** — newer encapsulation alternative to VXLAN.
- **MPLS** — Multiprotocol Label Switching; Wireshark dissects label stacks.

That's well over 120 vocabulary entries. The trick is not to memorize them — the trick is to have them all in one place when you hit a word in Wireshark and need to know what it means.

## Try This

Five exercises, in order. Each one builds on the last.

### 1. Capture a single ping

Open a terminal. Run:

```bash
$ sudo tshark -i any -f 'icmp' -a duration:5 -w /tmp/ping.pcapng &
$ sleep 1
$ ping -c 3 1.1.1.1
$ wait
$ tshark -r /tmp/ping.pcapng
```

You should see six packets — three echo requests and three echo replies. Read the source/destination columns. Notice that source and destination flip on each request/reply pair.

Now run it again with `-V` and read the dissected fields:

```bash
$ tshark -r /tmp/ping.pcapng -V | less
```

You'll see Ethernet, IP, ICMP layers all dissected out. Notice the ICMP type (8 = request, 0 = reply) and code.

### 2. Find your DNS queries

```bash
$ sudo tshark -i any -f 'udp port 53' -a duration:30 -w /tmp/dns.pcapng &
$ # in another terminal, browse some websites
```

After 30 seconds, look at the captured queries:

```bash
$ tshark -r /tmp/dns.pcapng -Y 'dns.flags.response==0' \
    -T fields -e frame.time -e dns.qry.name | sort -u
```

Every domain your computer asked about. You will be surprised how chatty your machine is.

### 3. Watch a TLS handshake

```bash
$ sudo tshark -i any -f 'host example.com and tcp port 443' -a duration:5 -w /tmp/tls.pcapng &
$ sleep 1
$ curl -s https://example.com >/dev/null
$ wait
$ tshark -r /tmp/tls.pcapng -V | less
```

Look for the Client Hello (handshake type 1), Server Hello (type 2), Certificate (type 11), and the encrypted handshake messages. Note the cipher suite the server picked.

### 4. Decrypt your own HTTPS

```bash
$ export SSLKEYLOGFILE=/tmp/sslkeys.log
$ sudo tshark -i any -f 'host example.com and tcp port 443' -a duration:5 -w /tmp/tls2.pcapng &
$ sleep 1
$ curl -s https://example.com >/dev/null
$ wait
$ tshark -r /tmp/tls2.pcapng -o tls.keylog_file:/tmp/sslkeys.log \
    -Y http -T fields -e http.host -e http.request.uri
```

If everything worked, you just decrypted your own TLS traffic and saw the cleartext HTTP request inside.

### 5. Find the noisy talker

```bash
$ sudo tshark -i any -a duration:60 -w /tmp/all.pcapng
$ tshark -r /tmp/all.pcapng -q -z conv,ip
```

Sort the list by bytes. The top entry is your noisiest pair — probably something like a video stream or a backup running. Now you can ask "is that thing supposed to be running?"

## Where to Go Next

- Read **ramp-up/tcp-eli5** if you haven't, then capture a real TCP handshake and verify each step.
- Read **ramp-up/dns-eli5**, then watch DNS queries while a browser loads a page.
- Read **ramp-up/tls-eli5**, then capture, decrypt, and follow a TLS stream end to end.
- Read **ramp-up/ebpf-eli5** to understand the kernel-side machinery that makes capture filters fast.
- Read **networking/tcpdump** for the smaller cousin and quick one-liners.
- Read **network-tools/wireshark** for the deeper reference (less narrative, more tables).
- Read **ramp-up/http3-quic-eli5** if you're seeing QUIC traffic dominate your captures.
- Open Wireshark's built-in user guide: **Help → Contents.** It is enormous and well-written.
- Watch one **SharkFest** keynote on YouTube. SharkFest is the annual Wireshark conference; talks are free and superb.

## See Also

- `network-tools/wireshark` — the full Wireshark reference sheet (less ELI5, more terse)
- `networking/tcpdump` — the smaller terminal cousin
- `networking/dig` — for chasing DNS captures with the canonical query tool
- `networking/dns` — DNS protocol reference
- `ramp-up/tcp-eli5` — gentle introduction to TCP
- `ramp-up/tls-eli5` — gentle introduction to TLS
- `ramp-up/dns-eli5` — gentle introduction to DNS
- `ramp-up/icmp-eli5` — gentle introduction to ICMP and ping
- `ramp-up/linux-kernel-eli5` — what the kernel is doing under the capture
- `ramp-up/bash-eli5` — for piping tshark output into other tools

## References

- **wireshark.org** — official site, downloads, docs, sample captures
- **wireshark.org/docs/wsug_html_chunked/** — Wireshark User Guide (the canonical manual)
- **wireshark.org/docs/dfref/** — Display filter reference (every field name)
- **"Practical Packet Analysis" by Chris Sanders** — the standard introductory book; reads like a series of case studies
- **"Wireshark Network Analysis" by Laura Chappell** — the deeper reference; encyclopaedic
- **`man tshark`** — terminal manual page
- **`man pcap-filter`** — the BPF capture filter syntax spec
- **`man dumpcap`** — manual for the actual capture engine
- **SharkFest videos** — annual Wireshark conference, all talks free on YouTube
- **RFC 1350 / 793 / 768 / 1035** — TFTP / TCP / UDP / DNS — the protocols you'll see most
- **NSS Key Log Format** — `firefox-source-docs.mozilla.org/security/nss/legacy/key_log_format/index.html`
- **PCAP-NG specification** — `pcapng.com`
