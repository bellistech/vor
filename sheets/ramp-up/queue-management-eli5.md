# Queue Management — ELI5

> A router is a doorman with a lobby. The lobby has a finite number of chairs. When the chairs fill up and more people keep coming, somebody has to be turned away. The art of deciding *who* and *when* — and *how loudly* the doorman has to slam the door — is queue management.

## Prerequisites

- `ramp-up/tcp-eli5` — you must already understand that TCP slows down when packets are lost and speeds up when they are not. The whole reason queue management is a hard problem is that TCP *reacts* to packet loss. If you don't know what TCP congestion control is, none of the rest of this sheet will land. Go read the TCP ELI5 sheet first, come back when you can explain "slow start" and "congestion window" to a friend.
- `ramp-up/ip-eli5` — you should know what an IP packet is, what a router does, what a "hop" is, and roughly what "buffering" means. The IP ELI5 sheet covers that. Come back when you can draw a packet on a napkin and label the source IP, destination IP, and payload.

If you have those two sheets under your belt, you are ready. If a word feels weird, look it up in the **Vocabulary** table near the bottom (chunk 4). Every weird word in this sheet is in that table with a one-line plain-English definition.

If you see a `$` at the start of a line in a code block, that means "type the rest of this line into your terminal." You do not type the `$`. The lines underneath that don't have a `$` are what your computer prints back at you.

## What Even Is Queue Management

### Imagine a coffee shop with one barista

Picture a coffee shop on the corner of a busy street. Inside, there is exactly one barista. The barista can make exactly one drink per minute. There is a line of customers waiting to order. The customers stand single-file, one behind the other. The first customer in line is the next to be served.

That line is a **queue**. A queue is just an English word for "a line of things waiting to be processed." The barista is the **server**. The customers are **work units**. The whole arrangement — line plus barista — is a **queueing system**.

Now imagine the door is open and people keep walking in. They keep adding themselves to the back of the line. As long as customers walk in slower than the barista can serve them, the line stays short. As long as customers walk in faster than the barista can serve them, the line grows. Forever.

But the coffee shop is not infinitely big. It has walls. There is only so much floor space. At some point, the line bumps up against the back wall and there is no more room. New customers can't get in. Now the doorman has a decision to make: "Sorry, we're full. Come back later."

That decision — *what to do when the line is full* — is **queue management**. There is no clever way around it. If the work is coming in faster than the work is going out, something has to give. The math is simple: arrivals minus departures equals queue growth. If arrivals exceed departures forever, the queue grows forever. Real queues are not infinite. So eventually you have to turn somebody away. The only question is *whom*.

### A router is a coffee shop

A router is doing exactly the same thing as that coffee shop, except it is selling packet-forwarding instead of espresso. A router takes packets in on one interface, looks at the destination IP address, decides which interface to send the packet out, and forwards it. The "barista" is the outgoing interface. The "drink time" is however long it takes to clock the packet's bits onto the wire (1500 bytes at 1 Gbps takes 12 microseconds; at 10 Mbps it takes 1.2 milliseconds; at 56 kbps dial-up it takes 214 milliseconds).

The "line" is the **packet buffer**, sometimes called the **transmit queue** or **egress queue**. It is RAM inside the router. When a packet arrives faster than the outgoing interface can drain it, the packet sits in the buffer until the interface is ready. The buffer holds packets in **first-in-first-out** order. Packet that arrived first leaves first. Just like the coffee shop.

```
            +-------------------+
            |  Router buffer    |
            |  (RAM, finite)    |
arrivals -->| [P5][P4][P3][P2][P1] |--> outgoing link (wire)
            |                   |
            +-------------------+
            "tail"           "head"
            new packets      next packet
            join here        to leave
```

The buffer has a finite size. On a home router it might be 64 packets. On an enterprise router it might be 1000 packets. On a big core router it might be megabytes or gigabytes. The size is set by the manufacturer or by the network operator. Once the buffer is full, the router has to decide what to do with the next packet that arrives.

That decision is queue management. Same word. Same problem. Slightly different vocabulary because we're talking about bytes instead of customers.

### Why does the buffer exist at all?

You might ask: "If the line is bad, why have a line?" The answer is **bursts**. Real network traffic does not arrive at a steady rate. It arrives in clumps. A web page loads, and suddenly fifty packets show up in 10 milliseconds. Then nothing for half a second. Then forty more packets. Then nothing. Then a Zoom call sends a steady 30 packets per second. Then a backup job kicks off and pushes 10,000 packets per second for two minutes.

If the router had **no buffer at all** — if every packet that arrived had to be transmitted on the same nanosecond — the router would have to drop every packet that came in during a burst, even if the average rate was tiny. That would be a disaster. So routers have buffers. The buffer absorbs bursts. Packets pile up briefly, the interface drains them at line rate, and as long as the average arrival rate is below the link rate, everybody gets through.

The buffer is good. The buffer is necessary. The problem is *how big the buffer should be* and *what to do when it fills up anyway*. Those are the two questions queue management answers.

## The Naive Approach: Tail Drop (and why it's terrible)

### What tail drop is

The simplest possible queue management policy is called **tail drop**. The rule is one sentence long:

> When the buffer is full, drop any new packet that arrives.

That's it. The word "tail" is there because new packets join the queue at the **tail end** (the back of the line). When the queue has no more room, those new packets — the ones at the tail — are dropped on the floor. Existing packets in the queue are untouched. They stay in line and get served in order. Only the new arrivals suffer.

Tail drop is the default behavior on almost every router that has ever existed. If you do not configure anything else, you get tail drop. It is the easiest thing in the world to implement: when you go to put a packet in the buffer, check if there's room. If yes, put it in. If no, free the packet's memory and increment a counter called something like `ifOutDiscards` or `tx_drops`.

```bash
# On a Linux box, see tail drops on an interface:
$ ip -s link show eth0
2: eth0: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 ...
    RX: bytes  packets  errors  dropped overrun mcast
    104857600  85234    0       0       0       12
    TX: bytes  packets  errors  dropped overrun carrier
    52428800   42117    0       1342    0       0
                                ^^^^^^
                                these are tail drops
```

The `dropped` counter on the TX line is incremented every time a packet was thrown away because the egress queue was full. That counter is your "doorman turned customers away" counter.

### Why tail drop sounds reasonable

If you have never thought about it before, tail drop sounds fine. You might even argue it is *fair*. Every packet has the same chance of being dropped: zero, until the buffer fills, then 100% if you happen to arrive after the buffer is full. The packets that got there first get served. The packets that got there second wait. The packets that got there too late get dropped. That sounds like how lines work in real life.

It is also extremely cheap. There is no math, no random number generator, no statistics, no measurement of latency. Just a comparison: `if (queue_size >= queue_max) drop;`. A router can do this on every packet without breaking a sweat.

For a long time, this is what every router did. From the 1960s ARPANET to the early 1990s Internet, tail drop was the only game in town. Nobody thought about it because nobody had to. Then traffic got heavy, and three problems showed up that no amount of buffer-sizing could fix.

### Problem 1: Tail drop punishes bursty flows

Imagine two flows sharing the router's outgoing link. Flow A is a steady stream of 100 packets per second. Flow B is a bursty stream that sends 50 packets in one millisecond, then nothing for half a second, then 50 more, then nothing, repeating.

Both flows have the same average rate. If the link can handle 200 packets per second, both flows fit comfortably on average. But during Flow B's burst, the buffer fills up. Tail drop activates. *Whose packets get dropped?* The packets at the tail — which are mostly Flow B's, because Flow B is the one currently bursting. Flow A's steady packets, which were trickling in during the burst, also get caught in the carnage, but Flow B takes most of the hits.

OK, fine, you say — Flow B was the noisy one, it deserves to lose packets. Except the bursts are the *normal* shape of HTTP traffic. Every web page load is a burst. Every video chunk download is a burst. By punishing bursts, tail drop is biased against ordinary, well-behaved traffic. That's annoying, but on its own it is not the killer problem.

### Problem 2: Tail drop wastes the buffer

The second problem is subtler. Look at what tail drop does over time. The buffer sits empty when traffic is light. As load grows, the buffer slowly fills. When the buffer hits 100% full, tail drop activates and packets start dropping. The buffer is now perpetually 100% full as long as load exceeds capacity. Every new packet finds the buffer full, so every new packet faces the worst-case latency: it has to wait behind every other packet in the buffer before it gets served.

That's bad. Big buffers were supposed to absorb bursts and let traffic flow. Instead, big buffers under tail drop become permanent reservoirs of latency. Every packet pays the full buffer's delay. We will come back to this idea — it is the core of bufferbloat. Read on.

### Problem 3: Tail drop synchronizes TCP

This is the killer. This is the one that broke the Internet for two decades and the one we will spend the next section unpacking. Tail drop creates a phenomenon called **TCP global synchronization**, where hundreds or thousands of independent TCP connections all back off at the same time, then all ramp up at the same time, then all hit the wall at the same time, then all back off at the same time. Forever. The link sits half-empty most of the time, then floods, then dies, then sits half-empty, then floods, then dies. Throughput collapses. Latency spikes. Everybody suffers.

This is not a hypothetical. This is what happened on real ISP networks in the 1990s, and it is what still happens today on misconfigured links. We need to understand it.

## TCP Global Synchronization (the saw-tooth dance of 1000 flows backing off in unison)

### Why TCP slows down when it loses packets

Recall from the TCP ELI5 sheet: when TCP detects a lost packet, it interprets that as a signal that the network is congested. Its response is to *slow down*. Specifically, TCP cuts its **congestion window** — the amount of data it is allowed to have in flight — roughly in half. Then it slowly grows the window back, adding one MSS (about 1460 bytes) per round-trip-time. That growth is called **additive increase**. The cut is called **multiplicative decrease**. Together: AIMD — additive increase, multiplicative decrease.

This rule is the entire foundation of TCP congestion control. It is why the Internet works at all. It is also why tail drop is a disaster when many flows share a bottleneck.

### The synchronization mechanism

Here is what happens, step by step, on a tail-dropping router with N TCP flows sharing the outgoing link.

1. Traffic is light. Buffer is small. All N flows are growing their congestion windows in additive-increase mode. Total throughput is climbing slowly.
2. Total traffic crosses the link's capacity. Packets start arriving faster than they can drain. The buffer fills.
3. The buffer hits 100%. Tail drop activates. The router drops the next packet that arrives. That packet belongs to one of the N flows. Call it Flow 1.
4. But wait — the buffer is still full on the next packet. Drop. That belongs to Flow 2. The buffer is *still* full. Drop. Flow 3. Drop. Flow 4. Drop. Drop. Drop. Drop.

In a single millisecond, the router drops one packet from every flow that happens to be sending. Maybe 100 flows. Maybe 1000 flows. Tail drop cannot tell them apart and does not care which flow each dropped packet belongs to. Every flow loses a packet at almost exactly the same time.

5. About one round-trip-time later (say 50 ms), every one of those flows notices the loss. Every one of them cuts its congestion window in half. *At the same time.*
6. Total offered traffic suddenly halves. The buffer drains. The link goes from 100% full to maybe 50% full. Then 30% full. Then 10% full. Then empty.
7. The link is now under-utilized. Lots of capacity is going to waste.
8. All N flows are now in additive-increase mode again. They all start growing their windows. They all grow at the same rate (one MSS per RTT). Their growth curves are nearly identical.
9. Some time later, total offered load again crosses the link's capacity. Buffer fills. Drops happen. Every flow loses a packet at the same time. Every flow halves its window. At the same time.
10. Go to step 6. Repeat. Forever.

### What it looks like

If you were to plot total link utilization over time during this dance, it would look like a saw-tooth wave:

```
 100%|         /|        /|        /|        /|
     |        / |       / |       / |       / |
util |       /  |      /  |      /  |      /  |
     |      /   |     /   |     /   |     /   |
  50%|     /    |____/    |____/    |____/    |____
     |    /          \         \         \
     |   /            (everybody       (and again)
     |  /              halved at once)
   0%|_/_____________________________________________
     +--------------> time
        ramp    drop    ramp    drop    ramp    drop
```

The link is sometimes nearly empty, sometimes 100% full. The *average* utilization is well below 100%. Some studies in the early 1990s found average utilization on synchronized links was as low as 30-50%, even when offered load was 200% of capacity.

That is the core insult: a fully-loaded link, busy enough to be dropping packets constantly, was carrying *less than half* of what it could carry. The router and the link had plenty of capacity. The flows were fighting for that capacity. Tail drop made them all fight in lockstep.

### Why "global"?

The word **global** means "every flow on the link, not just one or two." A localized synchronization (a few flows happening to back off at the same time by coincidence) is normal and unavoidable. *Global* synchronization is when essentially every active flow becomes phase-locked. That requires a coordinating mechanism. Tail drop is that mechanism: it drops packets from many flows at the exact same instant, because that is when the buffer fills.

In the 1980s and early 1990s, this was the dominant pathology of the Internet. Sally Floyd and Van Jacobson — both at the Lawrence Berkeley Laboratory — described it formally and proposed the first widely-deployed fix in their 1993 paper *Random Early Detection Gateways for Congestion Avoidance*. We will read that paper, in plain English, in two sections. First, we have to talk about what happened when network operators tried to fix the problem the wrong way.

## Bufferbloat (Jim Gettys' 2010 wake-up call: bigger buffer ≠ better)

### "Just add more memory"

By the 2000s, RAM was cheap. Network engineers had a problem — packets were being dropped, which TCP interpreted as congestion, which made TCP back off, which felt slow to users — and a tool sitting right there on the workbench: more buffer. If a 64-packet buffer drops packets, what about a 128-packet buffer? What about 1024? What about 8192? What about ten megabytes? What about a hundred?

This thinking was particularly common in consumer-grade equipment: home routers, cable modems, DSL modems, mobile phones. The vendors looked at it like this: "Our customers complain when they see packet loss. Memory is now free. Let's give them so much buffer they will never see a drop again." So they did. Some home cable modems shipped with multi-megabyte transmit buffers. Some 3G cellular base stations shipped with seconds of buffering.

This sounds like it would be great. Eliminate drops! Eliminate the trigger for TCP back-off! Faster Internet for everyone!

Instead, it broke the Internet differently.

### What bufferbloat is

In 2010, Jim Gettys, an engineer at Bell Labs and one of the original authors of the X Window System, started measuring his home cable modem and noticed that pings to anywhere on the Internet were taking *seconds* — not milliseconds — whenever he was uploading a file. He coined the term **bufferbloat** to describe the syndrome and wrote a series of blog posts and academic papers that became, over the next few years, one of the most influential changes to networking thinking since TCP itself.

The core insight of bufferbloat is this:

> A buffer absorbs bursts. A buffer that is *always full* does not absorb bursts; it just adds latency. Big buffers, under TCP, become permanent reservoirs of stale data.

Here is why. TCP has no way to know how big a buffer is. TCP only knows two things: (1) is the buffer ever dropping my packets? and (2) how long is the round-trip-time? TCP grows its congestion window until *something* happens to push back. Either a drop, or a persistently rising RTT. With tail drop and a tiny buffer, TCP grows until a drop happens. The buffer maybe gets half-full before the drop. RTTs are reasonable.

With tail drop and a *huge* buffer, TCP grows and grows and grows. The buffer fills with TCP's data. No drops happen, because the buffer is huge. TCP keeps growing. The buffer fills more. Eventually the buffer is 100% full. *Now* drops happen. But by that point, the buffer is full of stuff that has to drain at line rate. Every packet in the buffer pays the full latency of every packet ahead of it.

If the link is 1 Mbps and the buffer is 1 megabyte, the worst-case latency through the buffer is **8 seconds**. Eight thousand milliseconds. To send a single ping packet across a buffered link, you have to wait behind every byte already queued. That's why Gettys saw multi-second pings.

### A diagram of bufferbloat in action

```
Time = 0    [empty buffer]                   |link|
            "ping reply: 20 ms"

Time = 1 s  TCP upload starts.
            TCP grows window.

Time = 5 s  [############----]               |link|
            buffer 75% full
            "ping reply: 1500 ms"

Time = 10 s [################]               |link|
            buffer 100% full, drops start
            "ping reply: 2000 ms (worst case)"

Time = 11 s buffer always 100% full.
            new packets queue at tail.
            ping packets from your VoIP call:
            8 seconds of waiting.
            VoIP unusable.
```

While the buffer is full, every other connection sharing the link suffers, too. Your VoIP call drops. Your video chat freezes. Your DNS lookups time out. Your SSH session lags. Your web pages take forever to start loading because the SYN handshake is sitting in the buffer behind a megabyte of bulk data.

The link has plenty of throughput. The link is delivering data at full line rate. But it is delivering data with seconds of delay. *Throughput is fine; latency is destroyed.* That is bufferbloat.

### Who got hit hardest?

Bufferbloat was worst on:

1. **Asymmetric residential links** — DSL and cable, where upload bandwidth is much smaller than download. The upload buffer fills first, and ACKs for the download get stuck behind upload data, ruining download throughput too. (This is sometimes called "ACK clocking destruction.")
2. **Cellular networks** — 3G base stations had multi-second buffers because base station vendors did not want to drop packets during handoffs. Result: cellular ping times could be 5-10 seconds during data transfers.
3. **Home Wi-Fi** — early 802.11n equipment had large transmit queues. Mixing a backup with a Zoom call on the same Wi-Fi was a guaranteed Zoom call disaster.
4. **Anywhere with a lot of buffer and TCP not aware of it** — basically, anywhere.

### How to measure bufferbloat right now

If you are reading this in 2026 on a network behind a router you don't control, try this:

```bash
# Terminal 1: start a sustained download to fill the buffer
$ curl -o /dev/null https://example.com/big-file.iso

# Terminal 2: while the download is running, ping your gateway
$ ping -c 30 192.168.1.1
PING 192.168.1.1 (192.168.1.1): 56 data bytes
64 bytes from 192.168.1.1: icmp_seq=0 ttl=64 time=2.3 ms
64 bytes from 192.168.1.1: icmp_seq=1 ttl=64 time=812 ms     <-- yikes
64 bytes from 192.168.1.1: icmp_seq=2 ttl=64 time=1240 ms    <-- bufferbloat
64 bytes from 192.168.1.1: icmp_seq=3 ttl=64 time=1900 ms    <-- terrible
```

If the ping under load is more than 50-100 ms above the ping at idle, you have bufferbloat. The classic web tests for this are at `https://www.waveform.com/tools/bufferbloat` and `https://fast.com` (which Netflix added a "loaded latency" measurement to specifically because of bufferbloat awareness).

### The cure preview

The cure for bufferbloat is **not** "smaller buffers." Smaller buffers cause more drops, which cause TCP to slow down too much. The cure is *active* queue management — drop or signal congestion *before* the buffer is full, so TCP backs off proactively while the buffer stays mostly empty, preserving low latency. This is what RED, WRED, CoDel, PIE, and FQ-CoDel are all about. The next sections introduce them.

## The Latency-vs-Throughput Tradeoff

### Two things you want, and you can't have both at maximum

When you size a buffer, you are trading two desirable properties against each other.

**Throughput** is how much data per second you can move through the link. A bigger buffer absorbs bursts better, which means the link is more often doing useful work and less often sitting idle waiting for the next packet. Bigger buffer → higher throughput, up to a point.

**Latency** is how long an individual packet waits in the buffer before being served. A bigger buffer means more packets in front of any given new arrival, so each packet pays a higher worst-case delay. Bigger buffer → higher latency, always.

You cannot maximize both at once. If your buffer is zero, you have minimum latency (no waiting) but terrible throughput (every burst is dropped, TCP collapses). If your buffer is gigantic, you have maximum throughput (no drops, TCP runs at full speed) but terrible latency (multi-second delays). The right size is somewhere in between, and the sweet spot depends on (a) the link's bandwidth, (b) the typical RTT, and (c) the mix of latency-sensitive vs throughput-sensitive flows.

### The bandwidth-delay product (BDP) rule of thumb

The classical rule, derived by Van Jacobson and others in the 1980s, is:

> Buffer size should be roughly equal to the **bandwidth-delay product** (BDP) of the link.

The BDP is `bandwidth × round-trip-time`. For a 1 Gbps link with 50 ms RTT, the BDP is:

```bash
$ cs calc '1 Gbps * 50 ms / 8'
6.25 MB
```

That is the amount of data that can be "in flight" between sender and receiver at any one time. A buffer that size lets a single TCP flow keep the link 100% full. A smaller buffer leaves capacity unused (TCP's window can't grow large enough). A bigger buffer just adds latency without adding throughput, because TCP can't usefully exploit more in-flight data than the BDP.

That rule was good enough for routers handling one or two big TCP flows. It is not good enough for modern routers handling thousands of flows of mixed size and mixed priority. Hence active queue management, hence the rest of this sheet.

### Latency-sensitive vs throughput-sensitive flows

Different applications have different sensitivities:

- **Voice (SIP, RTP)** — needs latency below ~150 ms one-way to feel natural. Throughput is tiny (~64 kbps per call). Loss is OK if recovery is fast.
- **Video conferencing** — needs latency below ~200 ms. Throughput is moderate (1-5 Mbps). Some loss tolerable.
- **Web (HTTP)** — wants page-load times under a few seconds. A first-byte latency of 200 ms is fine. Throughput needs are bursty.
- **Bulk file transfer (rsync, backup)** — does not care about latency at all. Wants maximum throughput. Will happily fill any buffer.
- **Online gaming** — needs latency below ~50 ms. Throughput tiny (~50 kbps).
- **Streaming video (Netflix, YouTube)** — buffers locally for tens of seconds. Cares about *sustained* throughput, not latency.

A queue management policy that treats all of these equally is going to make somebody unhappy. A bulk backup will fill the buffer and ruin everybody else's latency. A simple "drop the latest" policy will hurt the bulk transfer's throughput without protecting the voice call effectively. Active queue management plus prioritization (CoDel, FQ-CoDel, WRED, plus QoS classes) is the toolkit for handling this mix.

## Active Queue Management (AQM) — the family of solutions

### What AQM is, in one sentence

Active queue management means **the router takes action on packets *before* the buffer fills up**, with the goal of giving TCP an early warning to slow down, so that the buffer stays mostly empty and latency stays low.

The "active" word is in contrast to **passive** queue management. Tail drop is passive: the router does nothing until the buffer is 100% full, at which point it has no choice but to drop. AQM is active: the router monitors the queue, and when the queue starts looking too full, it intervenes — either by dropping packets, or by marking packets, or by changing the order packets are served.

### The two AQM tools: drop and mark

There are exactly two ways for a router to signal congestion to a sender:

1. **Drop the packet.** The sender will eventually notice the missing packet (via duplicate ACKs or a timeout) and slow down. This is the classic mechanism. Works with any TCP. No protocol negotiation needed.
2. **Mark the packet.** The router sets a bit in the packet's header — specifically, the **ECN** (Explicit Congestion Notification) bits in the IP header — to say "this packet experienced congestion." The receiver echoes this back to the sender. The sender slows down without losing the packet. This requires both endpoints and the router to support ECN.

Marking is strictly better than dropping when both ends support it: the sender slows down without losing data, no retransmit needed, no extra latency. But ECN deployment was historically slow because middleboxes (NATs, firewalls) sometimes mangled the bits. As of 2026, ECN is widespread on Linux and modern OSes; AQM algorithms support both.

### The AQM family tree

```
                Queue Management
               /                \
        Passive                  Active (AQM)
        (Tail Drop)             /          \
                       Probabilistic     Latency-based
                       /        \         /          \
                     RED        WRED    CoDel        PIE
                                ^         ^           ^
                          (priority   (target     (proportional
                           classes)    latency)    integral)
                                                       \
                                                  Combined
                                                  /        \
                                               FQ-CoDel    CAKE
                                               (per-flow   (with
                                                fairness)  shaping)
```

The probabilistic family (RED, WRED) drops packets with rising probability as the queue gets longer. The latency-based family (CoDel, PIE) drops packets when the *time spent in the queue* gets too long, regardless of queue depth in bytes. The combined family (FQ-CoDel, CAKE) does both *and* gives each flow its own sub-queue so a bulk flow can't starve a voice flow.

We will spend the next section on RED — the original AQM, the one that started it all. CoDel and the rest come in chunks 2 and 3.

## RED (Random Early Detection) — Sally Floyd / Van Jacobson, 1993

### The setting

In 1993, Sally Floyd and Van Jacobson published *Random Early Detection Gateways for Congestion Avoidance* in IEEE/ACM Transactions on Networking. (A "gateway" in the language of that era is what we now call a "router.") The paper proposed a new queue management algorithm to replace tail drop, with three explicit goals:

1. **Avoid global synchronization** of TCP flows.
2. **Avoid bias against bursty traffic.**
3. **Keep average queue length low** so latency stays bounded, while still allowing transient bursts to absorb into the buffer.

The algorithm they proposed was called Random Early Detection. The "random" part is what breaks synchronization. The "early" part is what keeps latency low.

### The core idea

RED watches the **average queue length** over time, using an exponentially-weighted moving average. As the average grows, RED begins to randomly drop (or mark) packets, with a probability that rises as the average queue grows. Below a threshold, no drops. Above another threshold, all drops. In between, probabilistic drops.

The genius of the algorithm is that it lets TCP flows discover congestion *before* the buffer fills. Each flow sees a packet drop, slows down, and the queue stays small. Because the drops are random and spread over time, different flows back off at different times — breaking synchronization.

Think of it this way: tail drop is a brick wall that flows hit all at once. RED is a fog bank that flows hit gradually, one at a time, in a randomized order. By the time the heaviest flow has been told "slow down" twice, the lightest flow may not have been told anything yet.

### The drop probability curve

```
drop
prob
 1.0|                                       _____________
    |                                      /
    |                                     /
    |                                    /
max_p|. . . . . . . . . . . . . . . . . /
    |                                  /
    |                                 /
    |                                /
    |                               /
  0 +______________________________/__________________
    0           min_thresh    max_thresh    queue_max
                              |              |
                              +- discard zone +
                              (drop with rising
                               probability)
```

Three regions:

1. **Below `min_thresh`:** no drops. The queue is short enough that we don't need to push back.
2. **Between `min_thresh` and `max_thresh`:** probabilistic drops. The probability rises linearly from 0 to `max_p` as the average queue grows from `min_thresh` to `max_thresh`.
3. **Above `max_thresh`:** all packets dropped (or all marked). The queue is dangerously full and we want to give every flow a strong signal.

When a packet arrives, RED computes the current average queue length, looks up the drop probability from the curve, and then flips a biased coin. If the coin comes up "drop," the packet is discarded (or ECN-marked). Otherwise, it is enqueued normally.

### Why "average" not "instantaneous"?

If RED used the *instantaneous* queue length, it would react to every micro-burst by triggering drops, which would defeat the whole point of having a buffer. The exponentially-weighted moving average smooths over short-term fluctuation. A short burst raises the instantaneous queue to 80% but the average stays at 20%, no drops triggered. A sustained load raises the average gradually, drops kick in gradually.

The averaging weight is one of the configurable knobs (see next section). Lower weight = slower averaging = more tolerance for bursts. Higher weight = faster averaging = quicker reaction to load changes.

### What RED solved

RED, properly tuned, solved global synchronization. Instead of all flows losing a packet in the same millisecond, drops were spread over many milliseconds, hitting different flows at different times. Flows backed off independently. The link's average utilization jumped from 30-50% (under tail drop) to 80-95% (under RED). Latency dropped because the buffer stayed mostly empty.

It also reduced the bias against bursty traffic. Because RED begins dropping while the queue is mostly empty, a burst that arrives during an empty-queue period is *not* punished — it sails through, fills the buffer briefly, and drains. Only sustained pressure that grows the *average* triggers drops.

### What RED didn't solve

RED has knobs. Four of them, as we'll see in the next section. RED is famously hard to tune. The "right" values depend on link speed, RTT, traffic mix, and traffic volume. A RED config that works on a 1 Gbps backbone link with 50 ms RTT will not work on a 10 Mbps DSL link with 30 ms RTT, and a RED config that works during the day may not work at night when traffic patterns change.

In a 2001 paper called *RED in a Different Light*, Mikkel Christiansen, Kevin Jeffay, David Ott, and F. Donelson Smith showed that RED's parameter sensitivity made it hard to deploy correctly, and that misconfigured RED was sometimes worse than tail drop. This is the problem CoDel was designed to solve a decade later.

But RED was a giant leap. Every modern AQM algorithm builds on the foundation Floyd and Jacobson laid in 1993. You cannot understand CoDel without understanding RED first.

## RED Parameters (min_thresh, max_thresh, max_p, weight)

### The four knobs

RED has four configuration parameters. Tuning them well is the hard part of running RED. Tuning them poorly can make things worse than tail drop. Here is what each one means.

### `min_thresh` — the minimum threshold

`min_thresh` is the average queue length below which RED never drops. It is the floor. Set it too low and RED starts dropping when the queue is barely used, sacrificing throughput for nothing. Set it too high and RED never kicks in until the queue is dangerously full, which is basically tail drop.

The classic Floyd-Jacobson recommendation is roughly 5 packets, or larger if traffic is bursty. Modern guidance for high-speed links is usually a few hundred packets, scaled to BDP.

```bash
# Linux example: set RED on eth0 with min_thresh = 5000 bytes
$ tc qdisc add dev eth0 root red \
    limit 60000 \
    min 5000 \
    max 15000 \
    avpkt 1000 \
    burst 20 \
    probability 0.02
```

`min` is `min_thresh`. `max` is `max_thresh`. `limit` is the actual buffer size. `avpkt` is the average packet size for averaging math. `burst` is a parameter that affects the EWMA weight. `probability` is `max_p`.

### `max_thresh` — the maximum threshold

`max_thresh` is the average queue length above which RED drops every packet. It is the ceiling. The Floyd-Jacobson rule of thumb is `max_thresh = 3 × min_thresh`, giving a wide region of probabilistic dropping between the two thresholds. A narrow region (small gap between min and max) makes the drop-probability curve steep, which makes RED react sharply to small queue changes — close to tail drop's pathology. A wide region gives a gentle curve, which is more tolerant.

If `max_thresh` is set too close to the actual buffer size, you can run out of buffer before you reach `max_thresh`, and tail drop kicks in anyway. Always leave headroom: `max_thresh < buffer_size × 0.7` is a safe guideline.

### `max_p` — the maximum drop probability

`max_p` is the drop probability when the average queue length equals `max_thresh`. Typical values are 0.02 to 0.10 (2% to 10%). Setting `max_p` too low makes RED gentle but slow to react — you may overrun max_thresh before flows back off. Setting `max_p` too high makes RED aggressive — flows back off too hard, link utilization drops.

The original RED paper recommended `max_p = 0.02`. Later work (by Sally Floyd herself, in *Recommendations on Using the "Gentle" Variant of RED*) suggested the algorithm should keep increasing probability past `max_p` between `max_thresh` and the actual buffer limit, instead of jumping to 1.0. That variant is called **gentle RED** and is the default in most modern implementations.

### `weight` (or `wq`) — the EWMA weight

The exponentially-weighted moving average is computed as:

```
avg_queue = (1 - weight) × old_avg + weight × current_queue
```

`weight` controls how fast the average tracks the instantaneous queue. Typical values are 0.001 to 0.01 (1/1024 to 1/128). A small weight = slow tracking = tolerant of bursts but slow to react to sustained load changes. A large weight = fast tracking = reacts quickly but can be fooled by transient bursts.

In Linux's `tc`, the EWMA weight is configured indirectly via the `burst` parameter, which is the number of average-sized packets that can arrive in a burst without triggering averaging. Roughly, `weight ≈ 1 / (2 × burst + 1)` for `burst` in average packets. A `burst` of 20 gives `weight ≈ 0.024`.

### Putting them together

A well-tuned RED for a residential 100 Mbps link, average packet ~1000 bytes, target average queue ~50 ms of buffering:

```bash
# 100 Mbps × 50 ms = 625 KB ≈ 625 packets at 1000 bytes/packet
# min_thresh = 1/3 of target = ~210 packets ≈ 210 KB
# max_thresh = target = 625 packets ≈ 625 KB
# max_p = 0.05
# burst = ~70 (giving weight ≈ 0.007)

$ tc qdisc add dev eth0 root red \
    limit 1000000 \
    min 210000 \
    max 625000 \
    avpkt 1000 \
    burst 70 \
    probability 0.05
```

You can verify it is in place:

```bash
$ tc -s qdisc show dev eth0
qdisc red 8001: root refcnt 2 limit 1000000b min 210000b max 625000b ewma 6 ...
 Sent 105234123 bytes 89234 pkt (dropped 421, overlimits 421 requeues 0)
  marked 0 early 421 pdrop 0 other 0
```

`early 421` means RED's early-drop logic has fired 421 times. `marked 0` means no packets have been ECN-marked — either ECN is off, or no senders supported it. `overlimits 421` is the tail-drop count after RED let too many through; ideally this matches `early` (RED is doing all the work, no packets reaching the actual ceiling).

The takeaway: RED works, but RED is *fussy*. Four parameters times "what is your link rate" times "what is your traffic mix" gives a tuning matrix that humans rarely get right on the first try, and rarely re-tune as the network changes. That fragility was one of the motivations for the next generation of AQMs — but those are stories for chunks 2 and 3.

### Sanity checks before you ship a RED config

Before you turn RED on in production, run through this checklist. Every one of these has bitten somebody before.

1. **Does `min_thresh` actually fit your link?** A 5-packet `min_thresh` on a 10 Gbps link is nonsense — the link drains 5 packets in microseconds, and RED never has time to engage. Scale to BDP.
2. **Is `max_thresh` well below the buffer ceiling?** Always leave at least 30% headroom. If your buffer is 1000 packets, `max_thresh` should be 700 or less. Otherwise tail drop fires before RED reaches its full drop probability and the synchronization problem returns.
3. **Is the `min_thresh` to `max_thresh` ratio at least 1:3?** A narrow window (e.g., 100 to 120) makes the curve almost a step function, defeating the "gradual signal" goal. Wider is better.
4. **Is ECN enabled on your hosts?** ECN-marking is strictly better than dropping. Check `sysctl net.ipv4.tcp_ecn` on Linux. `2` = passive (respond if asked), `1` = active (request always). Most modern stacks default to `2`.
5. **Did you test under realistic load before turning it on for real?** A common mistake is to enable RED, run a single iperf flow, see "looks fine," and roll out. RED's whole job is to handle *many* flows. Test with `iperf3 -P 32` or, better, real traffic mirrored from production.
6. **Did you save the previous tail-drop config?** RED can be worse than tail drop if misconfigured. Have a known-good rollback ready: `tc qdisc replace dev eth0 root pfifo limit 1000`.

```bash
# Quick rollback to plain FIFO if RED misbehaves
$ tc qdisc replace dev eth0 root pfifo limit 1000

# Confirm no qdisc weirdness
$ tc -s qdisc show dev eth0
qdisc pfifo 8002: root refcnt 2 limit 1000p
 Sent 0 bytes 0 pkt (dropped 0, overlimits 0 requeues 0)
```

### Common RED error messages and what they mean

```bash
# Trying to add RED with min >= max:
$ tc qdisc add dev eth0 root red limit 1000000 min 500000 max 500000 \
    avpkt 1000 burst 20 probability 0.05
RTNETLINK answers: Invalid argument
# Fix: max must be strictly greater than min, ideally 3x.

# Trying to add RED with limit < max:
$ tc qdisc add dev eth0 root red limit 100000 min 50000 max 200000 \
    avpkt 1000 burst 20 probability 0.05
RTNETLINK answers: Invalid argument
# Fix: limit must be larger than max_thresh. limit is the absolute buffer size.

# Trying to add RED on a virtual interface with no queue support:
$ tc qdisc add dev lo root red ...
RTNETLINK answers: Operation not supported
# Fix: loopback and some virtual interfaces don't carry qdiscs. Use a real interface.
```

### A note on RED in 2026

You will not configure RED on a brand-new deployment in 2026. CoDel, FQ-CoDel, and CAKE have superseded it for most use cases. But RED is still everywhere — it ships in every major switch ASIC, it is still the default AQM in many enterprise routers, and millions of in-service devices use it. Understanding RED is understanding the foundation of every AQM that came after. WRED (next chunk) is RED with priority awareness. CoDel started life as "RED that doesn't need tuning." Knowing RED first makes everything that follows make sense.

## WRED (Weighted RED) — different drop probabilities per traffic class

Plain RED treats every packet the same. The router measures the queue length, gets a drop probability, rolls a die, and drops the loser. The die does not care whether the packet was a phone call, a video stream, a software update, or a backup job. Everybody gets the same odds. Everybody bleeds the same.

WRED says: hold on. Some packets matter more than others. Phone calls cannot be dropped without sounding terrible. Video calls go choppy when you drop their packets. Backups, on the other hand, do not care if you drop a packet here and there. The TCP layer underneath the backup will simply slow down and try again, and the user will never notice. So if we have to drop something, drop the backup, not the phone call.

### The bouncer at the club

Picture a nightclub with one bouncer at the door. The club is filling up. The bouncer needs to start turning people away. Plain RED is a bouncer that flips a coin for every person in line. Heads in, tails out, no matter who you are. Fair, in a cold mathematical way.

WRED is a bouncer with a list. Regulars, VIPs, members of the band — those people get a very low chance of being turned away even when the club is almost full. Random walk-up tourists in flip-flops get a high chance of being turned away. Everybody is still subject to randomness, but the dice are loaded depending on who you are.

In the router, "who you are" comes from the packet's **DSCP** (Differentiated Services Code Point) value, or its **IP precedence**, or sometimes a class of service tag. The operator builds a policy: "Class EF (voice) starts dropping at 90% queue full with maximum probability 5%. Class AF11 (gold data) starts dropping at 70% with max probability 15%. Default class starts dropping at 30% with max probability 60%."

That gives you three different RED curves running on the same physical queue. As the queue grows, the most disposable traffic gets dropped first and most often. Voice and signalling traffic stay safe until the queue is genuinely about to overflow. Only then do those high-priority classes start to feel any pain.

### Tradeoffs

WRED is more complex than RED. You have to define traffic classes, you have to mark packets at the network edge, you have to maintain consistent DSCP markings across every link, and you have to tune three sets of curves instead of one. If your DSCP markings are wrong (a really common operational problem), WRED happily protects the wrong traffic.

WRED also still relies on operators picking the right `min`, `max`, and `probability` values per class. Pick them wrong and you get the same RED-tuning misery, only multiplied by the number of classes you defined.

WRED does not actually reduce queue depth or latency on its own — it only reorganises *which* packets get dropped. If your queue is too deep, WRED will not save you from bufferbloat. You still need a sane queue limit and a sane scheduler.

WRED is also a single-queue mechanism. It does not separate traffic into different queues; it just biases drops within one shared queue. That means a heavy elephant flow in the "low priority" class can still build queue and add latency for everybody, even though it gets dropped more often.

### When you'd use it

WRED earned its keep in carrier and enterprise WAN networks where the operator has a strong, consistent QoS policy across every link. Carrier MPLS backbones, ISP peering edges, enterprise WAN routers, data-center top-of-rack switches feeding mixed traffic — anywhere you have a clear notion of "this traffic matters more than that traffic" and you've been disciplined about marking it consistently, WRED is a reasonable mechanism.

You generally would not use WRED on a single home router or a Linux server's egress NIC, because the better modern mechanisms (FQ-CoDel, CAKE) handle mixed traffic without requiring DSCP marking discipline.

### Real-world deployment notes

Cisco IOS WRED has been the default congestion-avoidance mechanism on Cisco service-provider routers for over two decades. The classic deployment pattern is:

- Mark packets with DSCP at the network edge (CE router or access switch).
- Trust DSCP markings in the core.
- Apply per-class WRED on each output interface that experiences congestion.
- Tune the curves with the help of NetFlow data showing actual drop patterns.

Juniper devices have an equivalent called **RED with drop profiles**, configured per forwarding class. Same idea, slightly different syntax.

On Linux, WRED is not a built-in qdisc, but you can approximate the behaviour with `tc` using `gred` (Generic RED), which supports up to 16 virtual queues each with its own RED curve, sharing one buffer.

### Paste-runnable example (Linux GRED, the closest thing)

```bash
# 16-DP GRED on eth0, total buffer 200 packets
# Each DP gets its own min/max/probability curve.
sudo tc qdisc add dev eth0 root handle 1: gred setup DPs 16 default 0 grio

# DP 0 (default class): drop early and often
sudo tc qdisc change dev eth0 root handle 1: gred DP 0 \
    prio 0 limit 200000 min 30000 max 90000 avpkt 1000 burst 55 \
    bandwidth 100Mbit probability 0.5

# DP 1 (silver): drop later, less aggressively
sudo tc qdisc change dev eth0 root handle 1: gred DP 1 \
    prio 1 limit 200000 min 60000 max 120000 avpkt 1000 burst 55 \
    bandwidth 100Mbit probability 0.2

# DP 2 (gold/voice): drop only when buffer is nearly full, very low probability
sudo tc qdisc change dev eth0 root handle 1: gred DP 2 \
    prio 2 limit 200000 min 150000 max 190000 avpkt 1000 burst 55 \
    bandwidth 100Mbit probability 0.05

# Inspect
tc -s qdisc show dev eth0
```

```text
# Cisco IOS-style WRED config (paste into the interface)
interface GigabitEthernet0/0/0
 random-detect dscp-based
 random-detect dscp ef    40 50 10
 random-detect dscp af31  30 40 10
 random-detect dscp af11  20 35 10
 random-detect dscp 0     10 30 10
```

The triplet after each DSCP is `min-threshold max-threshold mark-probability-denominator`. So `random-detect dscp ef 40 50 10` means: "for EF-marked packets, start dropping when the average queue depth hits 40 packets, drop with maximum probability 1/10 once it hits 50 packets."

### Parameter ranges

- `min` (minimum threshold): 5–30% of buffer for default class; 60–90% for premium classes.
- `max` (maximum threshold): 30–60% of buffer for default; 85–95% for premium.
- `probability` (max drop probability at `max`): 0.1–0.5 (10%–50%) for default; 0.01–0.1 (1%–10%) for premium.
- `avpkt` (average packet size): tune to ~1000 bytes for IP traffic.

## ECN (Explicit Congestion Notification) — mark, don't drop (RFC 3168)

Plain RED and WRED both deal with congestion by *dropping* packets. Dropping is a hammer. The packet is gone. The endpoint has to detect the loss (a missing TCP ACK, a timeout) and retransmit. That works, but it is wasteful — the network already spent effort moving that packet most of the way to its destination, and now we throw all that work away. It is also bursty: a single drop is invisible until the receiver eventually notices.

ECN says: instead of dropping the packet, just **scribble on it**. Set a special bit in the IP header that says "I had to queue this — please slow down." Forward the packet anyway. The receiver echoes the mark back to the sender, and the sender's TCP slows down exactly the same way it would have if you had dropped the packet, but without losing the data.

### The marker on the cookie sheet

Imagine you bake cookies and your kid sometimes eats them too fast. Plain RED is: when the cookies start running low, you randomly knock one off the table onto the floor. Now it is gone. Your kid notices because there is one less cookie, and they slow down a bit.

ECN is: instead of knocking a cookie onto the floor, you draw a little frowny face on it with food colouring. The cookie still exists. Your kid still gets to eat it. But when they bite into it and see the frowny face, they know "oh, mum is telling me to slow down" — and they slow down. No cookie wasted.

In packet-network terms, the router that sees congestion sets the **CE (Congestion Experienced)** codepoint in the two-bit ECN field of the IP header. The packet keeps going. The receiver, when it processes that packet, notices the CE mark and sets a flag in the TCP ACKs going back to the sender (the **ECE — ECN Echo** flag). The sender sees ECE, treats it like a packet loss for congestion-control purposes, halves its congestion window, and reduces send rate.

The net effect: same congestion signal, no packet wasted, no retransmission needed, no application-visible hiccup.

### The four ECN codepoints

The two-bit ECN field in the IP header encodes:

```
00 = Not-ECT      (this packet does not support ECN, treat normally)
01 = ECT(1)       (sender supports ECN, codepoint 1)
10 = ECT(0)       (sender supports ECN, codepoint 0)
11 = CE           (Congestion Experienced — set by a router under congestion)
```

If a router sees a Not-ECT (00) packet and wants to signal congestion, it has no choice but to drop it. If a router sees an ECT(0) or ECT(1) packet, it can change the codepoint to CE (11) instead of dropping, signalling congestion without losing the packet.

### Negotiating ECN

ECN is opportunistic. Both endpoints have to support it. TCP negotiates this in the SYN handshake using the `ECN-Echo` and `CWR` (Congestion Window Reduced) flags. If the SYN+ACK comes back without the ECE flag, the connection falls back to non-ECN behaviour and the sender marks every packet 00.

ECN works for TCP, SCTP, and increasingly for QUIC. For UDP it requires the application to handle marks itself.

### Tradeoffs

ECN is essentially free in terms of network performance. The only cost is the protocol complexity of negotiating it, plus a tiny bit of router logic to set the CE bit instead of dropping.

The historical problem with ECN was middlebox interference. For a long time, a meaningful percentage of NAT boxes, firewalls, and load balancers either stripped the ECN bits or zeroed out the entire IP TOS/DSCP byte (which contains the two ECN bits in the lower two positions). That meant connections trying to negotiate ECN sometimes got mysteriously dropped or stalled. Linux for years had `tcp_ecn = 2`, which means "accept ECN on incoming connections but do not initiate it outbound," to avoid breakage with old middleboxes.

By the mid-2020s, middlebox compatibility is much better. Linux defaults to `tcp_ecn = 2` in current kernels, but enabling `tcp_ecn = 1` (initiate ECN outbound) is increasingly safe.

A second historical issue: classic ECN gives the same coarse "halve cwnd" signal as a packet loss. It does not allow finer-grained pacing. This is what L4S is designed to fix (next section).

A third issue: not every queue policy supports ECN marking. Plain `pfifo_fast` does not. RED supports it (`ecn` flag). CoDel/FQ-CoDel/CAKE support it natively. WRED on Cisco supports it.

### When you'd use it

Always, if you can. Modern Linux TCP, FQ-CoDel, and CAKE will mark instead of dropping by default when the sender supports ECN. There is essentially no reason not to enable it, and the latency-under-load improvement is measurable. Datacenter TCP variants (DCTCP) rely on ECN for very fine-grained congestion signalling and would not work without it.

### Real-world deployment notes

- Linux kernel: `sysctl net.ipv4.tcp_ecn=1` for full-on outbound ECN, or `2` for receive-only (the default for many distros).
- Datacentre fabrics widely use **DCTCP**, which marks based on **instantaneous** queue depth (not average) and uses the fraction of marks per RTT to compute a smooth rate adjustment. This delivers very low queueing latency in datacentres and is the precursor to L4S.
- ECN is a hard requirement for L4S (the new low-latency profile, RFC 9330).
- Verify ECN is happening with `ss -ti` (look for `ecnseen` and `ecn` in the socket info) and `tcpdump -v` (look for `ECT(0)` / `CE` in packet captures).

### Paste-runnable examples

```bash
# Enable ECN on Linux (full bidirectional)
sudo sysctl -w net.ipv4.tcp_ecn=1

# RED with ECN (mark instead of drop where possible)
sudo tc qdisc add dev eth0 root handle 1: red \
    limit 1000000 min 50000 max 150000 avpkt 1000 \
    burst 55 bandwidth 100Mbit probability 0.1 ecn

# CoDel with ECN (default-ish settings, ECN enabled)
sudo tc qdisc replace dev eth0 root codel ecn

# Inspect ECN markings on a live TCP socket
ss -ti | grep -E 'ecn|ecnseen'

# Watch packets with ECN bits
sudo tcpdump -i eth0 -nv 'tcp' | grep -i 'ECT\|CE'
```

## L4S (Low Latency, Low Loss, Scalable Throughput, RFC 9330)

Classic ECN inherits TCP's "additive increase, multiplicative decrease" rhythm. When a congestion signal arrives, the sender halves its window. That is fine but coarse. It causes the bandwidth to oscillate up and down, and it requires queues to grow somewhat for the marks to be accurate. You still get tens of milliseconds of queueing latency in many cases.

L4S is a redesign of the congestion-signalling story aimed at sub-millisecond queue latency for a new generation of applications: cloud gaming, AR/VR, real-time control, ultra-low-latency video. It changes both the queue mark behaviour and the sender response.

### Two queues, two different stories

Imagine a restaurant with two pickup counters. The "classic" counter serves regular dine-in customers who don't mind waiting a few minutes. The "express" counter serves people picking up Uber Eats orders who absolutely must get out the door in 30 seconds.

If you put both kinds of customers in one line, you have to run the line at the speed of whichever group is more sensitive to delay — and inevitably the other group complains. So you run two queues, both fed from the same kitchen, but with different rules:

- The classic queue uses the old slow rhythm: bigger buffer, less frequent marks, each mark means "halve your output."
- The express queue uses a much faster rhythm: tiny buffer, very frequent fine-grained marks, each mark means "back off by a small fraction proportional to the mark rate."

That second rhythm is **scalable congestion control**. It uses high-frequency, small congestion signals to keep the queue tiny. Senders treat each mark as a small slow-down rather than a giant slow-down. The result is a queue that is nearly always empty, with milliseconds of latency rather than tens of milliseconds.

### The L4S codepoint trick

L4S re-purposes ECN codepoints. Where classic ECN uses ECT(0) for "supports ECN," L4S uses **ECT(1) (codepoint 01)** to mean "this flow uses scalable congestion control." Routers that understand L4S route ECT(1) packets into the L4S (low-latency) queue and apply a much more aggressive marking rule there: even very tiny queues get CE-marked.

Routers that do not understand L4S see ECT(1) and ECT(0) as just "ECN-capable" — they handle both the same. Backwards-compatible.

The full deployment requires:

- Senders that implement scalable congestion control (e.g. **TCP Prague**, **BBRv3** in some variants, **SCReAM** for media) and emit ECT(1).
- Routers/queues that implement the dual-queue AQM and treat ECT(1) flows aggressively in the L4S queue.
- Receivers that echo CE marks accurately back to the sender.

### Tradeoffs

L4S is genuinely transformative for latency-sensitive workloads. Cable operators (Comcast, Charter via the **DOCSIS 3.1 Low-Latency DOCSIS — LLD** spec) have been deploying it on cable modems. ISPs see queueing latency drop from ~30 ms p99 to <1 ms p99 for L4S-marked flows, with the same throughput.

The downside: L4S is a different rhythm than classic TCP. If you put L4S and classic flows in the **same** queue with shared marking, the L4S flow is more aggressive (responds less to marks) and starves the classic flow. That is why L4S deployments require either dual-queue AQM or strict isolation.

L4S also relies on accurate ECN echo. There has been debate in the IETF about whether the way classic ECN echoes a single bit per RTT is accurate enough for L4S (the **AccECN** extension provides finer-grained feedback, RFC 9341).

There is a long-running IETF debate about L4S vs. **SCE (Some Congestion Experienced)**, an alternative proposal that uses a different bit-allocation strategy. L4S won the IETF process and was published as RFC 9330 in 2023, but the technical debate continues.

### When you'd use it

L4S is appropriate when you control both endpoints and the bottleneck queue, AND you have a latency-sensitive workload. The clearest deployment is operator-managed access networks (cable, fibre, mobile) where the operator controls the access router queue and can roll out L4S-aware home gateways. Cloud-gaming providers, real-time-comms providers, and CDNs are the early adopters on the application side.

For general-purpose Internet traffic on a generic Linux server, you do not need L4S yet. FQ-CoDel and CAKE handle most workloads well.

### Real-world deployment notes

- **DOCSIS 3.1 LLD** specifies dual-queue L4S for cable modems. Major US cable operators are deploying it in 2024–2026.
- **TCP Prague** is the reference scalable congestion control for L4S, available as a Linux kernel patch.
- **DualPI2** is the reference dual-queue AQM, mergeable into Linux as a `tc` qdisc patch.
- **BBRv3** does not strictly require ECN but interacts well with L4S marks when paced by the queue.
- Apple's NQTCP and Google's BBR both include components compatible with the L4S philosophy.

### Paste-runnable example (DualPI2 — kernel patch required)

```bash
# DualPI2 qdisc (requires a kernel with the dualpi2 patch — not in mainline as of 2024)
# Replace eth0 with your interface
sudo tc qdisc replace dev eth0 root dualpi2 \
    target 15ms tupdate 16ms alpha 0.16 beta 1.0 \
    coupling_factor 2 step_thresh 1ms drop_on_overload 1

# Verify L4S is active
tc -s qdisc show dev eth0

# Set up scalable congestion control on the sender
sudo sysctl -w net.ipv4.tcp_congestion_control=prague

# Enable AccECN for finer-grained feedback
sudo sysctl -w net.ipv4.tcp_ecn=1
sudo sysctl -w net.ipv4.tcp_ecn_fallback=1
```

### Parameter ranges

- `target`: 1ms–15ms (much lower than classic CoDel)
- `tupdate`: 8ms–32ms (control loop period)
- `alpha`/`beta`: PI controller gains; defaults usually fine
- `step_thresh`: 0.5ms–2ms (the threshold above which L4S marking starts)

## CoDel (Controlled Delay) — Kathie Nichols / Van Jacobson, 2012

RED was hard to tune. Operators gave up trying to set `min`, `max`, and `probability` correctly. After a decade of watching that fail, Kathie Nichols and Van Jacobson designed a new algorithm that did not require tuning. Same goal as RED: keep queues short. New mechanism: measure the *time* a packet spends in the queue, not the queue length in bytes or packets.

CoDel's name says it: **Controlled Delay**. The metric is delay, not depth. The algorithm aims to keep the minimum sojourn time (the shortest time any packet spent in the queue, measured over a sliding interval) below a small target — usually **5 milliseconds**.

### The supermarket queue

Imagine you are at a supermarket. There is one checkout, and a long line is forming. A manager walks up and starts a stopwatch. The manager picks one customer and follows them: how long did they have to stand in the queue before reaching the till? If that time stays below 30 seconds, the manager does nothing. If it goes above 30 seconds, the manager starts opening another till — but only if it has been above 30 seconds *consistently* for a while, not just for one unlucky customer.

CoDel does this with packets. It tracks the **minimum** queue sojourn time over a sliding window (default 100 ms — the "interval"). If that minimum is above the target (default 5 ms) for the entire interval, CoDel enters "dropping state" and starts dropping packets. The drop rate accelerates as long as the queue stays full — the next drop comes after one interval, then 1/sqrt(2) intervals, then 1/sqrt(3), etc. As soon as the minimum sojourn time falls back below the target, CoDel stops dropping.

The genius is that CoDel does not care about queue *length*. A short queue with packets sitting in it for a long time (because the link is slow) gets the same treatment as a long queue draining quickly. CoDel measures what actually matters to the user: how long is your packet stuck waiting?

### Why "minimum" and not "average"?

Bursty traffic creates queues that fill and drain rapidly. Average queue depth over time can be misleading — a deep queue that rapidly drains looks bad on average but is not actually causing user-visible latency, because the *minimum* sojourn time during the drain is small. CoDel uses the minimum precisely so that healthy bursts don't trigger drops.

The minimum is "the smallest sojourn time we saw during this interval." If during the interval at least one packet sailed through quickly, the minimum is small, the queue is doing its job, no action needed. If every single packet during the entire interval was delayed, the minimum is large — that's a real, persistent queue that is actually hurting user latency.

### Tradeoffs

CoDel is *almost* tuning-free. The defaults work astonishingly well across a huge range of link speeds, from dialup to 100 Gbps. The only knob most operators ever touch is the `target` (and even then, only in odd corner cases like extremely slow links).

CoDel does not separate flows. A single elephant flow can fill the queue and force CoDel to drop on everybody. That's why CoDel is rarely used by itself in modern Linux — it is paired with fair queueing (FQ-CoDel, see below).

CoDel also does not reduce the *peak* queue depth on its own; if a sender bursts hard, the queue grows until CoDel notices and starts dropping. The effect is good on a rolling average, but you can still see brief excursions.

CoDel is not magic on links with extremely variable capacity (cellular, satellite). The `target` is in real time, but the real available bandwidth jitters. CoDel handles this acceptably well; it just doesn't perform quite as well as on stable links.

### When you'd use it

You almost always use **FQ-CoDel**, not raw CoDel. Raw CoDel is most useful on a single shared queue that genuinely needs no flow isolation — for example, a pure transit interface with one big flow class. FQ-CoDel and CAKE are the right answers on every multi-flow scenario.

### Real-world deployment notes

- CoDel has been in Linux mainline since kernel 3.5 (2012), originally by Eric Dumazet.
- Apple shipped CoDel in iOS and macOS in 2014.
- It is the basis for FQ-CoDel, default on Linux egress since kernel 4.20 (2018).
- Reference implementation by Nichols/Jacobson lives in `net/sched/sch_codel.c`.

### Paste-runnable example

```bash
# Replace eth0's qdisc with plain CoDel
sudo tc qdisc replace dev eth0 root codel \
    limit 1000 target 5ms interval 100ms ecn

# Inspect counts and current state
tc -s qdisc show dev eth0

# Try with a tighter target on a fast LAN
sudo tc qdisc replace dev eth0 root codel \
    limit 10000 target 1ms interval 20ms ecn

# Disable ECN (drop instead of mark)
sudo tc qdisc replace dev eth0 root codel \
    limit 1000 target 5ms interval 100ms noecn
```

### Parameter ranges

- `target`: 1 ms (datacentre / fast LAN) to 30 ms (very slow link). Default **5 ms** works well from ~1 Mbps upward.
- `interval`: typically 1× the worst-case RTT. Default **100 ms** works for most Internet links. Use 20–30 ms for LANs, 200–300 ms for satellite.
- `limit`: the hard ceiling on queue depth in packets. Default 1000 packets is fine; raise it for very-fast links if you see tail-drops.
- `ecn`/`noecn`: mark instead of drop where the packet is ECN-capable.
- `ce_threshold`: secondary, more aggressive ECN-mark target — like a "soft" target where CoDel marks but doesn't drop. Useful on links with a lot of ECN-capable traffic.

## PIE (Proportional Integral controller Enhanced, RFC 8033)

PIE is the other modern AQM that emerged at roughly the same time as CoDel. It tackles the same problem (control queue latency without per-operator tuning) using a different engineering style: a **proportional-integral controller** like the ones used in industrial process control, robotics, and HVAC.

PIE was designed by a Cisco team led by Rong Pan, with deployment goals in cable-modem and DOCSIS networks where simplicity and low CPU were essential. It became the AQM of choice for **DOCSIS 3.1** cable modems and was published as RFC 8033 in 2017.

### The thermostat analogy

You have a room with a thermostat. You set the target temperature to 20 °C. The thermostat measures the current temperature, compares it to the target, and turns the heater up or down based on the *error* — the difference between current and target.

A simple thermostat just turns the heater fully on or fully off. A *proportional* controller turns the heater on a little when the error is small and a lot when the error is large. A *PI controller* (the one in the name) also looks at how the error has been *trending* over time. If the error has been positive for a while (room too cold for a long time), the controller increases its response even more. That extra ingredient — the integral — prevents persistent steady-state error.

PIE uses exactly this kind of controller, but the variable being controlled is not temperature: it is **drop probability**. The error signal is the difference between current queue delay and the target queue delay (typically 15–20 ms). When the queue is over the target, PIE increases its drop probability. When the queue is under the target, PIE decreases it. The proportional and integral gains determine how quickly and smoothly the drop probability changes.

### Why a controller, not a heuristic?

CoDel is a state-machine: "above target for an interval — start dropping." PIE is a continuous controller: "compute drop probability every 16 ms based on current and recent queue delay, smooth it, apply it as a per-packet drop chance."

The controller approach has nicer mathematical properties (you can prove convergence and stability) and runs with very small CPU cost — just a couple of multiplies and adds per update interval, plus a coin-flip per packet. That low cost is why DOCSIS picked PIE: cable modems are aggressively cost-engineered devices and the AQM had to fit alongside everything else they do.

### Tradeoffs

PIE and CoDel benchmark very similarly in most scenarios. PIE has a slight CPU advantage on small embedded devices because it does not need to time-stamp every packet (it computes delay from queue depth and dequeue rate). CoDel has a slight robustness advantage on highly variable workloads because it is reading delay directly rather than estimating it.

PIE is single-queue, like CoDel. To get fairness across flows, you wrap it in FQ-PIE (which exists but is less commonly deployed than FQ-CoDel).

PIE does not handle bursts as gracefully as CoDel; it can over-react to a transient burst because the integral term ramps drop probability up faster than CoDel's interval-based mechanism. RFC 8033 includes a "burst allowance" timer to mitigate this, similar in spirit to a token bucket.

### When you'd use it

If you are operating cable modems / DOCSIS infrastructure or building an embedded device targeting that ecosystem, PIE is the right answer. If you are picking an AQM for general Linux egress, FQ-CoDel is the dominant choice and PIE is mostly of historical/comparative interest.

PIE also has a niche on hardware that has built-in PIE silicon support (some merchant switch ASICs), where it is cheaper than CoDel because the controller fits the hardware better.

### Real-world deployment notes

- PIE is the mandated AQM in **DOCSIS 3.1** cable modems for "active queue management" service flow profiles.
- Linux has PIE in the mainline kernel as the `pie` qdisc since kernel 3.14 (2014).
- FQ-PIE was added to Linux 5.7 (2020) for combined fairness + PIE.

### Paste-runnable example

```bash
# Plain PIE on eth0
sudo tc qdisc replace dev eth0 root pie \
    target 20ms tupdate 15ms alpha 2 beta 20 \
    bytemode ecn

# Inspect
tc -s qdisc show dev eth0

# Tighter target for low-latency interactive workloads
sudo tc qdisc replace dev eth0 root pie target 5ms tupdate 16ms ecn

# FQ-PIE for combined fairness + PIE
sudo tc qdisc replace dev eth0 root fq_pie \
    limit 10240 flows 1024 target 15ms tupdate 15ms ecn
```

### Parameter ranges

- `target`: 5–25 ms; default 20 ms (matches DOCSIS profiles).
- `tupdate`: control-loop period. Default 15 ms; range 8–32 ms.
- `alpha`/`beta`: PI controller gains (proportional and integral). Defaults are tuned for general Internet workloads; rarely changed.
- `bytemode`: drop probability scales with packet size (recommended).
- `ecn`/`noecn`: mark vs. drop.

## FQ-CoDel (Fair Queueing + CoDel) — default since Linux 4.20

CoDel solved "how long should the queue be?" but did not solve "how should we share bandwidth between flows?" Fair Queueing solved sharing but did not solve queue depth. Combine them: **FQ-CoDel** runs a separate CoDel instance on each of many flow buckets, gives each flow its own little queue, and rotates between flows fairly. The result is the closest thing to a no-tuning, no-bufferbloat, fair-and-low-latency qdisc that exists for general use.

### The food-court analogy

Imagine a food court with one big shared queue. People with single-item orders get stuck behind a person buying lunch for an entire office. Latency for everybody is awful.

Now redesign: each restaurant has its own line. Each line has its own till and its own little queue. A fair-share scheduler walks between the queues and serves one customer from each in rotation. People with quick orders are now blocked only by their own restaurant's queue, not by everybody.

FQ-CoDel does exactly this. Incoming packets are hashed into one of many "flow buckets" (default 1024) by 5-tuple (src/dst IP, src/dst port, protocol). Each bucket is its own independent queue. A scheduler rotates between buckets in a deficit-round-robin order, dequeueing one quantum (default 1514 bytes) from each in turn. CoDel runs *inside* each bucket, deciding when that bucket is too deep and packets need to be marked or dropped.

### Why this matters

Your interactive SSH session is one flow. Your big software download is another flow. Without FQ, both share one queue and the SSH session sees latency caused by the download's packets. With FQ, the SSH packets land in their own bucket which is almost always empty. The download fills its own bucket, CoDel marks/drops packets from that bucket, the download backs off, but the SSH bucket stays empty and SSH latency stays low.

You get **bufferbloat protection** (because each bucket has CoDel) AND **per-flow fairness** (because the scheduler rotates) at essentially zero extra CPU cost. It is the rare engineering win where two good ideas combined are strictly better than either alone.

### The "new flows / old flows" trick

FQ-CoDel adds one more piece of cleverness: when a bucket sees its first packet, the scheduler puts it in a special "new flows" list. New flows get *priority* over old flows (the ones that have been running for a while). This means latency-sensitive short flows — DNS queries, TCP SYNs, single HTTP request packets — get out of the queue immediately, even while a giant TCP download is filling its own bucket. New-flow priority lasts until the new flow has dequeued one full quantum, at which point it joins the regular old-flows rotation.

This is what makes FQ-CoDel feel *fast* even under load: the response to a click, an SSH keystroke, or a DNS lookup gets out the door in microseconds while bulk traffic continues elsewhere.

### Tradeoffs

FQ-CoDel is the modern default. The known weaknesses:

- Hash collisions: with 1024 buckets, two unrelated flows can end up in the same bucket. Generally only a problem in pathological cases (heavy-traffic shared host).
- It enforces per-5-tuple fairness, not per-host or per-user fairness. A single host opening 100 TCP connections gets 100 buckets and effectively 100x as much bandwidth as a host with 1 connection. CAKE (next section) addresses this with hierarchical fairness.
- It does not shape (limit) total bandwidth — for that you need an HTB or similar shaper above it.
- No DSCP awareness by default. CAKE adds that.

### When you'd use it

Use FQ-CoDel as the default qdisc on every Linux server, Linux desktop, and Linux router unless you have a specific reason not to. Modern kernels enable it automatically on most network namespaces. Even better: use **CAKE** if you need shaping or per-host fairness.

### Real-world deployment notes

- Default qdisc on Linux egress since kernel 4.20 (2018), via `net.core.default_qdisc=fq_codel`.
- Default on systemd-managed interfaces since systemd v220.
- Default on most major Linux distributions (Debian, Ubuntu, Fedora, Arch).
- Used by Google Fiber routers, OpenWrt, IPFire, pfSense, and Linksys EA-series consumer routers (which originally pioneered FQ-CoDel deployment in consumer hardware via the Bufferbloat project's research).
- Eliminated the user-visible bufferbloat problem on most home networks running modern firmware.

### Paste-runnable example

```bash
# Set FQ-CoDel as default for all newly-created interfaces
sudo sysctl -w net.core.default_qdisc=fq_codel

# Or apply to one interface explicitly
sudo tc qdisc replace dev eth0 root fq_codel \
    limit 10240 flows 1024 quantum 1514 target 5ms interval 100ms ecn

# Inspect (you'll see drop counters per bucket)
tc -s qdisc show dev eth0

# Adjust quantum on a fast LAN to reduce per-rotation overhead
sudo tc qdisc replace dev eth0 root fq_codel quantum 8192

# Pair with HTB for shaping
sudo tc qdisc add dev eth0 root handle 1: htb default 10
sudo tc class add dev eth0 parent 1: classid 1:10 htb \
    rate 80mbit ceil 80mbit
sudo tc qdisc add dev eth0 parent 1:10 fq_codel
```

### Parameter ranges

- `limit`: total packets allowed across all buckets. 1000 (small) – 100000 (large). Default 10240.
- `flows`: number of buckets. 64 (tiny) – 65536 (huge). Default 1024 is fine for almost everything.
- `quantum`: bytes dequeued per bucket per rotation. Default 1514 (one MTU). Raise for fast LANs (4096–16384).
- `target`/`interval`: as for CoDel.
- `ecn`/`noecn`: mark vs. drop.
- `ce_threshold`: optional aggressive ECN target for L4S-like behaviour.
- `memory_limit`: per-flow byte budget; default 32 MB total.

## CAKE (Common Applications Kept Enhanced) — Jonathan Morton, OpenWrt

CAKE is the latest evolution in the FQ-CoDel lineage. Designed by Jonathan Morton with support from the **Bufferbloat project** and **Free Software Foundation Europe**, CAKE bundles together every good idea from the previous decade of AQM research into one easy-to-deploy qdisc.

CAKE solves several of FQ-CoDel's residual issues:

- **Bandwidth shaping built in**: no need for HTB above it.
- **Hierarchical fairness**: per-host as well as per-flow.
- **DiffServ/DSCP awareness**: traffic classes routed to different priority bands.
- **Overhead compensation**: knows about ATM/PPPoE/ADSL/cellular framing overhead so the shaper is accurate.
- **Per-tin AQM**: each priority band has its own CoDel-derived AQM.
- **Acceleration/congestion-control hooks**: integrates with `tcp_pacing` for smoother behaviour.

### The food-court-with-floors analogy

If FQ-CoDel is a food court with one line per restaurant, CAKE is a multi-floor mall with food courts on each floor, and each floor has its own bandwidth budget. The mall as a whole has a total budget (the shaped rate). Each floor (priority tin) gets a guaranteed share of that budget. Within each floor, each restaurant (host) gets its share of the floor. Within each restaurant, each customer (flow) gets fair access. And CoDel-style queue management runs at the per-flow level inside each restaurant.

The mall analogy maps to CAKE's structure:

- **Total shaped rate** = mall capacity.
- **Tins** = floors. CAKE supports 1, 3, 4, or 8 tins, mapped from DSCP. Voice goes in the priority tin, video in the next, bulk in the lowest.
- **Hosts** = restaurants on a floor. CAKE hashes on src or dst IP to identify hosts.
- **Flows** = customers within a restaurant. Hashed on full 5-tuple within the host bucket.
- **CoDel** = the queue manager inside each flow.

### Tradeoffs

CAKE is a heavier qdisc than FQ-CoDel. On small embedded devices (low-end routers, OpenWrt boxes) it consumes meaningfully more CPU, especially at gigabit speeds. On modern x86 hardware the cost is negligible (microseconds per packet).

CAKE's overhead-compensation feature is genuinely useful for ADSL/VDSL/DOCSIS lines but requires the operator to know the link's framing overhead. Wrong setting yields slightly inaccurate shaping. CAKE ships with named presets (`atm`, `ptm`, `docsis`, `ethernet`) that cover most cases.

CAKE's DiffServ defaults are opinionated. By default CAKE re-marks DSCP into 8 tins using a built-in policy. Operators in carrier networks who already have a strict DSCP policy may prefer FQ-CoDel without re-marking, or use CAKE's `besteffort` mode to disable tinning.

CAKE is single-mode for ingress vs. egress. Ingress shaping requires `ifb` virtual interfaces and is documented in the `tc-cake` manpage.

### When you'd use it

Anywhere you need shaping plus AQM:

- Home routers (OpenWrt's SQM uses CAKE by default).
- Enterprise WAN egress where the operator wants fair-share between hosts plus DSCP-aware classification.
- Servers behind a constrained uplink (cloud gaming, video conferencing services).
- Any case where you would otherwise have stacked HTB + FQ-CoDel — CAKE replaces both with fewer moving parts.

### Real-world deployment notes

- Mainline Linux since kernel 4.19 (2018).
- Default in **OpenWrt SQM** (Smart Queue Management) for home routers.
- Tested extensively in the Bufferbloat community; widely regarded as the best general-purpose qdisc for asymmetric / metered links.
- Supports `ack-filter` (drops redundant TCP ACKs to cope with extreme up/down asymmetry like 3:300 Mbit cable links).

### Paste-runnable example

```bash
# Simple egress shaping at 80 Mbit on eth0 with default tins and DSCP policy
sudo tc qdisc replace dev eth0 root cake \
    bandwidth 80Mbit besteffort

# Realistic DSL setup: 18 Mbit down / 1.4 Mbit up with PPPoE-over-VDSL framing
# Egress on eth0
sudo tc qdisc replace dev eth0 root cake \
    bandwidth 1400Kbit ptm overhead 30 mpu 64 \
    diffserv4 nat dual-srchost ack-filter

# Ingress shaping via ifb (mirror packets to a virtual device, then shape that)
sudo modprobe ifb
sudo ip link add ifb0 type ifb
sudo ip link set dev ifb0 up
sudo tc qdisc add dev eth0 handle ffff: ingress
sudo tc filter add dev eth0 parent ffff: protocol all u32 match u32 0 0 \
    action mirred egress redirect dev ifb0
sudo tc qdisc replace dev ifb0 root cake \
    bandwidth 18Mbit ptm overhead 30 mpu 64 \
    diffserv4 nat dual-dsthost ingress

# Inspect (rich output: per-tin stats, per-host stats, drop counts)
tc -s qdisc show dev eth0

# Diffserv-disabled, simple fairness only
sudo tc qdisc replace dev eth0 root cake bandwidth 100mbit besteffort flows
```

### Parameter ranges and modes

- `bandwidth`: shaping rate, e.g., `80Mbit`, `1400Kbit`. Required for shaping.
- `besteffort` | `precedence` | `diffserv3` | `diffserv4` | `diffserv8`: number of tins.
- `flows` | `dual-srchost` | `dual-dsthost` | `triple-isolate`: fairness scheme.
- `nat`: detect post-NAT addresses for fairer hashing on home routers.
- `wash`: clear DSCP markings before forwarding (some operators want this).
- Overhead presets: `atm`, `ptm`, `docsis`, `ethernet`, plus `overhead <bytes>` and `mpu <bytes>` (minimum packet size on wire).
- `ack-filter` | `ack-filter-aggressive`: drop redundant ACKs on egress.
- `ingress`: behaviour adjusted slightly for ingress-direction shaping.
- `memlimit`: total memory cap.
- `rtt <time>`: hint expected RTT for tuning marking aggressiveness; default 100 ms.

CAKE is the closest thing to "set this and forget it" for home and SOHO links — and unlike FQ-CoDel, it has been battle-tested specifically for the messy reality of ADSL framing, DOCSIS framing, asymmetric links, and weird cable modem behaviour.

## BQL (Byte Queue Limits) — Linux NIC queue control

> BQL is the kernel saying to the network card, "I will not hand you more bytes than you can shoot out the door in the next blink, because if I do, those extra bytes just sit in your hardware buffer and get stale."

Imagine a mailroom. The mailroom has a clerk (the kernel). The clerk has a big stack of letters to mail. There is a mailbox out front (the NIC hardware queue). The clerk's job is to walk letters from the inside stack to the outside mailbox.

If the clerk dumps every letter from the inside stack into the outside mailbox all at once, the mailbox overflows. Letters fall on the ground. The mail truck shows up and grabs the first letter on top, which might be a letter from a week ago, while a new urgent letter is buried at the bottom.

What we want is for the clerk to put letters in the mailbox slowly. Just enough so the mailbox is never empty when the truck arrives, but never so full that letters pile up and get stale. If the truck always shows up and finds exactly one letter waiting, the clerk has it perfect — no waste, no delay, no stale mail.

That is BQL. BQL watches how fast the NIC drains bytes from its hardware ring (the mailbox) and tells the qdisc (the inside stack) how many bytes are safe to push down. If the NIC is draining slowly because the wire is slow, BQL keeps the hardware ring small. If the NIC is draining fast, BQL lets the hardware ring grow. The point of BQL is to keep almost all the queueing in software, where the qdisc can do AQM and fair queueing, instead of in hardware, where the NIC just FIFOs everything blindly.

### Why this matters

A modern NIC has a hardware transmit ring with hundreds or thousands of slots. Without BQL, the kernel happily fills that ring to the brim. Once a packet is in the hardware ring, the kernel has lost control: it cannot reorder, drop, mark, or prioritise. So if you've got fq_codel on the qdisc doing brilliant fair queueing and CoDel marking, but the hardware ring has 1000 packets sitting in it, your fq_codel might as well be a brick. The hardware ring becomes the new bottleneck and you're back to bufferbloat.

BQL fixes this by capping the hardware ring at the smallest size that still keeps it from going empty. The kernel keeps a moving estimate of how many bytes the NIC can drain in one interrupt cycle and limits in-flight bytes to about that. The result: the hardware ring is almost always small, and the qdisc is where the queue lives. Now fq_codel can actually do its job.

### How it works under the hood

BQL has two state variables per NIC tx-queue:

- `limit` — current cap on bytes-in-flight (bytes already handed to the NIC but not yet sent)
- `inflight` — current count of bytes-in-flight

Whenever the kernel calls `dql_queued()` (when handing a packet to the NIC), `inflight` goes up. Whenever the NIC reports completion, `dql_completed()` runs, `inflight` goes down, and BQL adjusts `limit` based on whether the queue went empty (limit too low — bump it up) or whether `inflight` stayed close to `limit` (limit might be too high — back off).

It is a feedback loop. Just like CoDel for the qdisc, BQL for the hardware. They were invented by the same person, Tom Herbert, around the same time, and they are designed to work together.

### Tuning BQL

For most users you do not tune BQL. The defaults work. But on a slow link with weird hardware, you might need to set a floor.

```bash
# show current BQL state for tx queue 0 on eth0
$ cat /sys/class/net/eth0/queues/tx-0/byte_queue_limits/limit
3028

# show min and max bounds
$ cat /sys/class/net/eth0/queues/tx-0/byte_queue_limits/limit_min
0
$ cat /sys/class/net/eth0/queues/tx-0/byte_queue_limits/limit_max
1879048192

# set a floor of 4500 bytes (3 full-size frames) so we don't underrun
$ echo 4500 | sudo tee /sys/class/net/eth0/queues/tx-0/byte_queue_limits/limit_min

# inspect every tx queue at once
$ for q in /sys/class/net/eth0/queues/tx-*/byte_queue_limits/limit; do
    echo "$q: $(cat $q)"
  done
```

If `limit` is jumping around by ±1500 every second, BQL is doing its job. If it's pegged at `limit_min` and never moves, you're underrunning the link and need to raise `limit_min`. If it's pegged at `limit_max` and the queue keeps filling, your hardware ring is sized too small or your wire is too slow.

### Tools that use BQL data

```bash
# bqlmon shows live BQL stats per queue (tx-0, tx-1, …)
$ bqlmon -i eth0

# ethtool can show ring sizes (BQL works under these)
$ ethtool -g eth0
Ring parameters for eth0:
Pre-set maximums:
RX:		4096
TX:		4096
Current hardware settings:
RX:		512
TX:		512

# shrink TX ring to 256 — less hardware buffer = less hidden bufferbloat
$ sudo ethtool -G eth0 tx 256
```

A smaller hardware TX ring is friendlier to AQM. The trade-off is more interrupts per second (the ring fills and drains more often). On a modern CPU this is essentially free. On a tiny embedded box it can matter; benchmark it.

### When BQL bites you

The two failure modes are:

1. **Underrun.** `limit_min` is too small for your hardware. The NIC keeps emptying its ring before the kernel hands it more, so the wire goes idle. Throughput tanks. Fix: raise `limit_min` to about 2-4 MTUs.
2. **No effect.** The NIC driver does not call `dql_queued()` and `dql_completed()`. BQL relies on the driver. If the driver is old or proprietary, BQL does nothing. Most upstream drivers (igb, ixgbe, mlx5, virtio_net) support BQL. Some out-of-tree drivers do not. Check `/sys/class/net/eth0/queues/tx-0/byte_queue_limits/` — if the directory exists, the driver supports BQL.

## TCP Small Queues (TSQ) — limits in-flight on the sender

> TSQ is the kernel telling each TCP connection, "do not push more bytes into the network stack than I have set aside for you, even if your congestion window says you can."

Imagine a single household with five kids and one bathroom. Even if the family rule book says "anyone can use the bathroom", reality is that only one person fits at a time. If five kids all rush in simultaneously, nobody can do anything.

TSQ is the rule that says: each kid gets a timed turn. The kid stays in the bathroom for a fixed amount of time (a fixed number of bytes) before another kid can enter. You don't queue up infinite bathroom-needers; you cap the in-flight requests.

Specifically, TSQ caps the number of bytes a single TCP connection can have queued in the kernel's qdisc-layer + NIC-layer combined. Default is 1 MB. If you have a fat congestion window (cwnd) of say 30 MB, TCP would happily dump 30 MB into the qdisc. Without TSQ that would queue 30 MB of one flow, starving every other flow. With TSQ, only 1 MB is in the qdisc at a time, and the rest waits in TCP's send queue, where it is fungible (TCP can retransmit, reorder, change congestion behaviour).

### Why TSQ matters

The classic problem TSQ solves: a fast TCP flow on a fast link can fill the qdisc with one flow's packets, even if fq_codel is supposed to be fair. fq_codel's fairness only kicks in when there are packets from multiple flows in the queue. If one flow can dump 30 MB before any other flow shows up, that flow wins.

TSQ paper-thins each flow's footprint in the qdisc to ~1 MB so that fq_codel and fq can quickly see and balance many flows. Combined with FQ-style scheduling, TSQ is what makes the modern Linux network stack actually fair under load.

### Tuning TSQ

```bash
# default TSQ limit per TCP socket — 1048576 = 1 MB
$ cat /proc/sys/net/ipv4/tcp_limit_output_bytes
1048576

# tighten to 256 KB on a low-bandwidth box
$ sudo sysctl -w net.ipv4.tcp_limit_output_bytes=262144

# loosen to 4 MB on a 100 Gbps server
$ sudo sysctl -w net.ipv4.tcp_limit_output_bytes=4194304
```

The right number depends on your bandwidth-delay product. A 10 Gbps link with 50 ms RTT has a BDP of ~62 MB. You don't want every flow to have 62 MB in the qdisc — you want each flow to inject a couple of milliseconds worth of bytes and then back off. 1 MB at 10 Gbps is roughly 0.8 ms of data — so you'd be slightly underclocking, but on a busy server with thousands of flows, 1 MB per flow is plenty.

### TSQ + fq

`fq` qdisc has its own per-flow byte-cap (`maxrate`, `quantum`) that interacts with TSQ. The two are layered: TSQ caps how much TCP shoves into the qdisc, fq caps how much the qdisc passes downward to the NIC.

```bash
# inspect the fq qdisc on eth0
$ tc -s qdisc show dev eth0
qdisc fq 8001: root refcnt 2 limit 10000p flow_limit 100p buckets 1024 …
                 quantum 3028b initial_quantum 15140b …
```

`flow_limit 100p` means each flow can have at most 100 packets queued in fq. With TSQ at 1 MB and average packet 1500 B, a single flow has 1 MB / 1500 = ~700 packets bottled up in TCP and at most 100 in fq. That gives fq breathing room to fair-share between many flows.

## BBR (Bottleneck Bandwidth and RTT) — Google's modern congestion control that obviates much queue management

> BBR is TCP that tries to figure out the actual capacity of the path and the actual round-trip latency, and then sends at exactly that rate, instead of guessing by stuffing bytes into the network until something drops.

Imagine you're driving on a freeway. The old way (Reno, CUBIC) is to floor the gas pedal and keep flooring until you crash into someone, then slow down a bit, then floor it again, then crash, then slow down a bit, then floor it again. You never know the actual speed limit; you only know it by hitting the wall.

BBR drives differently. BBR watches how fast traffic is moving around it. BBR notices "ah, the cars in front of me are doing 60 mph, and there's no slow-down anywhere in sight, so the road's true speed limit must be 60 mph." Then BBR drives at 60 mph. No crashing. No flooring and braking.

Specifically, BBR maintains two estimates:

- `BtlBw` (Bottleneck Bandwidth) — the maximum delivery rate the path supports, computed from `bytes_acked / elapsed_time` over recent windows.
- `RTprop` (Round-Trip Propagation) — the minimum RTT seen recently, which is the unloaded round-trip latency.

Then BBR sends at a pacing rate = `BtlBw`, with a congestion window cap of `BDP = BtlBw * RTprop`. Most of the time, this means BBR sends exactly the amount the path can carry, no more. The bottleneck queue stays empty. There is no buildup. There is no drop. Bufferbloat is sidestepped entirely.

### Why BBR is a game-changer for queue management

The whole reason we need RED, CoDel, fq_codel, and friends is because TCP fills queues until they overflow. Reno and CUBIC are loss-based — they only know the path is full when a packet drops. So they always probe upward until something drops. That's bufferbloat-as-a-feature.

BBR is rate-based. It does not rely on loss to detect the limit. It detects the limit by watching delivery rate flatten out. So BBR never deliberately fills queues. On a path where every flow is BBR, queues stay shallow even with no AQM. The bottleneck queue might have a few BDP of in-flight bytes during a bandwidth probe, but it never sustainably builds up.

In other words: if everyone ran BBR, you would not need RED or CoDel. The job AQM is doing — keeping queues short — is something BBR does to itself voluntarily.

### Caveats

BBRv1 has known fairness issues with CUBIC: BBR can crowd out CUBIC flows by ignoring loss signals that CUBIC respects. BBRv2 (and now BBRv3) fix this by being responsive to ECN marks and to loss in a more principled way.

BBR also has an issue on shallow-buffer paths where the BtlBw estimate undershoots (the path empties before BBR can probe) or overshoots (BBR keeps sending at last-known rate even after path narrows). Probe phases (`PROBE_BW`, `PROBE_RTT`) handle this but introduce small periodic queue spikes.

### Enabling BBR on Linux

```bash
# check what's available
$ sysctl net.ipv4.tcp_available_congestion_control
net.ipv4.tcp_available_congestion_control = reno cubic bbr

# check what's in use
$ sysctl net.ipv4.tcp_congestion_control
net.ipv4.tcp_congestion_control = cubic

# switch to BBR
$ sudo sysctl -w net.ipv4.tcp_congestion_control=bbr

# persist across reboots
$ echo 'net.ipv4.tcp_congestion_control = bbr' | sudo tee /etc/sysctl.d/99-bbr.conf

# BBR works best with fq qdisc (fq does pacing in software)
$ sudo sysctl -w net.core.default_qdisc=fq

# verify
$ tc qdisc show dev eth0
qdisc fq 8001: root refcnt 2 limit 10000p flow_limit 100p …
```

If you're on kernel 5.4+ you can use BBRv2 (out-of-tree patch) or BBRv3 (mainline as of 6.4). On older kernels you only get BBRv1 — works fine but be aware of the CUBIC fairness story.

### How BBR interacts with FQ_CODEL

With BBR, fq_codel still helps. Reasons:

- Other flows on the same box may not be BBR (e.g. CUBIC for HTTP/1, BBR for HTTP/3). fq_codel keeps them isolated from each other.
- BBR's PROBE_BW phase deliberately overshoots briefly. fq_codel marks ECN on the burst before it grows.
- Under heavy fan-in (lots of incoming flows merging), even BBR can briefly queue. fq_codel cleans it up.

So the modern best-practice stack is: BBR + fq_codel + ECN. BBR keeps queues shallow voluntarily, fq_codel polices the edge cases, ECN signals before any drop. Latency stays at the propagation floor under load.

## Queue Disciplines in Linux (qdisc): pfifo, pfifo_fast, htb, fq, fq_codel, cake, mq, mq+fq_codel, tbf, sfq

> A qdisc is a Linux kernel object that decides "of all the packets currently waiting to leave this network interface, which one goes next, and do any of them need to be dropped?"

In one Linux kernel there are dozens of qdiscs. Most are obscure or experimental. These are the ones you actually run into in 2026.

### pfifo

> Plain FIFO. Bounded by packet count.

```bash
# attach pfifo with a 1000-packet limit
$ sudo tc qdisc add dev eth0 root pfifo limit 1000
$ tc qdisc show dev eth0
qdisc pfifo 8001: root refcnt 2 limit 1000p
```

Cuts to the chase: it queues up to 1000 packets, drops tail when full, no fairness, no AQM. Useful for testing, never for production. pfifo is what you compare everything else against.

### pfifo_fast

> Three-band FIFO. The classic Linux default before kernel 3.5.

Three priority bands (high, mid, low) selected by the IP `tos` field. Within each band, FIFO. Across bands, strict priority — high drains before mid, mid before low.

```bash
$ tc qdisc show dev eth0
qdisc pfifo_fast 0: root refcnt 2 bands 3 priomap 1 2 2 2 1 2 0 0 1 1 1 1 1 1 1 1
```

Modern kernels prefer fq_codel as the default. pfifo_fast still appears as the default on some distributions and inside containers without sysctl tuning.

### htb (Hierarchical Token Bucket)

> Tree of token-bucket shapers. Lets you split bandwidth between classes with a strict cap and optional bursting.

```bash
# 100 Mbit total link, two child classes: 60 Mbit for "video", 40 Mbit for "everything else"
$ sudo tc qdisc add dev eth0 root handle 1: htb default 30
$ sudo tc class add dev eth0 parent 1: classid 1:1 htb rate 100mbit
$ sudo tc class add dev eth0 parent 1:1 classid 1:10 htb rate 60mbit ceil 100mbit
$ sudo tc class add dev eth0 parent 1:1 classid 1:30 htb rate 40mbit ceil 100mbit
```

Each class has `rate` (guaranteed) and `ceil` (burst max). Tokens accumulate in a bucket and are spent when a packet is dequeued. Idle classes can lend tokens to busy classes up to `ceil`.

htb is the workhorse of bandwidth shaping. Tin-stack qdiscs like cake replace it for most consumer cases, but htb is still king for serious multi-tenant rate limiting, ISP gateway shaping, and any case where you have a clear hierarchy of services.

### fq (Fair Queue, no CoDel)

> Per-flow fair queueing with TCP pacing support. No AQM.

```bash
$ sudo tc qdisc replace dev eth0 root fq
$ tc qdisc show dev eth0
qdisc fq 8001: root refcnt 2 limit 10000p flow_limit 100p buckets 1024 \
        orphan_mask 1023 quantum 3028b initial_quantum 15140b
```

fq is what BBR wants underneath: each flow gets its own bucket, packets are paced according to TCP's pacing rate, no queue grows long. fq does not do CoDel-style marking — it relies on TCP itself to keep queues shallow.

Use fq when you're running BBR. Use fq_codel when you're running CUBIC or mixed.

### fq_codel

> Per-flow fair queueing + per-flow CoDel AQM.

```bash
$ sudo tc qdisc replace dev eth0 root fq_codel
$ tc qdisc show dev eth0
qdisc fq_codel 8001: root refcnt 2 limit 10240p flows 1024 quantum 1514 \
        target 5ms interval 100ms memory_limit 32Mb ecn drop_batch 64
```

This is the right default for most boxes in 2026. Each flow has its own queue, CoDel marks/drops to keep queue sojourn time under 5 ms, ECN-capable flows get marked instead of dropped.

Defaults:

- `target 5ms` — sojourn time goal
- `interval 100ms` — minimum RTT estimate
- `quantum 1514b` — DRR quantum (bytes given per round to each flow)
- `flows 1024` — number of hash buckets
- `memory_limit 32Mb` — total queue memory cap
- `ecn` — turns ECN marking on (default since 4.20-ish; explicit on most distros)

Tune by raising `target` if your link is over-active and underutilising; lower `target` if you want extra-low latency and don't mind some throughput cost.

### cake

> CAKE = Common Applications Kept Enhanced. The "everything-but-the-kitchen-sink" qdisc for home gateways.

Includes:

- 8-tin DiffServ-aware shaping
- Flow isolation (per-host then per-flow, by default)
- DOCSIS, ATM, and Ethernet overhead compensation
- COBALT (CoDel + BLUE hybrid) AQM
- Native dual-stack ack-filter

```bash
# 200 Mbit down, 20 Mbit up cake — typical home setup
$ sudo tc qdisc replace dev eth0 root cake bandwidth 200mbit
$ sudo tc qdisc replace dev ifb0 root cake bandwidth 20mbit ingress
$ tc qdisc show dev eth0
qdisc cake 8001: root refcnt 2 bandwidth 200Mbit besteffort triple-isolate \
        nonat nowash no-ack-filter split-gso rtt 100ms noatm overhead 0
```

Use cake when you want a single qdisc that just handles everything for a whole-house router. Replaces htb+fq_codel for almost all home use cases.

### mq (Multi-Queue)

> A meta-qdisc. Doesn't queue anything itself; just maps to one child qdisc per hardware tx queue.

```bash
$ tc qdisc show dev eth0
qdisc mq 0: root
qdisc fq_codel 0: parent :1 limit 10240p flows 1024 …
qdisc fq_codel 0: parent :2 limit 10240p flows 1024 …
qdisc fq_codel 0: parent :3 limit 10240p flows 1024 …
qdisc fq_codel 0: parent :4 limit 10240p flows 1024 …
```

If your NIC has 4 tx queues (multi-queue NIC), mq attaches a separate fq_codel to each. This is essential — otherwise your single root qdisc becomes a CPU bottleneck because every tx happens through one lock.

On modern multi-queue NICs (mlx5, ixgbe, igb, virtio-net with multi-queue), the kernel auto-selects mq+fq_codel as the root. You usually don't need to configure it. To change the per-queue qdisc:

```bash
# replace each child with cake at 1 Gbit
$ for q in $(seq 1 $(nproc)); do
    sudo tc qdisc replace dev eth0 parent mq:$q cake bandwidth 1gbit
  done
```

### mq + fq_codel

The default-of-defaults on modern Linux. mq at the root, fq_codel under each hardware queue. This is what `net.core.default_qdisc=fq_codel` plus a multi-queue NIC gives you out of the box.

### tbf (Token Bucket Filter)

> Single shaper. Limits an interface to a fixed rate. Simpler than htb but less flexible.

```bash
$ sudo tc qdisc replace dev eth0 root tbf rate 50mbit burst 32kbit latency 50ms
$ tc qdisc show dev eth0
qdisc tbf 8001: root refcnt 2 rate 50Mbit burst 32Kb latency 50ms
```

Use tbf for quick "cap this interface at 50 Mbit, don't care about anything else" jobs. For multi-class shaping, use htb or cake.

### sfq (Stochastic Fair Queueing)

> The original fair-queue qdisc. Hash flows into buckets and round-robin between buckets.

```bash
$ sudo tc qdisc replace dev eth0 root sfq perturb 10
$ tc qdisc show dev eth0
qdisc sfq 8001: root refcnt 2 limit 127p quantum 1514b depth 127 flows 128 \
        divisor 1024 perturb 10sec
```

`perturb 10` rotates the hash every 10 seconds to avoid persistent collisions. sfq is from the 1990s, and fq_codel is its modern descendant with CoDel bolted on. Use fq_codel unless you have a very specific reason to want flat sfq.

## Hardware Queues vs Software Queues (NIC ring vs kernel qdisc)

> A packet leaving your machine traverses two queues: the kernel's qdisc (software) and the NIC's transmit ring (hardware). Each behaves differently. Understanding the split is the difference between a working AQM and a useless one.

Picture a fast-food drive-thru. The line of cars in the parking lot is the software queue (kernel qdisc). The pickup window itself is the hardware queue (NIC ring). Cars drift from the parking-lot line up to the window one at a time as the window finishes a customer.

If the parking-lot line is huge and the window is small, the manager (the kernel) controls everything: who goes first, who gets booted out for being too slow, who can cut in line. Useful intelligence happens in the parking lot.

If the pickup window is huge — say, 100 cars deep inside the building — the manager has lost control. By the time a car is at the window, the manager can't change its mind. The cars in the building queue in arrival order, no fairness, no AQM, no nothing. Bufferbloat happens inside the building.

The job of BQL is to keep the inside-the-building queue tiny. Just one or two cars deep. So the manager (kernel) keeps almost all of the line in the parking lot, where it can be reordered, dropped, marked.

### Software queue (kernel qdisc)

Lives in `/sys/class/net/eth0/queues/tx-N/`. Each tx queue has:

- a qdisc (set with `tc`)
- a BQL controller
- a per-CPU sender path

Software queue is where AQM happens. CoDel marks here. Fair queueing happens here. ECN flips here. The qdisc has visibility into per-flow state, can drop or mark intelligently, and is bounded by configurable byte/packet limits.

### Hardware queue (NIC ring)

Lives on the NIC itself, accessed via DMA descriptor rings. Each entry points to a buffer in host RAM. The NIC reads the descriptor, DMAs the buffer onto the wire, marks the descriptor done. Then it interrupts the CPU.

Hardware queue is dumb. It is FIFO. It has no per-flow state. It cannot mark ECN (that requires looking inside the IP header and recomputing checksum, which the NIC may or may not support depending on offloads). It cannot drop intelligently. The only thing the hardware ring does well is shoot bytes onto the wire fast.

The hardware ring's size is set with ethtool:

```bash
$ ethtool -g eth0
Ring parameters for eth0:
Pre-set maximums:
RX:		4096
TX:		4096
Current hardware settings:
RX:		512
TX:		512

# shrink TX ring to 256 — pushes more queue into software where AQM can help
$ sudo ethtool -G eth0 tx 256

# shrink RX ring to 256 — reduces RX bufferbloat
$ sudo ethtool -G eth0 rx 256
```

Smaller rings mean more interrupts (each ring drain triggers an interrupt). On modern hardware with interrupt coalescing, this is a non-issue.

### The split, drawn

```
program → socket → TCP → IP → qdisc (sw queue) → BQL gate → NIC ring (hw queue) → wire
                                ^^^^^^^^^^^^^^^^^^^^^
                                       AQM happens here

                                                         no AQM, just FIFO
                                                                      ^^^
```

BQL is the gate between the two. Without BQL, the qdisc would happily push all its packets into the NIC ring, where they sit and rot. With BQL, the qdisc only releases what the NIC can drain in the next interrupt cycle, and everything else stays in the qdisc where it can be managed.

## Multi-Queue NICs (RSS, RFS, RPS — receive-side scaling)

> A modern NIC has many tx queues and many rx queues, one per CPU core. Receive-side scaling spreads incoming packets across cores so a single core doesn't become the bottleneck. RSS, RFS, and RPS are three different ways to do that, each with trade-offs.

Imagine a big call center. There are 32 phone operators (32 CPU cores). Calls coming in from outside need to be routed to operators. If every call goes to operator 1, operator 1 burns out and the other 31 sit idle. You want to spread the calls.

The simplest approach is round-robin: call 1 to operator 1, call 2 to operator 2, … call 32 to operator 32, call 33 to operator 1, etc. But that's bad for cache locality — if operator 1 already knows the customer who's calling about call 33 (because she handled their earlier call too), you want call 33 to go back to operator 1, not start fresh on operator 5.

So you hash by some key (the customer's phone number = the connection's 4-tuple) and route to whichever operator that hash points to. Now every call from the same customer goes to the same operator. That's RSS.

But maybe operator 1 is currently busy with a different complicated call, and operator 5 is free. RFS says: route the incoming packet to whichever operator is currently running the program that will receive it. RPS is a software-only fallback when the NIC can't hash on its own.

### RSS — Receive Side Scaling (in hardware, in the NIC)

The NIC computes a Toeplitz hash over the (src_ip, dst_ip, src_port, dst_port) 4-tuple of each incoming packet. The low bits of the hash select an rx queue. Each rx queue has its own MSI-X interrupt vector pinned to a specific CPU core. So the packet ends up handled on a specific core, with rx irqs distributed across cores.

```bash
# how many rx queues does the NIC expose?
$ ethtool -l eth0
Channel parameters for eth0:
Pre-set maximums:
RX:		32
TX:		32
Combined:	32
Current hardware settings:
Combined:	32

# show RSS hash key
$ ethtool -x eth0
RX flow hash indirection table for eth0 with 32 RX ring(s):
    0:      0     1     2     3     4     5     6     7
    8:      8     9    10    11    12    13    14    15
   16:     16    17    18    19    20    21    22    23
   24:     24    25    26    27    28    29    30    31
…

# set RSS hash to use src_ip + dst_ip + src_port + dst_port for TCPv4
$ sudo ethtool -N eth0 rx-flow-hash tcp4 sdfn
```

`sdfn` = src ip + dst ip + src port + dst port. There are flags for udp4, tcp6, udp6, etc.

### RFS — Receive Flow Steering (software-assisted hardware steering)

RFS is RSS with a memory: the kernel records "the last time we delivered a packet on flow F to userspace, the application was running on CPU N." Next time a packet on flow F arrives, the kernel programs the NIC to steer it to CPU N's rx queue. This keeps the application's data hot in CPU N's cache.

```bash
# enable RFS — hash table size 32k entries
$ echo 32768 | sudo tee /proc/sys/net/core/rps_sock_flow_entries

# set per-queue flow count
$ for q in /sys/class/net/eth0/queues/rx-*; do
    echo 2048 | sudo tee $q/rps_flow_cnt
  done
```

RFS requires NIC support (`ntuple` filtering). Most modern enterprise NICs support it.

### RPS — Receive Packet Steering (pure software)

RPS does the same hashing-and-steering RSS does, but in software, after the packet has already been received on the legacy single rx queue. Useful if your NIC has only one rx queue (e.g. virtio-net without multi-queue, old NICs).

```bash
# spread RPS to all 8 CPUs for rx queue 0
$ echo ff | sudo tee /sys/class/net/eth0/queues/rx-0/rps_cpus
```

The hex bitmap `ff` = CPUs 0..7. Each bit set means RPS will deliver to that CPU.

### XPS — Transmit Packet Steering

XPS picks which tx queue a process's packets go out on. Pin process P to CPU C, set XPS so packets from C go to tx queue C. Now tx and rx are both core-local.

```bash
# tx queue 0 → CPU 0, tx queue 1 → CPU 1, …
$ for q in 0 1 2 3 4 5 6 7; do
    printf '%x\n' $((1 << q)) | sudo tee /sys/class/net/eth0/queues/tx-$q/xps_cpus
  done
```

### When this matters

Below 10 Gbps, a single core handles rx fine. Above 10 Gbps you must use RSS or you'll be CPU-bound on softirq. At 100 Gbps, you'll usually use 8-32 rx queues plus aggressive interrupt coalescing.

Mq+fq_codel pairs naturally with RSS: each tx queue has its own fq_codel instance, each rx irq pins to its own core, no shared lock contention. This is what default modern Linux gives you.

## Common Errors

Real error strings you will see, and the canonical fix.

### `Error: Specified qdisc kind is unknown.`

```
$ sudo sysctl -w net.core.default_qdisc=fq_codell
sysctl: setting key "net.core.default_qdisc": Invalid argument
$ dmesg | tail -1
[12345.678] sysctl: net.core.default_qdisc: cannot set 'fq_codell' (unknown qdisc)
```

`net.core.default_qdisc` was set to a qdisc name the kernel does not recognise. Typo (`fq_codell` instead of `fq_codel`), or the qdisc module is not loaded. Fix:

```bash
# list loaded qdiscs
$ tc qdisc show
# load the module if missing
$ sudo modprobe sch_fq_codel
# verify
$ ls /sys/module/sch_fq_codel
$ sudo sysctl -w net.core.default_qdisc=fq_codel
```

### `RTNETLINK answers: Invalid argument` on `tc qdisc add`

```
$ sudo tc qdisc add dev eth0 root fq_codel target 5
RTNETLINK answers: Invalid argument
```

The argument `target 5` is not a valid time. fq_codel wants `5ms` not `5`. Many tc parameters require a unit. Fix:

```bash
$ sudo tc qdisc replace dev eth0 root fq_codel target 5ms interval 100ms
```

Other common typos:

- `limit 10000` (packets) is fine, but `limit 10000b` (bytes) needs context — htb wants bytes, fq_codel wants packets.
- `rate 100M` is invalid; use `rate 100mbit`.
- `quantum 1514b` is correct; `quantum 1514` is interpreted as bytes anyway but some kernels reject it.

### `RX dropped > 0` (NIC ring overflow)

```
$ ip -s link show eth0
2: eth0: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 …
    RX: bytes  packets  errors  dropped overrun mcast
    102837419121 8123412   0       12873   0       12
```

`dropped` on RX is the NIC throwing away packets because the rx ring filled up before the kernel could drain it. Causes:

- rx ring too small → `ethtool -G eth0 rx 4096`
- softirq starved → check `mpstat -P ALL 1` for `%soft`; pin irqs and use RSS
- kernel busy in another softirq path — check `cat /proc/interrupts`

### `TX dropped` (qdisc full)

```
$ ip -s link show eth0
    TX: bytes  packets  errors  dropped carrier collsns
    72938472    827342    0       4823    0       0
```

`dropped` on TX = qdisc dropped because its limit was hit. Either:

- traffic is exceeding the rate cap and qdisc is correctly tail-dropping (working as intended for shaping)
- `limit` is too small, raise it: `tc qdisc change dev eth0 root fq_codel limit 20480p`

### `ECN-Capable Transport flag silently ignored by middlebox`

```
$ tcpdump -nv -i eth0 'tcp and (tcp[13] & 0xC0 != 0)'
…
IP (tos 0x2,ECT(0), ttl 64 …) 192.0.2.1.443 > 198.51.100.1.55432 …
…
# next hop strips ECN bits, downstream receiver sees tos 0x0
```

Some old middleboxes (especially carrier-grade NAT and ancient routers) zero the TOS byte, killing ECN. Symptoms: ECN negotiated, but no marks ever arrive, and CWR is never set.

Fix on your side: nothing you can do about the middlebox. You can run `tcpdump -nv 'tcp and (ip[1] & 0x03 != 0)'` at both ends to confirm the marking survives. If it doesn't, fall back to non-ECN AQM (`fq_codel noecn`).

### `CWR not propagated`

```
# capture at receiver
$ tcpdump -n -v 'tcp[tcpflags] & tcp-cwr != 0'
… 198.51.100.1.55432 > 192.0.2.1.443: Flags [P.W], …    # CWR set
# capture at sender
$ tcpdump -n -v 'tcp[tcpflags] & tcp-cwr != 0'
…                                                      # nothing — CWR never makes it back
```

The receiver-side ECE → sender-side CWR loop is broken. Usually a NIC offload bug. TSO/GSO can clobber TCP flags during segmentation on some old drivers.

```bash
# disable offloads to test
$ sudo ethtool -K eth0 tso off gso off gro off
# rerun the capture; if CWR now appears, the offload was the culprit
```

Update the driver, or live with offloads off if you can spare the CPU.

### `AQM enabled but bufferbloat persists` (hardware queue too big)

```
$ tc qdisc show dev eth0
qdisc fq_codel 8001: root refcnt 2 …
$ ping -c 100 -i 0.2 8.8.8.8 &
$ iperf3 -c speedtest.example.com -t 60   # heavy upload
…
64 bytes from 8.8.8.8: time=480 ms        # WTF latency under load
```

You configured fq_codel but the NIC's hardware tx ring is enormous and BQL is not active. The qdisc never sees packets in flight because they all ship to the NIC ring and queue there.

```bash
# check BQL
$ ls /sys/class/net/eth0/queues/tx-0/byte_queue_limits/
# if directory is empty or missing, BQL is not active — driver doesn't support it
# shrink the ring as a workaround
$ sudo ethtool -G eth0 tx 256
# rerun ping under load — should be near-zero rise
```

### `fq_codel not available — kernel < 3.5`

```
$ sudo tc qdisc add dev eth0 root fq_codel
RTNETLINK answers: No such file or directory
$ uname -r
3.2.0-…
```

fq_codel landed in kernel 3.5 (released 2012). Anything older needs a kernel upgrade. If you really cannot upgrade, fall back to `sfq perturb 10`. But seriously — upgrade.

### `/proc/sys/net/core/default_qdisc — should be fq_codel`

```
$ sysctl net.core.default_qdisc
net.core.default_qdisc = pfifo_fast
```

This is your distro defaulting to the ancient pfifo_fast. Set it:

```bash
$ echo 'net.core.default_qdisc = fq_codel' | sudo tee /etc/sysctl.d/99-qdisc.conf
$ sudo sysctl --system
$ sysctl net.core.default_qdisc
net.core.default_qdisc = fq_codel
```

Note: this only affects newly created interfaces. To apply to existing ones:

```bash
$ for d in $(ls /sys/class/net/); do
    [[ "$d" == "lo" ]] && continue
    sudo tc qdisc replace dev $d root fq_codel
  done
```

### `BQL warnings when limit too low`

```
$ dmesg | grep -i bql
[12345.678] bql: tx-0 dropped under-limit, limit_min=64
[12346.789] bql: tx-0 underrun, link idle for 50ms
```

BQL set its limit too tight and now the wire is going idle. Bump the floor:

```bash
$ echo 4500 | sudo tee /sys/class/net/eth0/queues/tx-0/byte_queue_limits/limit_min
```

### `RSS/RFS hash collision causing CPU imbalance`

```
$ mpstat -P ALL 1
CPU    %usr   %sys  %soft  %idle
all    0.50   1.20  18.00  80.30
0      0.00   0.00  85.00  15.00
1      0.00   0.00   2.00  98.00
2      0.00   0.00   1.00  99.00
3      0.00   0.00   0.00 100.00
```

CPU 0 is hammered with softirqs while 1-3 are idle. RSS is hashing all your traffic to the same rx queue (which is irq-pinned to CPU 0). Causes:

- only one TCP flow (RSS hashes the 4-tuple, one flow → one queue)
- bad hash key
- single-source-IP traffic and hash key only uses dst-ip

Fix:

```bash
# enlarge hash to include ports
$ sudo ethtool -N eth0 rx-flow-hash tcp4 sdfn
$ sudo ethtool -N eth0 rx-flow-hash udp4 sdfn

# enable RPS as a software fallback to spread single-flow rx
$ echo ff | sudo tee /sys/class/net/eth0/queues/rx-0/rps_cpus
```

For genuinely single-flow workloads (a single 40 Gbps flow), no amount of RSS will help — you need application-level sharding or aRFS.

### `tc: Specified class not found` when changing fq_codel parameters

```
$ sudo tc qdisc change dev eth0 root fq_codel target 3ms
Error: Specified class not found.
```

Subtle: `change` requires the qdisc to already exist with the same kind. If you try to `change` a fq_codel that's actually a pfifo_fast, you get this. Fix with `replace`:

```bash
$ sudo tc qdisc replace dev eth0 root fq_codel target 3ms interval 100ms
```

`replace` works whether or not the qdisc exists; `change` only modifies an existing matching qdisc.

### `ethtool: Operation not supported` on `-G`

```
$ sudo ethtool -G eth0 tx 256
Cannot set device ring parameters: Operation not supported
```

The driver does not support changing ring sizes at runtime. Common on virtio-net, some embedded NICs. The ring is hardcoded at probe. Workaround: pass module parameters at insmod time, or live with the default.

### `default_qdisc only applies to new interfaces`

```
$ sudo sysctl -w net.core.default_qdisc=fq_codel
$ tc qdisc show dev eth0
qdisc pfifo_fast 0: root refcnt 2 …      # still pfifo_fast!
```

Yes — `default_qdisc` applies to interfaces created after the sysctl, not retroactively. To switch existing interfaces:

```bash
$ sudo tc qdisc replace dev eth0 root fq_codel
```

Or bring the interface down and up:

```bash
$ sudo ip link set eth0 down && sudo ip link set eth0 up
```

### `tc: Cannot find specified qdisc.` on delete

```
$ sudo tc qdisc del dev eth0 root
Error: Cannot find specified qdisc.
```

The interface has no qdisc, or it has the implicit default which can't be deleted. Use `replace` to swap to another qdisc, or accept that the default can't be removed without bringing the interface down.

## See Also

- `aqm-deep-dive` — the AQM family in depth
- `bufferbloat` — the disease this all treats
- `tcp-congestion-control` — Reno, CUBIC, BBR detail
- `linux-network-stack` — the full path a packet takes
- `ethtool` — every offload, ring, and driver knob

## References

- Tom Herbert, *Byte Queue Limits*, LWN 2011, https://lwn.net/Articles/454390/
- Eric Dumazet, *TCP Small Queues*, LWN 2012, https://lwn.net/Articles/507065/
- Cardwell, Cheng, Gunn, Yeganeh, Jacobson, *BBR: Congestion-Based Congestion Control*, ACM Queue 2016
- Linux kernel source: `net/sched/sch_fq_codel.c`, `net/sched/sch_cake.c`, `net/sched/sch_fq.c`
- `man 8 tc`, `man 8 tc-fq_codel`, `man 8 tc-cake`, `man 8 tc-htb`
- `Documentation/networking/scaling.rst` in the kernel tree (RSS/RFS/RPS)

## Hands-On

You are about to type a lot of commands. Each one is short. Each one teaches one thing. Read what the computer prints back. The output is where the learning lives.

Some of these commands need `sudo` because they touch the network stack. If your prompt does not have `#` at the end, put `sudo` in front of every `tc`, `sysctl -w`, `ethtool -K`, and `ip link` command. Read-only commands like `tc qdisc show`, `cat`, and `ip -s link show` do not need `sudo`.

Replace `eth0` with whatever your real interface is called. Run `ip -br link` to see your interface names. On laptops it is often `wlan0` or `wlp3s0`. On servers it is often `eno1` or `ens33`. On Docker hosts it might be `enp0s3`.

If a command says "command not found," install the package: `tc` lives in `iproute2`, `ethtool` lives in `ethtool`, `ss` lives in `iproute2`, `bpftrace` lives in `bpftrace`, `perf` lives in `linux-tools-common` or `perf`.

### See the qdisc on every interface

```bash
$ tc qdisc show
qdisc noqueue 0: dev lo root refcnt 2
qdisc fq_codel 0: dev eth0 root refcnt 2 limit 10240p flows 1024 quantum 1514 target 5ms interval 100ms memory_limit 32Mb ecn drop_batch 64
qdisc fq_codel 0: dev wlan0 root refcnt 2 limit 10240p flows 1024 quantum 300 target 5ms interval 100ms memory_limit 4Mb ecn drop_batch 64
```

The line for `lo` (loopback) says `noqueue` — loopback never queues, it just hands packets straight back to the kernel. The line for `eth0` says `fq_codel`, which is the modern Linux default. The numbers after each parameter (`limit 10240p`, `flows 1024`, `target 5ms`) are the live tunables. Keep these in your head — we will edit them later.

### See just one interface

```bash
$ tc qdisc show dev eth0
qdisc fq_codel 0: root refcnt 2 limit 10240p flows 1024 quantum 1514 target 5ms interval 100ms memory_limit 32Mb ecn drop_batch 64
```

Same line as before, just the one we care about. `target 5ms` is CoDel's "if delay stays above 5ms for an interval, start dropping." `interval 100ms` is CoDel's measurement window. `ecn` means ECN marking is enabled — packets get the CE bit instead of being dropped when possible.

### Add fq_codel as the root qdisc

```bash
$ sudo tc qdisc add dev eth0 root fq_codel
```

No output means success. Most `tc` commands are silent on success. To check, run `tc qdisc show dev eth0` and look for `fq_codel` in the line. If you get `RTNETLINK answers: File exists` it means a qdisc is already attached — use `replace` instead of `add`.

### Add CAKE with a bandwidth limit

```bash
$ sudo tc qdisc replace dev eth0 root cake bandwidth 100mbit
$ tc qdisc show dev eth0
qdisc cake 8002: root refcnt 2 bandwidth 100Mbit diffserv3 triple-isolate nonat nowash no-ack-filter split-gso rtt 100ms raw overhead 0
```

CAKE is doing way more than fq_codel. `diffserv3` means three priority tins (bulk/best-effort/voice). `triple-isolate` means fairness across hosts AND across flows AND across destination IPs. `split-gso` means CAKE will break up large segmentation-offloaded super-packets so it can shape them accurately. `rtt 100ms` is its assumed RTT for sizing.

### Replace a qdisc atomically

```bash
$ sudo tc qdisc replace dev eth0 root fq_codel
```

`replace` does an `add` if nothing is there, or swaps the existing qdisc out if one is. Always prefer `replace` in scripts so you can re-run them safely.

### Show qdisc stats

```bash
$ tc -s qdisc show dev eth0
qdisc fq_codel 0: root refcnt 2 limit 10240p flows 1024 quantum 1514 target 5ms interval 100ms memory_limit 32Mb ecn drop_batch 64
 Sent 4823910283 bytes 4138291 pkt (dropped 142, overlimits 0 requeues 18)
 backlog 0b 0p requeues 18
  maxpacket 1514 drop_overlimit 0 new_flow_count 8312 ecn_mark 23 drop_overmemory 0
  new_flows_len 0 old_flows_len 0
```

`Sent` is total bytes/packets transmitted. `dropped 142` is packets the qdisc dropped (CoDel doing its job, or buffer overflow). `overlimits 0` is when the rate limiter said no — zero here because we have no rate limit. `ecn_mark 23` is the headline number for AQM — packets we marked CE instead of dropping. If `dropped` is huge and `ecn_mark` is zero, ECN isn't reaching peers. If `dropped` is zero and `backlog` is huge, the queue is filling up but isn't dropping yet — bad sign.

### Show class stats

```bash
$ tc -s class show dev eth0
class fq_codel :1 parent 0:
 (dropped 12, overlimits 0 requeues 0)
 backlog 0b 0p requeues 0
  deficit 1514 count 0 lastcount 0 ldelay 0us
class fq_codel :2 parent 0:
 (dropped 0, overlimits 0 requeues 0)
 backlog 0b 0p requeues 0
  deficit 1514 count 0 lastcount 0 ldelay 2.1ms
```

Per-flow detail. `ldelay` is the last measured queue delay for that flow. If you see `ldelay` above your target on a flow, that flow is eating the queue. `deficit` is the DRR deficit counter — how many bytes that flow has banked toward its next turn.

### NIC drop counters

```bash
$ ethtool -S eth0 | grep -i drop
     rx_dropped: 0
     tx_dropped: 0
     rx_fifo_errors: 0
     rx_missed_errors: 0
     rx_no_buffer_count: 0
     rx_csum_errors: 0
     tx_aborted_errors: 0
```

Hardware-level drop counts. `rx_dropped` going up means the NIC ring is full — the kernel didn't drain it fast enough (CPU overload, IRQ misrouting). `rx_no_buffer_count` is the same idea on Intel hardware. If these climb, increase `rx` ring or fix RPS/RSS, not the qdisc.

### Show ring buffer sizes

```bash
$ ethtool -g eth0
Ring parameters for eth0:
Pre-set maximums:
RX:             4096
RX Mini:        n/a
RX Jumbo:       n/a
TX:             4096
Current hardware settings:
RX:             1024
RX Mini:        n/a
RX Jumbo:       n/a
TX:             1024
```

`Pre-set maximums` is what the NIC supports. `Current` is what's set right now. `RX 1024` means 1024 descriptors — at 1500 bytes each that's about 1.5MB of buffer. Bigger isn't always better — bigger ring means more bufferbloat at the NIC layer.

### Resize ring buffers

```bash
$ sudo ethtool -G eth0 rx 4096 tx 4096
```

Silent on success. Some NICs require the link to bounce — be ready for a brief disconnect. Validate with `ethtool -g eth0`. Resist the temptation to maximize: leave defaults unless you have measured a real `rx_dropped` problem.

### Show offload features

```bash
$ ethtool -k eth0 | head -20
Features for eth0:
rx-checksumming: on
tx-checksumming: on
        tx-checksum-ipv4: on
        tx-checksum-ip-generic: off [fixed]
        tx-checksum-ipv6: on
scatter-gather: on
        tx-scatter-gather: on
        tx-scatter-gather-fraglist: off [fixed]
tcp-segmentation-offload: on
        tx-tcp-segmentation: on
        tx-tcp-ecn-segmentation: on
        tx-tcp-mangleid-segmentation: off
        tx-tcp6-segmentation: on
generic-segmentation-offload: on
generic-receive-offload: on
large-receive-offload: off [fixed]
rx-vlan-offload: on
tx-vlan-offload: on
ntuple-filters: off
receive-hash: on
```

`tcp-segmentation-offload (TSO)` lets the NIC split big TCP into MTU-sized packets in hardware. `generic-receive-offload (GRO)` coalesces inbound packets in software. Both reduce CPU. Both can hide latency from the qdisc by making the qdisc see fewer, bigger super-packets. CAKE's `split-gso` knob exists exactly to defeat this for shaping.

### Disable GRO

```bash
$ sudo ethtool -K eth0 generic-receive-offload off
$ ethtool -k eth0 | grep gro
generic-receive-offload: off
```

Useful when packet timing matters (real-time, gaming, telephony probes) or when you're debugging at low layers and want to see what really came in. Re-enable for general workloads — GRO is a big throughput win.

### Show interface stats

```bash
$ ip -s link show eth0
2: eth0: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 qdisc fq_codel state UP mode DEFAULT group default qlen 1000
    link/ether 52:54:00:c1:b3:7f brd ff:ff:ff:ff:ff:ff
    RX:  bytes packets errors dropped  missed   mcast
    481923012 4138291      0       0       0    1822
    TX:  bytes packets errors dropped carrier collsns
    482918321 4193827      0       0       0       0
```

`qdisc fq_codel` confirms the qdisc. `qlen 1000` is the kernel's per-device tx queue length (separate from the qdisc!). `RX dropped` here is software drops — the kernel ran out of socket buffer or the qdisc dropped. `errors` is CRC/framing failures.

### Doubly-stat link show

```bash
$ ip -s -s link show eth0
2: eth0: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 qdisc fq_codel state UP mode DEFAULT group default qlen 1000
    link/ether 52:54:00:c1:b3:7f
    RX:  bytes packets errors dropped  missed   mcast
    481923012 4138291      0       0       0    1822
    RX errors: length    crc   frame    fifo  missed
                    0      0       0       0       0
    TX:  bytes packets errors dropped carrier collsns
    482918321 4193827      0       0       0       0
    TX errors: aborted  fifo   window heartbeat transns
                     0     0        0         0       2
```

`-s -s` unlocks per-error-class breakdowns. `transns 2` means the link came up twice since boot.

### Show what the system default qdisc is

```bash
$ sysctl net.core.default_qdisc
net.core.default_qdisc = fq_codel
```

This is what new interfaces will use when they appear. On a modern distro it's `fq_codel`. On an older one it might still be `pfifo_fast` (tail-drop, the bad one). Change it.

### Set the default qdisc

```bash
$ sudo sysctl -w net.core.default_qdisc=fq_codel
net.core.default_qdisc = fq_codel
```

Make this stick across reboots by adding `net.core.default_qdisc = fq_codel` to `/etc/sysctl.d/99-network.conf`. Existing interfaces keep whatever qdisc is on them — change those with `tc qdisc replace`.

### Show TCP congestion control

```bash
$ sysctl net.ipv4.tcp_congestion_control
net.ipv4.tcp_congestion_control = cubic
```

`cubic` is the Linux default since 2.6.19 (2006). It is a loss-based congestion control: it grows the window until packets drop. With pure tail-drop queues that means CUBIC happily fills the queue, sees a drop, halves, and refills. CUBIC plus bufferbloat is the original sin.

### Switch to BBR

```bash
$ sudo sysctl -w net.ipv4.tcp_congestion_control=bbr
net.ipv4.tcp_congestion_control = bbr
```

BBR (Bottleneck Bandwidth and RTT) ignores loss and instead probes for RTT and bandwidth. It naturally avoids filling buffers because it stops sending faster than the bottleneck can drain. With BBR, bufferbloat is much less painful even when the qdisc is bad — but combine BBR with `fq_codel` for the best of both. If you get `Operation not permitted`, the `tcp_bbr` module isn't loaded — `sudo modprobe tcp_bbr` first.

### Show ECN setting

```bash
$ sysctl net.ipv4.tcp_ecn
net.ipv4.tcp_ecn = 2
```

`0` = disabled. `1` = always negotiate ECN inbound and outbound. `2` = accept ECN inbound, don't initiate outbound. Linux ships with `2` as a safe default. To enable fully:

```bash
$ sudo sysctl -w net.ipv4.tcp_ecn=1
net.ipv4.tcp_ecn = 1
```

You also want `net.ipv4.tcp_ecn_fallback=1` (default) so connections that try ECN and get blackholed retry without it.

### Backlog of softirq

```bash
$ cat /proc/sys/net/core/netdev_max_backlog
1000
```

This is the per-CPU softirq backlog — packets waiting to be moved off the NIC ring into the protocol stack. If this fills up, the kernel drops at the very front of the receive path. On 10G+ NICs you almost always need this raised: `sudo sysctl -w net.core.netdev_max_backlog=10000`.

### Network statistics

```bash
$ cat /proc/net/netstat | head -2
TcpExt: SyncookiesSent SyncookiesRecv SyncookiesFailed EmbryonicRsts PruneCalled RcvPruned OfoPruned OutOfWindowIcmps LockDroppedIcmps ArpFilter TW TWRecycled TWKilled PAWSActive PAWSEstab DelayedACKs DelayedACKLocked DelayedACKLost ListenOverflows ListenDrops TCPHPHits TCPPureAcks TCPHPAcks TCPRenoRecovery TCPSackRecovery TCPSACKReneging TCPSACKReorder TCPRenoReorder TCPTSReorder TCPFullUndo TCPPartialUndo TCPDSACKUndo TCPLossUndo TCPLostRetransmit TCPRenoFailures TCPSackFailures TCPLossFailures TCPFastRetrans TCPSlowStartRetrans TCPTimeouts TCPLossProbes TCPLossProbeRecovery TCPRenoRecoveryFail TCPSackRecoveryFail TCPRcvCollapsed TCPDSACKOldSent TCPDSACKOfoSent TCPDSACKRecv TCPDSACKOfoRecv TCPAbortOnData TCPAbortOnClose TCPAbortOnMemory TCPAbortOnTimeout TCPAbortOnLinger TCPAbortFailed TCPMemoryPressures TCPMemoryPressuresChrono TCPSACKDiscard TCPDSACKIgnoredOld TCPDSACKIgnoredNoUndo TCPSpuriousRTOs TCPMD5NotFound TCPMD5Unexpected TCPMD5Failure TCPSackShifted TCPSackMerged TCPSackShiftFallback TCPBacklogDrop PFMemallocDrop TCPMinTTLDrop TCPDeferAcceptDrop IPReversePathFilter TCPTimeWaits TCPSynRetrans TCPOrigDataSent TCPSynRetransOnSyn TCPDelivered TCPDeliveredCE TCPACKSkippedSynRecv TCPACKSkippedPAWS TCPACKSkippedSeq TCPACKSkippedFinWait2 TCPACKSkippedTimeWait TCPACKSkippedChallenge TCPWinProbe TCPKeepAlive TCPMTUPFail TCPMTUPSuccess TCPDelivered TCPDeliveredCE TCPAckCompressed TCPZeroWindowDrop TCPRcvQDrop TCPWqueueTooBig TCPFastOpenPassiveAltKey TcpTimeoutRehash TcpDuplicateDataRehash TCPDSACKRecvSegs TCPDSACKIgnoredDubious TCPMigrateReqSuccess TCPMigrateReqFailure
TcpExt: 0 0 0 0 0 0 0 0 0 0 0 0 0 0 13822 92831 0 198 0 0 1238211 8231 91728 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 12 1827 14 0 0 0 19 0 482 0 0 12 0 1 0 0 0 0 0 0 0 12 0 0 0 0 0 0 0 0 0 0 0 0 1827 0 8231910 0 8231928 198 0 0 0 0 0 0 0 0 0 0 8231928 198 0 0 0 0 0 0 0 8231 0 0 0
```

The interesting ones for queue management: `TCPDeliveredCE` is total packets delivered with the CE codepoint set (= ECN marking working). If that number is climbing, your AQM is talking to peers. If it's flat at zero on a busy box, ECN isn't crossing the path.

### RPS — receive packet steering

```bash
$ cat /sys/class/net/eth0/queues/rx-0/rps_cpus
00000000,00000000
```

Bitmask of CPUs that can process inbound packets from this RX queue in software. All zeros means RPS is off and only the CPU that handled the IRQ does the work. To spread load across CPU 0-3:

```bash
$ echo f | sudo tee /sys/class/net/eth0/queues/rx-0/rps_cpus
f
```

`f` = `0b1111` = CPUs 0,1,2,3.

### Byte Queue Limits

```bash
$ cat /sys/class/net/eth0/queues/tx-0/byte_queue_limits/limit_max
1879048192
$ cat /sys/class/net/eth0/queues/tx-0/byte_queue_limits/limit
3028
$ cat /sys/class/net/eth0/queues/tx-0/byte_queue_limits/limit_min
0
```

BQL caps how many bytes can be queued at the *device driver* layer below the qdisc. `limit` is the current dynamic value — DQL adjusts it based on observed completion. Without BQL the qdisc wastes effort because the driver hides another hidden queue. Modern Linux NICs almost all have BQL — verify with `ls /sys/class/net/eth0/queues/tx-0/byte_queue_limits/`.

### Old-school ifconfig

```bash
$ ifconfig eth0
eth0: flags=4163<UP,BROADCAST,RUNNING,MULTICAST>  mtu 1500
        inet 10.0.0.42  netmask 255.255.255.0  broadcast 10.0.0.255
        ether 52:54:00:c1:b3:7f  txqueuelen 1000  (Ethernet)
        RX packets 4138291  bytes 481923012 (459.6 MiB)
        RX errors 0  dropped 0  overruns 0  frame 0
        TX packets 4193827  bytes 482918321 (460.6 MiB)
        RX errors 0  dropped 0  overruns 0  carrier 0  collisions 0
```

`txqueuelen 1000` is the kernel-level egress queue length used by qdiscs that respect it (mq, prio, etc). `dropped` here is software drops, `overruns` is the NIC ring overflowing.

### Socket statistics

```bash
$ ss -tin
State     Recv-Q  Send-Q  Local Address:Port   Peer Address:Port
ESTAB     0       0       10.0.0.42:38211      140.82.114.4:443    cubic wscale:7,7 rto:204 rtt:3.187/2.412 ato:40 mss:1448 pmtu:1500 rcvmss:1448 advmss:1448 cwnd:10 bytes_sent:842 bytes_acked:843 bytes_received:5482 segs_out:8 segs_in:9 data_segs_out:2 data_segs_in:7 send 36.36Mbps lastsnd:1812 lastrcv:1808 lastack:1808 pacing_rate 72.7Mbps delivery_rate 36.36Mbps delivered:3 app_limited busy:23ms rcv_space:14600 rcv_ssthresh:64076 minrtt:3.103
```

`cubic` says CUBIC is in charge for this socket. `cwnd:10` is the congestion window. `pacing_rate` is the BBR/fq pacing budget. `rtt:3.187/2.412` is smoothed RTT and variance in ms. `delivered:3` and `delivery_rate` are BBR-style measurements. If `Recv-Q` is large the receiver isn't reading. If `Send-Q` is large the sender is congestion-blocked or peer rwnd-blocked.

### Socket statistics, more detail

```bash
$ ss -ntipo dst :443
State Recv-Q Send-Q Local Address:Port    Peer Address:Port    Process
ESTAB 0      0      10.0.0.42:38211       140.82.114.4:443
        cubic wscale:7,7 rto:204 rtt:3.187/2.412 mss:1448 pmtu:1500 cwnd:10 bytes_sent:842 ...
        timer:(keepalive,29sec,0)
```

`-p` adds process. `-o` adds timer info. Useful when something is mysteriously slow — look for `timer:(retrans,...)` showing repeated retransmissions, or `unacked:N` showing in-flight packets that haven't been ACKed.

### Path MTU discovery

```bash
$ ip route get 1.1.1.1
1.1.1.1 via 10.0.0.1 dev eth0 src 10.0.0.42 uid 1000
    cache mtu 1500 advmss 1460 ...
```

Tells you the resolved next-hop, the source IP the kernel will use, and the path MTU. If `mtu` here is below 1500, ICMP fragmentation-needed messages got through and PMTUD lowered it — possible network black hole. If a workload mysteriously hangs on big payloads check this.

### Watch ICMP in real time

```bash
$ sudo tcpdump -i any -nn icmp
tcpdump: data link type LINUX_SLL2
13:04:18.391829 eth0  In  IP 1.1.1.1 > 10.0.0.42: ICMP echo reply, id 12, seq 1, length 64
13:04:19.392133 eth0  Out IP 10.0.0.42 > 1.1.1.1: ICMP echo request, id 12, seq 2, length 64
```

Useful when probing latency under load. Run a `ping` in another window, run `iperf3` in yet another window pegging the link, watch ICMP RTT climb. With fq_codel, ICMP RTT under load stays close to baseline (this is the entire point of AQM). With pfifo_fast, ICMP RTT shoots into hundreds of ms.

### ECN-related TCP stats

```bash
$ netstat -s | grep -i ecn
    InCEPkts: 23
    InECT0Pkts: 18382
    InECT1Pkts: 0
    OutECT0Pkts: 23912
```

`InCEPkts` is packets we received with the CE bit (= a router on the path marked them). `InECT0Pkts` is packets we received with classic ECT(0). `OutECT0Pkts` is packets we sent with ECT(0) negotiated. Zero `InCEPkts` and high traffic = nobody on the path is doing AQM, you're either uncongested or a tail-drop is silently throwing your packets away.

### Trace transmit events with perf

```bash
$ sudo perf record -e net:net_dev_xmit -a sleep 5
[ perf record: Captured and wrote 8.231 MB perf.data (192831 samples) ]
$ sudo perf report --stdio | head -10
# Samples: 192K of event 'net:net_dev_xmit'
# Event count (approx.): 192831
#
# Overhead  Command          Shared Object        Symbol
# ........  ...............  ...................  ......................
#
    34.21%  iperf3           [kernel.kallsyms]    [k] dev_hard_start_xmit
    18.93%  swapper          [kernel.kallsyms]    [k] dev_hard_start_xmit
     5.12%  nginx            [kernel.kallsyms]    [k] dev_hard_start_xmit
```

`net:net_dev_xmit` is the tracepoint fired every time a packet leaves the device. Combine with `-g` for stacks to find what's actually generating tx traffic.

### bpftrace one-liner

```bash
$ sudo bpftrace -e 'kprobe:fq_codel_drop { @drops[comm] = count(); } interval:s:5 { exit(); }'
Attaching 2 probes...
@drops[iperf3]: 1382
@drops[curl]: 4
```

Counts CoDel drops by process for 5 seconds. The named process is whichever process owns the socket whose packet got dropped at egress. Useful for finding the loud neighbour.

### Interrupts per CPU

```bash
$ cat /proc/interrupts | head -3 | column -t
            CPU0    CPU1    CPU2    CPU3
   0:      27      0       0       0       IO-APIC   2-edge      timer
  43:    482918    0       0       0       PCI-MSI   524288-edge eth0-rx-0
```

If only CPU0 is handling `eth0-rx-0` IRQs, you have an IRQ pinning problem — that one CPU saturates and `rx_dropped` climbs. Spread IRQs across cores with `set_irq_affinity` (driver script) or `echo` into `/proc/irq/<n>/smp_affinity`.

### Set IRQ affinity

```bash
$ echo 2 | sudo tee /proc/irq/43/smp_affinity
2
```

`2` = `0b10` = CPU 1. Pins eth0-rx-0 to CPU 1. For a multi-queue NIC, run `set_irq_affinity all eth0` (a script shipped by the NIC vendor in `/usr/src/`) to map each queue to a distinct CPU.

### Multi-queue + fq_codel setup

```bash
$ sudo tc qdisc replace dev eth0 root mq
$ for i in 0 1 2 3; do sudo tc qdisc replace dev eth0 parent :$((i+1)) fq_codel; done
$ tc qdisc show dev eth0
qdisc mq 0: root
qdisc fq_codel 8001: parent :1 limit 10240p flows 1024 quantum 1514 target 5ms interval 100ms ecn
qdisc fq_codel 8002: parent :2 limit 10240p flows 1024 quantum 1514 target 5ms interval 100ms ecn
qdisc fq_codel 8003: parent :3 limit 10240p flows 1024 quantum 1514 target 5ms interval 100ms ecn
qdisc fq_codel 8004: parent :4 limit 10240p flows 1024 quantum 1514 target 5ms interval 100ms ecn
```

This puts a per-queue fq_codel under `mq` so each hardware queue has its own AQM. On modern kernels with multi-queue NICs this happens automatically when `default_qdisc=fq_codel`.

### HTB with classes

```bash
$ sudo tc qdisc replace dev eth0 root handle 1: htb default 10
$ sudo tc class add dev eth0 parent 1: classid 1:1 htb rate 100mbit ceil 100mbit
$ sudo tc class add dev eth0 parent 1:1 classid 1:10 htb rate 90mbit ceil 100mbit
$ sudo tc class add dev eth0 parent 1:1 classid 1:20 htb rate 10mbit ceil 50mbit
$ sudo tc qdisc add dev eth0 parent 1:10 fq_codel
$ sudo tc qdisc add dev eth0 parent 1:20 fq_codel
$ tc -s class show dev eth0
class htb 1:1 root rate 100Mbit ceil 100Mbit burst 1600b cburst 1600b
 Sent 0 bytes 0 pkt (dropped 0, overlimits 0 requeues 0)
 ...
class htb 1:10 parent 1:1 leaf 8001: prio 0 rate 90Mbit ceil 100Mbit burst 1480b cburst 1600b
 Sent 0 bytes 0 pkt
 ...
class htb 1:20 parent 1:1 leaf 8002: prio 0 rate 10Mbit ceil 50Mbit burst 1280b cburst 1500b
 Sent 0 bytes 0 pkt
 ...
```

This is the classic "bandwidth-shaping with priority classes" layout. `1:10` is the bulk class (90 Mbps guaranteed, 100 Mbps ceiling). `1:20` is the constrained class (10 Mbps guaranteed, 50 Mbps ceiling). Both have fq_codel underneath so each class gets its own AQM. `default 10` means unmatched traffic goes to class 1:10.

### Add a u32 filter

```bash
$ sudo tc filter add dev eth0 parent 1: protocol ip prio 1 u32 match ip dport 22 0xffff flowid 1:20
$ tc filter show dev eth0
filter parent 1: protocol ip pref 1 u32 chain 0
filter parent 1: protocol ip pref 1 u32 chain 0 fh 800: ht divisor 1
filter parent 1: protocol ip pref 1 u32 chain 0 fh 800::800 order 2048 key ht 800 bkt 0 flowid 1:20 not_in_hw
  match 00160000/ffff0000 at 20
```

Pushes SSH (port 22) traffic into the constrained class 1:20. Real production setups usually use DSCP marking from the application + a `tc filter` matching DSCP, but u32 ad-hoc filters are great for quick experiments.

### Bandwidth shaping with TBF

```bash
$ sudo tc qdisc replace dev eth0 root tbf rate 50mbit burst 32kbit latency 50ms
$ tc -s qdisc show dev eth0
qdisc tbf 8001: root refcnt 2 rate 50Mbit burst 4Kb lat 50.0ms
 Sent 38291 bytes 39 pkt (dropped 0, overlimits 0 requeues 0)
 backlog 0b 0p requeues 0
```

Token Bucket Filter — a simple shaper. `rate 50mbit` is the average rate, `burst 32kbit` is how much can go out in one go without rate-limiting, `latency 50ms` is how long packets can wait before being dropped. TBF on its own has no AQM, so for real use put fq_codel underneath:

```bash
$ sudo tc qdisc replace dev eth0 root handle 1: tbf rate 50mbit burst 32kbit latency 50ms
$ sudo tc qdisc add dev eth0 parent 1:1 fq_codel
```

### Ingress shaping with ifb

```bash
$ sudo modprobe ifb
$ sudo ip link set ifb0 up
$ sudo tc qdisc add dev eth0 handle ffff: ingress
$ sudo tc filter add dev eth0 parent ffff: protocol all u32 match u32 0 0 action mirred egress redirect dev ifb0
$ sudo tc qdisc replace dev ifb0 root cake bandwidth 100mbit
$ tc qdisc show dev ifb0
qdisc cake 8001: root refcnt 2 bandwidth 100Mbit diffserv3 triple-isolate ...
```

Ingress is the dirty trick. The kernel can't truly shape inbound (the bytes are already on the wire), but it can mirror inbound packets onto an Intermediate Functional Block (`ifb`) device and then put any qdisc on that device's egress. CAKE on `ifb0` will drop/mark inbound traffic as if it were egressing the ifb. Net effect: TCP senders back off.

### Test bufferbloat from CLI

```bash
$ # Terminal 1: hammer the link with iperf
$ iperf3 -c speedtest.example.com -t 60
...

$ # Terminal 2: measure RTT under load
$ ping -c 60 1.1.1.1
PING 1.1.1.1 (1.1.1.1): 56 data bytes
64 bytes from 1.1.1.1: icmp_seq=0 ttl=58 time=12.4 ms
64 bytes from 1.1.1.1: icmp_seq=1 ttl=58 time=11.8 ms
...
64 bytes from 1.1.1.1: icmp_seq=30 ttl=58 time=14.2 ms       ← still good with fq_codel
...
```

Compare two runs: one with `pfifo_fast` (you'll see RTT climb from 12ms to 800-1500ms), one with `fq_codel` (RTT stays under 20ms). That delta is the bufferbloat you fixed.

### dslreports speed test from CLI

```bash
$ pip install --user dslr-cli  # community wrapper
$ dslr-cli run
Phase: idle latency        12 ms
Phase: download             95 Mbit/s   bloat avg 25 ms   bloat max 47 ms   grade A
Phase: upload                9 Mbit/s   bloat avg 11 ms   bloat max 18 ms   grade A
```

`bloat` rows are the ones you want to read. Anything over 100ms is a fail. With fq_codel/CAKE properly configured these stay sub-50ms even on saturated links.

### CAKE with overhead and ingress

```bash
$ sudo tc qdisc replace dev eth0 root cake bandwidth 100mbit overhead 18 ingress
```

`overhead 18` accounts for Ethernet framing overhead so CAKE shapes accurately. `ingress` flips CAKE into a mode that's friendlier to inbound shaping (it tries harder to drop early and signal back to senders).

### sch_etf — earliest tx first (TSN)

```bash
$ sudo tc qdisc replace dev eth0 parent 1:10 etf clockid CLOCK_TAI delta 1500000 offload
```

For deterministic packet timing — Time-Sensitive Networking. `delta 1500000` is 1.5ms of slack. `offload` pushes scheduling into the NIC. Used in industrial / pro audio / pro video where jitter must be sub-microsecond.

### Disable a qdisc

```bash
$ sudo tc qdisc del dev eth0 root
$ tc qdisc show dev eth0
qdisc pfifo_fast 0: root refcnt 2 bands 3 priomap 1 2 2 2 1 2 0 0 1 1 1 1 1 1 1 1
```

`del dev eth0 root` removes the root qdisc. The kernel falls back to the default specified in `net.core.default_qdisc`, but if no qdisc is set at all the kernel uses `pfifo_fast` (tail-drop). Don't leave a system in this state.

### Check what was negotiated

```bash
$ ss -tin state established | grep -i ecn
        cubic ecnseen wscale:7,7 ...
```

`ecnseen` flag means we received ECN markings from the peer this session. If `ecnseen` is missing on long-lived connections, the path or peer isn't honouring ECN.

### Live qdisc stats

```bash
$ watch -n 1 'tc -s qdisc show dev eth0'
```

Re-runs every second. Watch `dropped`, `ecn_mark`, and `backlog` counters in real time as you drive load.

## Common Confusions

**Wrong:** `pfifo_fast` is fine because it's the historical default.

**Right:** `pfifo_fast` is the textbook bufferbloat villain. It's tail-drop with three priority bands and no AQM. Modern Linux defaults to `fq_codel` for exactly this reason — `sysctl net.core.default_qdisc` should read `fq_codel` on any current install. If it still says `pfifo_fast`, change it.

---

**Wrong:** ECN replaces AQM — turning on ECN means I don't need a smart queue.

**Right:** ECN is just a different way for AQM to *signal* — instead of dropping a packet to tell the sender to slow down, AQM marks the packet's CE bit. You still need an AQM (CoDel, PIE, RED) running underneath ECN. ECN with no AQM does nothing because nobody is making the marking decision. ECN is the messenger; AQM is the manager.

---

**Wrong:** Pause Frames (802.3x) solve queueing — let the switch tell the sender to stop.

**Right:** Pause Frames are link-layer flow control. They tell the *previous hop* to shut up, which freezes the entire link including unrelated flows (head-of-line blocking) and propagates congestion backward through the network ("congestion spreading"). Pause Frames are useful in a single-hop lossless fabric (RoCE, FCoE) but they are not AQM. AQM happens *inside* the queue and signals via drop/mark, not via stopping the upstream link.

---

**Wrong:** BBR makes queue management irrelevant because it doesn't fill buffers.

**Right:** BBR is much better at not bloating buffers than CUBIC, because it stops sending when it estimates the bottleneck is full instead of waiting for loss. But BBR + a bad qdisc still hurts: BBR's sender estimate is based on observed RTT, and a tail-drop queue on the path will introduce variable RTT, fooling BBR. BBR also competes badly with CUBIC neighbours on a shared bottleneck (BBR can crowd out loss-based flows). Combine BBR with `fq_codel` and you get good fairness and low latency.

---

**Wrong:** Bufferbloat is a hardware problem — buy a better router.

**Right:** Bufferbloat is a software problem on a hardware substrate. The hardware in your home router has plenty of buffer; the problem is the firmware uses a tail-drop FIFO. Most consumer routers running OpenWrt with `sqm-scripts` (which configures CAKE) become low-latency overnight on the same hardware. Hardware vendors do ship "smart queue" toggles now, but it's the algorithm that fixed it, not the silicon.

---

**Wrong:** A "good" queue depth is as large as possible.

**Right:** Good queue depth is approximately the bandwidth-delay product of the path. Bigger than that just adds latency without throughput gain. Smaller than that risks underutilization. For a 100 Mbps WAN with 50ms RTT, BDP = 100e6 * 0.05 / 8 = 625 KB, about 425 packets. Anything larger is just queue space TCP will fill with no benefit. AQM exists to keep the *occupancy* short even when the buffer itself is generous.

---

**Wrong:** `fq_codel` is the default since Linux 4.20 so it's running everywhere.

**Right:** Two caveats. First, `default_qdisc` is `fq_codel` on most distros now, but some embedded/server distros lag — always check `sysctl net.core.default_qdisc`. Second, the default qdisc applies only to *egress*. Ingress is harder — you need the `ifb` mirror trick, or to apply AQM at your upstream router instead of your end host. End-host fq_codel does nothing for inbound bufferbloat.

---

**Wrong:** I can't shape ingress so I have to live with inbound bufferbloat.

**Right:** You can absolutely shape ingress with `ifb`. Mirror ingress packets onto a virtual `ifb0` device, then attach any qdisc (CAKE works great) to that device's egress. The qdisc will drop or mark, and senders will back off via TCP feedback. Limit to slightly below your real downstream rate so the queue actually exists at *your* router instead of upstream at your ISP's silent tail-drop.

---

**Wrong:** Bufferbloat measurements are noisy — just look at peak throughput.

**Right:** Throughput numbers hide bufferbloat almost completely. The whole point of bufferbloat is that you get *high throughput at the cost of* latency. dslreports.com, waveform.com, or `flent` (Flexible Network Tester) all measure latency *under load* — that's the number that matters. A link with 100 Mbps throughput and 800ms RTT under load is unusable for video calls; a link with 80 Mbps and 25ms is delightful.

---

**Wrong:** Goodput equals throughput.

**Right:** Throughput is bytes per second on the wire. Goodput is bytes per second of *useful application data* delivered. Throughput includes headers, retransmissions, and ACKs. Heavily congested paths can have high throughput and low goodput because retransmissions dominate. Dropped packets cost double — they took bandwidth on the way out and they prompt a retransmit. AQM by reducing queueing variance reduces retransmit rate, raising goodput at constant throughput.

---

**Wrong:** Only routers and middleboxes need AQM — end hosts don't queue.

**Right:** End hosts queue at every NIC tx ring and at every qdisc on every interface. A virtual machine has qdiscs on its tap, on the bridge, and on the physical uplink — three places to bloat. A Kubernetes node has qdiscs on `cni0`, on every `vethN`, on `eth0`. Container egress can saturate the bridge qdisc and stall everyone else on the host. End-host AQM matters.

---

**Wrong:** CoDel's `target=5ms` is too aggressive — drops too much.

**Right:** 5ms is fine for the public internet. CoDel measures *standing* delay, not peak — packets can briefly spend more than 5ms in the queue without triggering drops. Drop kicks in only after 5ms of delay sustained over a 100ms interval. For pure-LAN workloads the target can drop to 1ms; for high-RTT wireless paths CAKE's auto-tuning bumps target up. The default works for almost everyone.

---

**Wrong:** CAKE replaces fq_codel — it's just a more complex version.

**Right:** CAKE is a different operating point. Use fq_codel when you don't have or don't need a hard rate limit (line-rate Ethernet, modern data center). Use CAKE when you have a known bottleneck rate to enforce (home WAN, throttled VM uplink, deliberate egress shaping). CAKE includes a built-in shaper and DSCP-based tin separation; fq_codel does not. CAKE is one knob (`bandwidth`); HTB+fq_codel is many knobs. CAKE is heavier; fq_codel is lighter.

---

**Wrong:** PIE is just as common as CoDel — both are default.

**Right:** PIE was standardized in RFC 8033 and is mandatory in DOCSIS 3.1 cable modems, but in Linux end-host land it's rare. Almost every Linux deployment that runs an AQM runs CoDel (via fq_codel) or PIE (via fq_pie or a vendor implementation in cable modems). When troubleshooting Linux, expect fq_codel. When troubleshooting cable internet, expect PIE in the modem.

---

**Wrong:** ECN works everywhere — Linux ships with it on, that means it's negotiated.

**Right:** Linux ships `tcp_ecn=2` (accept ECN inbound, don't initiate outbound by default). ECN negotiation can fail in three ways: a middlebox strips the ECT codepoint (rendering CE marking impossible), a middlebox returns RST on ECN-bit SYNs (blackholing the connection), or the peer's TCP stack doesn't support ECN. Linux's `tcp_ecn_fallback=1` retries without ECN if blackholed. Verify ECN is actually working in production with `netstat -s | grep -i ecn` — `OutECT0Pkts` non-zero confirms outbound, `InCEPkts` non-zero confirms a router on the path is marking.

---

**Wrong:** L4S is shipping in production now.

**Right:** L4S (Low Latency Low Loss Scalable, RFC 9330+) is *deploying* in 2024-2026. Cable operators (Comcast, Vodafone) have started enabling dual-queue AQM in DOCSIS 3.1+ networks. Linux 5.18+ has TCP Prague support. Apple shipped L4S support in macOS Ventura/iOS 16. But end-to-end L4S — where your laptop, your ISP's middleboxes, and the server's stack all support L4S — is still rare in 2025. Watch this space; the wins are massive (sub-1ms queueing latency under load), but deployment is gated on every hop.

---

**Wrong:** TCP and the queue are independent — model each separately.

**Right:** TCP and the queue are a closed-loop control system: TCP's cwnd grows, fills the queue, the queue drops, TCP halves cwnd, queue drains, repeat. The "fluid model" of TCP+queue (Misra/Gong/Towsley 1999) shows the system as a coupled differential equation: queue size oscillates around a fixed point determined by AQM parameters and TCP's congestion algorithm. AQM design *is* the design of that fixed point. Treat them together.

---

**Wrong:** Higher delay-bandwidth product means I need a bigger buffer.

**Right:** Higher BDP means TCP needs a bigger *cwnd* to fill the pipe. The buffer needs to be approximately equal to BDP to handle one round-trip's worth of in-flight data without underflow. But the buffer doesn't need to be *occupied* at BDP — AQM keeps occupancy small while still allowing peak in-flight equal to BDP. Buffer sizing rule of thumb: provision BDP, run AQM, queue stays empty most of the time and full only briefly during bursts.

---

**Wrong:** AQM requires ECN — without ECN you can't do AQM.

**Right:** Reverse: AQM existed before ECN. RED (1993) drops packets. CoDel drops packets unless `ecn` is enabled. PIE drops packets. ECN is a *bonus* signal — when both endpoints support ECN, AQM marks instead of dropping; when they don't, AQM falls back to dropping. AQM with ECN is strictly nicer (signal without retransmit) but AQM without ECN works fine.

---

**Wrong:** Policing and shaping are the same thing.

**Right:** Policing drops packets that exceed a rate (sharp clip). Shaping queues packets that exceed a rate and releases them at the target rate (smooth out). Policing has zero buffer and zero latency added; shaping has a buffer and adds latency by design. Policing causes nasty bursts and TCP retransmits because it drops without warning. Shaping is friendlier to TCP because the queue absorbs bursts. tc has both: `action police` is policing, `tc qdisc add tbf` or `htb` is shaping.

## Vocabulary

queue — ordered list of packets waiting to be processed.
buffer — memory area holding queued packets.
FIFO — First In First Out, packets leave in the order they arrived.
drop — discard a packet, never to be transmitted.
mark — set a bit in a packet's header to signal congestion (ECN).
tail drop — drop new arrivals when the queue is full.
head drop — drop the oldest packet when the queue is full.
random drop — drop a randomly selected packet from the queue.
congestion — more arrivals than the link can carry.
congestion control — algorithm at the sender that adjusts rate based on signals.
AQM — Active Queue Management, intelligent dropping/marking inside the queue.
RED — Random Early Detection, drops with probability rising linearly with avg queue length.
WRED — Weighted RED, RED with per-class drop curves.
ARED — Adaptive RED, auto-tunes the drop probability.
ECN — Explicit Congestion Notification, the in-header signaling protocol.
ECT(0) — ECN-Capable Transport codepoint 0 (legacy ECN).
ECT(1) — ECN-Capable Transport codepoint 1 (used by L4S/Prague).
CE — Congestion Experienced, the marking applied by AQM.
CWR — Congestion Window Reduced, sender's ack of seeing CE.
ECE — ECN-Echo, receiver echoing CE back to sender.
CoDel — Controlled Delay, AQM that drops based on dwell time.
target — CoDel parameter, max acceptable standing delay before dropping.
interval — CoDel parameter, the measurement window over which target must be exceeded.
ce_threshold — CoDel parameter, dwell time above which to mark instead of drop.
FQ-CoDel — Fair Queueing with CoDel per flow.
flows — FQ-CoDel parameter, hash buckets count.
quantum — DRR parameter, bytes credited per flow per round.
PIE — Proportional Integral controller Enhanced, AQM with rate-based dropping.
target delay — PIE parameter, the delay the controller tracks.
alpha/beta — PIE controller gains.
CAKE — Common Applications Kept Enhanced, full-stack home gateway qdisc.
ingress — packets coming in.
egress — packets going out.
shaping — slowing packets to a target rate using a queue.
policing — dropping packets that exceed a target rate.
htb — Hierarchical Token Bucket, classful shaping qdisc.
tbf — Token Bucket Filter, simple single-rate shaper.
sfq — Stochastic Fairness Queueing, hash-bucketed fair queueing.
prio — strict-priority bands qdisc.
pfifo_fast — three-band tail-drop FIFO, the legacy default.
mq — multi-queue dummy qdisc, exposes per-NIC-queue children.
qdisc — queueing discipline, Linux's pluggable queue object.
class — internal subdivision within a classful qdisc.
filter — rule that classifies packets into qdisc classes.
action — operation a filter applies (mirror, redirect, police, mark).
hash filter — high-performance multi-bucket filter.
u32 filter — match arbitrary bit patterns at byte offsets.
fw filter — match Netfilter mark.
basic filter — match using ematch language.
bpf filter — match using a BPF program.
ToS — Type of Service byte in the IPv4 header.
DSCP — Differentiated Services Code Point, top 6 bits of ToS.
Diffserv — Differentiated Services framework for class marking.
AF — Assured Forwarding, Diffserv class family (AF11..AF43).
EF — Expedited Forwarding, Diffserv class for low-jitter traffic.
CS — Class Selector, Diffserv class for IP precedence compatibility.
DP — Drop Precedence, secondary AF parameter.
PHB — Per-Hop Behavior, Diffserv contract per class.
bandwidth — capacity of the link in bits per second.
throughput — bytes per second of all data on the wire.
goodput — bytes per second of useful application data delivered.
latency — one-way or round-trip delay in milliseconds.
jitter — variance in latency.
RTT — Round-Trip Time.
srtt — Smoothed Round-Trip Time, exponentially-weighted average.
cwnd — congestion window, bytes the sender allows in-flight.
rwnd — receive window, bytes the receiver advertises.
slow start — TCP phase where cwnd doubles per RTT.
congestion avoidance — TCP phase where cwnd grows linearly.
fast retransmit — retransmit on three duplicate ACKs.
fast recovery — keep cwnd halved after fast retransmit, no slow-start.
CUBIC — loss-based congestion control with cubic-curve growth (Linux default).
Reno — original loss-based TCP congestion control.
NewReno — Reno with multiple-loss-per-RTT handling.
Vegas — delay-based congestion control.
Westwood — bandwidth-estimation congestion control.
BBR — Bottleneck Bandwidth and RTT, model-based congestion control.
BBRv2 — BBR with ECN response and fairness improvements.
BBRv3 — BBR with L4S/Prague compatibility.
scalable congestion control — small reactions to many small signals (DCTCP, Prague).
packet pacing — spacing packets at the sender to match estimated rate.
gso — Generic Segmentation Offload, software TCP segmentation deferred to driver.
tso — TCP Segmentation Offload, hardware TCP segmentation.
lso — Large Segment Offload, generic name for hardware segmentation.
lro — Large Receive Offload, hardware coalescing of inbound packets.
gro — Generic Receive Offload, software coalescing on receive.
BQL — Byte Queue Limits, per-driver-queue byte cap.
DQL — Dynamic Queue Limits, BQL's adaptive sizing algorithm.
L4S — Low Latency Low Loss Scalable, the new low-latency Internet architecture.
dual-queue AQM — L4S architecture with classic and L4S queues sharing capacity.
accurate ECN — RFC 9438 extension giving senders precise CE counts.
Prague — TCP variant for L4S, scalable congestion control.
DCTCP — Data Center TCP, ECN-based scalable congestion control.
DCQCN — Data Center Quantized Congestion Notification (RoCE).
RoCEv2 — RDMA over Converged Ethernet v2.
PFC — Priority-based Flow Control, per-class link-layer pause.
DCBX — Data Center Bridging Exchange, advertises PFC/ETS/QCN configs.
NIC ring buffer — circular descriptor array between driver and hardware.
RX queue — hardware receive queue on a NIC.
TX queue — hardware transmit queue on a NIC.
transmit timestamping — hardware/software stamping at TX completion.
sch_etf — Earliest Tx First qdisc for time-sensitive networking.
sch_taprio — Time-Aware Priority qdisc, IEEE 802.1Qbv scheduling.
ifb — Intermediate Functional Block, virtual device used to apply qdisc to ingress.
eBPF qdisc — sch_bpf, programmable qdisc using BPF programs.
clsact — classifier action, ingress+egress filter hook on every device.
bufferbloat — pathological queueing latency caused by oversized unmanaged buffers.
ILBQ — Idle Link Below Quota, FQ-CoDel's "old flow" condition.
TSQ — TCP Small Queues, sender-side per-socket buffer cap.
sk_pacing_rate — per-socket pacing rate set by congestion control.
fq qdisc — fair queueing per-flow with pacing support.
SFQ — Stochastic Fair Queueing, simpler hashed fair queueing.
DRR — Deficit Round Robin, fair scheduler with byte deficit accounting.
WFQ — Weighted Fair Queueing.
WRR — Weighted Round Robin.
SP — Strict Priority.
LLQ — Low Latency Queue, SP class with policing in Cisco gear.
CBWFQ — Class-Based Weighted Fair Queueing.
HQF — Hierarchical Queueing Framework (Cisco).
MDRR — Modified Deficit Round Robin (Cisco).
CBQ — Class-Based Queueing, classful WFQ-like.
LFQ — Low Flow Queueing, FQ-CoDel's "new flow" path.
CWND clamp — sysctl forcing a max cwnd.
TCP timestamps — RFC 7323 option for accurate RTT measurement.
window scaling — TCP option allowing rwnd > 65535 bytes.
SACK — Selective Acknowledgement, TCP option for non-contiguous ACKs.
DSACK — Duplicate SACK, signals received duplicates.
F-RTO — Forward RTO Recovery, spurious-retransmission detection.
RACK — Recent ACKnowledgement, time-based loss detection.
TLP — Tail Loss Probe.
PRR — Proportional Rate Reduction.
ER — Early Retransmit.
F-RACK — RACK with EWMA-tuned RTT.
HyStart — Hybrid Slow Start, CUBIC's slow-start exit heuristic.
HyStart++ — improved HyStart variant.
NV — TCP NV (New Vegas), GAE-flavor BBR-ish control.
DCTCP threshold — K parameter, queue length threshold for marking.
ECN-fallback — return to no-ECN if path blackholes ECN-marked SYN.
SYN — TCP synchronize, opens a connection.
SYN-ACK — TCP synchronize-acknowledge.
SYN cookie — server stateless SYN handling under flood.
TFO — TCP Fast Open, data on the SYN.
ICW — Initial Congestion Window, default 10 segments.
MTU — Maximum Transmission Unit.
MSS — Maximum Segment Size, MTU minus headers.
PMTUD — Path MTU Discovery.
PMTU black hole — ICMP-blocked path with mismatched MTU.
softirq — software interrupt, where Linux processes packets after an IRQ.
NAPI — New API, modern NIC poll/interrupt driver model.
GRO budget — max packets coalesced per NAPI poll.
RPS — Receive Packet Steering, software CPU spreading on receive.
RFS — Receive Flow Steering, RPS aware of flow-to-app affinity.
XPS — Transmit Packet Steering, choose tx queue based on CPU/flow.
RSS — Receive-Side Scaling, hardware hash to multiple RX queues.
flow hash — hash of 5-tuple used by RSS/RPS/fq.
cgroup net_cls — assign a class to a cgroup's traffic.
cgroup net_prio — assign a priority to a cgroup's traffic.
nftables — modern packet filter.
iptables — legacy packet filter.
conntrack — Netfilter connection tracking.
NIC — Network Interface Card.
PHY — physical layer, the cable-side electronics.
MAC — Media Access Control sublayer.
PCS — Physical Coding Sublayer.
SerDes — Serializer/Deserializer.
FEC — Forward Error Correction.
DP — Drop Precedence.
QFQ — Quick Fair Queueing.
HPFQ — Hierarchical Packet Fair Queueing.
FRL — Flow Rate Limiter.
WTP — Worst Time Phenomenon.
hashed buckets — fixed-size flow set used by SFQ/fq_codel.
flowid — tc identifier for a class.
classid — qdisc class id (major:minor).
handle — qdisc instance id.
parent — qdisc/class parent reference.
default — htb default classid for unmatched traffic.
priomap — pfifo_fast/prio TOS-to-band mapping.
peakrate — TBF peak rate.
mtu — TBF parameter, max packet size for shaping.
limit — generic qdisc parameter, max queue size in packets or bytes.
overhead — CAKE/HTB framing-overhead accounting parameter.
mpu — minimum packet unit for shaping accounting.
bandwidth-delay product — BW*RTT, a connection's worth of bytes in flight.
in-flight — bytes sent but not yet acknowledged.
inflight cap — sender's clamp on in-flight bytes.
ack clocking — TCP's self-clocking based on incoming ACKs.

## Try This

1. Run `tc qdisc show` on every machine you have. Note which use `fq_codel`, which use `pfifo_fast`. The legacy ones are the bufferbloat candidates.

2. On any Linux box: `iperf3 -c <server> -t 60` in one terminal, `ping -c 60 1.1.1.1` in another. Record the ping RTT distribution. Switch the qdisc to `pfifo_fast`, repeat. The difference is bufferbloat.

3. Attach `cake` with a deliberately low bandwidth (`bandwidth 10mbit`) to a test interface. Run a download and watch how RTT under load stays bounded — that's the value of integrated AQM+shaping.

4. `sysctl net.ipv4.tcp_congestion_control=bbr` on a sender. Curl a 1GB file from a remote server. Compare goodput and time-to-first-byte against `cubic` over the same path.

5. Watch `tc -s qdisc show dev eth0` once a second while running `iperf3` parallel streams. Watch `dropped` and `ecn_mark` counters. If your peer supports ECN, marks should dominate drops.

6. `bpftrace -e 'kprobe:fq_codel_drop_func { @[comm] = count(); }'` for 30 seconds during your daily workload. The top processes are your noisy flows.

7. Configure ifb-based ingress shaping at 90% of your real downlink. Re-test with dslreports.com. Ingress bloat should disappear.

8. Build an HTB tree with two classes: `bulk` (90 Mbps) and `latency` (10 Mbps with strict priority). Pin SSH to `latency`. Confirm SSH stays interactive while a `iperf3` saturates `bulk`.

9. On a quiet machine: `cat /proc/net/netstat | grep -E 'CEPkts|ECT'` before and after a known-ECN-capable connection (curl an HTTPS server with `tcp_ecn=1`). The counters should change.

10. Read the live state of one socket with `ss -tin dst :443`. Identify cwnd, srtt, pacing_rate. Run a download over that socket; rerun ss every second. Watch the numbers move.

11. Dump CAKE stats with `tc -s qdisc show dev eth0` and look at the per-tin rows (`bulk`, `besteffort`, `voice`). Run a Zoom call alongside a `iperf3` flood. Confirm voice tin stays drained.

12. Tweak `fq_codel target` from 5ms down to 1ms with `tc qdisc replace dev eth0 root fq_codel target 1ms`. Re-run your latency-under-load test on a low-RTT LAN. Smaller target should mean smaller standing delay.

## Where to Go Next

You've now got the queue management mental model. From here:

If you want to go deeper into the Linux side, read [`kernel-tuning/network-stack-tuning`](kernel-tuning/network-stack-tuning) for the full set of network sysctls and the order to tune them in.

If you want to go deeper into TCP, read [`ramp-up/tcp-eli5`](ramp-up/tcp-eli5) for the same gentle treatment of TCP itself, then [`networking/tcp`](networking/tcp) for the reference card.

If you want to go deeper into shaping and policy, read [`networking/qos-advanced`](networking/qos-advanced) for the multi-queue, classful, DSCP-marked variants, then [`networking/sp-qos`](networking/sp-qos) for service-provider-grade QoS.

If you want to study the data center side specifically (DCTCP, RoCE, PFC), read [`networking/cos-qos`](networking/cos-qos).

If you want to know why bufferbloat happens at the *kernel* layer, the [`ramp-up/linux-kernel-eli5`](ramp-up/linux-kernel-eli5) sheet is the prerequisite, then jump into the network-stack tuning sheet for actual sysctls.

If you're chasing absolute lowest latency (DPDK, AF_XDP, kernel-bypass), see [`networking/dpdk`](networking/dpdk) and [`networking/af-xdp`](networking/af-xdp). These bypass the qdisc entirely — different problem space, but related.

The endgame is L4S (RFC 9330 et al). Right now (2025) deployment is partial, but in a few years dual-queue AQM with TCP Prague will be the default for any latency-sensitive path. Keep an eye on the Linux changelogs for `tcp_prague`, `dualpi2`, and accurate ECN.

## See Also

- [networking/cos-qos](networking/cos-qos) — Class of Service / QoS reference card.
- [networking/qos-advanced](networking/qos-advanced) — Advanced QoS and shaping.
- [networking/sp-qos](networking/sp-qos) — Service-provider QoS.
- [networking/tcp](networking/tcp) — TCP reference.
- [networking/ecmp](networking/ecmp) — Equal-cost multipath.
- [networking/iptables](networking/iptables) — iptables filtering.
- [networking/nftables](networking/nftables) — nftables filtering.
- [networking/dpdk](networking/dpdk) — Data Plane Development Kit.
- [networking/af-xdp](networking/af-xdp) — AF_XDP express data path.
- [kernel-tuning/network-stack-tuning](kernel-tuning/network-stack-tuning) — Linux network sysctl tuning.
- [kernel-tuning/cpu-scheduler-tuning](kernel-tuning/cpu-scheduler-tuning) — CPU scheduler tuning (interacts with NAPI/softirq).
- [ramp-up/tcp-eli5](ramp-up/tcp-eli5) — TCP for absolute beginners.
- [ramp-up/ip-eli5](ramp-up/ip-eli5) — IP for absolute beginners.
- [ramp-up/linux-kernel-eli5](ramp-up/linux-kernel-eli5) — The Linux kernel for absolute beginners.
- [ramp-up/iptables-eli5](ramp-up/iptables-eli5) — iptables for absolute beginners.

## References

- RFC 2309 (1998), "Recommendations on Queue Management and Congestion Avoidance in the Internet" — Braden et al, the original AQM call to arms.
- RFC 3168 (2001), "The Addition of Explicit Congestion Notification (ECN) to IP" — Ramakrishnan, Floyd, Black.
- RFC 7567 (2015), "IETF Recommendations Regarding Active Queue Management" — Baker, Fairhurst, the modern AQM update to RFC 2309.
- RFC 8033 (2017), "Proportional Integral Controller Enhanced (PIE)" — Pan et al, PIE specification.
- RFC 8290 (2018), "The Flow Queue CoDel Packet Scheduler and Active Queue Management Algorithm" — Hoeiland-Joergensen, McKenney, Taht et al, FQ-CoDel specification.
- RFC 9330 (2023), "Low Latency, Low Loss, and Scalable Throughput (L4S) Internet Service: Architecture" — Briscoe et al, the L4S framework.
- RFC 9331 (2023), "The Explicit Congestion Notification (ECN) Protocol for Low Latency, Low Loss, and Scalable Throughput (L4S)" — Schepper, Briscoe.
- RFC 9332 (2023), "Dual-Queue Coupled Active Queue Management for Low Latency, Low Loss, and Scalable Throughput (L4S)" — Schepper, Briscoe, Tilmans.
- "Controlling Queue Delay" — Kathleen Nichols & Van Jacobson, Communications of the ACM, July 2012. The CoDel paper. The single most important paper on AQM.
- "Bufferbloat: Dark Buffers in the Internet" — Jim Gettys & Kathleen Nichols, Communications of the ACM, January 2012. The paper that named the problem.
- "BBR: Congestion-Based Congestion Control" — Neal Cardwell, Yuchung Cheng, C. Stephen Gunn, Soheil Hassas Yeganeh, Van Jacobson, Communications of the ACM, February 2017.
- "Computer Networking Problems and Solutions" — Russ White & Ethan Banks, Pearson, Chapter 8 covers queueing and scheduling end-to-end.
- "TCP Congestion Avoidance with a Misbehaving Receiver" — Savage, Cardwell, Wetherall, Anderson, ACM SIGCOMM 1999. Why congestion control assumes the receiver is honest.
- "Random Early Detection Gateways for Congestion Avoidance" — Sally Floyd & Van Jacobson, IEEE/ACM ToN 1993. The original RED paper.
- "Analysis and Design of Controllers for AQM Routers Supporting TCP Flows" — C. Hollot, V. Misra, D. Towsley, W. Gong, IEEE Transactions on Automatic Control, June 2002. The control theory of AQM.
- Linux kernel source: `net/sched/sch_codel.c`, `net/sched/sch_fq_codel.c`, `net/sched/sch_cake.c`, `net/sched/sch_pie.c`, `net/sched/sch_htb.c` — read the code.
- bufferbloat.net — community resources, mailing list archives, and `flent` test suite.
- `man tc`, `man tc-fq_codel`, `man tc-cake`, `man tc-htb`, `man tc-pie` — Linux man pages, surprisingly thorough.
- `Documentation/networking/scaling.rst` in the Linux kernel source — RPS/RFS/RSS/XPS authoritative reference.
- Toke Hoeiland-Joergensen's PhD thesis (2018), "Bufferbloat and Beyond" — the academic synthesis of the FQ-CoDel/CAKE work.
- Pollere LLC blog (Kathleen Nichols) — ongoing notes on CoDel/AQM evolution.
- The CeroWrt project — historical context, the testbed that proved CoDel/fq_codel in practice.

