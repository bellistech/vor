# TCP — ELI5 (Certified Mail With Delivery Receipts)

> TCP is certified mail for computers: every letter is numbered, every letter is signed for, and any letter that goes missing gets sent again until it arrives.

## Prerequisites

(none — but `cs ramp-up linux-kernel-eli5` and `cs ramp-up icmp-eli5` help)

You do not need to know anything about networks to read this. You do not need to know what a "packet" is. You do not need to know what an "IP address" is. By the end of this sheet you will know all of those things, and you will have run real commands in a real terminal to see TCP working with your own eyes.

If a word feels weird, look it up in the **Vocabulary** table near the bottom. Every weird word in this sheet lives in that table with a one-line plain-English definition.

If you see a `$` at the start of a line in a code block, that means "type the rest of this line into your terminal." You do not type the `$`. The lines underneath that don't have a `$` are what your computer prints back at you. We call that "output."

## What Even Is TCP?

### Imagine sending a really important letter

Picture this. You are sitting at home and you need to send a really important letter to your grandma. Maybe it's a thank-you note for a birthday present. Maybe it's a homework assignment your teacher said had to get to grandma somehow. The point is, you really, truly need this letter to reach grandma. Not "probably." Not "most of the time." It has to get there. And if you sent twenty letters, all twenty have to arrive, and they all have to arrive in the right order, and grandma has to be sure none of them are duplicates and none are fakes.

You walk into the post office. You ask, "What is the most reliable way to send a letter?" The clerk smiles. "You want **certified mail**," she says. "We will give your letter a number. The mail carrier will hand it directly to your grandma. Your grandma signs a little card saying she got it. The little card comes back to you. Now you know, for sure, that your letter arrived. If for some reason the letter never gets there, we will know, because no signed card will come back. We will send another copy. We will keep trying until it gets there. We will tell you the moment it does."

You think about this. "What if I want to send a hundred letters?" you ask.

"Same thing," says the clerk. "Each one gets its own number. Each one gets its own signed card. If any of them fail to arrive, we resend that exact one. Your grandma puts them all in order on her kitchen table by the numbers. She knows which one came first, which came second, all the way up to one hundred. If there is a missing letter — say letter number forty-seven never arrives — your grandma knows, because she has number forty-six and number forty-eight but no forty-seven. She tells the post office. The post office tells you. You send number forty-seven again."

That is TCP. That is the whole idea. **TCP is certified mail for computers.** Every chunk of data is a letter. Every letter has a number. Every letter gets a signed receipt. Anything missing gets sent again. By the time the conversation is over, both sides are absolutely certain every byte of data arrived in exactly the right order.

### Why this is harder than it sounds

You might think, "Pfft, just send the data. The internet works, it's fine."

Here is the secret of the internet that most people never learn: **the internet is unreliable.** Not a little. A lot. Down at the bottom layer, the actual wires and Wi-Fi signals that move your data around, things go wrong all the time. Packets get dropped. Packets show up in the wrong order. Packets sometimes show up twice. Packets sometimes show up at the wrong door entirely. Packets get garbled by static. Packets get held up at a busy router for a half-second and then released in a flood. Packets get eaten by black holes (we will get to those).

The wires under the ocean carry signals across thousands of miles. Sometimes a signal hits a glitch and a bit flips from a 1 to a 0. Sometimes a router somewhere is too busy and just throws away packets to keep up. Sometimes a Wi-Fi signal bounces off a microwave oven and gets mangled. The hardware does not promise anything. The wires do not promise anything. The radio waves do not promise anything. **The bottom layer is chaos.**

But you, sitting at your computer, want to download a movie or send an email or load a webpage, and you want all the bytes to arrive correctly, in order, without missing pieces. You don't want to think about packets. You don't want to think about retries. You just want it to work.

So somebody, a long time ago, said: "What if we built a system on top of the chaos that hides the chaos? What if we built a system that, no matter how broken the underneath layer is, always delivers the bytes correctly and in order, like certified mail?"

That is TCP. That is the entire purpose. **TCP turns an unreliable network into a reliable one.** It is a magic trick. The chaos is still happening underneath. Packets are still getting lost and reordered and duplicated. But TCP catches all of that and hides it from you, so all you see is a clean, perfect stream of bytes coming in and going out.

### A different picture: TCP as a phone call

Here is another way to think about it. Pretend two people want to have a phone call.

Before they talk, they have to dial each other up. The phone rings. One person picks up. They say "hello?" The other says "hi, it's me." The first one says "great, I hear you." Now they are connected. Now they can talk.

While they talk, if one person doesn't hear something clearly, they say "wait, what was that last thing? Could you say it again?" The other person repeats it. The conversation keeps going. Both people stay on the line until the call ends. When they're done, one person says "okay, I'm hanging up now." The other says "okay, bye." Click.

That is TCP. There is a setup at the beginning (the **three-way handshake**). There is a conversation in the middle, with each side asking the other to repeat anything that got garbled (the **acknowledgments and retransmissions**). There is a polite goodbye at the end (the **four-way close**). And all the while, both sides are staying connected. Both sides know who the other is. Both sides are keeping track of where they are in the conversation.

The opposite of a phone call is a postcard. You drop a postcard in the mail and walk away. You don't know if it arrived. You don't know when it arrived. You can't ask the recipient questions in real time. That is **UDP**, the other big protocol. UDP is fast and simple but doesn't guarantee anything. TCP is the phone call. TCP guarantees everything.

### A third picture: TCP as a careful video call between two robots

Imagine two robots having a video call. Both robots are very polite. They never talk over each other. Whenever one robot finishes a sentence, the other robot says "got it, please continue." If a sentence comes through with static, the listening robot says "I missed that, please say it again." Both robots have a notebook in front of them where they write down every single thing the other robot says, with a little number next to it. If the listening robot ever notices a number missing — like it has 1, 2, 3, 4, and 6, but no 5 — it stops and says "wait, where is sentence 5?"

The two robots also pay attention to how fast they are talking. If the listening robot is having trouble keeping up, it says "please slow down a bit." If both robots are talking really comfortably and the network is humming, they pick up the pace.

That is TCP. Polite. Numbered. Acknowledged. Re-asked when something is missing. Slowed down or sped up to match conditions. Two computers, two notebooks, every byte tracked.

## The Two Hard Problems TCP Solves

TCP exists because the internet has two problems that nobody can fix at the wire level. TCP solves them at the software level instead.

### Problem 1: The network is unreliable

Down at the wire level, on the actual cables and radio waves, anything can go wrong. We already talked about this above, but let's make it very concrete. Picture a packet of data — a little bundle of bytes wrapped in a header — getting sent from your laptop in California to a server in Europe.

That packet is going to travel through dozens of pieces of equipment on the way. Your Wi-Fi router. Your home modem. Your internet provider's first hop. Their next hop. A regional router. A national router. A big router on the coast. A submarine cable across the ocean. A landing station in Europe. Their regional routers. Their last-mile router. The European internet provider. The server's data center. The server's switch. The server's network card.

At every single step, the packet might:

- **Get dropped.** A router might be too busy to handle it and just throw it away. This happens billions of times per day on the internet. The router does not call you up to apologize.
- **Get reordered.** Two packets might take slightly different paths through the network and arrive in the wrong order. Packet 5 might come in before packet 4.
- **Get duplicated.** A weird glitch or a route change might cause the same packet to show up twice.
- **Get corrupted.** A bit might flip in transit. A 1 becomes a 0 or vice versa. The packet still arrives but its contents are wrong.
- **Get delayed.** The packet sits in a router's queue for a long time before being forwarded. By the time it arrives, the receiver might have given up waiting.
- **Get eaten by a black hole.** A misconfigured router somewhere just silently consumes the packet without telling anyone. It vanishes.

If you tried to use the network with no help, every single one of these would be your problem. Your application would have to detect drops, fix reorders, throw out duplicates, recompute checksums, retry, and time out. Every program that talks to the network would have to do all this. It would be a nightmare.

So instead, **TCP does it for you.** TCP sits below your application and above the unreliable network. It catches every drop, fixes every reorder, throws out every duplicate, verifies every checksum, retries every loss, and only hands clean, in-order, correct bytes up to your application. Your application sees a clean stream. The chaos stays hidden.

### Problem 2: The network has variable speed

The second problem is just as bad. The network is sometimes fast, sometimes slow, and **the speed changes from second to second.**

Imagine your home Wi-Fi when nobody else is using it. You might get really fast speeds. Then your sister starts streaming a movie. Now your speed drops. Then your sister stops, but somebody three houses over starts a giant download on the same shared cable. Now your speed drops more. Then they stop. Now it's fast again. The speed is bouncing around all the time, and you have no way to know what speed will be available at the next millisecond.

If TCP just blasted data out at the maximum theoretical speed of your connection, three things would go wrong:

- **The receiver would get overwhelmed.** Maybe the receiver is a slow phone with limited memory. Even if your laptop can send at gigabit speeds, the phone can only swallow at megabit speeds. Sending faster than the phone can handle just causes the extra to get dropped.
- **The network in the middle would get overwhelmed.** Routers have limited buffers. If you fill them up, they start dropping packets. Now you're causing the very losses you're trying to avoid.
- **You'd be a bad neighbor.** If everyone blasted at full speed, the whole shared network would melt. Your traffic would crowd out everyone else's. The internet only works because every TCP sender voluntarily slows down when the network is busy.

So TCP has to **adapt.** It has to figure out, in real time, how fast it can send without overwhelming the receiver and without overwhelming the network. When things look good, it speeds up. When things look bad, it slows down. It does this constantly, every fraction of a second, for every connection.

This is called **flow control** (don't overwhelm the receiver) and **congestion control** (don't overwhelm the network). Together they are the magic that lets the internet work for billions of users at once without melting.

### Two problems, one protocol

So TCP has two big jobs:

1. Make the unreliable network look reliable.
2. Adapt to the network's variable speed.

Everything in TCP — the sequence numbers, the acks, the windows, the timers, the slow start, the congestion avoidance, the retransmissions — exists to solve one of those two problems. If you remember the two problems, the rest of TCP makes sense.

## The Three-Way Handshake

Before two computers can have a TCP conversation, they have to set up the connection. This is the **three-way handshake.** It is exactly three messages, no more, no less. Each one is short. Each one has a special purpose. Together they do four things:

- Confirm both sides are alive and reachable.
- Agree on the starting sequence numbers each side will use.
- Tell each other their initial window size and any options.
- Move both sides into the **ESTABLISHED** state.

Here is the flow:

```
   CLIENT                                                 SERVER
   ------                                                 ------

     |                                                       |
     |  1. SYN (seq=1000)                                    |
     |  "Hi! I want to talk. My starting number is 1000."    |
     |------------------------------------------------------>|
     |                                                       |
     |                                                       |
     |  2. SYN-ACK (seq=5000, ack=1001)                      |
     |  "OK! I want to talk too. My starting number          |
     |   is 5000. I confirm I got your 1000+1=1001."         |
     |<------------------------------------------------------|
     |                                                       |
     |                                                       |
     |  3. ACK (seq=1001, ack=5001)                          |
     |  "Great. I confirm your 5000+1=5001. Let's go."       |
     |------------------------------------------------------>|
     |                                                       |
     |                                                       |
     |          === CONNECTION ESTABLISHED ===               |
     |                                                       |
```

### Why three messages, not two?

This is a question every kid asks when they first see this, and it is a great question. Wouldn't two be enough? "Hi, I want to talk." "OK, let's talk." Done?

Almost. The reason we need three is that **both sides need to know that both sides got the message.**

Think about it this way. After message 1, the server knows the client wants to talk. Good. After message 2, the client knows the server agreed. Good. But after only two messages, the **server doesn't know that the client got the agreement.** The server sent its agreement, but it doesn't know if it arrived. Maybe it got dropped. Maybe the client never heard it.

So we need a third message: the client sends one final acknowledgment back to the server saying "I got your agreement." Now both sides know the other side knows.

This is one of those cute logic puzzles. To establish a connection over an unreliable network, you actually need three messages. Two is mathematically not enough. There is even a famous theorem about this called the "Two Generals' Problem," but the gist for TCP is: **three messages is the minimum number to be sure both sides agree.** TCP uses exactly that minimum.

### Walking through each message

Let's go through each of the three messages and what they really say.

**Message 1: SYN (synchronize)**

The client picks a random starting number, say 1000. Why random? Because if the client always started at 0, an attacker could guess the numbers and inject fake packets. Picking randomly makes it hard to guess. The client sends a packet with the SYN flag set and that initial sequence number.

In real TCP the starting number is a 32-bit value, so it can be anything from 0 to about 4 billion. Modern systems pick it from a cryptographic random source.

**Message 2: SYN-ACK (synchronize and acknowledge)**

The server receives the SYN. It picks its own random starting number, say 5000. It also acknowledges the client's number by sending back the client's number plus one (1001 in our example). The "plus one" is key: it says, "I got your 1000, and I'm telling you the next byte I expect is 1001."

The server sends back a packet with both the SYN flag and the ACK flag set. Two flags in one packet to save a round trip.

**Message 3: ACK (acknowledge)**

The client receives the SYN-ACK. It now knows the server's starting number (5000). It sends back an ACK packet acknowledging the server's number by adding one (5001). Now both sides have confirmed they received each other's starting numbers.

After this third packet, both sides move from SYN_SENT and SYN_RECEIVED into the **ESTABLISHED** state. They can now exchange data.

### Why pick random starting numbers?

Two reasons. The first is security: if numbers were always 0, an attacker could craft a fake packet that looks like it belongs to your connection, and the receiver wouldn't know it was fake. By picking a random starting number, only the two real endpoints know what numbers are valid. An attacker has to guess from 4 billion possibilities. (Modern attackers can sometimes still do this, which is why we layer encryption on top with TLS, but that's another sheet.)

The second reason is to handle old connections. Imagine you had a TCP connection a minute ago, and it ended. Now you're starting a new one. If both connections used sequence number 0, a packet from the old connection that was delayed in the network might show up and get accepted by the new connection, which would corrupt the data. By picking new random numbers each time, the chance of collision is tiny.

### What happens if the SYN gets lost?

If the very first SYN is lost — the client sends it, but a router dropped it — the client will never get a SYN-ACK back. After a timeout (usually 1 second on Linux for the first try), the client retries by sending the SYN again. If that one is lost too, it doubles the wait time and retries: 2 seconds. Then 4. Then 8. This is **exponential backoff.** Linux gives up after a few retries (configurable, often 5 or 6 attempts) which adds up to about 2 minutes of trying. After that you get "Connection timed out."

Same for the SYN-ACK — if the server's reply is lost, the client never moves to ESTABLISHED, the server thinks the connection is starting but the client gives up, and the server eventually times out the half-open connection.

### What happens during a SYN flood?

Bad guys figured out that they could flood a server with SYN packets but never send the third ACK. The server sets aside memory for each half-open connection waiting for the third message that never arrives. If they send millions of SYNs, the server runs out of memory. This is a classic denial-of-service attack called a **SYN flood.**

Modern systems defend against this with **SYN cookies.** Instead of allocating memory for each half-open connection, the server encodes the connection's state into the SYN-ACK's sequence number itself, like a magic decoder ring. When the third ACK comes in, the server reads the state out of the ack number and reconstructs the connection. No memory is held during the wait. SYN flood prevented.

You can see if SYN cookies are enabled on Linux:

```bash
$ cat /proc/sys/net/ipv4/tcp_syncookies
1
```

A `1` means they're on (the default on most modern Linux). A `0` means off.

## Sequence Numbers and Acknowledgments

The heart of TCP's reliability magic is **sequence numbers.** Every byte of data sent over a TCP connection has a number. Not every packet, every **byte.**

### Every byte has a number

Imagine you want to send the message "HELLO". That's five bytes: H, E, L, L, O. If your starting sequence number is 1000, then:

- H is byte 1000
- E is byte 1001
- L is byte 1002
- L is byte 1003
- O is byte 1004

When you send these bytes in a TCP packet, you put a sequence number in the header that says "this packet starts at byte 1000." The receiver sees the header, knows the bytes are 1000 through 1004, and can put them in the right place. If the next packet starts at 1005, the receiver knows that's the byte that comes right after.

This is the trick that makes ordering work. Even if packets arrive out of order, the receiver can sort them by sequence number. It just looks at the sequence number on each packet, drops it into the right slot in its buffer, and waits for any holes to fill in.

### What is an ACK?

An ACK ("acknowledgment") is a tiny packet the receiver sends back to say "I got your bytes up to here." The ACK contains an acknowledgment number, which is **the next byte the receiver expects.**

If the receiver got bytes 1000 through 1004, it sends back an ACK with ack=1005. That means: "I have everything up to and including byte 1004. Send me 1005 next."

This is called a **cumulative ACK.** It acknowledges everything up to the ack number. You don't have to send a separate ACK for every byte. One ack=1005 tells the sender "all five bytes are received."

### What if a packet is missing?

Imagine the sender sends three packets:

- Packet A: bytes 1000-1099
- Packet B: bytes 1100-1199
- Packet C: bytes 1200-1299

Packet A arrives. Packet B gets dropped. Packet C arrives.

The receiver got A and C but not B. With cumulative ACKs alone, the receiver sends back ack=1100 (acknowledging A only) when it receives A. When it receives C, it can't say "I got everything up to 1300" because B is missing. So it sends ack=1100 again.

That's a **duplicate ACK.** The sender sees the ack number didn't move forward, even though more data was sent. That's a strong hint that something got lost. We'll see how senders react to duplicate ACKs later — that's the foundation of fast retransmit.

### What is SACK?

Cumulative ACKs are nice but they have a problem: if many packets are missing in the middle, the receiver can only acknowledge up to the first hole. The sender doesn't know what came after. The sender might end up retransmitting packets that already arrived.

**SACK** ("Selective Acknowledgment") fixes this. With SACK, the receiver can say, "I have bytes 1000-1099 (the cumulative ACK) AND I also have bytes 1200-1299 (the SACK block). The hole is 1100-1199." Now the sender knows exactly what to retransmit.

SACK was introduced in RFC 2018 in 1996 and is now standard. Linux enables it by default. You can check:

```bash
$ cat /proc/sys/net/ipv4/tcp_sack
1
```

A `1` means SACK is on.

### A picture of sequence numbers in action

```
   SENDER                                   RECEIVER
   ------                                   --------

   Send seq=1000 (100 bytes)  ---->         Receives bytes 1000-1099
                              <----  ACK ack=1100
                                            "I have everything up to 1099,
                                             send me 1100 next."

   Send seq=1100 (100 bytes)  ---->         Receives bytes 1100-1199
                              <----  ACK ack=1200

   Send seq=1200 (100 bytes)  --X            (LOST in network)

   Send seq=1300 (100 bytes)  ---->         Receives 1300-1399 but
                                             still missing 1200-1299
                              <----  ACK ack=1200, SACK=[1300-1400]
                                            "I still need 1200, but FYI
                                             I also got 1300-1399."

   (Sender sees duplicate ACK + SACK info,
    retransmits seq=1200)
   Send seq=1200 (100 bytes)  ---->         Now has everything up to 1399
                              <----  ACK ack=1400
                                            "Great, all caught up."
```

This is the dance happening millions of times per second across the internet, in every TCP connection.

## The Receive Window (Flow Control)

Now we get to the speed problem. The sender doesn't want to overwhelm the receiver. So the receiver constantly tells the sender, "Here is how much data I can swallow right now." That number is called the **receive window**, or **rwnd** for short.

### Picture a bucket

Imagine the receiver has a bucket. Data comes in and fills the bucket. The receiving application drains the bucket as it processes data. The receive window is the empty space in the bucket — how much more data the sender can pour in before the bucket overflows.

If the bucket is half full, the receive window is half the bucket's size. If the bucket is almost empty, the receive window is almost the whole bucket. If the bucket is full, the receive window is **zero** and the receiver tells the sender "stop! Don't send anything more until I drain some out."

The receiver puts the current window size in every ACK it sends back. The sender reads it and adjusts. "Oh, the receiver says I can send up to 64 kilobytes more. OK." If the next ACK says the window is now 32 kilobytes, the sender slows down. If the next one says 0, the sender stops entirely until a later ACK comes with a bigger window.

This is **flow control.** The sender never sends more than the receiver's current window allows.

### Why does the window go up and down?

The window changes because the receiving application drains the bucket at variable speeds. Maybe the application is busy doing something else and isn't reading data right now. Then the bucket fills up and the window shrinks. When the application catches up and starts reading, the bucket drains and the window grows again.

The receive window is also limited by how much memory the kernel has reserved for the receive buffer for this socket. On Linux, you can see those limits:

```bash
$ cat /proc/sys/net/ipv4/tcp_rmem
4096    131072  6291456
```

Three numbers: minimum, default, maximum. So this kernel will give a TCP socket a receive buffer between 4 KB and 6 MB, starting at 128 KB by default. The window is bounded by whatever the buffer is.

### Window scaling: dealing with fast modern networks

Original TCP from 1981 had a 16-bit window field. The maximum window size was 65,535 bytes. That was plenty in 1981 when networks were slow and round trips were short.

Today, a transcontinental link might have a round-trip time of 100 milliseconds and a bandwidth of 10 gigabits per second. To fully use that pipe, you need to send 10 Gbps × 0.1 s = 1 Gigabit = 125 MB worth of data "in flight" at any moment. But the maximum window from 1981 was only 65 KB. **You'd be limited to about 5 Mbps over a 10 Gbps link.** That would be terrible.

So in 1992 (RFC 1323, later updated by RFC 7323) people invented **window scaling.** Window scaling adds an option in the SYN that says "multiply my window value by 2^N." N can be up to 14, which means you can scale by up to 16,384. So a window value of 65535 with a scale of 14 gives an effective window of 65535 × 16384 = about 1 GB. That's enough for any modern network.

Window scaling is negotiated at handshake time. Both sides send their scale factor in the SYN. Both sides apply both scales for the duration of the connection. If you only see SYN-ACK without window scale, it falls back to the original 64 KB limit.

Linux turns this on by default:

```bash
$ cat /proc/sys/net/ipv4/tcp_window_scaling
1
```

### Zero windows and window probes

If the receiver's bucket fills up, it sends an ACK with window=0. The sender stops. But what if the receiver's update ACK (saying "OK, I have room again, window=large_number") gets lost? The sender would wait forever.

To prevent this, the sender sends a **zero-window probe** — a tiny packet — every so often when the window is zero. The receiver responds with the current window. If the receiver opened up but the previous ACK was lost, the probe surfaces the new window. Cute trick.

## The Congestion Window (Congestion Control)

Now we move to the second speed problem: not overwhelming the **network.** Even if the receiver has plenty of room (rwnd is huge), the network in the middle might not. So in addition to rwnd, TCP keeps another window called **cwnd** — the **congestion window.**

The sender sends no more than the **smaller** of rwnd and cwnd. rwnd protects the receiver. cwnd protects the network. The smaller one wins.

### How does the sender know the network is busy?

Here is the clever part. The sender has no direct way to ask the network "are you busy?" The internet doesn't report congestion. There's no dashboard. The only signal the sender has is whether packets are getting lost or delayed.

So the sender uses **packet loss as a signal of congestion.** If packets are getting lost, the network is full, slow down. If packets are getting through and acks are coming back fast, the network has room, speed up.

This is a beautiful idea. The sender can't see the network, but it can feel its way through the network by watching what happens to packets. When the network pushes back (drops packets), TCP backs off. When the network is calm, TCP grows.

### Slow start

When a connection first starts, TCP doesn't know how fast the network is. So it starts cautiously. It sets cwnd to a small value (originally 1 segment, now usually 10 segments — about 14 KB) and **doubles it every round trip** until something goes wrong.

This is called **slow start.** The name is misleading — slow start is actually fast, because doubling grows exponentially. But it's "slow" compared to the alternative of just blasting at full speed from the first packet.

So the cwnd grows: 10 segments, then 20, then 40, then 80, then 160. Each time, all the packets are sent, all the acks come back, and the next round the sender doubles its rate. This continues until either:

- A packet is lost. The sender saw too much. Back off.
- The cwnd hits **ssthresh** (slow start threshold), a value that says "we think this is about as fast as the network can handle." Switch from slow start to congestion avoidance.

### Congestion avoidance

Once cwnd reaches ssthresh, slow start ends and **congestion avoidance** begins. In congestion avoidance, cwnd grows much more slowly: roughly +1 segment per round trip, instead of doubling.

This is gentler. The sender is now in a phase where it is probably close to the network's capacity. It increases slowly to avoid causing a sudden flood. It is feeling its way up to find the limit.

### Fast retransmit and fast recovery

What happens when a packet is lost during congestion avoidance? The sender notices because of duplicate ACKs (we talked about these earlier). When the sender sees three duplicate ACKs, it concludes a packet was lost and retransmits immediately, without waiting for a full timeout. This is **fast retransmit.**

After fast retransmit, the sender enters **fast recovery.** Instead of going all the way back to slow start, it cuts cwnd in half (or some factor) and continues in congestion avoidance. The idea is: a single lost packet doesn't mean the network is totally saturated. Cut the rate back, but don't restart from scratch.

### Reno → CUBIC → BBR (the three main flavors)

There have been many congestion control algorithms over the decades. The big ones you'll hear about are Reno, CUBIC, and BBR.

**Reno** (from the 1990s) is the classic AIMD ("Additive Increase, Multiplicative Decrease") algorithm. Grow cwnd by +1 each round trip. On loss, cut it in half. Simple, robust. Reno is the foundation everything else is built on.

**CUBIC** (2008) is what Linux uses by default today. It uses a cubic curve (the math kind: c × t³) to grow cwnd. The curve is shaped so that after a loss, cwnd grows quickly back to the previous high, then plateaus near the limit. It works much better than Reno on high-bandwidth, high-latency networks. You can think of it as: "rush back up to where you were, then carefully creep higher."

**BBR** (2016, from Google) takes a totally different approach. Instead of using packet loss as the signal, it measures the actual **bottleneck bandwidth** and the **round-trip time** of the path. It then sends at the rate that fills the pipe without queueing in routers. The idea is, by the time you see packet loss, you've already overfilled buffers somewhere. BBR tries to find the perfect rate without ever causing loss.

BBR can be way faster than CUBIC on certain types of paths (especially long, lossy ones). But it can also be aggressive against CUBIC flows. Linux supports both. You can switch:

```bash
$ cat /proc/sys/net/ipv4/tcp_congestion_control
cubic

$ cat /proc/sys/net/ipv4/tcp_available_congestion_control
reno cubic bbr
```

Switching to BBR (root):

```bash
$ sudo sysctl -w net.ipv4.tcp_congestion_control=bbr
net.ipv4.tcp_congestion_control = bbr
```

BBR has a v2 (BBRv2) that is more polite to other flows. It's an active research area.

### Putting it all together

```
   cwnd
    |
    |                                          ____________________________
    |                                         /
    |                                  ______/
    |                                 /
    |                          ______/
    |                         /
    |                  ______/  <-- congestion avoidance: +1 per RTT
    |                 /
    |                /
    |               /  <-- ssthresh reached, switch to CA
    |              /
    |             /
    |            /
    |           /
    |          /
    |         /
    |        /  <-- slow start: doubling per RTT
    |       /
    |      /
    |     /
    |    /
    |   /
    |  /
    | /
    |/
    +------------------------------------------------------------- time
```

Slow start grows fast. Congestion avoidance grows slow. On loss, drop and try again. That's the basic shape.

## Retransmissions and RTO

We've talked about how the sender knows packets are missing (duplicate ACKs and SACK blocks). But what if there are no ACKs at all? What if the entire connection just goes silent?

That can happen for many reasons. Maybe a router went down. Maybe a Wi-Fi signal got really bad. Maybe the receiver's network is gone entirely. The sender doesn't know what happened. All it sees is silence.

So TCP has a timer called the **RTO** — the **Retransmission Timeout.** If a packet is sent and no ACK comes back before the RTO expires, the sender retransmits the packet.

### How long should the RTO be?

This is a tricky question. Set it too short, and you'll retransmit packets that were just delayed but not lost — wasting bandwidth and confusing the receiver. Set it too long, and recovery from real losses is painfully slow.

The answer is to base RTO on the actual round-trip time of the path. Measure RTT. RTO is some multiple of RTT. The current formula (RFC 6298) is:

```
RTO = SRTT + 4 * RTTVAR
```

Where SRTT is the smoothed round-trip time and RTTVAR is the variance. If RTTs are stable, RTO is just a bit above RTT. If RTTs are bouncing around, RTO is much higher to avoid spurious retransmissions. There's also a minimum (usually 200ms on Linux) and a maximum (usually 60 seconds).

### How does TCP measure RTT?

When the sender sends a packet, it remembers when it sent. When the matching ACK comes back, it subtracts. That difference is the RTT for that packet.

There's a subtlety: what if the packet was retransmitted? You can't tell whether the ACK is for the original or for the retransmission. Karn's algorithm (named after Phil Karn) says: **don't measure RTT on retransmitted packets.** Only use the unambiguous samples. Wait for a clean send-and-ack pair. Otherwise the measurement is biased.

There's also a **TCP timestamp** option (RFC 7323) that puts the send time in the packet header, so the ACK can echo it back. This gives unambiguous RTT measurement even on retransmits. Most modern systems use timestamps.

### Why exponential backoff on repeated losses?

If an RTO fires and no ACK comes back even after retransmission, it might mean the network is really broken. Retransmitting at the same rate would just pile on. So TCP **doubles the RTO** after each consecutive timeout. RTO becomes 2× then 4× then 8× and so on, up to some maximum.

This way, if the network really is dead, the sender doesn't keep blasting it with packets every fraction of a second. It backs off, gives the network a chance to recover, and only retries occasionally. If the network comes back, the next retransmission gets through, the connection wakes up, and RTO returns to normal.

### Tail loss probe (TLP)

There's an annoying edge case: what if the **last** packet of a burst is lost? The sender sent some packets, then went idle. The last one was lost. There's no traffic to generate duplicate ACKs (because no later packets are being sent). So fast retransmit can't fire. The sender has to wait for the full RTO to fire, which might be 200ms or more.

**Tail Loss Probe (TLP)** is a clever fix. After a short delay (about 2× RTT), the sender probes by re-sending the last packet. If it was lost, the receiver gets it now and acks. If it wasn't lost, no harm done. TLP cuts tail latency dramatically for short transactions like web requests.

### RACK (Recent Acknowledgment)

**RACK** is a more modern loss-detection mechanism (RFC 8985) that replaces the old "three duplicate ACKs" rule with a smarter time-based rule. RACK looks at the timestamps of received ACKs and concludes: "any packet I sent more than X time ago, that hasn't been acknowledged yet, must be lost." This works better with reordering and complex modern networks.

Linux has RACK on by default in modern kernels:

```bash
$ cat /proc/sys/net/ipv4/tcp_recovery
1
```

## The Four-Way Close

Just like setup needed three messages, teardown needs four. (Actually it can be three sometimes, but let's start with four.)

```
   CLIENT                                                 SERVER
   ------                                                 ------

     |                                                       |
     |  1. FIN                                               |
     |  "I'm done sending. Closing my side."                 |
     |------------------------------------------------------>|
     |                                                       |
     |  2. ACK                                               |
     |  "OK, I see you're done."                             |
     |<------------------------------------------------------|
     |                                                       |
     |  --- server can still send if it wants ---            |
     |  --- this is called the half-closed state ---         |
     |                                                       |
     |  3. FIN                                               |
     |  "Now I'm done sending too."                          |
     |<------------------------------------------------------|
     |                                                       |
     |  4. ACK                                               |
     |  "OK, full close."                                    |
     |------------------------------------------------------>|
     |                                                       |
     |        === BOTH SIDES CLOSED ===                      |
     |                                                       |
     |  Client enters TIME_WAIT for 2 * MSL                  |
     |                                                       |
```

### Why four messages?

Because TCP connections are **bidirectional** and each direction has to be closed independently. The client says "I'm done sending" with a FIN. The server acks. Now data only flows server-to-client. The server can keep sending until it's done. When it's done, the server says "I'm done sending too" with its own FIN. The client acks. Now both directions are closed.

It's possible to combine messages 2 and 3 into one (an ACK and a FIN in the same packet) if the server is ready to close as soon as it sees the client's FIN. That's the "three-way close" path. But the four-way version is the general case.

### What is TIME_WAIT?

After the client sends the final ACK, it enters a state called **TIME_WAIT.** It hangs out there for a duration of **2 × MSL** (Maximum Segment Lifetime). MSL is usually 30 to 120 seconds, so TIME_WAIT is typically 1 to 4 minutes. On Linux, the default is 60 seconds, so TIME_WAIT is 120 seconds (effectively).

Why? Two reasons.

**Reason 1: Make sure the final ACK got there.** If the client's last ACK is lost, the server will retransmit its FIN. The client needs to be around to ack it again. If the client closed the connection immediately, it would receive the retransmitted FIN and not know what to do. By staying in TIME_WAIT for a while, the client can handle the retransmitted FIN cleanly.

**Reason 2: Avoid mixing up old and new connections.** Imagine the same two endpoints make a new connection on the same port pair right after closing the old one. If old packets from the previous connection were still floating around in the network, they might show up at the new connection and confuse it. By waiting 2 × MSL, all old packets are guaranteed to have died (because MSL is the maximum lifetime of any packet), so the new connection starts clean.

TIME_WAIT is on the side that initiates the close. If the server closes first, the server gets TIME_WAIT. If the client closes first, the client gets TIME_WAIT. Often servers close first (in HTTP/1.0, after sending a response, the server closes), which is why high-traffic servers can accumulate lots of TIME_WAIT sockets.

### Tuning TIME_WAIT

If a server has a million TIME_WAIT connections, that consumes memory and ports. There are a few ways to manage this:

- `net.ipv4.tcp_tw_reuse=1` — allows reusing TIME_WAIT sockets for new outgoing connections. Safe.
- `net.ipv4.tcp_tw_recycle=1` — used to be a thing but **was removed in Linux 4.12** because it broke things behind NAT. Don't use this even if you find it.
- `net.ipv4.tcp_fin_timeout=60` — controls how long FIN_WAIT_2 lasts (related but not the same as TIME_WAIT).
- `SO_LINGER` socket option with timeout 0 — abruptly close with RST instead of FIN, no TIME_WAIT, but you lose all the safety. Almost never the right answer.

The right answer for most servers is: tcp_tw_reuse=1, leave the rest alone. TIME_WAIT is usually fine.

### What is CLOSE_WAIT?

**CLOSE_WAIT** is the opposite problem and a common cause of bugs. CLOSE_WAIT happens on the side that **received** a FIN but hasn't called close() yet.

If the remote peer sent a FIN, the kernel acked it (message 2 above) and put the socket in CLOSE_WAIT state. The kernel is waiting for the local application to also call close() so it can send its own FIN (message 3).

If the application forgets to call close(), the socket sits in CLOSE_WAIT forever. If this happens millions of times, you've got a leak. **A growing CLOSE_WAIT count almost always means a bug in the application: it isn't closing connections when the peer hangs up.**

If you see CLOSE_WAIT piling up:

```bash
$ ss -tan | awk '{print $1}' | sort | uniq -c
```

And there are thousands of CLOSE_WAIT entries, find the application that owns those sockets and fix the close() bug.

## TCP States

A TCP connection is a state machine. There are exactly **11 states** (well, 12 if you count CLOSED as a state). Here they all are:

```
                              +---------+
                              | CLOSED  | (no connection)
                              +---------+
                                 |   ^
       passive open (server)     |   | application close
       socket() + listen()       |   |
                                 v   |
                              +---------+
                              | LISTEN  | (waiting for SYN)
                              +---------+
                                 |
       receive SYN, send SYN-ACK |
                                 v
                              +-----------+
                              | SYN_RCVD  | (sent SYN-ACK, waiting for ACK)
                              +-----------+
                                 |
       receive ACK               |
                                 v
                              +-------------+
                              | ESTABLISHED |  <----- both sides talk freely
                              +-------------+
                                 |        ^
                                 |        | active open (client)
                                 |        | socket() + connect()
                                 |        | sends SYN
                                 |        |
                                 |     +-----------+
                                 |     | SYN_SENT  |
                                 |     +-----------+
                                 |        ^   |
                                 |        |   | receives SYN-ACK,
                                 |        |   | sends ACK
                                 |        |   |
                                 |        +---+
                                 |
                                 |  application calls close()
                                 |  (active close)
                                 v
                            +-----------+
                            | FIN_WAIT_1| (sent FIN, waiting for ACK)
                            +-----------+
                                 |
                receive ACK      |
                                 v
                            +-----------+
                            | FIN_WAIT_2| (got ACK, waiting for peer's FIN)
                            +-----------+
                                 |
                receive FIN      |
                                 v
                            +-----------+
                            | TIME_WAIT |  (wait 2 * MSL, then CLOSED)
                            +-----------+


       receive FIN              receive FIN+ACK  receive FIN
       (passive close)
       ESTABLISHED              SYN_RCVD         FIN_WAIT_1
            |                      |                  |
            v                      v                  v
       +-----------+        +-----------+        +---------+
       | CLOSE_WAIT|        | CLOSE_WAIT|        | CLOSING |
       +-----------+        +-----------+        +---------+
            |                                          |
   app close() sends FIN                       receive ACK
            v                                          v
       +---------+                              +-----------+
       | LAST_ACK|                              | TIME_WAIT |
       +---------+                              +-----------+
            |
      receive ACK
            v
       +---------+
       | CLOSED  |
       +---------+
```

Let's walk through each state:

- **CLOSED.** No connection exists. This is the starting state and the ending state.
- **LISTEN.** A server has called listen() and is waiting for incoming SYNs.
- **SYN_SENT.** A client has called connect() and sent a SYN. Waiting for SYN-ACK.
- **SYN_RECEIVED.** A server got a SYN, sent SYN-ACK. Waiting for the final ACK.
- **ESTABLISHED.** The connection is set up. Both sides can send data freely. This is where most of the work happens.
- **FIN_WAIT_1.** The local side called close() and sent a FIN. Waiting for the ACK of that FIN.
- **FIN_WAIT_2.** Got the ACK of our FIN. Waiting for the peer to send its own FIN.
- **CLOSE_WAIT.** The peer sent a FIN, we acked it. Waiting for our application to call close().
- **CLOSING.** Both sides sent FINs simultaneously. Waiting for ACK of our FIN.
- **LAST_ACK.** We sent our FIN (after the peer's), waiting for the final ACK.
- **TIME_WAIT.** Both sides have closed. Hanging out for 2 × MSL to make sure all old packets are dead.

You can see all your sockets and their states with:

```bash
$ ss -tan
State    Recv-Q  Send-Q  Local Address:Port  Peer Address:Port
LISTEN   0       128     0.0.0.0:22          0.0.0.0:*
ESTAB    0       0       192.168.1.10:54322  142.250.80.46:443
TIME-WAIT 0      0       192.168.1.10:54320  142.250.80.46:443
```

### What does each state look like in the wild?

A typical listening server has lots of sockets in LISTEN. A busy web server has lots of sockets in ESTABLISHED. After clients disconnect, sockets briefly enter FIN_WAIT_2 or TIME_WAIT. CLOSE_WAIT is rare in healthy code; CLOSING is rare too (only happens with simultaneous close).

If you see thousands of FIN_WAIT_2 sockets, the peer isn't closing properly. If you see thousands of CLOSE_WAIT sockets, your application isn't closing properly. If you see thousands of TIME_WAIT sockets, that's normal for a busy server but you might want tcp_tw_reuse=1.

## The Nagle Algorithm and TCP_NODELAY

There's a subtlety about how data gets packetized into segments. If your application writes one byte at a time, and TCP sent each byte as its own segment, you'd be wasting bandwidth: the TCP/IP header is at least 40 bytes, so a one-byte packet is 41 bytes total of which only 1 byte is data. That's 2.4% efficiency.

**Nagle's algorithm** (named after John Nagle, who invented it in 1984) prevents this. The rule is: if there's already unacked data in flight, hold onto small writes briefly until either (a) more data arrives to fill a full segment, or (b) the previous segment gets acked. This batches small writes into bigger packets.

### When Nagle is good

For bulk data transfers — file uploads, video streams, large web pages — Nagle is fine. The application is sending lots of data, segments fill up naturally, and Nagle barely affects anything.

### When Nagle is bad

For chatty interactive applications — SSH, real-time games, RPC calls — Nagle adds latency. You write a small message, Nagle holds it for up to 200ms waiting for more, but you don't have any more to send. The other side waits for it. Nothing happens. Eventually Nagle gives up and sends, but you've wasted up to 200ms.

The classic example is a hostile interaction with delayed ACKs: Nagle holds a packet waiting for an ACK, but the receiver delays the ACK waiting for more data to ack-piggyback, and both sides sit there waiting for each other for 200ms.

### TCP_NODELAY: disabling Nagle

If you have a chatty app and want every write to go out immediately, set `TCP_NODELAY` on the socket. This disables Nagle. Every write becomes its own segment.

```c
int flag = 1;
setsockopt(fd, IPPROTO_TCP, TCP_NODELAY, &flag, sizeof(flag));
```

Most interactive protocols (SSH, real-time games, low-latency RPCs) set TCP_NODELAY. Most bulk-transfer protocols don't.

### TCP_CORK: the opposite

`TCP_CORK` is the opposite of NODELAY. It tells the kernel to **not send anything** until the cork is removed. This lets the application batch writes into one big segment manually. Useful for sending headers and bodies together (e.g., HTTP servers that want headers + small response in one packet).

### TCP_QUICKACK

`TCP_QUICKACK` tells the receiver to **ack immediately**, skipping delayed ACKs. Useful for interactive protocols that need fast feedback.

## Keepalive

If a TCP connection sits idle, neither side knows if the other is still alive. Maybe the peer crashed. Maybe the network died. Maybe a router in the middle dropped the connection's state. Without traffic, you can't tell.

**SO_KEEPALIVE** is a socket option that asks the kernel to send periodic empty probes to detect dead peers.

```c
int on = 1;
setsockopt(fd, SOL_SOCKET, SO_KEEPALIVE, &on, sizeof(on));
```

After a configurable idle period, the kernel sends a keepalive probe. If the peer is alive, it responds normally. If the peer is dead (or the network is broken), the kernel retries a few times and eventually marks the connection broken.

The defaults on Linux are usually:

- `tcp_keepalive_time = 7200` (2 hours of idle before first probe)
- `tcp_keepalive_intvl = 75` (seconds between probes after first)
- `tcp_keepalive_probes = 9` (number of probes before giving up)

Two hours of idle is usually too long. If a connection idles for 30 minutes and the peer dies, you'd still wait 90 more minutes before noticing. For most production servers, you'd want shorter values:

```bash
$ sudo sysctl -w net.ipv4.tcp_keepalive_time=300    # 5 min
$ sudo sysctl -w net.ipv4.tcp_keepalive_intvl=30    # 30 sec
$ sudo sysctl -w net.ipv4.tcp_keepalive_probes=4    # 4 tries
```

That gives you "dead connection detected within ~7 minutes."

You can also set per-connection keepalive with `TCP_KEEPIDLE`, `TCP_KEEPINTVL`, `TCP_KEEPCNT` socket options.

Many application protocols build their own keepalive at the application layer (HTTP/2 pings, WebSocket pings, etc.) for finer-grained control.

## TCP Options (Brief Tour)

The TCP header has a fixed 20-byte minimum, but it can carry options at the end. Options are how features get added without breaking old TCP. Each option is a TLV: type, length, value. Here are the options you'll see most often.

**MSS (Maximum Segment Size).** Sent in SYN. Tells the peer the largest segment we can receive (typically MTU - 40 = 1460 bytes for Ethernet). Both sides use the smaller of the two MSSes.

**Window Scale.** Sent in SYN. Multiplies the receive window field by 2^N. Up to N=14. Negotiated once, applies to all subsequent packets in this connection.

**SACK Permitted.** Sent in SYN. Says "I support selective acknowledgments." If both sides send this, SACK can be used.

**Timestamps.** Each segment carries a send timestamp; ACKs echo it back. Lets the sender measure RTT exactly even on retransmits. Also helps detect old wrapped sequence numbers (PAWS — Protection Against Wrapped Sequences).

**TCP MD5 Signature.** Adds a 16-byte signed hash of the segment + a shared secret. Used by BGP to authenticate router peering. Mostly only seen between routers.

**TCP-AO (Authentication Option).** Modern replacement for MD5. Better algorithms.

**TFO (TCP Fast Open).** Lets a client send data in the SYN packet itself, eliminating one round trip for repeat connections. Requires a cookie negotiated on a previous connection. Saves real time on every HTTPS connection if both sides support it.

You can see whether TFO is enabled:

```bash
$ cat /proc/sys/net/ipv4/tcp_fastopen
1
```

## TCP and the Linux Kernel

Just to peek under the hood: in the Linux kernel, every TCP connection lives in a struct called `tcp_sock`. It has hundreds of fields tracking state: the sequence numbers, the windows, the timers, the congestion control state, the buffers, everything.

The retransmission queue is a list of segments that have been sent but not yet acknowledged. As ACKs come in, they get pulled off the front of the queue. If RTO fires, the segment at the front gets retransmitted.

Congestion control is **pluggable.** You can switch between Reno, CUBIC, BBR, and others without restarting:

```bash
$ ls /lib/modules/$(uname -r)/kernel/net/ipv4/ | grep tcp_
tcp_bbr.ko
tcp_bic.ko
tcp_cubic.ko
tcp_dctcp.ko
tcp_diag.ko
tcp_highspeed.ko
tcp_hybla.ko
tcp_htcp.ko
tcp_illinois.ko
tcp_lp.ko
tcp_nv.ko
tcp_scalable.ko
tcp_vegas.ko
tcp_veno.ko
tcp_westwood.ko
tcp_yeah.ko
```

Each one is a kernel module implementing a specific congestion algorithm.

You can change the default for new connections:

```bash
$ sudo sysctl -w net.ipv4.tcp_congestion_control=bbr
```

And see which is active:

```bash
$ cat /proc/sys/net/ipv4/tcp_congestion_control
bbr
```

## Common TCP Errors and Their Causes

When TCP goes wrong, you see specific error messages. Here's what each one means and how to fix it.

### "Connection refused"

You tried to connect to a host:port, the host responded, but the port has no listener. The kernel sent back a TCP RST (reset) packet. Your application got back ECONNREFUSED.

**Causes:**
- The service isn't running on that port.
- The service crashed.
- A firewall is intercepting and synthesizing the RST (less common; usually firewalls just drop).

**How to check:**
```bash
$ ss -tlnp | grep :8080
LISTEN 0  511  0.0.0.0:8080  0.0.0.0:*  users:(("myserver",pid=1234,fd=5))
```

If nothing shows, no listener.

### "Connection timed out"

You sent a SYN, got no SYN-ACK back. The kernel retried for some time (tcp_syn_retries) and gave up.

**Causes:**
- The host is offline or unreachable.
- A firewall silently drops your SYN.
- The port is firewalled.
- Network routing is broken.

**How to check:**
```bash
$ tracepath example.com
$ mtr -rwbz4 example.com | head -20
$ nc -zv example.com 443  # try the connection itself
```

### "Connection reset by peer"

You had an established connection. The peer sent a RST. Your read or write got back ECONNRESET.

**Causes:**
- The peer process crashed.
- The peer process was killed.
- The peer's kernel is dropping the connection (no socket exists for incoming traffic).
- A NAT or firewall in the middle dropped state and is sending RSTs to clean up.
- An attacker is injecting RSTs.

**How to check:**
- Look at peer's logs for crashes.
- Check NAT/firewall state.
- `tcpdump` to see if RST is coming from peer or middle.

### "Address already in use"

You tried to bind a socket to a local port, but something already has it.

**Causes:**
- An old process is still running on that port.
- The port is in TIME_WAIT (rebinding requires SO_REUSEADDR).
- Two processes both want the same port.

**How to check:**
```bash
$ ss -tanp | grep :8080
```

**Fix:**
- Set `SO_REUSEADDR` on the socket before binding.
- Wait for TIME_WAIT to clear.
- Kill the conflicting process.

### "Broken pipe" or "EPIPE"

You wrote to a connection that the peer has closed. The kernel sent SIGPIPE to your process (which by default kills it) or returned EPIPE on write.

**Causes:**
- The peer closed normally before you finished writing.
- The peer crashed.
- You wrote after a half-close.

**Fix:**
- Ignore SIGPIPE (`signal(SIGPIPE, SIG_IGN)`) and handle EPIPE in your code.
- Check that the connection is still alive before writing.

### "Too many open files"

You ran out of file descriptors. Each TCP connection eats a file descriptor.

**Causes:**
- File descriptor limit is too low.
- File descriptor leak (you're not closing connections).

**Check:**
```bash
$ ulimit -n
1024
$ cat /proc/sys/fs/file-max
9223372036854775807
$ ls -l /proc/$(pgrep myapp)/fd | wc -l
```

**Fix:**
- Raise ulimit: `ulimit -n 65535` (or in /etc/security/limits.conf).
- Fix the leak.

### "No buffer space available"

The kernel ran out of memory for socket buffers. You can't send.

**Causes:**
- Memory pressure on the host.
- TCP send/recv buffers exhausted.
- Network queue tail-drop somewhere.

**Check:**
```bash
$ cat /proc/net/sockstat
$ sysctl net.ipv4.tcp_mem
```

## Hands-On

These commands work on Linux. You can run all of them safely. Let's see what TCP looks like on your own machine.

### See all TCP sockets

```bash
$ ss -tan
State    Recv-Q  Send-Q  Local Address:Port    Peer Address:Port
LISTEN   0       4096    127.0.0.53%lo:53      0.0.0.0:*
LISTEN   0       128     0.0.0.0:22            0.0.0.0:*
ESTAB    0       0       192.168.1.10:54123    142.250.80.46:443
ESTAB    0       0       192.168.1.10:36012    52.84.150.10:443
TIME-WAIT 0      0       192.168.1.10:35221    93.184.216.34:80
```

`-t` = TCP, `-a` = all (including listeners), `-n` = numeric (don't resolve names).

### See only listening sockets

```bash
$ ss -tanl
State   Recv-Q  Send-Q  Local Address:Port  Peer Address:Port
LISTEN  0       128     0.0.0.0:22          0.0.0.0:*
LISTEN  0       4096    127.0.0.53%lo:53    0.0.0.0:*
LISTEN  0       511     127.0.0.1:631       0.0.0.0:*
```

`-l` filters to listening only. These are servers waiting for connections.

### See process info per socket (needs sudo)

```bash
$ sudo ss -tanp
State  Recv-Q  Send-Q  Local Address:Port  Peer Address:Port  Process
LISTEN 0       128     0.0.0.0:22          0.0.0.0:*          users:(("sshd",pid=1234,fd=3))
ESTAB  0       0       192.168.1.10:54123  142.250.80.46:443  users:(("firefox",pid=4567,fd=42))
```

Now you see *which* process owns each socket.

### See detailed info per socket

```bash
$ sudo ss -tnpi state established
ESTAB 0 0 192.168.1.10:54123 142.250.80.46:443 users:(("firefox",pid=4567,fd=42))
   cubic wscale:7,7 rto:204 rtt:3.42/1.05 ato:40 mss:1460 cwnd:10 ssthresh:7 bytes_sent:1234 bytes_acked:1234 segs_out:42 segs_in:38 send 34Mbps lastsnd:1024 lastrcv:512 lastack:512 pacing_rate 67Mbps rcv_space:14600 minrtt:2.1
```

Look at all that detail. `cubic` is the congestion control. `cwnd:10` is the congestion window in segments. `rtt:3.42` is the smoothed RTT in ms. `mss:1460` is the maximum segment size. `bytes_acked:1234` is how much data has been acknowledged.

### Filter by destination port

```bash
$ sudo ss -ti dst :443
ESTAB 0 0 192.168.1.10:54123 142.250.80.46:443
   cubic wscale:7,7 rto:204 rtt:3.42/1.05 ...
ESTAB 0 0 192.168.1.10:36012 52.84.150.10:443
   cubic wscale:7,7 rto:208 rtt:5.12/2.31 ...
```

All connections to port 443 (HTTPS).

### TCP summary

```bash
$ ss -s
Total: 1234
TCP:   456 (estab 12, closed 421, orphaned 0, timewait 19)

Transport Total     IP        IPv6
RAW       0         0         0
UDP       8         5         3
TCP       35        20        15
INET      43        25        18
FRAG      0         0         0
```

Quick overview of how many sockets exist, by type.

### See TCP via the older netstat

```bash
$ netstat -ant | head -10
Active Internet connections (servers and established)
Proto Recv-Q Send-Q Local Address           Foreign Address         State
tcp        0      0 0.0.0.0:22              0.0.0.0:*               LISTEN
tcp        0      0 127.0.0.53:53           0.0.0.0:*               LISTEN
tcp        0      0 192.168.1.10:54123      142.250.80.46:443       ESTABLISHED
```

Older tool. `ss` is faster and more featureful, but `netstat` still works.

### Read raw kernel TCP table

```bash
$ cat /proc/net/tcp | head -5
  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode
   0: 0100007F:0277 00000000:0000 0A 00000000:00000000 00:00000000 00000000     0        0 23456 1
   1: 0100007F:0035 00000000:0000 0A 00000000:00000000 00:00000000 00000000   101        0 12345 1
   2: 0100007F:0019 00000000:0000 0A 00000000:00000000 00:00000000 00000000     0        0 11111 1
```

This is the raw kernel data structure. Addresses are hex little-endian. `st` is the state in hex (`0A` = LISTEN, `01` = ESTABLISHED, `06` = TIME_WAIT, etc.). Useful for parsing programmatically.

### See socket statistics

```bash
$ cat /proc/net/sockstat
sockets: used 312
TCP: inuse 18 orphan 0 tw 12 alloc 19 mem 5
UDP: inuse 8 mem 4
UDPLITE: inuse 0
RAW: inuse 0
FRAG: inuse 0 memory 0
```

How many sockets are in use, in TIME_WAIT, etc.

### Current congestion control algorithm

```bash
$ cat /proc/sys/net/ipv4/tcp_congestion_control
cubic
```

### Available congestion control algorithms

```bash
$ cat /proc/sys/net/ipv4/tcp_available_congestion_control
reno cubic bbr
```

### Look at all TCP-related sysctls

```bash
$ sysctl -a 2>/dev/null | grep -E "tcp_(rmem|wmem|fin_timeout|tw_reuse|keepalive|fastopen)"
net.ipv4.tcp_fastopen = 1
net.ipv4.tcp_fin_timeout = 60
net.ipv4.tcp_keepalive_intvl = 75
net.ipv4.tcp_keepalive_probes = 9
net.ipv4.tcp_keepalive_time = 7200
net.ipv4.tcp_rmem = 4096    131072  6291456
net.ipv4.tcp_tw_reuse = 2
net.ipv4.tcp_wmem = 4096    16384   4194304
```

A peek at how TCP is tuned on your system.

### Capture some TCP packets

```bash
$ sudo tcpdump -i any -n -c 20 'tcp and port 443'
listening on any, link-type LINUX_SLL
14:23:01.123456 IP 192.168.1.10.54123 > 142.250.80.46.443: Flags [S], seq 12345, win 64240, options [mss 1460,sackOK,TS val 1,nop,wscale 7], length 0
14:23:01.234567 IP 142.250.80.46.443 > 192.168.1.10.54123: Flags [S.], seq 67890, ack 12346, win 65535, options [mss 1460,sackOK,TS val 9999,nop,wscale 8], length 0
14:23:01.234890 IP 192.168.1.10.54123 > 142.250.80.46.443: Flags [.], ack 67891, win 64240, length 0
```

See the SYN, SYN-ACK, ACK three-way handshake right there. `[S]` is SYN, `[S.]` is SYN+ACK, `[.]` is ACK.

### Filter to only SYNs and FINs

```bash
$ sudo tcpdump -i any -n -c 20 'tcp[tcpflags] & (tcp-syn|tcp-fin) != 0'
14:23:01.123456 IP 192.168.1.10.54123 > 142.250.80.46.443: Flags [S], seq 12345
14:23:01.234567 IP 142.250.80.46.443 > 192.168.1.10.54123: Flags [S.], seq 67890, ack 12346
14:23:05.567890 IP 192.168.1.10.54123 > 142.250.80.46.443: Flags [F.], seq 12345
14:23:05.678901 IP 142.250.80.46.443 > 192.168.1.10.54123: Flags [F.], seq 67890, ack 12346
```

Just connection setups and teardowns.

### Listen on a TCP port (manual server)

```bash
$ nc -l 8080
```

Now `nc` is listening on port 8080. In another terminal:

### Connect to that listener

```bash
$ nc 127.0.0.1 8080
```

Type something. It shows on the listener's screen. Type something on the listener's side. It shows on the connector's screen. Now you have a real TCP connection between two terminals.

### Quick port check

```bash
$ nc -zv example.com 443
Connection to example.com (93.184.216.34) 443 port [tcp/https] succeeded!
```

`-z` = scan only (don't send any data), `-v` = verbose. Just tells you if the port is open.

### See an HTTPS connection in detail

```bash
$ curl -v --max-time 5 https://example.com 2>&1 | head -20
*   Trying 93.184.216.34:443...
* Connected to example.com (93.184.216.34) port 443 (#0)
* ALPN, offering h2
* ALPN, offering http/1.1
* TLSv1.3 (OUT), TLS handshake, Client hello (1):
* TLSv1.3 (IN), TLS handshake, Server hello (2):
* TLSv1.3 (IN), TLS handshake, Encrypted Extensions (12):
* TLSv1.3 (IN), TLS handshake, Certificate (11):
...
```

Watch the connection happen blow-by-blow.

### Trace path to a host

```bash
$ tracepath example.com
 1?: [LOCALHOST]                      pmtu 1500
 1:  192.168.1.1                                          1.234ms
 1:  192.168.1.1                                          1.345ms
 2:  10.0.0.1                                             5.678ms
 3:  ...
```

Shows each hop on the route to a destination.

### Better trace with mtr

```bash
$ mtr -rwbz4 example.com | head -20
HOST: myhost              Loss%   Snt   Last   Avg  Best  Wrst StDev
  1.|-- 192.168.1.1        0.0%    10    1.2   1.3   1.1   1.5   0.1
  2.|-- 10.0.0.1           0.0%    10    5.6   5.7   5.5   5.9   0.1
  3.|-- 172.16.0.1         0.0%    10   12.3  12.4  12.1  12.6   0.1
  ...
```

Continuous path probing, useful for finding intermittent issues. `-r` report mode, `-w` wide, `-b` show IP, `-z` AS numbers, `-4` IPv4.

### Show cached TCP metrics per peer

```bash
$ ip tcp_metrics show | head -20
142.250.80.46 age 142.123sec cwnd 16 rtt 3500us rttvar 1500us source 192.168.1.10
93.184.216.34 age 67.456sec cwnd 12 rtt 18000us rttvar 4500us source 192.168.1.10
```

Linux caches per-peer TCP metrics so new connections to the same peer can start with a sensible cwnd. Aging out after a while.

### Flush the TCP metrics cache (root)

```bash
$ sudo ip tcp_metrics flush all
```

Useful if you're testing and want a clean slate.

### See TCP-related NIC offload features

```bash
$ ethtool -k eth0 | grep -i tcp
tcp-segmentation-offload: on
tx-tcp-segmentation: on
tx-tcp-ecn-segmentation: on
tx-tcp-mangleid-segmentation: off
tx-tcp6-segmentation: on
```

Modern NICs do a lot of TCP work in hardware. TSO (TCP Segmentation Offload) lets the NIC split big chunks into MSS-sized segments. Saves CPU.

### See what the kernel has logged about TCP

```bash
$ dmesg | grep -i tcp | tail -10
[12345.678] TCP: request_sock_TCP: Possible SYN flooding on port 80. Sending cookies.
[12346.789] tcp_metrics: Hash table 16384 entries
```

Useful for spotting attacks or weird events.

### eBPF: count connections by process

```bash
$ sudo bpftrace -e 'kprobe:tcp_sendmsg { @[comm] = count(); }'
Attaching 1 probe...
^C

@[chrome]: 234
@[firefox]: 1245
@[curl]: 12
@[ssh]: 4
```

Hooks the kernel's `tcp_sendmsg` function and counts calls by process name. eBPF is amazing for this kind of observation.

### Watch a connection grow

```bash
$ watch -n 1 'ss -tnpi state established | head -20'
```

Refreshes once per second so you can see RTT, cwnd, etc. evolve.

## Common Confusions

### "TIME_WAIT is dangerous, I should disable it!"

No. TIME_WAIT is **good.** It exists to make sure delayed packets from old connections don't corrupt new ones. The right tuning is `tcp_tw_reuse=1` (allow reusing TIME_WAIT sockets for new outgoing connections — safe). Don't try to "disable" TIME_WAIT, and especially don't use the long-deprecated `tcp_tw_recycle` which was removed in Linux 4.12 because it broke things behind NAT.

### "Why did my connection RESET?"

A RST means one of three things: (a) the peer crashed and its kernel cleaned up the socket; (b) a stateful firewall or NAT in the middle dropped the connection's state and is sending RSTs to clean up; (c) somebody is injecting fake RSTs (a real but specific attack). Capture with tcpdump and look at the source of the RST. If it comes from the peer's IP, the peer crashed or closed abruptly. If it comes from somewhere else, that's your firewall.

### "Why is my throughput stuck below my link speed?"

The most common cause is that your **bandwidth-delay product (BDP)** exceeds your receive window. Throughput is limited by `min(rwnd, cwnd) / RTT`. If your link is fast and the RTT is high, you need a big window. Check `tcp_rmem` and `tcp_wmem` — the kernel auto-tunes but might not hit your needs for very high BDP. You might need to bump up the maximums. Also try BBR instead of CUBIC for long-fat-network paths.

### "Why does Nagle hurt my latency?"

Nagle (the algorithm, not the person) holds small writes briefly to batch them. For chatty interactive apps (small messages with quick replies), this delay is a killer. The fix is `setsockopt(fd, IPPROTO_TCP, TCP_NODELAY, &on, sizeof(on))`. Most interactive protocols already set this. Most bulk-transfer protocols don't bother.

### "Should I use SO_LINGER with timeout 0 for clean close?"

Almost never. SO_LINGER with timeout 0 sends a RST instead of a FIN, which avoids TIME_WAIT but discards any unsent data and tells the peer "abort." That breaks most protocols. The right answer is to call close() normally and let the kernel handle TIME_WAIT (which is fine).

### "Why does my server leak CLOSE_WAIT sockets?"

This is always a bug in the application. CLOSE_WAIT means the peer sent a FIN but the application hasn't called close() yet. If thousands accumulate, your code is forgetting to close sockets when reads return 0 (EOF) or when errors occur. Find the missing close() in your code.

### "Why is my SYN_SENT count growing?"

Either the destination is unreachable (you're trying lots of dead hosts) or your DNS is returning bad addresses. Check with `ss -tnp state syn-sent` and look at the targets.

### "Why are my connections slow to start?"

Slow start. TCP starts with a small cwnd and ramps up. For short transactions (like loading a small webpage), slow start is the dominant cost. TCP Fast Open helps for repeat connections. HTTP/2 helps by reusing connections.

### "Why does packet capture show retransmits but the connection still works?"

TCP recovers from loss. Some retransmissions are normal on lossy paths. Look at the **rate** of retransmissions: if it's a few percent, fine. If it's 30%, you have a real problem.

### "Why is my server's accept queue full?"

The accept queue (between SYN_RECEIVED and the application's accept() call) is bounded by `min(somaxconn, listen() backlog)`. If your application is too slow to accept, the queue fills up and new connections get dropped. Check `ss -tlnp` — the `Send-Q` for a LISTEN socket shows the queue depth. Tune `net.core.somaxconn` upward and pass a bigger backlog to listen().

### "Why are bytes appearing out of order in my application?"

They aren't. TCP guarantees in-order delivery to the application. If you see out-of-order data, you're not actually using TCP, or you're using `recv` wrong, or you have multiple connections you're treating as one.

### "Why does my client get an immediate RST when the server is up?"

Either the server's listening on a different port than you think, or the listening socket's accept queue is full and the kernel is rejecting new SYNs (look at `dmesg` for SYN flood messages). Check `ss -tlnp` to see what's actually listening.

### "Why is throughput great in one direction but terrible in the other?"

Asymmetric paths and asymmetric tuning. The path back might have higher loss, smaller buffers, or different congestion control. Captures in both directions help.

### "Why does my long-running connection get killed by a NAT?"

Stateful NATs have an idle timeout. If you don't send anything for some minutes, the NAT forgets your mapping and silently drops. Subsequent packets get dropped (or get a RST from the NAT). Fix: enable TCP keepalive with a shorter interval than the NAT timeout, so traffic always flows.

## Vocabulary

This is the dictionary. If you ever forget what a word means, look here.

- **TCP** — Transmission Control Protocol. Reliable, in-order, connection-oriented protocol on top of IP.
- **UDP** — User Datagram Protocol. Unreliable, connectionless. The fast-and-loose cousin of TCP.
- **IP** — Internet Protocol. The layer below TCP. Routes packets but doesn't guarantee delivery.
- **segment** — A TCP-layer packet. The TCP unit of data.
- **packet** — Generic term for a chunk of network data.
- **datagram** — A self-contained packet, especially in UDP context.
- **frame** — A link-layer chunk. Below packets. Ethernet frames carry IP packets which carry TCP segments.
- **sequence number** — The byte offset within a TCP stream. 32-bit. Increments by 1 per byte.
- **ACK (acknowledgment)** — A TCP message confirming receipt of bytes.
- **SACK (Selective ACK)** — A TCP option for saying "I have these specific ranges of bytes" beyond the cumulative ACK.
- **FIN** — A TCP flag meaning "I'm done sending."
- **RST** — A TCP flag meaning "abort this connection now."
- **SYN** — A TCP flag meaning "synchronize starting sequence numbers" (used in handshake).
- **PSH** — A TCP flag meaning "push this data to the application now."
- **URG** — A TCP flag meaning "urgent data." Largely obsolete.
- **MSS (Maximum Segment Size)** — The largest amount of data a TCP segment can carry. Negotiated at handshake.
- **MTU (Maximum Transmission Unit)** — The largest IP packet a link can carry. MSS = MTU - 40 bytes for typical TCP/IPv4.
- **MSL (Maximum Segment Lifetime)** — The longest a packet can live in the network before being discarded.
- **TIME_WAIT** — The state after closing where you wait 2 × MSL to clean up.
- **CLOSE_WAIT** — The state where the peer has sent FIN but you haven't called close() yet.
- **ESTABLISHED** — The state where data flows freely.
- **LISTEN** — The state where a server waits for incoming connections.
- **SYN_SENT** — The client state after sending SYN, waiting for SYN-ACK.
- **SYN_RECEIVED** — The server state after sending SYN-ACK, waiting for ACK.
- **FIN_WAIT_1** — Sent FIN, waiting for ACK of it.
- **FIN_WAIT_2** — Got ACK of our FIN, waiting for peer's FIN.
- **half-open** — A connection where one side thinks it's open and the other doesn't.
- **half-close** — A connection where one direction is closed but the other isn't.
- **three-way handshake** — SYN, SYN-ACK, ACK. The setup.
- **four-way teardown** — FIN, ACK, FIN, ACK. The graceful close.
- **congestion window (cwnd)** — How much TCP thinks the network can handle. Sender-side.
- **receive window (rwnd)** — How much the receiver can swallow. Receiver-side.
- **slow start** — The initial phase where cwnd doubles per RTT.
- **ssthresh** — Slow start threshold. When cwnd hits this, switch to congestion avoidance.
- **congestion avoidance** — The phase where cwnd grows by +1 per RTT.
- **fast retransmit** — Retransmit on 3 duplicate ACKs without waiting for RTO.
- **fast recovery** — After fast retransmit, cut cwnd in half instead of going back to slow start.
- **Reno** — Classic TCP congestion control algorithm.
- **CUBIC** — Modern Linux default. Uses cubic curve.
- **BBR** — Google's bandwidth-based algorithm.
- **BBRv2** — Improved BBR, friendlier to other flows.
- **RTT (Round-Trip Time)** — The time for a packet to go to the peer and the ACK to come back.
- **RTO (Retransmission Timeout)** — How long to wait for an ACK before retransmitting.
- **RTTVAR** — Variance in RTT measurements.
- **SRTT** — Smoothed RTT. The averaged RTT.
- **Karn's algorithm** — Don't measure RTT on retransmitted segments.
- **RACK** — Recent Acknowledgment. Modern loss detection. RFC 8985.
- **TLP (Tail Loss Probe)** — Probe to recover the last lost packet of a burst quickly.
- **Nagle** — Algorithm that batches small writes.
- **TCP_NODELAY** — Disables Nagle.
- **TCP_QUICKACK** — Disables delayed ACKs.
- **TCP_CORK** — Holds writes until uncorked, for batching.
- **TCP_KEEPIDLE** — Idle time before first keepalive probe.
- **TCP_KEEPINTVL** — Interval between keepalive probes.
- **TCP_KEEPCNT** — Number of keepalive probes before declaring dead.
- **SO_KEEPALIVE** — Socket option to enable keepalive probes.
- **SO_LINGER** — Controls behavior of close().
- **SO_REUSEADDR** — Socket option to allow rebinding to a TIME_WAIT port.
- **SO_REUSEPORT** — Socket option for multiple processes to bind the same port.
- **accept queue** — The queue of fully-established connections waiting for accept().
- **somaxconn** — Kernel cap on the accept queue depth.
- **syn queue** — The queue of half-open connections (SYN received, awaiting ACK).
- **tcp_max_syn_backlog** — Kernel cap on the syn queue.
- **syncookies** — A SYN flood defense that encodes state into the SYN-ACK.
- **TFO (TCP Fast Open)** — Lets clients send data in the SYN.
- **MD5 signature** — Old TCP authentication option, mostly used by BGP.
- **TCP-AO** — Modern TCP authentication option.
- **ECN (Explicit Congestion Notification)** — Routers can mark packets to signal congestion without dropping.
- **CWR** — Congestion Window Reduced flag. Part of ECN.
- **ECT** — ECN-Capable Transport. Marks packets as ECN-eligible.
- **URG pointer** — Field in the TCP header for urgent data. Largely obsolete.
- **push flag** — Tells the receiver to deliver data to the application now.
- **half-duplex close** — Close one direction at a time.
- **self-clocking** — TCP regulates send rate based on incoming ACK rate.
- **bandwidth-delay product (BDP)** — bandwidth × RTT. The amount of data that fits in the pipe at once.
- **AQM (Active Queue Management)** — Routers managing their queues actively.
- **CoDel** — A modern AQM algorithm.
- **FQ** — Fair Queueing.
- **FQ-CoDel** — Fair Queueing + CoDel. Common Linux qdisc.
- **tc** — Linux traffic control utility.
- **qdisc** — Queueing discipline. The thing that queues packets.
- **ip_local_port_range** — Range of ports the kernel uses for ephemeral connections.
- **ephemeral port** — A short-lived port assigned by the kernel for an outgoing connection.
- **listen backlog** — How many pending connections a listen() can queue.
- **SYN flood** — DoS attack of sending many SYNs without completing handshakes.
- **RST attack** — Injecting fake RSTs to terminate connections.
- **port scan** — Probing many ports to find which are open.
- **conntrack** — Linux's connection tracking subsystem.
- **NAT** — Network Address Translation. Mapping internal addresses to external.
- **hairpin** — Traffic from inside a NAT going to another inside host via the NAT.
- **MSS clamping** — Modifying the MSS option in passing SYNs to fit into a tunnel.

## Try This

Real experiments you can run right now to see TCP in action.

### Experiment 1: Capture a complete three-way handshake

In one terminal:
```bash
$ sudo tcpdump -i any -n -c 10 'tcp and port 8080'
```

In another terminal, start a listener:
```bash
$ nc -l 8080
```

In a third terminal:
```bash
$ nc 127.0.0.1 8080
```

Watch the tcpdump output. You should see:

```
14:23:01.111 IP 127.0.0.1.50000 > 127.0.0.1.8080: Flags [S], seq 1
14:23:01.111 IP 127.0.0.1.8080 > 127.0.0.1.50000: Flags [S.], seq 1, ack 2
14:23:01.111 IP 127.0.0.1.50000 > 127.0.0.1.8080: Flags [.], ack 2
```

That's the SYN, SYN-ACK, ACK. You just watched TCP set up a connection.

### Experiment 2: Watch a connection enter TIME_WAIT

In one terminal:
```bash
$ nc -l 8080
```

In another:
```bash
$ nc 127.0.0.1 8080
```

Press Ctrl+D on the **client** (the second nc). It closes. Now in a third terminal, watch the state:

```bash
$ ss -tan | grep 8080
TIME-WAIT  0  0  127.0.0.1:50000  127.0.0.1:8080
LISTEN     0  1  *:8080           *:*
```

You'll see TIME_WAIT for about 60 seconds (Linux default), then it disappears.

### Experiment 3: Switch congestion control and watch

```bash
$ cat /proc/sys/net/ipv4/tcp_congestion_control
cubic

$ sudo sysctl -w net.ipv4.tcp_congestion_control=bbr
net.ipv4.tcp_congestion_control = bbr

$ curl -o /dev/null https://example.com   # use BBR

$ sudo sysctl -w net.ipv4.tcp_congestion_control=cubic   # back to default
```

Compare throughput with both. The difference is most noticeable on lossy long paths.

### Experiment 4: Force a SYN flood log

(Don't actually do this against any production system. Localhost only.)

Start a tiny listener with a small queue:
```bash
$ python3 -c "import socket; s = socket.socket(); s.bind(('127.0.0.1', 8888)); s.listen(1); import time; time.sleep(60)"
```

In another terminal, blast SYNs (mock with hping3 or a similar tool, or just make a lot of fast connections). If you flood enough, dmesg will log:
```bash
$ dmesg | grep -i syn
[12345.678] TCP: Possible SYN flooding on port 8888. Sending cookies.
```

### Experiment 5: Look at your bandwidth-delay product

Find a high-RTT host:
```bash
$ ping -c 4 example.com
PING example.com (93.184.216.34) 56(84) bytes of data.
64 bytes from 93.184.216.34: icmp_seq=1 ttl=56 time=85.2 ms
```

So RTT is about 85ms. Estimate your bandwidth (e.g., 100 Mbps from a speed test). BDP = 100 Mbps × 0.085 s = 8.5 Mbits = ~1 MB. That's how much data needs to be "in flight" to saturate the link. Your TCP window must be at least that big.

### Experiment 6: Compare CUBIC and BBR throughput

```bash
$ sudo sysctl -w net.ipv4.tcp_congestion_control=cubic
$ time curl -o /dev/null --silent https://your-test-server.example.com/big-file
real  0m12.345s

$ sudo sysctl -w net.ipv4.tcp_congestion_control=bbr
$ time curl -o /dev/null --silent https://your-test-server.example.com/big-file
real  0m8.765s
```

BBR often wins on high-RTT paths. Repeat a few times for a fair comparison.

### Experiment 7: Watch RTT change live

```bash
$ ss -tnpi state established | grep -A1 your-target
```

The `rtt:` field shows the smoothed RTT in milliseconds. Run it repeatedly while doing other things on the network. You'll see RTT change as the network gets more or less busy.

### Experiment 8: Make CLOSE_WAIT pile up

Write a buggy server in Python that doesn't close sockets:

```python
import socket
s = socket.socket()
s.bind(('127.0.0.1', 9999))
s.listen()
while True:
    c, a = s.accept()
    # Forgot to close c
```

Then have a client connect and disconnect a few times. Run `ss -tan | grep 9999`. You'll see growing CLOSE_WAIT. That's exactly what bug-leaking-CLOSE_WAIT looks like.

## Where to Go Next

- `cs networking tcp` — dense reference card
- `cs detail networking/tcp` — congestion math, RTT estimation, BBR formula
- `cs networking udp` — the connectionless cousin
- `cs networking ip` — what TCP rides on
- `cs ramp-up udp-eli5`
- `cs ramp-up ip-eli5`
- `cs ramp-up icmp-eli5`
- `cs networking tcpdump` — capture traffic
- `cs ramp-up linux-kernel-eli5` — what's running below

## See Also

- `networking/tcp`
- `networking/udp`
- `networking/ip`
- `networking/ipv4`
- `networking/ipv6`
- `networking/dns`
- `networking/dhcp`
- `networking/tcpdump`
- `networking/quic`
- `troubleshooting/tcp-errors`
- `kernel-tuning/network-stack-tuning`
- `ramp-up/udp-eli5`
- `ramp-up/ip-eli5`
- `ramp-up/icmp-eli5`
- `ramp-up/bgp-eli5`
- `ramp-up/tls-eli5`
- `ramp-up/linux-kernel-eli5`

## References

- RFC 9293 — Transmission Control Protocol (current consolidated)
- RFC 793 — original TCP (1981, historical)
- RFC 5681 — TCP Congestion Control
- RFC 6298 — RTO Calculation
- RFC 7323 — TCP Extensions for High Performance (Window Scale, Timestamps)
- RFC 2018 — TCP Selective Acknowledgment Options
- RFC 7413 — TCP Fast Open
- RFC 8985 — RACK-TLP
- RFC 9293 — Latest TCP roll-up (2022)
- BBR paper (Cardwell et al., Google, 2016)
- man tcp(7), man socket(2), man netstat, man ss
- "TCP/IP Illustrated, Vol 1" by Stevens, Fall

### One last thing before you go

You now know more about TCP than most working programmers. You know what the three-way handshake actually does, why TIME_WAIT exists, what slow start is, why Nagle hurts interactive apps, and how to read `ss -tnpi` output. The next time somebody at work says "the connection is RESETing" or "we have CLOSE_WAIT pileup" or "throughput is stuck," you will know what they mean and where to look.

The internet is a stack of clever ideas piled on top of more clever ideas. TCP is one of the cleverest. It takes a chaotic, lossy, unreliable network and makes it look like a clean stream of bytes to your application. It's been doing that since 1981, on every major operating system, in every data center, on every phone, in every browser. Billions of TCP connections open every second. Each one quietly does the dance you just learned.

Go capture some packets. Watch a real handshake. Read your own kernel's TCP stats. The only way this stuff really clicks is to see it for yourself, on your own machine, in real time. The terminal is the place. You don't need a web browser. You don't need to google anything. You have everything you need right here.
