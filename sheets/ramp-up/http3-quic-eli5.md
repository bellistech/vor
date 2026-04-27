# HTTP/3 & QUIC — ELI5 (The Modern Internet's New Roads)

> HTTP/3 over QUIC is what happens when the internet gets tired of waiting in line and builds a separate fast lane for every conversation, with the security guard already standing at the door.

## Prerequisites

(none — but if you have time, `cs ramp-up tcp-eli5`, `cs ramp-up udp-eli5`, and `cs ramp-up tls-eli5` will let you see what HTTP/3 is improving on)

This sheet is for somebody who has heard the word "HTTP" and maybe used a web browser, but who has never actually thought about what is going on under the hood when a web page shows up. You do not need to know what TCP is. You do not need to know what UDP is. You do not need to know what TLS is. You do not need to know what a "round trip" is or what "head-of-line blocking" means or what "0-RTT" is. By the end of this sheet you will know all of those things, in plain English, and you will have run real commands that show real HTTP/3 traffic flying out of your computer in real time.

If a word feels weird, look it up in the **Vocabulary** table near the bottom. Every weird word in this sheet is in that table with a one-line plain-English definition.

If you see a `$` at the start of a line in a code block, that means "type the rest of this line into your terminal." You do not type the `$`. The lines underneath that don't have a `$` are what your computer prints back at you. We call that "output."

## What Even Is HTTP/3?

### Pretend the internet is the post office

Imagine you live in a town and you want to send a letter to your friend across the country. You write the letter, you put it in an envelope, you put a stamp on it, you write the address on the front, and you drop it in a big blue mailbox. Some days later your friend opens their mailbox and finds the letter. They write a reply. They put it in another envelope. They drop it in their mailbox. A few days later you find it in yours. You and your friend have just had a conversation, but each "turn" of the conversation took days. That is fine for letters but it would be terrible for web pages.

Now imagine the post office is much faster. Imagine you can put a letter in a mailbox and the letter shows up at your friend's mailbox a quarter of a second later. That is what the internet feels like. Letters whip back and forth so fast that when you click "next page" on a website, the next page appears almost instantly. The internet is just the post office, sped up to ridiculous speeds and made to carry numbers instead of paper.

The post office in this story is **the internet**. The letter is a thing called a **packet**. The address on the envelope is the **IP address**. The stamp is roughly the **port number** (which lane in the post office your letter is supposed to go down). And the rules for how the letters are written, folded, and stuffed in envelopes — those rules are called **protocols**.

A protocol is just a rule for how two computers talk to each other. It is no different from saying "in English, the question goes first and the answer comes second." Or "if I say HELLO you should say HELLO back." Or "if I send you a number, you should send back the same number plus one to prove you got it." A protocol is just a tiny set of agreements about who says what, in what order.

**HTTP** is the protocol that web browsers and web servers use to talk to each other. HTTP stands for "HyperText Transfer Protocol." HyperText is the fancy name for "web page with links in it." Transfer means "moving." Protocol means "rules." So HTTP is just "the rules for moving web pages around." That is all. The whole web — every cat picture, every YouTube video, every Wikipedia page, every TikTok, every banking website, every email web client, every cloud document — every single piece of it gets moved around using HTTP, in some flavor or other.

There have been four big flavors of HTTP. The numbers go up. Each flavor is faster than the last. But they all do the same fundamental job: moving web pages and pictures and videos and forms and JSON between your browser and a server.

### HTTP/1.0 — one letter per envelope

HTTP/1.0 is the oldest version of HTTP that is still talked about. It came out in 1996. The way HTTP/1.0 works is so simple it almost sounds like a joke.

You want to fetch a web page. So your browser opens a brand new connection to the server. A connection is like a phone line — both sides agree to be connected to each other for a while. Once the line is open, your browser says one thing. It says "GET /index.html, please." The server hears that, looks up the page, and sends it back. As soon as the page is sent, the server hangs up. The connection is over.

If the page has a picture in it, your browser sees the `<img>` tag, opens **another brand new connection**, says "GET /cat.jpg, please," gets the picture, and the connection is over again. If there are ten pictures, ten more connections. Plus connections for the CSS file, the JavaScript file, the favicon, every font, every tracking pixel.

Each connection takes time to set up. Setting up a connection is like dialing a phone number, waiting for the other end to ring, waiting for them to pick up, exchanging a quick "hi, hi back," and only then can you say what you actually wanted to say. With HTTP/1.0 you have to do this dance for every single thing on the page. If a page has fifty things on it, that is fifty dial-and-handshake sequences. It is slow. It is sad. Nobody likes it.

```
HTTP/1.0
========
Browser:   "Hi, can I open a connection?"
Server:    "Sure."
Browser:   "GET /page.html"
Server:    "<html>...</html>"
Server:    *hangs up*

Browser:   "Hi again, can I open ANOTHER connection?"
Server:    "Sure."
Browser:   "GET /cat.jpg"
Server:    "<binary jpg data>"
Server:    *hangs up*

Browser:   "Hi AGAIN, can I open ANOTHER connection?"
...and so on, for every file on the page.
```

### HTTP/1.1 — keep the line open, but still serial

HTTP/1.1 came out in 1997. It is still everywhere on the internet today. Whoever made HTTP/1.0 said "wait, opening a new connection for every single thing is silly. Let's keep the connection open." So in HTTP/1.1, your browser opens one connection, asks for the page, gets the page, asks for the cat picture, gets the cat picture, asks for the CSS file, gets the CSS file, all over the same line. That is called **keep-alive** or **persistent connection**. The connection is reused.

This is like calling a friend on the phone, and instead of hanging up after every sentence and dialing again, you just keep the line open and have a whole conversation. Way faster.

But HTTP/1.1 has one big problem: it is still **serial**. Serial means "one at a time, in order." Your browser asks for the page, then waits for the whole page to come back. Then asks for the cat picture, then waits for the whole picture to come back. Then asks for the CSS, then waits. If one of those things is huge — like a giant video — the browser is stuck waiting for it before it can ask for anything else over that connection.

Browsers cheated around this by opening **six connections at once** to the same server. Six is a magic number that browsers picked. So you would have six concurrent HTTP/1.1 conversations going to the same server, each one fetching a different thing. That helped a lot. But it is still wasteful (six handshakes instead of one) and still has the serial problem inside each individual connection.

```
HTTP/1.1
========
Browser:   "Hi, open a connection?"
Server:    "Sure."
Browser:   "GET /page.html"
Server:    "<html>...</html>"
Browser:   "GET /cat.jpg"     (same connection!)
Server:    "<jpg data>"
Browser:   "GET /style.css"
Server:    "<css data>"
...etc, all on one line, but still ONE AT A TIME.
```

### HTTP/2 — many letters at the same time

HTTP/2 came out in 2015. The people who made HTTP/2 said "wait, why are we doing things one at a time on the same connection? What if we could ask for ten things at once and get them back in any order?"

So HTTP/2 introduced **multiplexing**. Multiplexing is a fancy word that just means "many things sharing the same line at the same time." HTTP/2 takes one connection and chops up everything you want to send into little pieces called **frames**. Each frame has a tag on it that says which "stream" it belongs to. A stream is just one request/response pair. So stream 1 might be the page, stream 3 might be the cat picture, stream 5 might be the CSS file, and so on. The frames for all those streams get jumbled together and sent down the same line. The other end pulls them apart by looking at the tag on each frame.

This is like one phone line where you and your friend are having ten different conversations at once, with little number stickers on every word, and at the other end somebody reads the stickers and stacks each conversation back into the right pile. It sounds insane but computers are good at this kind of bookkeeping.

HTTP/2 is way faster than HTTP/1.1 for pages with lots of small things on them. Modern web pages have hundreds of small things — icons, ads, fonts, scripts. HTTP/2 was a huge upgrade.

But HTTP/2 has a problem too. The problem is called **head-of-line blocking**. We are going to look at it carefully in its own section, because it is the whole reason HTTP/3 exists. Hold that thought.

```
HTTP/2
======
Browser opens ONE connection.
Many streams ride that one connection at the same time.

  Stream 1 (HTML):  [F1.A][F1.B][F1.C]
  Stream 3 (JPG):   [F3.A][F3.B][F3.C][F3.D]
  Stream 5 (CSS):   [F5.A][F5.B]

The frames go down the wire mixed together:
  F1.A  F3.A  F5.A  F1.B  F3.B  F5.B  F1.C  F3.C  F3.D
  ^all sharing the same TCP connection^

Receiver re-assembles each stream from the labeled frames.
```

### HTTP/3 — every conversation gets its own private channel

HTTP/3 came out as a finished standard in 2022 (RFC 9114). It does not run on TCP. It runs on a new thing called **QUIC**, which runs on UDP. We will spend a whole section on QUIC in a minute. The short version is: HTTP/3 fixes head-of-line blocking by making each stream truly independent. If one stream is having trouble, the other streams keep flowing.

HTTP/3 is also faster to **set up** than HTTP/2. HTTP/2 over TCP plus TLS takes three round trips before any real data flows. HTTP/3 takes one. And if you have visited the site before, HTTP/3 can send your first request along with the handshake — that is called **0-RTT**, zero round trips. Fast as it gets.

So in summary:

```
HTTP version timeline (ELI5)
============================

  1996  HTTP/1.0    one connection per thing. slow.
  1997  HTTP/1.1    keep-alive. still serial inside a connection.
  2015  HTTP/2      multiplexed streams over one TCP connection.
                    head-of-line blocking still hurts.
  2022  HTTP/3      runs on QUIC instead of TCP.
                    independent streams. fast handshake. survives
                    a network change. mandatory TLS.
```

Each version solves a problem that the previous version had. None of the older versions are gone — your browser still talks HTTP/1.1 to plenty of old servers. But the modern internet, especially big sites like Google, Cloudflare, Facebook, YouTube, is mostly HTTP/2 and HTTP/3 now.

### One more way to picture the four versions

If words and protocol diagrams are not clicking, here is the food-court version.

Imagine you are at a giant food court with one cashier per stall. You have ten friends, each wants something different from a different stall.

- **HTTP/1.0** is one friend at a time. They walk to a stall, order, wait for the food, walk back to the table, sit down. Then the next friend gets up. Then the next. By the time everybody has eaten, hours have passed.

- **HTTP/1.1** is one friend at a time, but the cashier remembers them and is faster on subsequent orders ("oh you again, what'll it be?"). Still one at a time, but less per-order overhead.

- **HTTP/2** is one friend goes up to the cashier and says "I have ten orders, please make them all and serve them as they finish, in any order." The cashier yells the orders into the kitchen. Food comes back jumbled but each friend can grab their own. Way faster. **Except**: if the cashier has one shared tray and one item is delayed in the kitchen, the cashier waits for it before bringing out the tray, even though everything else is ready. That is head-of-line blocking.

- **HTTP/3** is each friend has their own miniature courier shuttling food to the table independently, in their own dedicated lane. If one courier gets stuck in line, the other nine couriers keep delivering. And the courier service has a faster sign-up process so you can start ordering almost immediately. And if you walk to a different table mid-meal, the couriers find you.

The food court is the internet. The friends are streams. The cashier is the protocol. The kitchen is the server. The shared tray is the TCP connection. The independent couriers are the QUIC streams. It is a stretched metaphor but it is sometimes the one that lands.

## What Even Is QUIC?

### A name and a sales pitch

QUIC stands for **Quick UDP Internet Connections**. The name is half-marketing. Some people pronounce it "quick" (which is what the inventors meant). Other people say each letter (Q-U-I-C). Both are fine. Engineers will know what you mean.

If you imagine TCP — the old trusty pipe-style protocol that the whole internet has used for forty years — and then you imagine TCP got an upgrade where it was rewritten from scratch with everything we have learned in those forty years baked in, that is roughly QUIC. Same fundamental purpose: get bytes from one computer to another, in order, reliably, even if some packets get lost along the way. But every detail of how that gets done is different.

QUIC was started inside Google around 2012. Google had millions of users on slow mobile networks and was sick of how slow TCP was to start up. They built QUIC inside Chrome and YouTube as an experiment. It worked. Other companies and the IETF (the standards body that runs the internet) picked it up, polished it, and turned it into a real standard. The first real RFCs landed in 2021. Today QUIC is everywhere.

### The four big things QUIC gives you

1. **Reliability over UDP.** UDP is normally a "fire and forget" protocol. You send a UDP packet and there is no guarantee anybody received it. QUIC adds reliability on top: every packet has a number, every packet is acknowledged, and lost packets are retransmitted. The reliability is **per-stream**, not per-connection. That fixes head-of-line blocking.

2. **Built-in encryption.** Every QUIC packet (after the very first handshake bytes) is encrypted with TLS 1.3. There is no "non-secure mode" for QUIC. Encryption is mandatory. This is a security win and also a performance win — encryption and connection setup happen together instead of one after the other.

3. **Much faster handshake.** Setting up a TCP+TLS 1.2 connection took three round trips. Setting up a TCP+TLS 1.3 connection took two. QUIC does it in one. And if you have talked to the same server before, QUIC can do **0-RTT** — first request goes along with the handshake.

4. **Connection migration.** Your phone is on Wi-Fi at home. You walk out the door and your phone switches to cellular. Your IP address changes. With TCP, every connection breaks the moment your IP changes — TCP identifies a connection by the four-tuple (your IP, your port, server IP, server port). Change one and the connection is gone. QUIC identifies connections by a **connection ID** that is independent of IP, so you can keep the same connection alive across network changes.

### The RFCs that define QUIC

Standards on the internet are written down in documents called RFCs (Request For Comments). They are free to read. The QUIC standard is split across several:

- **RFC 9000** — the main QUIC transport spec. This is the big one. It defines packets, frames, streams, connection IDs, version negotiation, the whole thing.
- **RFC 9001** — how TLS 1.3 is integrated into QUIC. Tells you how the handshake works, what gets encrypted with what key, when keys rotate.
- **RFC 9002** — how QUIC detects lost packets and how it manages congestion. CUBIC is the default. BBR is an option. Implementations can plug in others.
- **RFC 9221** — an extension that lets you send unreliable datagrams over QUIC, for things like real-time games and voice.
- **RFC 9114** — HTTP/3. The layer on top of QUIC that carries actual web pages.
- **RFC 9204** — QPACK. Header compression for HTTP/3 (replaces HPACK).
- **RFC 9460** — Service Binding (SVCB and HTTPS DNS records). How clients find out a server supports HTTP/3 in the first place.

You do not need to read these to use QUIC. But it is nice to know they exist, because if you ever see something like "RFC 9000 §17.2.2.1" mentioned in a bug report, that is what people are pointing at.

### QUIC lives in user space, not the kernel

This is a big deal. TCP lives inside the **kernel** of your operating system. Every operating system ships its own TCP. To upgrade TCP you have to upgrade the kernel. That takes years. The internet is full of computers that will never get a kernel upgrade.

QUIC lives in **user space**, which means it is just regular application code. Each application can ship its own QUIC. Chrome ships its own. Firefox ships its own. nginx ships its own. Cloudflare's `quiche` is one. Microsoft's `msquic` is one. Mozilla's `neqo` is one. Cloudflare can update their QUIC tomorrow without anyone changing their kernel. This is why QUIC can evolve fast. It also means there are many implementations, which is good for diversity but bad if any of them have bugs.

```
Where TCP lives                   Where QUIC lives
===============                   ================
   APP                                APP
    | (system call)                    | (function call)
    v                                  v
  KERNEL: TCP                        QUIC LIBRARY (user space)
    |                                  |
    v                                  v (system call)
  NIC <-> wire                       KERNEL: UDP
                                       |
                                       v
                                     NIC <-> wire
```

The kernel still handles UDP for QUIC, but UDP is so simple that everybody's UDP works fine. The complicated stuff — reliability, encryption, congestion control — moves into the application.

### Why didn't they just fix TCP?

Good question. The honest answer is that fixing TCP is almost impossible. Here is why.

TCP is implemented in every operating-system kernel on every device on the internet. To change TCP, you have to ship a new kernel to every device. Phones get kernel updates eventually. Laptops get kernel updates eventually. Routers, smart TVs, factory equipment, satellites, IoT gadgets, twenty-year-old industrial control systems — these get a kernel update never. So even if every kernel developer agreed on a TCP change tomorrow, it would take a decade for the change to be common enough to rely on.

And it is worse than that. The middle of the internet is full of middleboxes — firewalls, load balancers, NAT devices, DPI systems, transparent proxies — that look at TCP packets and make decisions based on what they see. If you change a TCP bit, those middleboxes might drop your packet, modify it, or panic and reboot themselves. This phenomenon is called **protocol ossification**. The protocol gets locked in concrete by the things that depend on its current shape, and you literally cannot change it without breaking the network.

QUIC dodges all of this by being on UDP (which middleboxes mostly leave alone, because UDP packets don't have anything interesting to mess with) and by encrypting almost every byte (so middleboxes can't see anything to ossify). Future QUIC versions can change anything they want and nothing in the middle of the network will notice.

It is a clever bit of judo. Instead of trying to upgrade TCP, the inventors of QUIC built a "TCP-shaped thing" inside an encrypted UDP wrapper, beyond the reach of the ossified middle of the internet.

## Head-of-Line Blocking — the HTTP/2 Problem

This is the single most important thing to understand if you want to know why HTTP/3 was invented. We are going to walk through it slowly.

### What HTTP/2 promised

HTTP/2 said: you can send many streams over one connection, and the streams will not block each other. The browser asks for ten things at once. The server starts sending all ten things back, with the frames mixed together. Each stream makes progress on its own.

That is mostly true. But there is a catch.

### What TCP guarantees

TCP is the layer underneath HTTP/2. TCP's whole job is to deliver bytes **in order** and **without loss**. If a packet on the wire gets dropped (which happens all the time on real networks), TCP at the receiving end notices, asks for a retransmission, waits for it, and only then hands the bytes up to the application — **in order**.

TCP does not know anything about streams. To TCP, everything is one giant byte stream. If packet 47 is missing, TCP holds back packets 48, 49, 50, 51 etc. until packet 47 arrives. Even if packets 48–51 are perfectly fine and ready to go.

### What that does to HTTP/2

Now imagine HTTP/2 has ten streams running. The frames for those streams get jumbled together and sent as TCP segments. Suppose one TCP segment carrying frames for stream 3 gets lost. TCP at the receiver has to wait for the retransmission. While it waits, **all** the bytes after the lost segment are stuck — including frames for streams 1, 2, 4, 5, 6, 7, 8, 9, 10, which had nothing to do with stream 3.

So the application (HTTP/2) sees all ten streams stall, even though only stream 3 had a problem. That is **head-of-line blocking at the transport layer**. The "head of the line" is the lost packet, and it is "blocking" everything behind it.

```
HTTP/2 over TCP — head-of-line blocking
=======================================
Wire: [S1][S3][S5][LOST][S5][S3][S1][S2]...
                  ^ packet dropped on S3

TCP: "I can't deliver anything past the lost packet
      until I get the retransmission."

HTTP/2 streams 1, 2, 4, 5 all pause. They had no problem.
But TCP can't tell, so they wait.
```

This is mostly invisible on a fast network with no packet loss. But on a flaky mobile connection, head-of-line blocking shows up constantly. Your videos stutter. Your pages hang. The famous case is "the 0.1% packet loss link where HTTP/2 is slower than HTTP/1.1." Because HTTP/1.1 used six separate connections, a loss on one connection only stalled one connection. HTTP/2 collapsed everything to one connection, so a single loss stalled everything.

### How QUIC fixes it

QUIC does reliability **per stream, not per connection**. Each stream has its own packet numbering and its own retransmission logic. If a packet on stream 3 is lost, only stream 3 waits. Streams 1, 2, 4, 5, 6, 7, 8, 9, 10 keep flowing. The application gets whatever data is ready to be delivered, in stream order, without waiting on unrelated streams.

```
HTTP/3 over QUIC — no head-of-line blocking
===========================================
Wire: [S1][S3][S5][LOST-S3][S5][S3-retry][S1][S2]
                  ^ S3 packet dropped

QUIC: stream 3 waits for its own retransmission.
      streams 1, 2, 4, 5, 6, 7, 8, 9, 10 keep flowing.

HTTP/3 sees: "stream 3 paused, the others are fine."
```

Same wire, same loss, very different behavior at the application. That is the magic.

There is one more subtle layer to this. Even in HTTP/3 + QUIC, **HTTP/3 itself** can have a kind of head-of-line blocking in its header compression, because QPACK references a dynamic table that has to be kept in sync. The QPACK people thought hard about this and gave you knobs to control how aggressive the dynamic-table use is. Default settings are sensible. But it is worth knowing the phrase "QPACK head-of-line blocking" exists, in case you ever see it in a bug report.

### When does head-of-line blocking actually hurt?

In a lab on a fiber link with zero packet loss, you will not see head-of-line blocking. Every packet arrives in order. TCP never has to wait. HTTP/2 looks great. HTTP/3 also looks great. They look about the same.

The trouble starts on real networks. Mobile networks routinely have 1% packet loss and bursts of 5%–10% loss in bad conditions (subway, elevator, basement, edge of cell coverage). Wi-Fi at a busy conference has loss. Satellite has loss. Long-haul international links have loss. Anything where a fiber gets bumped, a router buffer overflows, a wireless collision happens, or a tiny optical fault flips a bit causing CRC failure — that is loss.

When loss happens, HTTP/2-over-TCP stalls **everything** for one round trip waiting for the retransmission. That round trip might be hundreds of milliseconds. With a busy page that has 200 streams in flight, that one stall is felt across all 200. The user sees a hiccup. Their video buffers. Their game lags.

HTTP/3-over-QUIC stalls **only the affected stream**. The other 199 keep flowing. The user sees nothing.

Real-world measurements at Cloudflare and Google have shown HTTP/3 wins are most dramatic at the **tail** of the distribution. The median page load might be 2% faster with HTTP/3. The 99th-percentile page load might be 30% faster. The top 1% of users — the ones with the worst networks — get the most dramatic improvement. Those are usually the users on phones, on slow networks, in places where the rich-country-default of "fast wired internet" does not apply.

This is one reason HTTP/3 deployment was driven so hard by mobile-first companies. The improvement at the long tail of bad networks is precisely where their users live.

## The QUIC Connection Setup

Setting up a connection is the part of any conversation where two computers say hi to each other and agree on the rules. It is also the part where most of the time gets wasted, because each "hi" takes a round trip across the internet. A round trip is the time for a packet to go to the server and come back. On a typical broadband link this is somewhere between 10 ms and 100 ms. On a satellite link it is 600 ms. On a mobile phone link in a basement it can be a second.

### TCP + TLS 1.2 — three round trips

For comparison, here is the old-school way. You want to fetch `https://example.com/`. The full handshake is:

```
TCP + TLS 1.2 handshake (3 round trips total)
=============================================

         CLIENT                          SERVER
           |                                |
RTT 1:     |---- SYN ------------------>    |
           |                                |
           |    <------------- SYN-ACK -----|
           |                                |
           |---- ACK ------------------>    |
           |                                |
                  TCP is now open.
                  Now do TLS 1.2.
           |                                |
RTT 2:     |---- ClientHello ---------->    |
           |                                |
           |    <----- ServerHello + cert --|
           |    <----- ServerKeyExchange ---|
           |    <----- ServerHelloDone -----|
           |                                |
           |---- ClientKeyExchange ---->    |
           |---- ChangeCipherSpec ----->    |
           |---- Finished ------------->    |
           |                                |
RTT 3:     |    <----- ChangeCipherSpec ----|
           |    <----- Finished ------------|
           |                                |
                  TLS is now open.
                  NOW you can send your HTTP request.
```

Three round trips just to be ready to talk. If your round trip is 100 ms, that is 300 ms of doing nothing before the first byte of your actual request. On a satellite link, almost two seconds.

### TCP + TLS 1.3 — two round trips

TLS 1.3 fixed some of this:

```
TCP + TLS 1.3 handshake (2 round trips total)
=============================================

RTT 1:     SYN -> ; <- SYN-ACK ; ACK ->
RTT 2:     ClientHello -> ; <- ServerHello + Finished ; Finished ->
           NOW you can send.
```

Better. Still two round trips.

### QUIC — one round trip

QUIC merges TCP and TLS into one handshake. There is no separate "open the connection" phase before the "negotiate encryption" phase. They happen together.

```
QUIC handshake (1 round trip total)
===================================

         CLIENT                          SERVER
           |                                |
RTT 1:     |---- Initial (CRYPTO: ClientHello,
           |       transport params,
           |       chosen versions) ---->   |
           |                                |
           |    <----- Initial (CRYPTO: ServerHello)
           |    <----- Handshake (CRYPTO: EncryptedExtensions,
           |              Certificate, CertificateVerify,
           |              Finished) --------|
           |                                |
           |---- Handshake (CRYPTO: Finished)
           |---- 1-RTT (your first HTTP/3 request!) -->
           |                                |
                       Done.
```

One round trip from "let's connect" to "I'm sending my actual request." On a 100 ms link, you save 200 ms. On a satellite link, you save more than a second. For mobile users on flaky connections, this is the difference between "the page snaps in" and "the page is visibly slow."

### QUIC 0-RTT — zero round trips

If you have talked to this server before recently, QUIC remembers. Specifically, the client kept a little ticket from the previous conversation. On the next connection, the client can send the very first packet **with both the handshake AND the actual HTTP/3 request bundled together**. Zero round trips before data flows. The server may not respond to the request until it has finished the handshake on its side, but the request is in flight from byte zero.

```
QUIC 0-RTT handshake (resumption — first byte is data!)
=======================================================

         CLIENT                          SERVER
           |                                |
T = 0:     |---- Initial (CRYPTO + ticket)
           |---- 0-RTT (HTTP/3 request) -> |
           |                                |
           |    <----- Initial (CRYPTO)
           |    <----- Handshake (CRYPTO + Finished)
           |    <----- 1-RTT (HTTP/3 response!) -|
           |                                |
                  ZERO round trips before data.
```

This sounds magical. There is a catch. **0-RTT data can be replayed**. If somebody captures your 0-RTT packet and replays it later, the server cannot tell the difference. So 0-RTT is only used for **idempotent** requests — requests that are safe to repeat, like `GET /index.html`. You should never put `POST /transfer-money` in a 0-RTT packet. The client and server have anti-replay logic, but it is not perfect, and protocols like HTTP/3 explicitly forbid using 0-RTT for non-idempotent methods.

### Visualizing the savings

```
Time-to-first-byte (one-way, before data flows)
================================================

  HTTP/1.1 plaintext (TCP only):     1 RTT   |#
  HTTPS / TLS 1.2 / HTTP/2 (TCP):    3 RTT   |#########
  HTTPS / TLS 1.3 / HTTP/2 (TCP):    2 RTT   |######
  HTTP/3 over QUIC, fresh:           1 RTT   |###
  HTTP/3 over QUIC, 0-RTT resume:    0 RTT   |
```

You can see why mobile companies and big web companies care so much about this. On a flaky link with 200 ms RTT, the difference between 0 RTT and 3 RTT is more than half a second, every single time you open a new connection.

### What gets exchanged in the QUIC handshake?

A bit more detail for the curious. The QUIC handshake carries a lot of information all at once, in a small number of packets:

- **Version negotiation** — the two sides confirm they speak the same QUIC version. With QUIC v1 dominant, this is mostly a quick "v1, v1, agreed."
- **Connection IDs** — both sides choose CIDs for the new connection.
- **TLS 1.3 ClientHello** — inside a CRYPTO frame, the client offers cipher suites, key shares, ALPN protocols (`h3`, `hq-29`, etc.), and SNI.
- **TLS 1.3 ServerHello and Finished** — server picks the cipher, finishes its key exchange, sends its certificate, signs a CertificateVerify, and finishes.
- **Transport parameters** — both sides advertise their tuning knobs: max idle timeout, initial max data, initial max stream data, max UDP payload size, ack delay exponent, active connection ID limit, etc. These are sent inside the TLS extension `quic_transport_parameters` (codepoint 0x39) so they get protected by the same handshake authentication that protects everything else.
- **Optional retry token** — if the server demanded address validation, the client echoes the retry token back.
- **Optional 0-RTT** — if the client has a session ticket, it can include 0-RTT data in the first flight.

All of this fits into one round trip. That is the result of fifteen years of careful design.

### Why packets are padded to 1200 bytes

You may notice in tcpdump that the very first QUIC packet from the client is exactly 1200 bytes. That is not a coincidence. The QUIC spec mandates that the client's first Initial packet be padded to at least 1200 bytes. The reason is to prevent **amplification attacks**.

Without padding, a small handshake-start packet from a spoofed source IP could be answered with a much larger handshake response. An attacker could spoof the victim's IP, send tiny QUIC opens to many servers, and have those servers blast big responses at the victim. Padding the first packet to 1200 bytes ensures the server's response is roughly the same size — no amplification. The 1200 also matches the minimum reliable IPv6 MTU and is a safe size for almost any path.

You can see this in `tcpdump`: the very first UDP packet of a QUIC handshake is `length 1232` (1200 bytes of QUIC plus 32 of UDP/IPv6 headers, give or take). After the handshake, packets get smaller again.

## Connection Migration

This is one of QUIC's most magical features. To explain why it is magical, we have to look at what TCP does when your network changes.

### How TCP identifies a connection

TCP identifies a connection by four numbers, called the **four-tuple**:

- your IP address
- your port number
- server IP address
- server port number

If any of those four change, TCP treats it as a brand-new (and therefore unauthorized) connection. The kernel sends a RST and tears it down. Your existing TCP socket dies.

### When does the four-tuple change?

All the time, especially on mobile devices.

- You walk out of the house. Your phone leaves Wi-Fi and switches to LTE. Your IP changed. All your TCP connections die.
- You unplug your laptop's ethernet cable and switch to Wi-Fi. Your IP changed. Connections die.
- You move from one Wi-Fi access point to another (different floor of an office). Some Wi-Fi setups will give you a new IP. Connections die.
- A NAT box in front of you decides to remap your port (a thing called a **NAT rebinding**). Your port changed without you doing anything. Connections die.

You may not have noticed because applications usually re-establish connections quickly. But every "die and reconnect" costs you a fresh handshake, an interruption, lost in-flight data, and probably a re-login somewhere.

### How QUIC identifies a connection

QUIC adds a layer of indirection. Every QUIC connection has a **Connection ID (CID)**. The CID is a random number agreed on during the handshake. Every QUIC packet carries the CID in its header. The receiving side looks up the connection by CID, not by four-tuple.

When your IP changes, your packets keep going to the same server (server's IP didn't change), and they still carry the same CID. The server looks at the CID and says "ah yes, this is still my friend, but coming from a different IP now. Let me verify and continue."

The "verify" step is called **path validation**. The server sends a small challenge packet to the new IP. If the client responds correctly, the server is satisfied that this is really the same client and the connection migrates. The connection state — keys, stream data, congestion window, everything — moves to the new path. No new handshake. No interruption you can see.

```
QUIC connection migration
=========================

Time 0:   Client (Wi-Fi, IP 192.0.2.10) <-----> Server
          CID=ABCD1234

Client switches to cellular:

Time 1:   Client (LTE, IP 198.51.100.42) ----> Server
          packet has CID=ABCD1234

Server:   "I see CID ABCD1234 from a new IP. Let me check."
          ----> path challenge to 198.51.100.42 ----->
          <---- correct response, signed with the right key

Server:   "OK, this is the same client. Keep going."

Connection state preserved. No handshake. No re-login.
The client may not even notice.
```

Connection migration is huge for mobile applications. A user on a video call who walks from the kitchen to the back yard losing Wi-Fi and gaining cellular: with TCP, the call hiccups or drops. With QUIC, the call keeps going.

### Why was this so hard for TCP?

TCP cannot do connection migration because its identity is the four-tuple. The four-tuple is baked into how the kernel finds the right socket when a packet arrives. The kernel uses the four-tuple as a hash-table key to figure out which application the packet belongs to. Change one element and the lookup fails — or worse, hits the wrong socket and gets rejected.

People have proposed extensions to TCP to fix this — Multipath TCP (MPTCP), TCP migrate options, etc. — but each proposal has run into deployment problems. Middleboxes in the network see the new options and either drop the packet or strip the option, breaking the connection. So MPTCP works in environments where you control both endpoints and the path (Apple uses it for some iCloud traffic), but it does not work as a generic internet protocol.

QUIC sidesteps the problem entirely. By moving connection identity into the cryptographic part of the protocol, and by hiding everything from middleboxes via encryption, QUIC can move connections wherever it likes and the network is none the wiser.

### The CID actually rotates

To prevent the network from tracking you across migrations, QUIC clients and servers can keep a pool of "spare" CIDs and rotate which one they use. So even if a passive observer sees you on Wi-Fi with CID ABCD and on cellular with CID WXYZ, they cannot tell those are the same connection without breaking the encryption.

There are also things called **Source Connection ID** and **Destination Connection ID**, where each side picks the CID it wants the other side to use when sending packets to it. It sounds confusing. The short version: the spec keeps things flexible so that load balancers, NAT boxes, and migration can all coexist.

## QUIC Streams

A **stream** is a logical flow of bytes inside a QUIC connection. You can have lots of streams in one connection. Each stream is independent — its own data, its own flow control, its own reliability.

### Bidirectional and unidirectional

QUIC has two flavors of stream:

- **Bidirectional streams** — both sides can send. Like a phone call. HTTP/3 uses these for normal request/response: client sends the request on the stream, server sends the response back on the same stream.
- **Unidirectional streams** — only one side sends. Useful for control messages. HTTP/3 uses one of these for SETTINGS and another for QPACK encoder/decoder messages.

### Stream IDs

Every stream has a number called the **Stream ID**. The stream ID is a 62-bit integer. The lowest two bits of the stream ID encode two things:

- bit 0: who initiated this stream? 0 = client, 1 = server
- bit 1: is this stream bidirectional or unidirectional? 0 = bidirectional, 1 = unidirectional

So:

```
Stream ID  binary low 2 bits  meaning
=========  =================  =============================
0          00                 client-initiated, bidirectional
1          01                 server-initiated, bidirectional
2          10                 client-initiated, unidirectional
3          11                 server-initiated, unidirectional
4          00                 client-initiated, bidirectional
5          01                 server-initiated, bidirectional
6          10                 client-initiated, unidirectional
7          11                 server-initiated, unidirectional
...
```

Each side picks new IDs in order from its own pool. This way nobody ever clashes.

### Flow control

Each stream has a buffer at the receiving end. If the receiver is slow at consuming bytes, the buffer fills up. QUIC has **per-stream flow control** so that one slow stream cannot eat all the buffer space and starve the others. There is also **per-connection flow control** for the total bytes across all streams.

Flow control limits are advertised with **MAX_DATA** (per connection) and **MAX_STREAM_DATA** (per stream) frames. The sender stops when it hits the limit and waits for the receiver to advertise more. This is similar to TCP's window, but split per stream so a clogged stream cannot stall the rest.

### Stream lifecycle

A stream starts when somebody sends data on it. It ends when both sides have signaled they are done. A side can send a **RESET_STREAM** frame to abandon a stream early (e.g., the user closed the tab, no point downloading the rest of the JPEG). The other side can send a **STOP_SENDING** frame to say "please stop sending on this stream, I'm not interested."

```
Normal stream lifecycle
=======================

  open  ---->   data, data, data  ---->   FIN (last byte has the fin flag)
                                          stream half-closed
        <---- data, data, data <----      FIN
                                          stream fully closed

Abandoned stream
================

  open  ---->   data, data, data
        <----   STOP_SENDING (please stop)
        ---->   RESET_STREAM (ok, abandoning)
```

### Stream priority

HTTP/3 has its own way of expressing priority between streams via PRIORITY_UPDATE frames and an "Extensible Prioritization Scheme" (RFC 9218). Each stream gets two attributes: **urgency** (0 to 7, low number = more urgent) and **incremental** (whether it can be progressively rendered). Browsers use this to tell servers "the HTML is most urgent, the CSS is next, the visible images come after that, the fonts can wait." Servers honor it best-effort.

This is a big simplification compared to HTTP/2's tree-structured priority, which was so complex that few servers got it right. The HTTP/3 scheme is small, simple, and easier to implement consistently.

### How many streams can there be?

A LOT. The stream ID is 62 bits, so 4.6 quintillion possible streams per direction. In practice both sides advertise a `initial_max_streams_bidi` and `initial_max_streams_uni` transport parameter that limits how many streams the peer may initiate concurrently. Typical values are 100 to 1000. As streams close, the peer can advertise a new higher cap with `MAX_STREAMS` frames, freeing up room for more.

This means a single QUIC connection can carry hundreds of concurrent HTTP/3 requests in flight, vastly more than HTTP/1.1's six and on the same order as HTTP/2's typical 100 concurrent streams.

## TLS in QUIC

In the old world, you ran TCP, then on top of TCP you ran TLS, then on top of TLS you ran HTTP. They were three separate layers. In QUIC, they are not separate. TLS 1.3 is **baked in**. The TLS handshake bytes ride inside QUIC packets. The keys derived from the handshake are used to encrypt later QUIC packets. The two handshakes are one handshake.

### Why bake it in?

A few reasons:

- **Speed.** A merged handshake takes one round trip instead of two or three.
- **Security.** Almost everything in a QUIC packet is encrypted, including bits that TCP traditionally left in plaintext (sequence numbers, flags, etc.). This makes it much harder for middleboxes to peek inside or to mess with packets.
- **Evolvability.** Because middleboxes cannot peek inside, they cannot ossify the protocol the way they did with TCP. Anyone trying to be clever with QUIC packet bits will find them encrypted and have to leave them alone.

### Encryption levels

QUIC has four "encryption levels," used at different stages of the handshake. Each level has its own set of keys.

```
QUIC encryption levels timeline
===============================

  T = 0          T = 1 RTT          T = many RTT
   |                |                  |
  Initial -------> Handshake -------> 1-RTT
   |                |                  |
  also: 0-RTT (sent by client at T=0 if resuming)

  Initial:     keys derived from a public salt + connection ID.
               Used for ClientHello / ServerHello CRYPTO frames.
               Anyone can derive these keys (so they protect
               against off-path attackers, not eavesdroppers).

  Handshake:   keys derived from the Diffie-Hellman exchange.
               Used for the rest of the TLS handshake messages.

  0-RTT:       keys derived from a previous session ticket.
               Used by the client to send early data (replayable).

  1-RTT:       fully forward-secure keys after the handshake.
               Used for all real traffic. Keys can be rotated
               periodically with KEY_UPDATE.
```

Each level has its own packet number space. Each level has its own keys. The four levels exist because, at different stages of the handshake, the two sides have different amounts of cryptographic material to work with.

### Key rotation

QUIC supports rotating keys mid-connection without interrupting traffic. Either side can decide to rotate. After rotation, packets are encrypted with the new key. This is mostly invisible to the application.

### What this means for Wireshark

If you point Wireshark at a QUIC packet capture, you will see a small unencrypted header (the public part) and the rest is encrypted bytes. To decode the contents, Wireshark needs the TLS keys. Browsers can write the keys to a file specified by the **SSLKEYLOGFILE** environment variable. Point Wireshark at that file and it can decrypt your QUIC sessions on the fly. We will do this in the Hands-On.

### Header protection — the layer almost nobody knows about

Inside QUIC there is a clever bit called **header protection**. The packet number and a few flag bits in every QUIC packet header are encrypted with a separate key derived from the same TLS handshake. Even if you can see the long-header fields (version, CIDs), the packet number is scrambled.

Why? Because the packet number is reused for nonces in the AEAD encryption, and if an attacker could see and confirm packet numbers they could correlate streams and infer traffic patterns. By hiding the packet number in the header itself, QUIC makes traffic analysis harder.

Header protection adds tiny CPU cost — one more AES-CTR or ChaCha20 invocation per packet — but it pays off in security. It is a thing you do not need to worry about as a user, but if you ever look at QUIC dissection in Wireshark and see "header protection mask" you will know what it is.

### Post-quantum is coming

TLS 1.3 in QUIC currently uses classical Diffie-Hellman key exchange, typically X25519 or P-256. These will not survive a large quantum computer. The IETF has been adding post-quantum hybrid key exchange to TLS 1.3 (Kyber + X25519, etc.). Cloudflare, Google, and Apple have started rolling out experiments with post-quantum hybrid handshakes. Because QUIC's handshake is TLS, QUIC inherits these upgrades automatically. By the time large quantum computers arrive (probably the 2030s), we should already be on post-quantum hybrid handshakes everywhere.

## HTTP/3 Frames and Streams

HTTP/3 sits on top of QUIC. QUIC gives HTTP/3 streams. HTTP/3 puts its own structure inside each stream.

### How HTTP/3 maps onto QUIC streams

For a normal request/response, HTTP/3 uses **one bidirectional QUIC stream per request/response pair**. The client opens the stream, writes the HEADERS and DATA frames for the request, the server writes HEADERS and DATA frames for the response on the same stream, both sides FIN, done. Next request, new stream.

HTTP/3 also uses a few special unidirectional streams for control:

- **Control stream** — both sides have one. Carries SETTINGS, GOAWAY, MAX_PUSH_ID, and other control frames.
- **QPACK encoder stream** — for sending dynamic-table updates.
- **QPACK decoder stream** — for sending acknowledgments back to the encoder.
- **Push streams** — used by server push (rarely supported in practice).

### HTTP/3 frame types

Inside an HTTP/3 stream, bytes are organized into frames:

- **HEADERS** — the request headers, encoded with QPACK.
- **DATA** — the body bytes.
- **SETTINGS** — exchanged once at the start on the control stream.
- **GOAWAY** — "I am about to stop accepting new requests, please don't send any more."
- **MAX_PUSH_ID** — limits how many server pushes the client will accept.
- **CANCEL_PUSH** — cancels a push.
- **PUSH_PROMISE** — server announcing a push.

A normal request looks like:

```
Stream N (request/response)
==========================
  client sends:  HEADERS (method=GET, :path=/, :authority=example.com, ...)
                 (no DATA for a GET; DATA frames for a POST body)
                 FIN
  server sends:  HEADERS (status=200, content-type=..., ...)
                 DATA <bytes>
                 DATA <bytes>
                 ...
                 FIN
```

### QPACK — header compression for QUIC

HTTP/2 used a header-compression scheme called HPACK. HPACK works fine over TCP because TCP guarantees in-order delivery. But QUIC streams are independent and can be reordered. So HPACK does not work as-is on QUIC.

QPACK is the QUIC-friendly successor. It separates header compression into:

- A **static table** of common headers (`:method: GET`, `content-type: application/json`, ...). The full list is in RFC 9204.
- A **dynamic table** that the encoder maintains and updates. Updates flow on a separate QPACK encoder stream.
- An acknowledgment flow on a QPACK decoder stream.

The trick is: the request stream can reference dynamic-table entries, but if the encoder-stream update has not arrived yet, the request stream might have to pause. This is the QPACK head-of-line blocking we mentioned earlier. QPACK lets you trade off compression ratio against blocking risk via `SETTINGS_QPACK_BLOCKED_STREAMS` and `SETTINGS_QPACK_MAX_TABLE_CAPACITY`. Most implementations pick sensible defaults.

You do not normally have to think about this. Just know the words exist.

### HTTP/3 vs HTTP/2 frame mapping

```
HTTP/2 frame  ->  HTTP/3 frame    notes
============      ============    =====
HEADERS           HEADERS         in HTTP/3, no priority data here
DATA              DATA            same idea
SETTINGS          SETTINGS        on control stream now
PRIORITY          (gone)          replaced by PRIORITY_UPDATE / extensible scheme
RST_STREAM        (gone, sort of) handled by QUIC's RESET_STREAM at transport layer
PING              (gone)          QUIC has its own PING frame
GOAWAY            GOAWAY          same idea, slightly different semantics
WINDOW_UPDATE     (gone)          handled by QUIC flow control
PUSH_PROMISE      PUSH_PROMISE    works the same, mostly unused
                  MAX_PUSH_ID     new — limits push count
                  CANCEL_PUSH     new
```

## ALPN and Discovery

When your browser opens a connection to `https://example.com/`, how does it know whether the server speaks HTTP/3?

### ALPN — picking the protocol after the connection opens

ALPN stands for "Application-Layer Protocol Negotiation." It is a tiny extension to TLS where the client lists the protocols it can speak (`h3`, `h2`, `http/1.1`) and the server picks one. The picked one is used.

For HTTP/2 and HTTP/1.1 over TLS, ALPN is enough — you open TCP, do TLS, ALPN tells you whether to do HTTP/2 or HTTP/1.1. For HTTP/3 over QUIC, ALPN is also used to confirm `h3` support during the handshake. But there is a chicken-and-egg problem: HTTP/3 uses UDP, not TCP, so the browser has to know **before opening any connection** that the server speaks UDP-based QUIC.

### Alt-Svc header — "next time, try QUIC"

The first solution was an HTTP response header called **Alt-Svc** (Alternative Services). When you make an HTTPS request to a server over HTTP/2 or HTTP/1.1, the server can respond with:

```
Alt-Svc: h3=":443"; ma=86400
```

This means: "I also support HTTP/3 on UDP port 443 for the next 86400 seconds (24 hours). Feel free to try that next time." The browser caches this. On the next request to the same origin, it tries QUIC first. If QUIC works, it uses QUIC and stops using TCP. If QUIC fails (firewall, packet drops, whatever), it falls back to TCP/HTTP/2.

This means you always pay one TCP+HTTP/2 round trip on the very first visit. Subsequent visits get HTTP/3.

### HTTPS RR / SVCB DNS records — discover before the connection

A newer mechanism uses DNS. Two new record types — **HTTPS** and **SVCB** — let a domain advertise its supported protocols and parameters in DNS itself. The browser does a DNS query, sees `h3` in the alpn parameter, and tries QUIC on the very first request.

```
$ dig +short -t HTTPS cloudflare.com
1 . alpn="h3,h2" ipv4hint=104.16.132.229 ipv6hint=2606:4700::6810:84e5
```

This eliminates the "first visit always uses TCP" warm-up. It is becoming standard but is still rolling out. Some resolvers strip the new record types. Some networks block the queries. Browsers fall back gracefully if the DNS lookup fails.

### Fallback

In all cases, browsers fall back. If QUIC is blocked, dropped, or broken, they retry over TCP. The user sees a working page but with HTTP/2 instead of HTTP/3. This is called "happy eyeballs for QUIC" in some implementations. It is what makes QUIC safe to deploy in the wild.

## Congestion Control in QUIC

Congestion control is the part of any reliable protocol that says "go fast when the network can take it, slow down when the network is dropping packets, ramp back up when things clear." It is what stops one greedy connection from using all the bandwidth and starving everybody else.

### TCP's default — CUBIC

For the last fifteen years or so, the default congestion control on Linux TCP has been **CUBIC**. CUBIC ramps up its congestion window in a cubic curve, which fills bandwidth fast on long fat pipes (lots of bandwidth, lots of latency, e.g. transatlantic links). CUBIC works well in many cases. It is the default for QUIC too.

### BBR

**BBR** stands for "Bottleneck Bandwidth and Round-trip propagation time." It was developed at Google. Instead of treating packet loss as the only signal of congestion (the way CUBIC does), BBR builds a model of the network's actual bandwidth and round-trip time and tries to operate just at the right point — full bandwidth, low queue. In practice BBR is much faster on lossy links (mobile, transcontinental) but can be unfair to other CUBIC flows in some scenarios. Most QUIC implementations let you choose between CUBIC and BBR.

### Pluggable

Because QUIC lives in user space, switching congestion control is just a code path. Cloudflare's quiche, Google's QUIC, msquic, and others all have multiple algorithms available. Some let you pick at compile time, some at runtime. Server operators can run experiments. Developers can plug in research algorithms (Copa, Vegas, Reno, Westwood, etc.) without touching the kernel.

### ECN

Explicit Congestion Notification (ECN) is a way for the network to say "I am about to drop your packet, please slow down" by setting two bits in the IP header instead of actually dropping the packet. QUIC supports ECN. So does TCP, but TCP's ECN deployment is famously fragile. QUIC implementations have a fresher chance to get it right.

### Pacing

Beyond the high-level CC algorithm, modern QUIC implementations also **pace** their packets — spreading them out over time rather than blasting them in a burst. Pacing helps avoid filling buffers and triggering loss. The pacing rate is derived from the congestion window divided by the smoothed round-trip time. With pacing on, a connection of 10 Mbps over 100 ms RTT looks more like a steady stream than a series of bursts.

### Why this matters for HTTP/3 vs HTTP/2

HTTP/2 is bottlenecked by Linux's TCP CC, which is set system-wide and only changeable with root privileges. HTTP/3 is bottlenecked by the application's chosen QUIC CC, which a server operator can switch with a config change. This means HTTP/3 is much more deployable for experiments. If BBR works better for your workload, you can flip to BBR in your QUIC server config. With TCP, you would have to coordinate with the kernel team.

## Common HTTP/3 / QUIC Errors

### 0-RTT replay risk

If you put a non-idempotent request in 0-RTT, an attacker can replay it. Buying coffee with your debit card twice. Don't do this.

```
Wrong:   client sends  POST /transfer-money  in 0-RTT
         attacker captures the packet
         attacker replays it later
         server happily transfers the money again

Right:   client sends  GET /static-asset.js  in 0-RTT
         client sends  POST /transfer-money  AFTER 1-RTT keys established
```

HTTP/3 mandates that POST and certain other methods are not allowed in 0-RTT. Servers should reject them.

### Handshake failures: UDP filtered by middleboxes

Many enterprise networks, hotel Wi-Fi, and corporate firewalls block or rate-limit UDP. QUIC uses UDP. If UDP/443 is blocked, your QUIC handshake never completes. The client times out and falls back to TCP. You will see this in `curl --http3-only`:

```
$ curl --http3-only https://example.com/
curl: (7) Failed to connect to example.com port 443 after 10003 ms: Timeout was reached
```

The fix: the client retries with TCP. The server still works, just slower.

### "QUIC version not supported"

The client offers a QUIC version (e.g. v1, draft-29) and the server offers a different set. The server replies with a version-negotiation packet listing what it does support. The client picks one and retries. If there is no overlap, the connection fails. With QUIC v1 (RFC 9000), this is mostly historical now — almost everybody supports v1.

### "alt-svc cache invalidated"

The browser tried HTTP/3 because Alt-Svc said it would work, but the server stopped responding on UDP. The browser invalidates the Alt-Svc entry, falls back to TCP, and won't try HTTP/3 again until the next valid Alt-Svc header arrives.

### "stream reset"

Either side sent RESET_STREAM. The application sees this as an EPROTO or `RESET_STREAM` error in the QUIC library API. Common reasons: user closed tab, server canceled response, request rejected by application logic.

### "connection close: TRANSPORT_PARAMETER_ERROR"

The other side's transport parameters were missing, malformed, or contradictory. Usually a sign of an implementation bug or version mismatch. The connection is closed immediately.

### Middleboxes that reset connections without sending UDP back

Some firewalls drop QUIC packets silently. The client thinks the network is congested, retransmits, eventually times out. From the server's side, no UDP ever arrived. From the client's side, no UDP ever returned. Both sides are stuck waiting. This is why fallback to TCP is so important.

### DPI / SPI firewalls dropping QUIC handshake

Some deep-packet-inspection systems specifically look for QUIC handshakes and drop them, either because the operator does not understand the protocol or because they cannot inspect the encrypted traffic. The fix is usually "ask the operator to allow UDP/443" or "fall back to TCP."

### Path MTU issues — no fragmentation in QUIC

QUIC packets must fit inside a single UDP datagram, and QUIC explicitly does not allow IP-level fragmentation. The minimum MTU QUIC can use is 1200 bytes. Many networks have a smaller path MTU than the local interface MTU because of tunnels. QUIC includes a Path MTU Discovery (PMTUD) mechanism to find a safe size. If PMTUD picks a size that the path silently drops, the connection stalls. Implementations have to be conservative.

### "encryption_failure" or "decryption_failure"

Keys do not match. Could be a transport parameter mismatch, a botched key update, a packet sent at the wrong encryption level, or genuine corruption. Implementation bug, usually.

### Packet number rollover

QUIC packet numbers are 62 bits but the on-wire encoding is shorter (1–4 bytes). If endpoints get out of sync about how to expand the truncated number into the full one, decryption fails. Almost never happens in practice but the spec spends pages on it.

## Hands-On

Let's see HTTP/3 and QUIC actually working on your computer. You will need:

- a recent `curl` (built with HTTP/3 support; check with `curl -V | grep HTTP3`)
- `tcpdump` or `tshark` to see packets
- `dig` for DNS
- a network connection that does not block UDP/443

Some commands need root/sudo (the packet capture ones). If you do not have a curl with HTTP/3, you can install one — Cloudflare publishes a static binary, or you can build curl with the boringssl + nghttp3 stack. The non-HTTP/3 commands all work without that.

### Inspect a real HTTP/3 site

```
$ curl --http3-only -v https://cloudflare-quic.com/ 2>&1 | head -30
* Host cloudflare-quic.com:443 was resolved.
* IPv6: 2606:4700:7::a29f:8a55
* IPv4: 162.159.138.85
*   Trying [2606:4700:7::a29f:8a55]:443...
* QUIC cipher selection: TLS_AES_128_GCM_SHA256:TLS_AES_256_GCM_SHA384:TLS_CHACHA20_POLY1305_SHA256
* Skipped certificate verification
* Connected to cloudflare-quic.com (2606:4700:7::a29f:8a55) port 443
* using HTTP/3
* [HTTP/3] [0] OPENED stream for https://cloudflare-quic.com/
* [HTTP/3] [0] [:method: GET]
* [HTTP/3] [0] [:scheme: https]
* [HTTP/3] [0] [:authority: cloudflare-quic.com]
* [HTTP/3] [0] [:path: /]
* [HTTP/3] [0] [user-agent: curl/8.6.0]
* [HTTP/3] [0] [accept: */*]
> GET / HTTP/3
> Host: cloudflare-quic.com
> User-Agent: curl/8.6.0
> Accept: */*
>
* Request completely sent off
< HTTP/3 200
< date: ...
< content-type: text/html
< server: cloudflare
```

Notice `using HTTP/3` and the pseudo-headers `:method`, `:scheme`, `:authority`, `:path`. Those are the HTTP/2/3 way of representing what HTTP/1.1 spread across multiple lines.

### Compare against a server that may not support HTTP/3

```
$ curl --http3 -v https://www.google.com 2>&1 | head -20
* Host www.google.com:443 was resolved.
* IPv6: 2607:f8b0:4005:80c::2004
* IPv4: 142.250.190.36
*   Trying [2607:f8b0:4005:80c::2004]:443...
* using HTTP/3
* [HTTP/3] [0] OPENED stream for https://www.google.com/
* [HTTP/3] [0] [:method: GET]
> GET / HTTP/3
< HTTP/3 200
< content-type: text/html; charset=ISO-8859-1
```

Google's frontend speaks HTTP/3. Good.

### Force HTTP/2 for comparison

```
$ curl --http2 -s -o /dev/null -w "%{http_version}\n" https://www.google.com
2

$ curl --http3-only -s -o /dev/null -w "%{http_version}\n" https://www.google.com
3
```

The `%{http_version}` curl variable prints which HTTP version was used. You can see HTTP/2 and HTTP/3 talking to the same server.

### Watch QUIC packets in tcpdump

```
$ sudo tcpdump -i any -n -c 20 udp port 443
tcpdump: data link type LINUX_SLL2
listening on any, link-type LINUX_SLL2 (Linux cooked v2), snapshot length 262144 bytes
14:02:11.123  IP 192.0.2.10.55313 > 162.159.138.85.443: UDP, length 1232
14:02:11.144  IP 162.159.138.85.443 > 192.0.2.10.55313: UDP, length 1252
14:02:11.145  IP 192.0.2.10.55313 > 162.159.138.85.443: UDP, length 75
14:02:11.166  IP 162.159.138.85.443 > 192.0.2.10.55313: UDP, length 184
14:02:11.167  IP 192.0.2.10.55313 > 162.159.138.85.443: UDP, length 75
...
```

Every line is a QUIC packet. `length 1232` is a typical Initial-with-padding QUIC packet, padded up so the server's response can be similarly large without amplification concerns.

### Decode QUIC with tshark

```
$ sudo tshark -i any -Y 'quic' -V 2>&1 | head -50
Capturing on 'any'
Linux cooked capture v2
QUIC IETF
    QUIC Connection information
    [Packet Length: 1232]
    1... .... = Header Form: Long Header (1)
    .1.. .... = Fixed Bit: True
    ..00 .... = Packet Type: Initial (0)
    Version: 1 (0x00000001)
    Destination Connection ID Length: 8
    Destination Connection ID: ab12cd34ef567890
    Source Connection ID Length: 0
    Token Length: 0
    Length: 1217
    Packet Number: 0
    [Decoded Frame: PADDING]
    [Decoded Frame: CRYPTO]
        Frame Type: CRYPTO (0x06)
        Offset: 0
        Length: 512
        Crypto Data: 010001fe...
```

You can see the long header, the QUIC version, the connection IDs, packet number, and even individual frames inside. The CRYPTO frame contains the TLS ClientHello.

### Look up HTTPS DNS records

```
$ dig +short -t HTTPS cloudflare.com
1 . alpn="h3,h2" ipv4hint=104.16.132.229 ipv6hint=2606:4700::6810:84e5
```

That `alpn="h3,h2"` is what tells modern browsers "this server supports HTTP/3, please try it before TCP."

```
$ dig +short -t SVCB cloudflare.com

```

Cloudflare uses HTTPS RR (which is the HTTP-specific subtype of SVCB) rather than raw SVCB, so the SVCB query returns nothing. That is normal.

### Resolve a HTTP/3-only test domain

```
$ getent hosts cloudflare-quic.com
2606:4700:7::a29f:8a55 cloudflare-quic.com
162.159.138.85   cloudflare-quic.com
```

### Probe ALPN with openssl

```
$ openssl s_client -connect cloudflare-quic.com:443 -alpn h3,h2 -tls1_3 < /dev/null 2>&1 | grep -i 'alpn\|protocol'
ALPN protocol: h2
Protocols available: TLSv1.3
```

`openssl s_client` only does TCP, so the ALPN negotiation here picks `h2`. The point is it confirms the server speaks HTTP/2 over TCP as a fallback. The HTTP/3 path goes via QUIC and is not exercised by `openssl s_client`.

### Use nghttp clients

```
$ nghttp -v --http3 https://cloudflare-quic.com/ 2>&1 | head -30
[  0.011] Connected via h3 to https://cloudflare-quic.com:443
[  0.011] HTTP Upgrade success
[  0.012] send HEADERS frame <length=64, flags=0x05, stream_id=0>
          ; END_STREAM | END_HEADERS
          (padlen=0)
          ; Open new stream
          :method: GET
          :path: /
          :scheme: https
          :authority: cloudflare-quic.com
          accept: */*
          accept-encoding: gzip, deflate
          user-agent: nghttp3/1.1.0
[  0.034] recv HEADERS frame <length=...> stream_id=0
          :status: 200
          ...
```

`nghttp` (with HTTP/3 support compiled in) is the gold standard for inspecting HTTP/3.

```
$ nghttp2 -v --no-content-length --get https://www.google.com/ 2>&1 | head -30
[  0.018] Connected via h2 to https://www.google.com:443
[  0.018] HTTP Upgrade success
[  0.018] send HEADERS frame <length=...>
          :method: GET
          :path: /
          :scheme: https
          :authority: www.google.com
[  0.040] recv HEADERS frame <length=...>
          :status: 200
```

Same shape, but HTTP/2 over TCP.

### Try other QUIC implementations

```
$ quiche-client https://cloudflare-quic.com/ 2>&1 | head -30
sending request to https://cloudflare-quic.com/
HTTP/3 200
date: ...
content-type: text/html
server: cloudflare

<html>
...
```

Cloudflare's `quiche-client` is a tiny test client built into the quiche library. If it works, your QUIC stack is healthy.

### Python QUIC with aioquic

```
$ python3 -c "import aioquic; print(aioquic.__version__)"
0.9.25
```

`aioquic` is a Python QUIC implementation, useful for testing and writing custom clients. There is also `httpx` with HTTP/3 support if you install the right extras.

### Look at the kernel UDP table

```
$ cat /proc/net/udp | head -5
   sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode ref pointer drops
  168: 00000000:0044 00000000:0000 07 00000000:00000000 00:00000000 00000000     0        0 12345 2 ffff... 0
  187: 0100007F:1538 00000000:0000 07 00000000:00000000 00:00000000 00000000     0        0 23456 2 ffff... 0
```

These are all UDP sockets the kernel knows about, including any QUIC sockets your applications have open.

```
$ ss -uan | grep ":443" | head -10
UNCONN  0  0  0.0.0.0:443  0.0.0.0:*
UNCONN  0  0  [::]:443     [::]:*
```

`ss -uan` shows UDP sockets. UDP sockets on `:443` are typically QUIC servers (or QUIC clients with a fixed source port).

### Kernel UDP buffer settings

```
$ cat /proc/sys/net/ipv4/udp_rmem_min
4096

$ cat /proc/sys/net/ipv4/udp_wmem_min
4096
```

These are minimum UDP socket buffer sizes. For high-throughput QUIC servers you may want to raise the maximum (`net.core.rmem_max`, `net.core.wmem_max`).

### Check NIC offloads

```
$ ethtool -k eth0 2>&1 | grep -i udp
tx-udp_tnl-segmentation: on
tx-udp_tnl-csum-segmentation: on
udp-fragmentation-offload: off [requested off]
```

Modern NICs offer UDP segmentation offload (USO) which speeds up QUIC servers significantly by letting the NIC slice up large UDP buffers.

### Check what your web server has compiled in

```
$ nginx -V 2>&1 | grep -i quic
configure arguments: --with-http_v3_module --with-quic ...
```

Newer nginx versions (1.25+) ship with experimental HTTP/3 support behind `--with-http_v3_module`. You enable it in config with `listen 443 quic reuseport;` and `add_header Alt-Svc 'h3=":443"; ma=86400';`.

```
$ caddy version
v2.7.6 h1:...
```

Caddy enables HTTP/3 by default since v2. No flag, no config change.

### Look at kernel logs for QUIC mentions

```
$ dmesg | grep -i quic | tail -10
```

This is usually empty unless your kernel has explicit QUIC integration (some new kernels do — there is in-kernel QUIC work in progress in Linux 6.x).

### bpftrace UDP packet count by process

```
$ sudo bpftrace -e 'tracepoint:net:net_dev_start_xmit / args->protocol == 0x86dd / { @[comm] = count(); }'
Attaching 1 probe...
^C

@[chrome]: 4127
@[firefox]: 1980
@[curl]: 12
@[caddy]: 8210
```

This counts IPv6 packets per process. Replace `0x86dd` with `0x0800` for IPv4. You can also filter by UDP port. This is the eBPF angle that the source binder also covered — `cs ramp-up ebpf-eli5` goes deep on that.

### Curl one-liner: HTTP/2 vs HTTP/3 timing

```
$ curl -o /dev/null -s --http2 \
    -w "h2 connect=%{time_connect}s appconn=%{time_appconnect}s starttransfer=%{time_starttransfer}s\n" \
    https://cloudflare-quic.com/
h2 connect=0.022s appconn=0.069s starttransfer=0.108s

$ curl -o /dev/null -s --http3-only \
    -w "h3 connect=%{time_connect}s appconn=%{time_appconnect}s starttransfer=%{time_starttransfer}s\n" \
    https://cloudflare-quic.com/
h3 connect=0.000s appconn=0.044s starttransfer=0.085s
```

`time_connect` is the TCP handshake (zero for QUIC because there is no TCP). `time_appconnect` is the TLS+ALPN ready point. `time_starttransfer` is when the response started arriving. HTTP/3 wins by about 20–30 ms here, which matches the 1-RTT-vs-2-RTT theory.

### See QUIC traffic to a specific destination

```
$ sudo tcpdump -i any -n udp port 443 and host cloudflare-quic.com
```

Now hit `cloudflare-quic.com` in another terminal with curl and watch the packets fly.

### Save QUIC traffic with keys for later decryption

```
$ SSLKEYLOGFILE=$HOME/quic-keys.txt curl --http3-only -o /dev/null https://cloudflare-quic.com/
$ sudo tcpdump -i any -w /tmp/quic.pcap -s0 udp port 443 &
$ # then point Wireshark at /tmp/quic.pcap and configure
$ # Edit > Preferences > Protocols > TLS > (Pre)-Master-Secret log filename = $HOME/quic-keys.txt
```

When you reload the pcap in Wireshark with the keys file pointed at, you can see the decrypted QUIC frames, decrypted HTTP/3 headers, and decrypted body bytes. This is how you debug real QUIC.

## Common Confusions

### "If TCP works, why change?"

TCP works for traffic that does not care about head-of-line blocking, slow handshakes, or network changes. Most desktop traffic from a stable wired connection. But the modern internet is mobile, lossy, latency-sensitive, and full of network changes. TCP shows its age in those conditions. QUIC was designed for those conditions.

### "Is QUIC just UDP?"

No. QUIC uses UDP as a transport, but QUIC adds reliability, ordering, encryption, multiplexed streams, flow control, congestion control, and connection migration on top. UDP gives you almost nothing — QUIC gives you everything TCP+TLS gives you, plus more. Calling QUIC "just UDP" is like calling TCP "just IP."

### "Why does my QUIC connection fail through enterprise networks?"

Enterprise firewalls often block or rate-limit UDP, especially on port 443. Some DPI systems specifically drop QUIC. Symptoms: handshake hangs, then falls back to TCP. Fix: get the firewall operator to allow UDP/443, or accept the fallback.

### "Is HTTP/3 always faster than HTTP/2?"

Usually, especially on lossy networks (mobile, satellite, transcontinental). On a stable wired link with negligible loss, HTTP/2 over TCP can be competitive, especially since browsers spend a lot of time on the same long-lived connection where the handshake cost amortizes. Real-world measurements consistently show HTTP/3 wins on the median, with bigger wins at the tail.

### "Does QUIC need TLS?"

Yes. QUIC v1 (RFC 9000) requires TLS 1.3. There is no plaintext mode. There is no "QUIC with no TLS" option. Every byte after the very early handshake is encrypted. This is a deliberate design choice — it prevents middleboxes from ossifying the protocol the way they did with TCP.

### "Why does Wireshark struggle to decode QUIC?"

Because the packet contents are encrypted. Wireshark can see the long header (version, CIDs, packet numbers in some cases) but the rest is opaque without keys. Set `SSLKEYLOGFILE` in your client and feed that file to Wireshark and you can decode everything.

### "Is HTTP/3 an upgrade you have to do, or does it happen automatically?"

For users: automatic. Browsers support it, servers advertise it, fallback handles failures.
For server operators: opt-in. You enable HTTP/3 in your web server config (nginx, Caddy, Apache mod_http3 patches, HAProxy with QUIC, etc.), open UDP/443 in your firewall, and watch the metrics.

### "Does HTTP/3 break my existing tools?"

Most tools that "speak HTTP" work over HTTP/1.1 and HTTP/2. Many do not yet speak HTTP/3. `curl` does (recent versions). Most browsers do. Many CLI tools (wget, basic Python `requests`) do not yet, though `httpx` and `aiohttp` are catching up. If you have a script that hits an HTTP/3-advertising server, your script will use HTTP/2 over TCP because the server still listens on TCP.

### "What's the relationship between QUIC and gRPC?"

gRPC is normally carried over HTTP/2 over TCP. There is a gRPC-over-HTTP/3 spec, but it is much less common in practice. The framing translates cleanly because HTTP/3 streams behave like HTTP/2 streams from gRPC's point of view.

### "Is HTTP/3 the last HTTP version?"

No. HTTP versioning is not done. HTTP/3 is the version of HTTP-the-application-protocol that runs on QUIC instead of TCP. Future "versions" might evolve QUIC itself (multipath QUIC, post-quantum hybrid TLS, QUIC v2) without bumping HTTP's number. Or there might be an HTTP/4. The IETF has not announced one.

### "What about WebSockets?"

WebSockets traditionally run over HTTP/1.1 with the Upgrade mechanism. There is RFC 8441 for WebSockets over HTTP/2, and RFC 9220 for WebSockets over HTTP/3. Adoption of WebSockets-over-H3 is just starting.

### "Is QUIC slower because encryption is mandatory?"

The cost of encryption is small relative to the cost of round trips. By eliminating one or two round trips, QUIC saves more time than the encryption costs. On modern CPUs with AES hardware acceleration, AES-GCM costs almost nothing. ChaCha20-Poly1305 (the alternative for ARM and older CPUs) is also very fast.

### "Does QUIC work over IPv6?"

Yes. QUIC uses UDP, UDP works over both IPv4 and IPv6, and QUIC has no IPv4-specific assumptions. In fact, IPv6 is often where QUIC shines — many IPv6 paths are cleaner because there are fewer middleboxes.

### "What is HTTP/2 over cleartext (h2c) and is there an HTTP/3 equivalent?"

`h2c` was HTTP/2 directly over TCP without TLS. Some people used it server-to-server inside trusted networks. There is **no** HTTP/3 equivalent. QUIC v1 cannot run without TLS. If you want HTTP/2-style internal traffic without TLS, you stick with `h2c`. If you want HTTP/3, you bring TLS.

### "Should I disable HTTP/3 on my server?"

Probably not. The cost of having it enabled is small (one extra UDP listener, a bit more CPU on handshakes), and clients fall back to TCP gracefully if QUIC is broken on a particular path. The benefit is real for users on bad networks. The major web servers (nginx, Apache, Caddy, HAProxy) and CDNs (Cloudflare, Fastly, Cloudfront, Akamai) all support HTTP/3 now. The deployment success stories are convincing.

### "Does HTTP/3 increase CPU use on the server?"

Yes, somewhat. Encryption is mandatory and there is more state to track per connection. Early QUIC implementations had pathological CPU profiles. Recent versions are much better — within 1.5–2x of the equivalent HTTP/2 server in most measurements. NIC features like UDP segmentation offload (USO) and GRO close the gap further. For very high-throughput servers, the gap matters; for normal websites, it is fine.

### "Why does my CDN log show h3 but my server only sees h2?"

Because your CDN is terminating QUIC at the edge and proxying to your origin over HTTP/2 (or HTTP/1.1). The user-facing connection is HTTP/3. The origin-facing connection is whatever your CDN talks to your server. This is normal and gives you the latency benefits of HTTP/3 without changing your origin.

### "Can I run HTTP/3 over a port other than 443?"

Yes, but no client will discover it without help. Browsers expect HTTPS-on-443 by default. You would have to explicitly point clients at the alternate port via Alt-Svc, HTTPS RR, or a custom URL scheme. For private services this is fine; for public services it is unnecessary friction.

### "What's the difference between QUIC and DTLS?"

DTLS is "TLS but for datagrams" — it provides encryption over UDP without reliability. QUIC provides encryption AND reliability AND multiplexing AND congestion control over UDP. DTLS is a small piece of what QUIC does. WebRTC uses DTLS. QUIC is much bigger.

### "Why is HTTP/3 sometimes called 'HQ' or 'hq'?"

Earlier draft versions used the ALPN identifier `hq-29`, `hq-30`, etc. — short for "HTTP-over-QUIC, draft 29." When the standard finalized, the ALPN became plain `h3`. You may still see `hq-` references in old code or test servers.

### "Can I downgrade an HTTP/3 connection to HTTP/2?"

Not in the middle of a connection. You can refuse to negotiate `h3` and the client falls back to opening a fresh TCP connection and trying `h2` there. There is no in-band downgrade — they are different transports.

## Vocabulary

| Word | Plain English |
|------|---------------|
| HTTP | The rules for moving web pages around between browsers and servers. |
| HTTPS | HTTP with TLS encryption — every byte secure. |
| HTTP/1.0 | The very first widely-deployed HTTP. One request per TCP connection. 1996. |
| HTTP/1.1 | HTTP/1.0 with persistent connections. Still serial inside a connection. 1997. |
| HTTP/2 | Multiplexed HTTP over one TCP connection with binary framing. 2015. |
| HTTP/3 | Multiplexed HTTP over QUIC. 2022, RFC 9114. |
| QUIC | Quick UDP Internet Connections. A reliable, encrypted, multiplexed transport on UDP. |
| TLS | Transport Layer Security. The encryption layer on top of TCP (or inside QUIC). |
| TLS 1.3 | The current version of TLS. Faster handshake, fewer ciphers, better security. |
| ALPN | Application-Layer Protocol Negotiation. A TLS extension that picks `h3` / `h2` / `http/1.1`. |
| h3 | The ALPN identifier for HTTP/3. |
| h2 | The ALPN identifier for HTTP/2. |
| h2c | HTTP/2 over plain TCP without TLS. No HTTP/3 equivalent. |
| SNI | Server Name Indication. The TLS extension that names the host you want, so a server can serve many sites on one IP. |
| ECH | Encrypted Client Hello. Hides the SNI from passive observers. |
| ESNI | Older name for the precursor to ECH. Mostly historical. |
| Frame | A small chunk of HTTP/2 or HTTP/3 protocol data. HEADERS, DATA, etc. |
| Stream | A logical bidirectional or unidirectional channel inside an HTTP/2 or QUIC connection. |
| Push | Server-sent extra resources without the client asking. Rarely used. |
| HEADERS | An HTTP/2 or HTTP/3 frame carrying request or response headers. |
| DATA | An HTTP/2 or HTTP/3 frame carrying body bytes. |
| SETTINGS | An HTTP/2 or HTTP/3 frame announcing per-connection options. |
| GOAWAY | An HTTP/2 or HTTP/3 frame saying "I'm shutting down, don't send new requests." |
| MAX_PUSH_ID | An HTTP/3 frame limiting how many server pushes the client will accept. |
| RESET_STREAM | A QUIC frame abruptly ending a stream. |
| STOP_SENDING | A QUIC frame asking the peer to stop sending on a particular stream. |
| PRIORITY_UPDATE | An HTTP/3 frame updating relative priority among streams. |
| QPACK | The header-compression scheme for HTTP/3. RFC 9204. |
| HPACK | The header-compression scheme for HTTP/2. The predecessor to QPACK. |
| Dynamic table | A QPACK structure of recently-seen header fields, encoder-controlled. |
| Static table | The fixed list of common header fields baked into HPACK / QPACK. |
| Connection ID | A random number identifying a QUIC connection across IP changes. |
| CID | Short for Connection ID. |
| Source CID | The CID a sender wants the receiver to use when replying. |
| Destination CID | The CID the receiver advertised earlier; used when sending to it. |
| Retry packet | A QUIC packet a server can send to demand the client prove they own their address. |
| Version negotiation | The packet exchange where client and server agree on a QUIC version. |
| Stateless reset | A way for a server to drop unknown connections without keeping state. |
| Encryption level | One of Initial / Handshake / 0-RTT / 1-RTT. Each has its own keys. |
| Initial | The first encryption level. Keys derived from public salts. |
| Handshake (level) | The encryption level used after the Diffie-Hellman exchange. |
| 1-RTT | The fully forward-secure encryption level used for normal traffic. |
| 0-RTT | An encryption level used to send data with the first packet on a resumed connection. |
| Replay | Resending a captured packet later. 0-RTT data can be replayed; later data cannot. |
| Anti-replay window | A receiver mechanism to detect and reject replayed 0-RTT packets. |
| Retry token | An opaque token the client must echo back to prove address ownership. |
| Packet number | A monotonically increasing number on every QUIC packet within an encryption level. |
| PADDING | A QUIC frame whose only job is to pad packets to a chosen size. |
| PING | A QUIC frame used as a keep-alive or RTT probe. |
| ACK | A QUIC frame acknowledging received packets. |
| ECN | Explicit Congestion Notification. IP-level signal that the network is filling up. |
| CONNECTION_CLOSE | A QUIC frame ending the entire connection with an error code. |
| Transport parameter | A QUIC handshake field tuning the connection (limits, idle timeouts, etc.). |
| Max idle timeout | How long the connection can be idle before being declared dead. |
| Max UDP payload size | The largest UDP datagram each side will send. |
| Initial max data | Per-connection flow-control limit, total bytes. |
| Initial max stream data | Per-stream flow-control limit. |
| Ack delay | How long an endpoint may delay sending an ACK to coalesce more. |
| Congestion control | The algorithm that decides how fast you can send. |
| CUBIC | A common congestion-control algorithm. Default in Linux TCP and QUIC. |
| BBR | "Bottleneck Bandwidth and Round-trip propagation time." A model-based CC. |
| NewReno | An older, simpler congestion-control algorithm. |
| Datagrams (RFC 9221) | Optional unreliable datagrams over QUIC, for real-time use cases. |
| Multipath QUIC | Experimental QUIC extension using multiple paths concurrently. |
| Path validation | The challenge-response check QUIC does before migrating to a new path. |
| NAT rebinding | When a NAT device gives your connection a new external port without warning. |
| Connection migration | QUIC moving a connection from one IP/port pair to another. |
| Spin bit | An optional 1-bit field in QUIC headers letting passive observers measure RTT. |
| Alt-Svc | An HTTP header advertising that the same origin is reachable on another protocol. |
| HTTPS RR | A new DNS record type advertising HTTPS-specific service info, including ALPN and ports. |
| SVCB | The general service-binding DNS record. HTTPS RR is a specialization of SVCB. |
| Secure connection | One protected by TLS or QUIC; any modern browser will refuse less. |
| Post-quantum hybrid | A TLS / QUIC handshake combining classical and post-quantum key exchange. |
| qlog | A QUIC-implementation-agnostic log format for debugging. |
| quiche | Cloudflare's QUIC + HTTP/3 library, written in Rust. |
| msquic | Microsoft's QUIC implementation. |
| lsquic | LiteSpeed's QUIC implementation. |
| picoquic | A small academic QUIC implementation. |
| ngtcp2 | A QUIC library used by curl and others. |
| aioquic | A Python QUIC library. |
| neqo | Mozilla's QUIC implementation, used in Firefox. |
| mvfst | Facebook / Meta's QUIC implementation. |
| Round trip | The time for a packet to go from one end to the other and back. |
| RTT | Short for round-trip time. |
| MTU | Maximum Transmission Unit. The largest single packet a network can carry. |
| Path MTU | The smallest MTU along the entire path between two endpoints. |
| PMTUD | Path MTU Discovery. The process of finding the path MTU. |
| Long header | A QUIC packet header used during the handshake. Includes version + CIDs. |
| Short header | A QUIC packet header used after the handshake. Smaller. |
| User space | Code that runs as a normal program, not in the kernel. |
| Kernel space | Code that runs as part of the operating system kernel. |
| Middlebox | Anything between client and server that fiddles with packets — NATs, firewalls, load balancers, DPI boxes. |
| DPI | Deep Packet Inspection. A firewall feature that looks inside packets. |
| SPI | Stateful Packet Inspection. A firewall that tracks connection state. |
| Idempotent | A request that can be safely retried — no side effects on repeat. |
| GOAWAY frame | An HTTP/3 frame indicating the sender will accept no new streams above a given ID. |
| qpack encoder stream | A unidirectional QUIC stream carrying QPACK dynamic-table updates. |
| qpack decoder stream | A unidirectional QUIC stream carrying QPACK acknowledgments. |
| Control stream | A unidirectional QUIC stream carrying HTTP/3 SETTINGS, GOAWAY, etc. |
| Push stream | A unidirectional QUIC stream carrying a server-pushed response. |
| Stream ID | A 62-bit integer identifying a QUIC stream. |
| Bidirectional stream | A QUIC stream both sides can send on. |
| Unidirectional stream | A QUIC stream only one side sends on. |
| Initial keys | Encryption keys derived from a public salt + Destination CID. |
| Handshake keys | Encryption keys derived from the TLS handshake. |
| 1-RTT keys | Final, forward-secure keys after the handshake completes. |
| 0-RTT keys | Resumption keys derived from a previous session ticket. |
| Key update | A QUIC mechanism to rotate 1-RTT keys mid-connection. |
| Forward secrecy | A property that means past traffic stays safe even if a key leaks later. |
| AEAD | Authenticated Encryption with Associated Data. The cipher mode QUIC uses. |
| AES-GCM | A common AEAD cipher. Hardware-accelerated on most CPUs. |
| ChaCha20-Poly1305 | An AEAD cipher that is fast on CPUs without AES hardware. |
| Header protection | A QUIC mechanism that encrypts parts of the packet header. |

## Try This

Pick any of these to learn by doing. Each one takes ten minutes or less.

### 1. Compare HTTP/2 vs HTTP/3 latency for the same site

Use the curl-with-timing one-liner from Hands-On against several big sites: Cloudflare, Google, Facebook, Netflix. Run each version five times and compare the medians. Try it from a few different networks (home, coffee shop, mobile tether). Note when HTTP/3 wins and when it ties.

### 2. Capture QUIC packets in tcpdump and decode them in Wireshark

Set `SSLKEYLOGFILE=$HOME/quic-keys.txt` before running curl. Run a `tcpdump -w /tmp/quic.pcap -i any udp port 443` in another terminal. Hit a QUIC site with curl. Stop tcpdump. Open the pcap in Wireshark. Tell Wireshark about the keys file (Edit > Preferences > Protocols > TLS > (Pre)-Master-Secret log filename). Watch QUIC frames decrypt before your eyes. Click into a single QUIC packet and find the CRYPTO frame, the STREAM frame, the ACK frame.

### 3. Watch a connection migrate

Set up a long-lived QUIC client (a streaming video, or a `quiche-client` looping). While it is running, switch your laptop from Wi-Fi to a USB tether (or vice versa). The connection should keep going if the server supports migration. You can watch in tcpdump as the source IP changes mid-connection but the same Destination CID keeps showing up.

### 4. Try to break HTTP/3 with a firewall

Block UDP/443 outbound on your local machine with iptables or pf. Try `curl --http3-only`. It should time out. Now try `curl --http3` (which falls back). It should still work, over TCP. This shows the fallback path. Re-enable UDP and confirm `--http3-only` works again.

### 5. Decode an Alt-Svc cache miss

Open Chrome's `chrome://net-export/`. Start logging. Visit a fresh site that supports HTTP/3 (clear your cache first). Then visit it again. Stop logging. Open the JSON in `chrome://net-internals/#events` (or import it into the netlog viewer). Find the Alt-Svc entry that got cached. Find the second visit using HTTP/3 because of that cache.

### 6. Run nginx with HTTP/3

Compile or install nginx with `--with-http_v3_module`. Use a self-signed TLS cert. Add `listen 443 quic reuseport;` and `listen 443 ssl;` to a server block, plus `add_header Alt-Svc 'h3=":443"; ma=86400';`. Hit it with `curl --http3-only`. Watch your own server speak QUIC.

### 7. Watch the QPACK dynamic table grow

Most clients let you tune QPACK settings via library options. Send a long sequence of requests with the same headers and watch the on-wire bytes shrink as QPACK references the dynamic table. The first request will be large; subsequent ones will be tiny.

### 8. Look at qlog output

Some QUIC libraries can emit qlog (a JSON log of every event). Cloudflare's `quiche` does. Run `quiche-client` with `--qlog-file out.qlog`. Open in `qvis.quictools.info` for a beautiful timeline visualization. Find the Initial packet, the Handshake, the first 1-RTT bytes.

### 9. Compare congestion control algorithms

If you have control of both ends, run quiche or msquic with CUBIC, then with BBR, against the same lossy link (use `tc qdisc add dev eth0 root netem loss 1% delay 50ms` to fake one). Measure throughput. BBR usually wins on lossy links.

### 10. Read RFC 9000 §17

Don't read all of RFC 9000. But §17 ("Packet Formats") is short, beautiful, and shows you exactly what every QUIC packet looks like on the wire. Twenty minutes of reading and you will understand QUIC at a level most engineers never reach.

## Where to Go Next

You now know what HTTP/3 and QUIC are, why they exist, what they fix, and how to inspect them on a real machine. The natural next stops:

- `cs networking http3` — the practical reference sheet.
- `cs networking quic` — the QUIC-specific reference sheet, with more on packet types and frames.
- `cs networking http2` — what HTTP/3 replaced, including HPACK and stream priority.
- `cs networking http` — the parent sheet with HTTP/1.x details, status codes, methods.
- `cs detail networking/http3` — the deep-dive theory page.
- `cs ramp-up tcp-eli5` — the protocol underneath HTTP/2, and what QUIC is escaping.
- `cs ramp-up udp-eli5` — the protocol underneath QUIC.
- `cs ramp-up tls-eli5` — the security layer baked into QUIC.
- `cs ramp-up ebpf-eli5` — the BPF angle the source binder also covered. Different sheet because eBPF is its own giant world.
- `cs ramp-up linux-kernel-eli5` — the kernel that handles the UDP layer for QUIC.
- `cs web-servers nginx` — server-side HTTP/3 with nginx 1.25+.
- `cs web-servers caddy` — server-side HTTP/3 by default, no flags needed.

## See Also

- `networking/http`
- `networking/http2`
- `networking/http3`
- `networking/quic`
- `networking/tcp`
- `networking/udp`
- `networking/dns`
- `security/tls`
- `web-servers/nginx`
- `web-servers/caddy`
- `web-servers/haproxy`
- `ramp-up/tcp-eli5`
- `ramp-up/udp-eli5`
- `ramp-up/tls-eli5`
- `ramp-up/ebpf-eli5`
- `ramp-up/ip-eli5`
- `ramp-up/linux-kernel-eli5`

## References

- RFC 9000 — QUIC: A UDP-Based Multiplexed and Secure Transport
- RFC 9001 — Using TLS to Secure QUIC
- RFC 9002 — QUIC Loss Detection and Congestion Control
- RFC 9114 — HTTP/3
- RFC 9204 — QPACK
- RFC 9221 — Unreliable Datagram Extension to QUIC
- RFC 9220 — Bootstrapping WebSockets with HTTP/3
- RFC 9460 — Service Binding (SVCB) and HTTPS RR
- RFC 9312 — Manageability of the QUIC Transport Protocol
- "HTTP/3 explained" by Daniel Stenberg — free book at https://http3-explained.haxx.se/
- nghttp2.org — nghttp2 / nghttp3 / ngtcp2 documentation
- quiche — Cloudflare, https://github.com/cloudflare/quiche
- msquic — Microsoft, https://github.com/microsoft/msquic
- lsquic — LiteSpeed, https://github.com/litespeedtech/lsquic
- aioquic — Python, https://github.com/aiortc/aioquic
- neqo — Mozilla, https://github.com/mozilla/neqo
- mvfst — Meta, https://github.com/facebookincubator/mvfst
- qvis — visualize qlog files at https://qvis.quictools.info/
- HTTP/3 deployment statistics — https://w3techs.com/technologies/details/ce-http3
- IETF QUIC working group archive — https://datatracker.ietf.org/wg/quic/about/
- IETF MASQUE working group — proxy mechanisms over HTTP/3
- Cloudflare blog "HTTP/3 from A to Z" series — long-form explainers
- Daniel Stenberg's curl + HTTP/3 build guide — https://github.com/curl/curl/blob/master/docs/HTTP3.md
- Robin Marx's QUIC deep-dive blog series at https://calendar.perfplanet.com/2020/head-of-line-blocking-in-quic-and-http-3-the-details/
- "QUIC is Now RFC 9000" by the IETF, May 2021 announcement
- Internet Society "An Introduction to QUIC" briefing
- Google's original QUIC paper (SIGCOMM 2017): "The QUIC Transport Protocol: Design and Internet-Scale Deployment"
- Cloudflare radar — live HTTP version share at https://radar.cloudflare.com/
- The QUIC interop matrix at https://interop.seemann.io/ — see which implementations talk to which
- Cloudflare's "Even faster connection establishment with QUIC 0-RTT" — security caveats explained
- "Path MTU Discovery for QUIC" — RFC 8899 (DPLPMTUD), the algorithm QUIC uses to find safe packet sizes
