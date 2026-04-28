# Anycast — ELI5

> Anycast is one phone number that rings the closest pizza store. You dial 1.1.1.1 from anywhere on Earth and the nearest copy of the server picks up.

## Prerequisites

- `ramp-up/ip-eli5` — what an IP address is and why every computer on the internet has one.
- `ramp-up/bgp-eli5` — how routers tell each other which neighborhoods of the internet they can reach.

This sheet uses words like "IP address," "router," "BGP," and "AS path" without re-explaining them in full. If those words make you go "huh?" then go read the two prereq sheets first. They are short. Come back here. We will wait.

If a word feels weird, look it up in the **Vocabulary** table near the bottom of this sheet. Every weird word has a one-line plain-English definition.

If you see a `$` at the start of a line in a code block, that means "type the rest of this line into your terminal." You do not type the `$`. The lines underneath that don't have a `$` are what your computer prints back. We call that "output."

## What Even Is Anycast

### The first thing to know

Before we get into pizza analogies, here is the one-sentence summary you can carry around forever and pull out at parties:

**Anycast is when a bunch of different physical computers, sitting in different cities, all answer to the same IP address, and the internet's routing system automatically picks the closest one for you.**

That's the whole thing. That is anycast. Every other detail in this 2,000-line sheet is a footnote on that sentence. We are going to spend the rest of the sheet picking that sentence apart, because every word in it is doing real work and most of those words have surprises hiding inside.

The surprising words, in case you missed them:

- "**different physical computers**" — not one computer; many. Could be five, could be five hundred.
- "**different cities**" — geographically scattered. Tokyo, London, São Paulo, Frankfurt, Sydney.
- "**all answer to the same IP address**" — yes, the **same** IP. They lie about who they are. Or rather, they all agree to play the same role.
- "**internet's routing system**" — specifically BGP, the Border Gateway Protocol, which is the thing that decides how packets get from one network to another.
- "**automatically picks the closest one**" — you didn't ask for the closest one. Nothing in your packet says "give me the closest server." But you got it anyway. That is the magic of anycast.
- "**for you**" — for **you**, specifically. Two different people in two different cities will both send packets to the same IP and get answers from totally different machines. They are both right. Each is hitting their nearest copy.

Now we will dive into pictures and analogies and concrete examples until your gut understands all of those words.

### The pizza-store picture

Pretend you live in a country with a very famous pizza chain called **Anycast Pizza**. Every Anycast Pizza store across the entire country shares one phone number: `555-PIZZA`. There is no "the New York store number" and "the Tokyo store number." There is one number. One. Every store has the same number.

You pick up the phone. You dial `555-PIZZA`.

The phone rings.

But here is the trick: the **closest** Anycast Pizza store is the one whose phone actually rings. If you live in New York, the New York store rings. If you live in Tokyo, the Tokyo store rings. If you live in London, the London store rings. You did not pick. The phone company picked. The phone company looked at where you were calling from, and it sent your call to the nearest store that owns that number. You dialed one number. A store nearest to you picked up.

That is anycast.

The "phone number" is an **IP address**. The "stores" are **servers**. The "phone company" is the **internet's routing system** (BGP). When you send a packet to `1.1.1.1`, every router on the path looks at its routing table, sees that `1.1.1.1` is reachable through several different neighbors, and picks the neighbor that gets there fastest (in terms of "AS hops"). The packet flows that way. It arrives at whichever Anycast Pizza store is closest. That store answers.

You did not configure anything. You did not say "give me the New York server." You just sent a packet to `1.1.1.1`. The internet picked the server for you.

### Why is this useful?

Three reasons. We will keep coming back to these.

**Speed.** The closest server answers, so the round-trip is short. Short round-trips mean fast websites. Fast websites mean happy people.

**Failure tolerance.** If the New York store burns down, the New York store stops "answering the phone." Routers notice (because the BGP announcement disappears) and they re-route calls to the next-closest store. Phone calls keep working. People keep getting pizza. The New York people get slightly slower service while the New York store rebuilds, but nobody gets dial-tone-of-death.

**Load.** Lots of people calling the pizza chain at once? That's fine — the calls are spread across every store automatically. The Tokyo store handles Tokyo calls. The London store handles London calls. No single store gets crushed.

That's anycast in three sentences. A shared phone number. The closest one rings. If it dies, the next closest takes over.

### A second picture: emergency services

Where you live, there is probably one emergency phone number. In the US it's 911. In the UK it's 999. In most of the EU it's 112. You dial that number and **a** call center picks up. Not "the" call center. **A** call center. Whichever one is closest to your geographic location.

When you dial 911 in New York City, the New York City 911 dispatch center answers. When you dial 911 in Los Angeles, the Los Angeles 911 dispatch center answers. Same number. Different physical building. Different staff. Different ambulance fleet. But it does not matter. The number gets you to **the right** dispatch center for **where you are**. The phone system routes you there.

That is exactly how `1.1.1.1` works on the internet. You send packets to `1.1.1.1`. They arrive at the nearest Cloudflare data center. There are roughly 300+ Cloudflare data centers in the world. Each one runs servers that answer for `1.1.1.1`. Each data center is essentially a "1.1.1.1 dispatch center." You don't pick. The internet picks.

### A third picture: branch offices of one company

Imagine a giant law firm called Same Address Lawyers Inc. They operate offices in 50 cities. Their corporate address is "1 Lawyer Plaza." Every single office, in every single city, **claims** to be at "1 Lawyer Plaza." There is a 1 Lawyer Plaza in New York. A 1 Lawyer Plaza in San Francisco. A 1 Lawyer Plaza in Frankfurt.

When you ask Google Maps for directions to 1 Lawyer Plaza, Google Maps sends you to whichever 1 Lawyer Plaza is closest to your starting point. If you set off from Brooklyn, you go to the New York office. If you set off from Berkeley, you go to the San Francisco office. If you set off from Frankfurt, you walk down the street to the Frankfurt office. You don't pick the office. The directions pick the office for you.

That is anycast. The "address" is the IP. The "directions" are BGP. Every office has agreed to use the same address, and the directions service automatically routes you to the closest one.

### A fourth picture: the radio station with many transmitters

Think of a national radio station. Maybe a sports radio network that broadcasts in dozens of cities. The station has one name (let's say "Sports Radio 88"). Every city has its own physical transmitter for Sports Radio 88. Each transmitter is on the same frequency: 88.0 MHz. Each transmitter plays the same content (the same baseball game, the same talk show, the same commercials).

When you tune to 88.0 MHz on your car radio, you don't pick a transmitter. You don't even know there are many transmitters. You pick the **frequency**. Whichever transmitter is closest to your car, that is the one you hear. As you drive across the country, your car radio quietly switches transmitters at some point — when one transmitter's signal gets weaker than the next one's signal — but you don't notice because they all play the same content.

The "frequency" is the IP address. The "transmitters" are the anycast servers. The "content" is the data each server is configured to serve. The "car radio's automatic switching" is BGP picking the shortest path. You drive across the country and your packets quietly shift between data centers as the routing system rebalances. You don't notice because the data centers are all serving the same content.

This is a particularly good picture because it captures something important: **all the transmitters serve the same content**. If one transmitter played a different baseball game than the others, drivers near the boundary would hear weird audio cuts as their radios switched between transmitters. This would be confusing and bad. So the radio station goes to a lot of trouble to keep all transmitters in sync.

Anycast operators do the same thing for their servers: they go to **a lot** of trouble to keep all the anycast nodes serving identical content. We'll see why this matters when we discuss "common pitfalls" later.

### A fifth picture: the chain of identical hotels

A worldwide hotel chain — let's call it Anycast Inn. Anycast Inn has 300 hotels around the world. Every hotel has the same name. Every hotel has the same room layout. Every hotel has the same kind of beds, the same kind of breakfast, the same kind of robes in the bathroom. They are deliberately identical. The whole point of the chain is that you check into Anycast Inn anywhere on Earth and the experience is the same.

When you book "an Anycast Inn room," you don't have to specify a city. The booking system looks at where you are and gives you the nearest one. You arrive. The room is what you expect. The breakfast is what you expect.

Two days later you fly to a different country and book "an Anycast Inn room" again. Same brand, same booking system, same expectation. You get the room. It's the same kind of room. You're happy.

That is anycast. The "brand" is the IP address. The "hotels" are the anycast nodes. The "booking system" is BGP. The "identical experience" is the requirement that every node serves the same content.

What if Anycast Inn had a hotel in Frankfurt that served bacon for breakfast and a hotel in Tokyo that served only fruit? Customers traveling between cities would get whiplash. They would say "this brand is broken." So Anycast Inn invests in keeping every hotel as identical as possible. Same with anycast operators.

### Why we keep using these analogies

You cannot see anycast. It is invisible. There is no light that goes on. There is no door that opens. The whole thing happens inside routers and inside the BGP protocol and inside packet headers. You ping `1.1.1.1`, and you have no way to **see** that the response came from Tokyo and not from London. The only clues are timing (round-trip time) and indirect signals (different POPs put slightly different things in `chaosnet` DNS responses). For a learner, you need pictures. The phone-number picture, the 911 picture, the branch-office picture — they all describe the same thing.

The thing they describe: **one IP address, many physical locations, the routing system delivers each packet to the closest location.**

### What anycast is NOT

Before we move on, let's nail down a few things anycast is **not**, because confusion abounds.

**Anycast is not load balancing.** A load balancer is one machine (or a small cluster) that receives all the traffic and distributes it to backend servers. The load balancer's IP is unicast. With anycast, there is no central load balancer; the **routing fabric itself** distributes traffic to the nearest copies. (You can do anycast in front of load balancers, and many CDNs do, but anycast and load balancing are different things.)

**Anycast is not DNS round-robin.** DNS round-robin returns multiple A records for one name, and the client picks one. Anycast uses **one** A record (one IP), and the network routes to multiple machines. The client thinks it's hitting one server. Different mechanism.

**Anycast is not VRRP / HSRP / first-hop redundancy.** VRRP gives you a "virtual IP" shared between two routers on the same LAN, where one is active and one is standby. That's local-LAN failover, not global routing. Anycast is global, BGP-driven, and **all** copies are active simultaneously.

**Anycast is not VIP migration.** Some systems can move a VIP between servers (cloud load balancers, keepalived). That's still unicast — the VIP is on **one** machine at a time, and migrates only on failure. Anycast has the same IP on **many** machines at the same time, and traffic is split routinely (not just on failure).

**Anycast is not "magic."** It just exploits BGP's shortest-path-wins behavior. Once you understand BGP, anycast is a 5-minute concept. Most of the complexity comes from making applications survive the routing-instability edge cases.

**Anycast is not a layer-7 thing.** It happens at layer 3 (network/IP) and layer 4 (transport, in the sense that 4-tuple flows can be re-routed). The application above it has no direct knowledge anycast is happening.

### A note on terminology

Sometimes people call the IP "an anycast IP." Sometimes they call it "an anycast address." Sometimes they call it "an anycast prefix" (referring to the routed prefix, like `1.1.1.0/24`, that contains the IP). These are roughly synonymous in casual conversation. In strict BGP terms, the **prefix** is what's announced, and the **address** is the specific IP within the prefix. So you announce `1.1.1.0/24` and you serve `1.1.1.1` (the address) inside that prefix.

Sometimes people call the per-site server "the anycast node," sometimes "the anycast POP," sometimes "the anycast peer," sometimes just "a copy." These are also roughly synonymous. We'll use "POP" most often, which is the CDN industry's term.

## Unicast vs Multicast vs Broadcast vs Anycast

There are four ways an IP address can map to physical machines. You need to know all four to understand anycast properly. Let's run through them.

### Unicast: one address, one machine

This is the boring normal kind. The kind you use every day without thinking about it. Your laptop has an IP. Your phone has an IP. Your friend's web server has an IP. Each of those addresses goes to **one** physical machine. You send a packet to `192.0.2.50`, the packet arrives at one specific server somewhere, and that server answers.

Picture: one telephone, one number, one house. You dial the number, the house's phone rings. End of story.

Most of the internet works this way. When you `ssh` into a cloud VM, you are unicasting. When you SCP a file to a friend, you are unicasting. When you visit a small website, you are usually unicasting (until the site grows big enough to want anycast).

### Multicast: one address, many machines, all of them get a copy

Multicast is "one-to-many delivery." You send one packet to the multicast address `224.0.0.5`, and the network duplicates that packet so that **every machine that has subscribed to `224.0.0.5`** receives a copy. The sender does not have to know who they are. The sender just sends. The network does the fan-out.

Picture: a school PA system. The principal speaks into a microphone. Every classroom that has plugged in a speaker hears the same announcement at (roughly) the same time. The principal didn't dial 200 phone numbers. The principal just spoke into one microphone. The network copied the audio.

Use cases: video streaming inside a corporate network, OSPF routers exchanging hellos with each other, financial market data feeds, Wake-on-LAN. On the public internet, multicast is mostly dead — ISPs don't carry it well between networks. Inside a single network, multicast still works fine.

### Broadcast: one address, every machine on this network gets a copy

Broadcast is "send to absolutely everyone on this LAN." There is a special address (`255.255.255.255` or the subnet's broadcast address like `192.168.1.255`) that means "every device on this layer-2 segment, please listen up."

Picture: a fire alarm. It does not know who is in the building. It does not care. It just shrieks loudly enough that **every** human on the floor reacts.

In IPv4 you use broadcast for things like DHCP discovery (your laptop yells "anyone got an IP for me?" to the whole subnet, and the DHCP server answers). In IPv6, broadcast is **gone** — replaced entirely by multicast.

### Anycast: one address, many machines, ONLY ONE of them gets each packet

Now we get to the star of the show.

Anycast is: same IP address announced from multiple physical locations, but **only one** of those locations receives any given packet. Not all of them. Not none of them. Exactly one. The network's routing decides which one.

Picture: that pizza chain again. You dial `555-PIZZA`. The phone company hands your call to **one** store — the closest one. The other stores don't ring. They never even know you tried. Each call goes to exactly one store, and the store that gets it depends on where the call came from.

So if you compare them all in a tiny table:

| Cast type   | "One address" maps to                          | Who gets the packet?              |
|-------------|-----------------------------------------------|-----------------------------------|
| Unicast     | Exactly one machine                           | That one machine                  |
| Multicast   | A "group" — many machines that subscribed     | All members of the group          |
| Broadcast   | A subnet — all machines in that LAN           | All machines in that LAN          |
| Anycast     | Many machines that share the address          | Exactly one — the "closest"       |

The "closest" word is in quotes because **closest** in routing-speak doesn't mean closest in geographic miles. It means "shortest BGP AS-path" or "lowest IGP metric" or whichever knob the routers use to decide. Often the geographically closest copy is also the routing-closest copy. Not always. We'll see exceptions later.

### A thing that trips people up

People sometimes think "broadcast and anycast are similar — they both fan out to multiple servers." They are **not** similar.

Broadcast = every machine receives a copy.
Anycast = exactly one machine receives a copy.

Anycast is much more like unicast (one packet, one destination, one reply) than it is like broadcast. The only weird part is that **which** one destination changes based on where you sent it from. A packet from New York might land on the New York server. The very next packet from London might land on the London server. Neither of them ever sees the other.

### A worked example for each cast type

Let me walk through one concrete, end-to-end example for each, just to nail it down.

**Unicast example.** You SSH from your laptop to your friend's home server at `203.0.113.42`. Your packet has destination IP `203.0.113.42`. It travels through your home router, your ISP, several internet backbones, your friend's ISP, and arrives at your friend's home router, which forwards it to her server. **Exactly one** machine in the world has that IP. Your packet goes to that one. Your friend's server sends back replies to your IP, and they make the reverse journey. Boring, classic, works perfectly.

**Multicast example.** A bunch of OSPF routers on a corporate WAN need to exchange "hello" messages. They all join the multicast group `224.0.0.5` ("OSPFIGP-AllSPFRouters"). When one router needs to send a hello, it puts `224.0.0.5` in the destination IP. The network duplicates the packet to every router that's joined the group. **Every member** sees the hello. Nobody else does.

**Broadcast example.** Your laptop boots up. It needs an IP. It doesn't know any servers' IPs yet (it has no IP itself!). So it sends a DHCP DISCOVER packet with destination IP `255.255.255.255`. Every machine on the local LAN sees this packet (because that IP means "everyone here"). The DHCP server sees it and replies. Other machines see it too but ignore it. Broadcast is local-only — your laptop's broadcast doesn't escape your home network.

**Anycast example.** You type `dig @1.1.1.1 google.com` into your terminal. Your packet has destination IP `1.1.1.1`. It travels through your home router, your ISP. Your ISP has learned via BGP that `1.1.1.0/24` is reachable through three different upstream peers. Two of those peers have heard about it from Cloudflare-Tokyo (one path) and Cloudflare-LAX (another path). Your ISP picks the path with the shortest AS-path — say, the Tokyo path, because you're in Asia. The packet flows through the Tokyo path, arrives at the Cloudflare-Tokyo POP, and is answered. **One** Cloudflare server saw your packet. Many other Cloudflare servers also "own" `1.1.1.1` but didn't see your packet. The reverse reply comes back from `1.1.1.1` (sourced by Cloudflare-Tokyo) and arrives at your laptop.

If you'd been in California, the same packet would have gone to Cloudflare-LAX and never visited Tokyo at all. Different physical machine, different physical location, same IP, same answer.

## The Insight: BGP Already Picks Shortest Path — Just Lie About Where The Server Is

Why does anycast work? Why is it allowed?

The honest answer: anycast is a **trick**. It exploits something BGP was already doing for an unrelated reason.

### What BGP was doing

BGP, the Border Gateway Protocol, is the routing protocol that glues the internet together. Every internet provider runs BGP. Every router learns from its neighbors which IP prefixes (chunks of address space) are reachable through which neighbor. When a router has multiple paths to the same destination, it runs a **best-path algorithm** to pick one. The most famous tiebreaker in that algorithm is "**shortest AS-path**" — the route that goes through the fewest networks wins.

BGP runs the best-path algorithm so that every packet is delivered through a relatively short path. That's a normal optimization. The internet would be slower otherwise.

### How anycast cheats this

Now imagine you are Cloudflare. You have data centers in 300 cities. From every one of those data centers, you announce the prefix `1.1.1.0/24` to your BGP neighbors. You're saying, to every neighbor: "hey, send packets for `1.1.1.0/24` over here."

Each of your neighbors says "okay, got it" and tells **its** neighbors the same thing. The announcement spreads. After a few minutes, every router on the internet has learned that `1.1.1.0/24` is reachable through some path. But each router has learned from **its closest** Cloudflare announcement, because that's the one with the shortest AS-path.

A router in Tokyo says "I see `1.1.1.0/24` through Cloudflare-Tokyo, AS-path length 1." Yes, the Tokyo router could **also** see Cloudflare's announcement coming from London, but the London path's AS-path is longer (because it has to traverse other ASes to get there). So the Tokyo router picks the Cloudflare-Tokyo announcement. Its packets to `1.1.1.1` go to Cloudflare-Tokyo.

A router in Frankfurt picks Cloudflare-Frankfurt. Its packets go there.

A router in Reykjavík picks whatever Cloudflare data center has the shortest AS-path. Maybe London. Maybe Frankfurt. Whichever one its ISP peers more closely with.

Nobody told any router "send Tokyo's traffic to Cloudflare-Tokyo." The routers figured it out themselves, just by running normal BGP best-path selection. Cloudflare just announced the same prefix everywhere, and BGP did the rest.

That is the central trick of anycast. You don't need a new protocol. You don't need a new packet header. You don't need any new routing logic. You just announce the **same prefix** from **multiple places**, and BGP's normal "shortest path wins" rule sends each user to their closest copy.

### A picture in ASCII

```
                   ┌─────────────────┐
                   │  User in Tokyo  │
                   └────────┬────────┘
                            │ packet to 1.1.1.1
                            ▼
              ┌─────────────────────────────┐
              │ Tokyo ISP router            │
              │ Sees 1.1.1.0/24 from:       │
              │   Cloudflare-Tokyo  AS:1    │ ← shortest, picked
              │   Cloudflare-LON    AS:5    │
              │   Cloudflare-NYC    AS:8    │
              └────────────┬────────────────┘
                           ▼
                ┌──────────────────┐
                │ Cloudflare-Tokyo │
                │ answers as 1.1.1.1│
                └──────────────────┘
```

Every router in the world plays this same game. Each picks its closest copy. The result: **every user is automatically served by the nearest copy** without anybody configuring per-user routing.

### Why this is genius

Other approaches to "give each user the closest copy" are clunky.

**Geo-DNS** does it at the DNS layer: when a client looks up `cdn.example.com`, the DNS server inspects the client's source IP, guesses where the client is geographically, and returns a different IP for different geographies. This works but is fragile — the DNS server's idea of where the client is can be wrong (especially if the client uses a public resolver), and the IP changes can break TLS certs, caching, etc.

**HTTP redirects** do it at the application layer: client connects to a "main" site, the main site looks at the client's IP, and redirects them to a regional copy. This works but adds an extra round-trip every connection.

**Anycast** does it at the routing layer, before the client even knows what's happening. Client sends a packet. Network delivers it to the closest copy. No DNS tricks. No redirects. No client-side configuration. It just works, all the way down at layer 3.

That is why almost every major piece of "fast, global infrastructure" is anycast: DNS root servers, public resolvers, CDN edges, NTP pools, DDoS scrubbing services, even some cloud-provider load balancers. They share a single IP. The closest copy answers. Always.

### A picture of the path-decision

Here's a more elaborate ASCII diagram showing what happens inside a router that learns about an anycast prefix from multiple peers.

```
                     ┌─────────────────────────────────┐
                     │        Router in your ISP       │
                     │                                 │
                     │   BGP Routing Information Base  │
                     │                                 │
                     │   Prefix: 1.1.1.0/24            │
                     │   ┌────────────────────────┐    │
                     │   │ Path 1                 │    │
                     │   │  Next hop: AS 174      │    │
                     │   │  AS_PATH: [174 13335]  │    │
                     │   │  MED: 0                │    │
                     │   │  Local pref: 100       │    │
                     │   │  ✓ Best                │    │
                     │   ├────────────────────────┤    │
                     │   │ Path 2                 │    │
                     │   │  Next hop: AS 6939     │    │
                     │   │  AS_PATH: [6939 13335] │    │
                     │   │  MED: 0                │    │
                     │   │  Local pref: 100       │    │
                     │   │  (not best — same      │    │
                     │   │   length as 1, but 1   │    │
                     │   │   has lower router-id)│    │
                     │   ├────────────────────────┤    │
                     │   │ Path 3                 │    │
                     │   │  Next hop: AS 2914     │    │
                     │   │  AS_PATH: [2914 1299   │    │
                     │   │            13335]      │    │
                     │   │  (longer AS path,      │    │
                     │   │   not best)            │    │
                     │   └────────────────────────┘    │
                     │                                 │
                     │   Best path → Path 1 → AS 174   │
                     └─────────────────────────────────┘
```

The router has three candidate paths to `1.1.1.0/24`. Each path has the same destination AS (Cloudflare = 13335) but goes through a different intermediate AS. The router runs the BGP best-path algorithm:

1. Compare local preferences. All equal (100). Move on.
2. Compare AS-path lengths. Paths 1 and 2 are both length 2. Path 3 is length 3. **Path 3 eliminated.**
3. Compare origin codes. All equal. Move on.
4. Compare MEDs. All equal. Move on.
5. Prefer eBGP over iBGP. All eBGP. Move on.
6. Prefer the path with lower IGP cost to next-hop. Assume equal. Move on.
7. Prefer the path with lower router-ID. Path 1's neighbor has the lower router-ID. **Path 1 wins.**

So the router installs Path 1 in its forwarding table. Every packet for `1.1.1.0/24` goes to AS 174.

This whole decision is happening, with subtle variations, in **every router on the entire internet**, all the time. Each router has its own view of the AS topology and its own neighbors. Each router picks the locally-best path. Collectively, the entire internet routes each user to "their" closest Cloudflare POP — without any central coordinator deciding anything.

That decentralized convergence is BGP's superpower. Anycast just rides on top of it.

## IPv4 Anycast Limitations

Now the bad news: IPv4 doesn't actually have anycast as an explicit concept.

In IPv4, every IP address looks the same to the protocol. There is no "anycast bit" in the IP header. There is no field that says "this address belongs to multiple machines." From the protocol's perspective, anycast is **just unicast** — you send a packet to a unicast IP, and it gets delivered to a host that owns that IP. The fact that **many** hosts own the same IP is invisible to IP. The illusion of anycast lives entirely in the routing protocols (BGP, OSPF, IS-IS) above IP.

This has consequences.

### Consequence 1: it works because routing converges, not because IP says so

Anycast works in practice because BGP and IGP routes are stable. If routes flap (change rapidly), packets in a single session might land on different anycast nodes. The protocol doesn't care that they did, but the **application** running on the node may care a lot — especially if it's stateful.

When a route flaps, the network might send packet 1 of a TCP connection to Node-A and packet 2 of the same connection to Node-B. Node-B has no idea about the connection. It sends back a TCP RST (reset). Connection dies.

You can absolutely have anycast in IPv4. You just have to design for **routing stability** and for protocols that tolerate node changes.

### Consequence 2: there is no way to tell if an IP is anycast

You cannot look at a packet header and say "this packet was sent to an anycast address." The packet looks the same as any other packet. Tools like `traceroute` will trace one path, but if you trace twice you might trace different paths to different nodes. There is no flag.

### Consequence 3: middleboxes can break things

Some middleboxes (firewalls, NATs, load balancers) keep state per-connection. If they expect every packet of a connection to go through them, but anycast routing changes mid-flow and packets start going through a different middlebox, the new middlebox has no state and drops the packets. This is mostly a problem for symmetric-state middleboxes; stateless ones don't care.

### Consequence 4: source-address conflicts

If two anycast nodes both source packets from `1.1.1.1`, and both reply to the same client, the client might see weird interleaved responses. (In practice, only one node sees a given request, so only one node replies — but if your routing changes between request and reply, you can get into trouble.)

### Why we still do it

Even with all these caveats, IPv4 anycast is **everywhere**. The 13 DNS root servers? Anycast. Cloudflare's `1.1.1.1`? Anycast. Google's `8.8.8.8`? Anycast. Your CDN edge? Anycast. The whole internet runs on IPv4 anycast for resolvers, CDNs, NTP, and a thousand other things. It works because:

1. Most internet routing is stable enough that flow re-routing is rare.
2. The protocols that anycast is most useful for (DNS over UDP) are stateless or near-stateless.
3. CDNs that need stateful anycast have built clever workarounds (connection sync, flow tables) to handle re-routing.

So: IPv4 anycast is a hack. A widely-deployed, battle-tested, "the internet runs on it" hack. But still a hack.

## IPv6 Anycast

IPv6, on the other hand, has anycast as a **first-class concept**. It is in the spec. RFC 4291 ("IP Version 6 Addressing Architecture") defines three address types:

- **Unicast** — one machine.
- **Multicast** — group of machines.
- **Anycast** — multiple machines, packet goes to "one of them," typically the nearest.

IPv6 dropped broadcast entirely. Where IPv4 used broadcast, IPv6 uses multicast.

Importantly, IPv6 anycast is **syntactically the same** as unicast. You can't tell from looking at an IPv6 address whether it's intended for unicast or anycast. The protocol doesn't reserve specific bits for "this is anycast." The only formal distinction is at the operations level: an address that's deliberately assigned to multiple hosts is "anycast"; an address assigned to one host is "unicast."

That said, IPv6 does reserve specific addresses as **predefined anycast addresses**. The most famous is the **Subnet-Router Anycast Address**.

### The Subnet-Router Anycast trick

In IPv6, every subnet has an anycast address that is "the subnet prefix followed by all zeros in the interface ID portion." So if your subnet is `2001:db8:1234::/64`, the subnet-router anycast address is `2001:db8:1234::`. (No `::1` — that's `::1` and would be a unicast loopback. Just plain `::` in the host part.)

Every router on that subnet **must** answer to that address. So if you send a packet to `2001:db8:1234::`, you reach **a** router on that subnet. Specifically, the router that the network can deliver to most easily. You don't have to know which router. You just send.

This is super handy for things like "send a packet to whichever default gateway is on this LAN." Without subnet-router anycast, you'd need to learn the specific gateway's IP. With it, you just send to the well-known anycast address and any router answers.

There are also **reserved anycast IDs** for specific roles, defined in RFC 2526. These were never used heavily in practice, but the spec allows for things like "Mobile IPv6 Home Agents" anycast addresses for finding home-agent routers automatically.

### IPv6 anycast in BGP

For Cloudflare-style global anycast on IPv6, you do exactly what you do on IPv4: announce the same `/48` (or `/56` or whatever your prefix length is) from multiple sites. BGP picks the shortest path. Same trick. Same outcome.

The ICANN root servers do exactly this on IPv6 today. So does `2606:4700:4700::1111` (Cloudflare's IPv6 DNS resolver — yes, it's anycast, just like the IPv4 sibling `1.1.1.1`).

### Picking IPv6 anycast addresses

There's a subtle wrinkle in IPv6. The default address selection rules (RFC 6724) prefer using a unicast address as a **source** address. When a host uses an anycast address as a source, surprising things can happen — the reply comes back to whichever node is "closest to the receiver," which might not be the original sender.

So IPv6 anycast is generally used as a **destination** address only, not a source. If you have a service announcing `2606:4700:4700::1111` from many servers, those servers reply using their **unicast** addresses, not the anycast address. The reply comes back through normal unicast routing. (Or, the service uses a clever trick to reply from `1.1.1.1` and just hopes the routing is stable enough that the reply makes it back. CDNs do this all the time.)

## Stateful vs Stateless Anycast

Anycast's biggest enemy is **state**. Servers that remember things about you across multiple packets do not love anycast.

### Stateless protocols love anycast

DNS over UDP is the perfect anycast workload. Why?

1. Each query is one UDP packet. No connection. No handshake. No "this query is part of session 17."
2. Each answer is one UDP packet. The server doesn't have to remember anything about the client between queries.
3. If a query gets routed to a different node next time, that's fine — every node has the same data and can answer any query.
4. If a packet is lost, the client retries. The retry might land on a different node. Doesn't matter. The retry will be answered correctly.

UDP DNS is the anycast poster child. The 13 DNS root servers — `a.root-servers.net` through `m.root-servers.net` — are each anycast. There are roughly 1,500 physical machines globally, all answering for those 13 logical addresses. Resolvers don't notice or care which physical machine they hit. The answer is the same either way.

NTP (network time) is similar — each NTP query is essentially a single packet exchange, and any time server can answer.

### Stateful protocols struggle with anycast

TCP is stateful. The server tracks the connection: sequence numbers, window sizes, congestion-control state, whether the connection's been established, what data's been sent and ACK'd, what data's been received but not yet read by the application. **All of that lives in memory on one specific server.**

If the network reroutes mid-flow — say, the BGP path changes because a peering link went down — the next packet might land on a **different** anycast node. That node has never heard of this connection. It looks at its connection table, sees nothing matching the 4-tuple `(client_ip, client_port, server_ip=1.1.1.1, server_port=443)`, and concludes "this packet must be from a stale connection." The standard TCP response to a packet on an unknown connection is **RST** — the new node sends back a reset. The client's TCP socket dies. The application sees an error.

TLS is stratospherically more stateful than TCP — there's a key exchange, session keys, record sequence numbers, the ticket cache, etc. Same problem applies, with extra fireworks.

QUIC is also stateful, but QUIC has a connection-ID field independent of the IP/port 4-tuple, so QUIC handles connection migration much more gracefully than TCP. We'll come back to this.

### The practical impact

In practice, on the modern internet:

1. **DNS over UDP**: trivially anycast. Always anycast. Has been for 15+ years. Works.
2. **HTTPS via TCP**: anycast works **as long as the routing is stable enough that flows don't switch nodes mid-connection**. In a stable network, packets from a given client all go to the same node throughout the connection.
3. **HTTPS via QUIC**: handles re-routing better. Less of a problem.
4. **Long-lived TCP** (e.g., 12-hour SSH sessions, long video streams over TCP): risky. If routing changes, the connection dies.

CDNs solved the long-lived TCP problem by introducing **connection synchronization** between anycast nodes. We'll cover that when we get to "Solutions."

### A long list of "stateful enough to care" examples

Just to drive home which protocols struggle:

- **HTTP/1.1 keep-alive** — keeps a TCP connection open across multiple requests. If routing changes mid-keep-alive, the next request bombs out.
- **HTTP/2** — multiplexes many streams over one TCP connection. One TCP stall affects every stream. Anycast re-routing kills all streams simultaneously.
- **WebSocket** — long-lived TCP. Often runs for hours. Re-routing **definitely** breaks it.
- **MQTT over TCP** — IoT pub/sub. Long-lived. Same risk.
- **gRPC** — usually over HTTP/2. Long-lived streaming RPCs. Same risk as HTTP/2.
- **Database connections** (PostgreSQL, MySQL, MongoDB clients) — long-lived TCP. Bad fit for anycast.
- **VPN tunnels** (IKE/IPsec, WireGuard) — IPsec is hilarious because it has its **own** sequence numbers; if you re-route mid-tunnel, both sides desynchronize and the tunnel drops. WireGuard handles re-keying gracefully but still won't love it.
- **SSH sessions** — long-lived TCP, holds shell state. Re-routing kills the session.
- **TCP-based RTSP for video streaming** — long-lived, large state. Bad fit.
- **Long-poll HTTP** — pretends to be short, actually long-lived. Risky.
- **SMTP STARTTLS** — short connections usually, but TLS adds state. Reasonably safe.

And the protocols that **don't** care about anycast:

- **DNS over UDP** — single-packet exchanges. Perfect.
- **NTP over UDP** — single-packet exchanges. Perfect.
- **stateless HTTP/1.0** — open, request, close, every time. No keep-alive, no problem.
- **simple TCP RPCs that complete in <1 second** — short enough that mid-flow re-routing is statistically rare.
- **QUIC with connection migration support** — built to handle this case.

The lesson: anycast favors short, stateless exchanges. The longer-lived and more stateful your protocol, the more anycast hurts.

## DNS Anycast

The grand poster child for anycast is the DNS root server system.

### The "13 root servers" myth

You may have heard that the internet has 13 DNS root servers. That number is famous. Wrong, but famous.

There are **13 logical** root server identities, named `a.root-servers.net` through `m.root-servers.net`. Each one has a single IPv4 address (and a single IPv6 address). For example:

- `a.root-servers.net`: `198.41.0.4` and `2001:503:ba3e::2:30`
- `f.root-servers.net`: `192.5.5.241` and `2001:500:2f::f`
- `m.root-servers.net`: `202.12.27.33` and `2001:dc3::35`

Why only 13? Because in 1999, when the system was set up, you could only fit responses for 13 root server addresses in a single DNS UDP packet (capped at 512 bytes by the original DNS spec). EDNS0 has lifted that 512-byte cap, but the "13 root servers" number stuck for political and operational reasons.

The fascinating part: each of those 13 IPs is **anycast**. There are around **1,500 physical machines** in the world, distributed across over **160 countries**, all answering for those 13 logical addresses.

`a.root-servers.net` alone is operated by Verisign and is anycast across dozens of physical sites. When you query `a.root-servers.net` from Tokyo, you hit a physical server in Tokyo (or as close as Verisign has a deployment). When you query it from São Paulo, you hit a physical server in São Paulo or Buenos Aires. Same IP. Different machine.

### Why anycast for DNS roots?

Three reasons.

**Latency.** DNS lookups are on the critical path for everything else. A slow root lookup means a slow page load. Putting a root server in every region cuts root-lookup latency from "across the ocean" to "across the city."

**Resilience.** Imagine a DDoS attack against `a.root-servers.net`. If `a` were a single physical box, the whole world's `a` queries would be down. Because `a` is anycast across 50+ sites, an attacker can DDoS one site's `a` and the BGP routes from that site simply withdraw. Other sites carry the load. The "a" service stays up.

**Geographic resilience.** A natural disaster takes out a region? That region's anycast nodes go silent. Their BGP announcements stop. The neighboring regions' nodes pick up the slack within a few minutes (the BGP convergence time).

### The .com TLD is also anycast

Verisign also operates `.com` and `.net`. Those NS servers are anycast across many sites globally. When your resolver looks up `google.com`, it queries an `.com` NS server, and that query is answered by a Verisign anycast node nearest to your resolver.

Same for almost every public TLD: anycast.

### How resolvers benefit

When you run `dig @1.1.1.1 google.com`, your packet goes to whichever Cloudflare resolver is closest to you. That resolver answers from local cache (fast) or recursively asks the root, then `.com`, then `google.com`'s authoritative NS — and **all three of those queries** also benefit from anycast, because the root is anycast and `.com` is anycast and Google's NS is anycast. The whole chain is anycast.

This is why DNS feels instant. Every link in the chain is geographically near you.

### A picture of the root server fleet

```
       The "13" DNS root servers (logical)
       Each = one IP, hundreds of physical machines

       a.root-servers.net  198.41.0.4   (Verisign)
       ┌─────────────────────────────────────────┐
       │ Tokyo  Sydney  Frankfurt  Ashburn  ...  │ ← anycast nodes
       └─────────────────────────────────────────┘

       b.root-servers.net  170.247.170.2 (USC ISI)
       ┌─────────────────────────────────────────┐
       │ Los Angeles  Marina del Rey  ...        │
       └─────────────────────────────────────────┘

       ... (k, l, m similarly distributed worldwide)

       Each "letter" is operated by a different organization.
       Each operator runs many physical sites for their letter.
       All physical sites for one letter share that letter's IP.
       This is anycast at planetary scale.
```

The whole point of having 13 letters operated by 13 different organizations is **diversity of failure**. If Verisign has a software bug that takes down `a`, you can still reach `b` through `m`. Each operator runs its own infrastructure, with its own engineering team, on its own software stack. They all serve the same root zone data, but their implementations are independent. So the system is resilient against both DDoS attacks (anycast spread) and software bugs (operator diversity).

This is one of the most successful pieces of internet infrastructure ever designed. It has run essentially uninterrupted for 25+ years. Anycast is a foundational reason it works at all.

## HTTP/HTTPS Anycast

CDNs are the second-biggest anycast users. Cloudflare, Fastly, Akamai, AWS CloudFront, Google Cloud CDN, Bunny, KeyCDN — every modern CDN uses anycast at scale.

The basic CDN setup:

1. CDN runs **edge POPs** (Points of Presence) in many cities. Maybe 50, maybe 300.
2. Every edge POP answers to the same handful of "anycast IPs" assigned to that customer.
3. When a user hits the customer's anycast IP, BGP routes the user to the nearest POP.
4. The POP terminates TLS, looks at the request, serves cached content if it has it, or fetches from the customer's origin if it doesn't.

### The connection-time decision

Critically, the choice of "which POP" happens **at the very first packet** of the connection. Once the TCP handshake is complete and the TLS session is up, the POP and the client are bound together. As long as the routing stays stable, every subsequent packet of this connection goes to the same POP.

This is why CDNs care so much about **routing stability**. If routes change a lot, connections die. If routes are stable, connections are happy. CDNs invest heavily in monitoring routing stability and in BGP-level traffic engineering to keep things calm.

### TLS certificates

A user hits Cloudflare's anycast IP from Tokyo and lands on Cloudflare-Tokyo. The user expects a TLS cert for `customerwebsite.com`. Cloudflare-Tokyo serves that cert.

The same user from London lands on Cloudflare-London. London serves the **same** cert.

How? Cloudflare-Tokyo and Cloudflare-London both have the same private key for `customerwebsite.com`. They have to. Otherwise the cert wouldn't validate on different POPs.

Distributing private keys to hundreds of POPs is, uh, a security challenge. Cloudflare invented "Keyless SSL" to solve it — keys never leave the customer's origin; POPs just ask the origin to do the signing. Other CDNs use HSMs or careful key distribution. The point is: the **same** cert (and matching private key) has to be available from **every** anycast node, or anycast TLS doesn't work.

### Backend split

A CDN's anycast layer is the **frontend**. The backend (origin servers) is usually unicast. The CDN POP terminates the user's anycast connection, then opens a separate **unicast** connection back to the origin to fetch the data.

This is the classic CDN architecture: **anycast at the edge, unicast at the core.** Most modern CDNs look exactly like this.

### A picture of the CDN edge

```
    User in Tokyo
       │
       │ HTTP GET https://customerwebsite.com
       │ DNS: customerwebsite.com → 104.16.0.1 (anycast)
       ▼
    ┌──────────────────────┐
    │ Cloudflare-Tokyo POP │
    │   - Terminates TLS   │
    │   - Checks cache     │
    │   - Cache hit → reply│
    └──────┬───────────────┘
           │ (cache miss)
           ▼ unicast back to origin
    ┌──────────────────────┐
    │ Origin server        │
    │   (one IP, one box)  │
    │   203.0.113.42       │
    └──────────────────────┘

           Frontend = anycast (closest POP for each user)
           Backend  = unicast (one origin, no anycast magic)
```

This split is the standard CDN pattern. Anycast handles the "fast frontend for users globally" part. Unicast handles the "single source of truth at the origin" part. The CDN's job is to bridge the two, caching aggressively at the edge so that most user requests never reach the origin.

### Tier-1 vs. Tier-2 vs. Tier-3 POPs

Big CDNs categorize their POPs:

- **Tier-1 POPs** — biggest, most-redundant, present at major IXPs. They handle the heavy lifting and have rich BGP peering. These are usually announced **most aggressively** so that the most traffic flows there.
- **Tier-2 POPs** — smaller, in mid-sized cities. Provide good latency to nearby users without the cost of a Tier-1 site.
- **Tier-3 POPs** — small, sometimes inside ISPs (called "**embedded caches**"). Provide great latency but limited capacity. Usually only carry a fraction of the customer base's traffic.

All three tiers may announce the same anycast prefixes, but with different BGP attributes (prepends, MEDs, communities) to influence which tier each user lands on. A small POP might prepend its AS twice, making it look "farther," so users only land there if there's no better alternative.

## Anycast for Failover

One of anycast's superpowers is "graceful site failure."

### How a site failure plays out

Suppose Cloudflare-Tokyo is humming along, serving `1.1.1.1` to all the users in Japan. Suddenly there's a fiber cut, or a power outage, or a software bug, and Cloudflare-Tokyo's BGP speaker stops announcing `1.1.1.0/24`.

Within seconds, the routers that were peering with Cloudflare-Tokyo notice the announcement is gone. They withdraw the route from their tables. They send BGP UPDATE messages to **their** peers saying "I no longer have a path to `1.1.1.0/24` through Cloudflare-Tokyo." Those peers update too. The withdrawal propagates outward.

Each router in the world that had been routing Tokyo-area traffic via Cloudflare-Tokyo now sees a withdrawn route. It re-runs its best-path algorithm. The next-best path is **Cloudflare-Singapore** (or Hong Kong, or Sydney, depending on geography). The router switches.

User packets to `1.1.1.1` from Tokyo now flow to Cloudflare-Singapore. They take a few extra milliseconds to get there (it's a longer path), but they arrive. The service stays up. Users notice slightly slower DNS, but nothing breaks.

### Convergence time

How fast does this happen? Typically **15 seconds to 3 minutes**, depending on:

1. How fast the original POP withdraws (BGP keepalive timers, hold timers, BFD if used).
2. How fast the withdrawal propagates through the internet (each AS-hop adds a small delay).
3. How aggressively each AS along the path is configured to react.

Modern CDNs use **BFD** (Bidirectional Forwarding Detection) to detect link death in **sub-second** time and trigger BGP withdrawal almost instantly. So the typical "site dies" scenario for a well-configured CDN looks like a 5–30 second blip in the affected region.

### The flip side: graceful re-introduction

When Tokyo comes back online, it starts re-announcing `1.1.1.0/24`. Routers learn about the new path. The Tokyo path is shorter (1 AS-hop) than the Singapore path (2+ AS-hops), so they switch back. Tokyo regains its traffic.

This re-introduction also takes seconds to minutes. During the transition, packets might briefly bounce between Singapore and Tokyo, which can break long-lived TCP sessions. CDNs sometimes use **graceful re-introduction** — adding a freshly-restored POP with a slightly worse BGP attribute (longer prepend, lower MED, lower local-pref) for a few minutes to give existing connections time to drain elsewhere before fully shifting traffic.

### A picture of withdraw-and-converge

```
   T=0s  Tokyo POP active
   ┌───────────────────────────────────────┐
   │ Tokyo POP announcing 1.1.1.0/24       │
   │ Singapore POP announcing 1.1.1.0/24   │
   │                                       │
   │ Japanese users → Tokyo POP            │
   │ Singapore users → Singapore POP       │
   └───────────────────────────────────────┘

   T=10s  Tokyo POP withdraws (maintenance)
   ┌───────────────────────────────────────┐
   │ Tokyo POP NOT announcing              │
   │ Singapore POP announcing 1.1.1.0/24   │
   │                                       │
   │ Japanese routers learn withdrawal     │
   │ Withdrawal propagates                 │
   │ Each router re-runs best-path        │
   └───────────────────────────────────────┘

   T=30s  Convergence reached
   ┌───────────────────────────────────────┐
   │ Tokyo POP NOT announcing              │
   │ Singapore POP announcing 1.1.1.0/24   │
   │                                       │
   │ Japanese users → Singapore POP       │
   │   (slightly higher latency, but up)  │
   │ Singapore users → Singapore POP       │
   │   (unchanged)                         │
   └───────────────────────────────────────┘

   T=300s  Tokyo POP back online (maintenance done)
   ┌───────────────────────────────────────┐
   │ Tokyo POP re-announcing 1.1.1.0/24    │
   │ Singapore POP announcing 1.1.1.0/24   │
   │                                       │
   │ Japanese users → Tokyo POP again     │
   │ Singapore users → Singapore POP       │
   └───────────────────────────────────────┘
```

The whole transition is invisible to users — well, mostly. UDP DNS users see no impact. TCP users **may** see brief connection drops if the routing changed mid-flow. WebSocket users **definitely** see disconnects (and reconnect, because well-written clients reconnect automatically).

## Anycast for Load Distribution

Anycast distributes load **automatically** based on geographic proximity, with no DNS tricks. Each user goes to their nearest POP, so load is naturally spread.

### The shape of the distribution

If a CDN has 300 POPs and serves users globally, the load per POP is roughly proportional to how much internet traffic originates from each POP's coverage area. POPs near big internet hubs (Frankfurt, Ashburn, San Jose, Singapore, Tokyo) carry a lot of traffic. POPs in less-densely-populated regions carry less.

A CDN can shape this distribution by:

1. Building POPs only where there's enough load to justify them.
2. **AS-path prepending** to make a POP look "farther" in BGP and discourage routing — this can move some traffic to a less-loaded neighbor POP.
3. **Selective announcement** — withdrawing from one peer to push traffic to another.
4. **Communities** — using BGP communities to ask upstream ASes to attach attributes that change traffic flow.

### The "geo-DNS without DNS" framing

Before anycast became widespread, "send users to their nearest server" was done with DNS-based geo-IP. DNS servers would inspect the resolver's IP, guess where the user was (using a geo-IP database), and hand back a different A record for different geographies.

This works but has problems:

1. The DNS server only sees the **resolver's** IP, not the user's. A user in Seattle who uses a Google DNS resolver looks like they're in Mountain View.
2. DNS records have TTLs. If you change which IP a name resolves to, clients may keep using the old IP for the TTL period.
3. A DNS-based geo system requires keeping a fresh geo-IP database, which is annoying.

Anycast sidesteps all of that. Anycast looks at the **user's** packet directly, in real time, and routes it to the nearest copy. No geo-IP database needed. No DNS changes needed. Just BGP.

This is why "anycast = geo-DNS without DNS" is a useful slogan. Anycast does what geo-DNS was trying to do, but at a lower layer with fewer moving parts.

### Approximate latency map

A rough mental picture of how anycast distributes users by latency:

```
   User location            Closest Cloudflare POP    Approx RTT
   ─────────────────────────────────────────────────────────────
   Tokyo                    Tokyo                       2-5 ms
   Berlin                   Frankfurt                   5-10 ms
   New York City            New York / Newark           1-3 ms
   Mumbai                   Mumbai                      5-15 ms
   São Paulo                São Paulo                   3-8 ms
   Sydney                   Sydney                      2-6 ms
   Cape Town                Johannesburg                10-20 ms
   Reykjavík                Stockholm or London         30-50 ms
   Anchorage                Seattle                     30-60 ms
   Antarctica research stn  Sydney (via satellite!)     500+ ms
```

The pattern: most users are within a few milliseconds of the nearest POP. Underserved regions (small countries, high latitudes) get higher latency because there's no local POP. Anycast can't fix physics — if there's no POP within 5,000 km, you'll see proportional latency.

## The Connection Pinning Problem

We've alluded to this several times. Let's spell it out.

### What goes wrong

A user in Tokyo opens a TCP+TLS connection to `1.1.1.1:443`. The TCP SYN packet's BGP path leads to Cloudflare-Tokyo. The handshake completes. The connection is established between user and Cloudflare-Tokyo.

10 minutes later, while the connection is still open, a peering link between two ISPs in Asia goes down. The Tokyo user's BGP path to `1.1.1.1` shifts. New packets now flow to Cloudflare-Singapore.

The next packet the user sends — a TLS application_data record — arrives at Cloudflare-Singapore. Singapore looks up the connection in its table. Nothing. Singapore has never seen this 4-tuple. To Singapore, this packet looks like a stray packet from a long-dead connection.

Singapore's TCP stack sends back a **RST** (reset). The user's TCP socket gets an `ECONNRESET` error. The application sees a broken connection. The user reloads the page.

This is **connection pinning failure**. Mid-flow rerouting kills the connection.

### How often does it happen?

In a stable BGP environment: rarely. Maybe once a day for a busy CDN, across millions of connections. Most users never see it.

In an unstable BGP environment: depressingly often. Networks with flaky peering, congested links, or under-provisioned BGP timers can re-route flows constantly. Stateful protocols hate that.

### Why it's not a death sentence

Two reasons.

1. Most internet routes are stable. BGP doesn't change paths every few seconds. It changes them rarely. Most TCP connections live and die without ever seeing a path change.
2. When a connection does break, the application usually retries. The retry establishes a new connection (probably going to Singapore now). Things keep working with one painful 1-second hiccup.

Browsers and HTTP/2/3 are particularly resilient because they have built-in retry logic. SSH and other persistent connections are more fragile.

### A picture of mid-flight TCP rerouting failure

```
   T=0s    User opens TCP connection to 1.1.1.1:443
   ┌────────────────────────────────────────────┐
   │ Client (Tokyo) ──── SYN ───→ POP-Tokyo     │
   │ Client          ←── SYN-ACK── POP-Tokyo    │
   │ Client ──── ACK + TLS hello ──→ POP-Tokyo  │
   │   Connection established with POP-Tokyo    │
   └────────────────────────────────────────────┘

   T=2m    Routing changes (peering link issue)
   ┌────────────────────────────────────────────┐
   │ Client (Tokyo) ──── ACK ───→ POP-Singapore │
   │   Singapore: "I have no entry for this     │
   │   4-tuple. This must be a stale connection.│
   │   Sending RST."                            │
   │ Client          ←── RST ────  POP-Singapore│
   │   Client TCP socket: broken!               │
   │   Application: ECONNRESET                  │
   └────────────────────────────────────────────┘
```

This is the worst-case anycast failure mode. The fix is connection sync between POPs (next section), or just accepting that some long-lived connections will die during routing changes and relying on the application to reconnect.

## Solutions: Connection Sync via L7 Mesh

To make TCP/TLS over anycast more robust against re-routing, modern CDNs use **connection synchronization** between POPs.

### The basic idea

Every CDN POP keeps a table of active TCP connections it's serving. POPs share these tables — over a private mesh network — so that other POPs know which connections exist, who owns them, and how to forward packets along.

When a packet arrives at the "wrong" POP because routing changed mid-flow, the wrong POP looks up the 4-tuple in its synced table, sees that this connection is owned by another POP, and **forwards** the packet to that POP via the mesh. The owning POP responds. The response comes back to the wrong POP, which forwards it to the user.

This adds latency (because the wrong-POP-to-right-POP hop is extra) but **preserves the connection**. The user never sees a RST.

### Cloudflare's Argo

Cloudflare's "Argo Smart Routing" includes a global private mesh between POPs. When a connection lands on POP-A but actually "belongs" to POP-B, Argo carries the packet over the mesh from A to B, lets B respond, and carries the response back. The user is none the wiser.

Argo also does smart routing for the **forward** direction (POP to origin) and chooses the best path through the global mesh, avoiding congested links. That's a separate trick that also benefits from the same mesh.

### Other implementations

**Fastly** has a similar internal connection-sync system.

**Akamai** uses Akamai's mapping system to keep flows pinned, plus internal forwarding when re-routing happens.

**AWS Global Accelerator** does anycast in front of regional ALBs/NLBs and uses AWS's backbone to forward connections to the right region.

### The design tradeoff

Connection sync makes anycast TCP rock-solid — but it requires a global private mesh, complex software, and a serious operations team. Small companies running anycast don't usually do this. They just accept that occasional re-routing kills some connections and rely on application-level retries.

### Design tradeoffs in connection sync

When you build a connection-sync mesh, you make several big tradeoffs:

**1. Latency overhead.** Forwarding misrouted packets adds a hop. If the wrong POP is geographically far from the right POP, the user's packets take a long path. A user in Tokyo who lands on Cloudflare-Tokyo (correct) sees 2 ms latency. The same user, if their packets get mis-routed to Singapore which then forwards to Tokyo, sees 100+ ms. Sync only saves the connection from RST; it doesn't preserve the latency advantage.

**2. Mesh bandwidth cost.** All those forwarded packets between POPs cost real money. The CDN pays for bandwidth between its sites. If 5% of packets get misrouted, that's 5% of total traffic going across the private mesh. For a CDN moving petabytes a day, that's a non-trivial bill.

**3. Sync staleness.** The mesh has to share connection state quickly, but state changes constantly (new connections, closed connections, sequence numbers advancing). Perfect global synchronization is impossible. The mesh has to tolerate some staleness — a POP might forward a packet to another POP that has just closed the connection. Then **that** POP sends a RST. Edge cases multiply.

**4. Security boundary.** The mesh carries connection metadata (who's connected to whom, where their packets are flowing). That metadata is sensitive. The mesh has to be encrypted, authenticated, and protected against malicious nodes. If an attacker compromises one POP, they shouldn't get the keys to the entire mesh.

**5. Operational complexity.** Adding the mesh means another moving part. Failures in the mesh can cause global problems. CDNs have learned (sometimes the hard way) to make the mesh fail closed — if the mesh fails, the POPs fall back to "each POP for itself" rather than locking up.

These tradeoffs are why connection sync is a serious engineering effort. Small companies running anycast usually don't bother — they accept the occasional connection drop and rely on application-level retries. Cloudflare-scale operators **must** do this work because their users won't tolerate frequent disconnects on long video streams or WebSocket apps.

## Anycast for Mitigation

DDoS attacks try to drown servers in junk traffic. Anycast helps by **diluting** the attack across geography.

### Why anycast naturally absorbs DDoS

Imagine an attacker has a botnet of 1 million infected machines, all sending junk traffic to `1.1.1.1`. With unicast, all 1 million machines would send their packets to the **same one server**, which would melt instantly.

With anycast, the botnet's 1 million machines are spread across the world. Each machine's packets go to **its closest anycast POP**, not to one central server. So the 1 million packets get spread across maybe 300 POPs, with each POP receiving some fraction. Each POP only has to absorb its local fraction.

If POPs are well-provisioned, each POP can handle its share. The attack is **diluted** geographically.

### The "soak it up" model

Cloudflare publicly absorbs single-digit-Tbps attacks regularly. They do it by spreading the attack across hundreds of POPs and dropping the junk locally at each POP. No single POP is overwhelmed because each POP only sees its share.

This is one reason why "DDoS protection" services almost always run on anycast networks. The geography of the deployment **is** the defense.

### Selective scrubbing

A CDN can also concentrate clean traffic by **withdrawing** the route from POPs that are overwhelmed. The bad traffic continues going to the still-announced POPs, where dedicated scrubbers (specialized DDoS-mitigation hardware/software) drop the junk. The clean POPs serve real users without getting dragged into the fight.

### Real-world DDoS soak: Cloudflare's 2.5 Tbps story

In 2017 Cloudflare publicly absorbed an attack of around 1.7 terabits per second. By 2020 they'd absorbed 2+ Tbps. By 2023 they'd absorbed peaks above 70 million requests per second. They did this **without** dedicated bandwidth-purchase agreements scaling each year — they did it by spreading the attack across hundreds of POPs that already existed for normal traffic.

The math: if you have 300 POPs and an attack is 2 Tbps, each POP averages ~6.7 Gbps of attack traffic. Most modern POPs have multi-tens-of-Gbps capacity. So no single POP is overwhelmed. The attack is just background noise that the POPs filter out alongside normal serving.

This is why DDoS providers genuinely **want** to be on lots of geographies. More geography = more capacity dilution = more attacks they can absorb without breaking a sweat. Anycast is the foundation that makes the math work.

### When anycast scrubbing isn't enough

If an attack concentrates from one specific region, the POPs in that region get hammered while others sit idle. Anycast doesn't load-balance the attack against POPs that aren't local to the attackers; it sends each attacker to their nearest POP. So a botnet of 1 million machines all in Brazil hits the São Paulo POP hard while Frankfurt sits underused.

For these regional concentrations, CDNs use:

1. **Selective withdrawal** — temporarily withdraw the prefix from the attacked POPs and let routing send users (and attackers) elsewhere.
2. **BGP FlowSpec** — instruct upstream peers to drop attack traffic before it even reaches the POP.
3. **Scrubbing centers** — dedicated DDoS-mitigation hardware in nearby sites that pre-filter junk.

But these are last-resort moves. Most attacks are absorbed silently by anycast spread alone.

## Anycast for Multipath

Modern transport protocols handle anycast better than old ones. QUIC is the poster child.

### TCP's problem with anycast

TCP identifies a connection by the 4-tuple `(src_ip, src_port, dst_ip, dst_port)`. If any of those four change mid-connection — including the destination IP, which doesn't really change in anycast but the **destination machine** can change — TCP says "this is a new connection" and breaks the old one.

TCP also doesn't have a great way to migrate between paths. There's no "hey, my path is changing, reuse this connection" mechanism in vanilla TCP.

### QUIC's connection ID

QUIC uses **connection IDs** (CIDs) instead of (or in addition to) the 4-tuple to identify connections. The CID is in the QUIC header, not the IP/UDP layer. Two packets with different IPs/ports but the **same CID** are the same connection.

This means QUIC can survive **migration**. If a user's packet ends up at a different anycast node, that node can look at the CID, find the connection, and (if the new node has the connection state, via sync) keep going. Or the new node can forward to the original node.

### MASQUE and connection-ID-based routing

Some advanced CDN setups route packets based on the QUIC CID rather than (or in addition to) the IP 4-tuple. The QUIC CID can encode a hint about which POP "owns" this connection, so even if anycast routes the packet to a "wrong" POP, that POP can quickly figure out the owner and forward correctly.

This is a Cloudflare/Google-scale optimization, not something most networks need. But it's why QUIC-based services tolerate anycast re-routing better than TCP-based services.

### MPQUIC

Multipath QUIC (a research/experimental extension) lets a single connection use **multiple paths** simultaneously. In an anycast world, this could mean a single QUIC connection to two different anycast nodes at once. We're not deploying this at scale yet, but it's coming.

### Why QUIC migration matters for anycast

Picture a mobile user. They're walking from their house to the train. Their phone is connected to home wifi (one IP). They walk out, their phone drops wifi and switches to LTE (different IP). They arrive at the train station, their phone connects to the station's wifi (another IP). All in 10 minutes.

If they're streaming a video over TCP, every network change kills their TCP connection. The video stutters. The app has to reconnect. They have to re-buffer. Annoying.

If they're streaming over QUIC, the connection migrates. The QUIC connection ID stays the same, even as the underlying IPs change. The video keeps playing seamlessly. The app never has to reconnect.

Anycast benefits from this because anycast routing changes effectively look like the same kind of "your IP-port mapping shifted." A connection-migration-capable transport can survive both kinds of changes (mobile network handoff, anycast re-routing) gracefully.

This is one of several reasons HTTP/3 (which uses QUIC) is taking over from HTTP/2 (which uses TCP) for high-traffic services. Anycast plays better with HTTP/3.

## BGP Anycast Setup

Let's get concrete. How do you actually set up anycast?

### The basic pattern

You need:

1. An IP prefix that you control (a `/24` for IPv4 or a `/48` for IPv6 are the typical minimums for global routing).
2. At least two physically-separated sites.
3. At each site, a router speaking BGP to upstream ISPs.
4. At each site, servers configured to receive traffic destined for that prefix.

### Same-AS or different-AS?

You have two choices.

**Single AS, multiple POPs.** Your whole organization has one ASN. Each POP runs a router that's part of that ASN. Each POP announces the same prefix, with the same AS-path (just `[your_asn]`). The internet sees "this prefix is reachable through AS X" from multiple geographic angles. BGP best-path picks the closest external announcement.

**Multiple ASes.** Some big organizations operate distinct ASes for distinct regions. Each AS announces the same prefix from its region. The AS-paths look like `[AS_eu]` from Europe and `[AS_us]` from North America. Same outcome — BGP picks the shortest path — but with regional autonomy.

For most setups, single-AS is simpler.

### A toy single-AS setup

```
                    ┌──────────────────────┐
                    │   AS 65000 (you)     │
                    │   ──────────────     │
                    │   POP-NYC (router)   │
                    │   POP-LON (router)   │
                    │   POP-TYO (router)   │
                    └──────────────────────┘

   POP-NYC announces 1.1.1.0/24 to its NYC upstream peer (AS 174 e.g.)
   POP-LON announces 1.1.1.0/24 to its LON upstream peer (AS 6939 e.g.)
   POP-TYO announces 1.1.1.0/24 to its TYO upstream peer (AS 2914 e.g.)

   Internet sees:
     1.1.1.0/24 via AS 174 → AS 65000  (path length 2)
     1.1.1.0/24 via AS 6939 → AS 65000 (path length 2)
     1.1.1.0/24 via AS 2914 → AS 65000 (path length 2)

   Each upstream router picks its local POP because the AS-path
   length is the same and the local peering is preferred.
```

### The minimum IP prefix

For IPv4: a `/24` is usually the smallest globally-routable prefix. Anything more specific (e.g., `/25`, `/26`) is filtered by most large ISPs. So your anycast prefix should be at least a `/24`.

For IPv6: a `/48` is the de-facto minimum. Some networks accept down to `/56` but `/48` is safer.

### Origin AS and RPKI

You should publish a **Route Origin Authorization (ROA)** in RPKI saying "AS 65000 is authorized to originate `1.1.1.0/24`." Without an ROA, RPKI-validating networks may drop your announcements as RPKI-invalid.

For anycast specifically, you should authorize **all** ASes that originate the prefix. If you're using single-AS, that's just your AS. If you're using multi-AS, list all of them in the ROA.

### Withdrawal

To remove an anycast site (e.g., for maintenance), the site stops announcing the prefix. BGP withdrawal propagates. Traffic shifts to other sites. After a brief convergence period, no traffic flows to the maintenance site.

### A complete anycast setup checklist

If you wanted to deploy anycast tomorrow, here's a rough checklist:

1. Get an ASN from your RIR (or use a private ASN if you're testing internally).
2. Get an IP prefix (a `/24` for v4, a `/48` for v6 minimum). RIRs sell these or your provider can sub-allocate.
3. Publish an RPKI ROA authorizing your AS to originate the prefix.
4. Configure your BGP routers at each POP to announce the prefix.
5. Configure servers at each POP to bind the anycast IP on a loopback interface.
6. Configure servers' applications to listen on the anycast IP.
7. Test reachability from multiple geographies (use looking glasses or globalping.io).
8. Test failure scenarios — withdraw from one POP, confirm traffic shifts.
9. Monitor BGP sessions, prefix announcements, and per-POP traffic.
10. Document your traffic-engineering policies (which POP serves which region, in normal operation).

Mistake to avoid: announcing the prefix only as `/24` and forgetting that more-specific prefixes win. If your network has any `/25` covering the same range, traffic will follow the `/25` and bypass anycast entirely.

Mistake to avoid: forgetting to bind the anycast IP on the loopback interface, so your kernel doesn't know it's local. The packet arrives, the kernel says "this isn't for me," and drops it. Always bind on `lo`.

Mistake to avoid: configuring each POP's BGP router differently, then wondering why traffic distribution is weird. Use config management (Ansible, NixOS, configuration-as-code) to keep POP configs consistent.

## iBGP vs eBGP for Anycast

Big organizations announce anycast prefixes from many internal points. There's a question: do those points talk **iBGP** to each other, or do they all talk **eBGP** to the upstream ISPs independently?

### eBGP: each POP independent

In the simplest model, each POP runs eBGP to its upstream ISP. POPs don't share BGP state with each other directly. Each POP announces the prefix to its upstreams. The internet sees each announcement separately.

This works fine for a small number of POPs. It scales poorly because each POP has to be configured individually.

### iBGP: POPs share BGP state internally

Larger organizations run **iBGP** internally — every POP's BGP router has an iBGP session with every other POP's BGP router (or to a route reflector). They share information about which prefixes they're announcing externally. This lets the organization implement **internal traffic engineering** decisions (which POP serves which user, based on internal preferences).

For anycast specifically, iBGP lets you use the **MED** (Multi-Exit Discriminator) attribute to influence inbound traffic. You can ask all your upstreams "please send traffic for prefix X preferentially via POP-LON if path lengths are equal." This is a fine-grained knob.

### Route reflectors

In a fully-meshed iBGP, every router peers with every other router. With N routers, that's `N*(N-1)/2` sessions. For 50 POPs, that's 1,225 sessions. Doesn't scale.

So organizations use **route reflectors** — specialized BGP speakers that take iBGP state from many "client" routers and re-distribute it. With route reflectors, each router only peers with the reflectors, not with every other router. Sessions go from quadratic to linear.

For anycast organizations, route reflectors are essential at scale.

### Same-prefix conflicts

iBGP carries the prefix from each POP. Each POP says "I have `1.1.1.0/24`." If those announcements meet at a route reflector, the reflector picks **one** as the iBGP-best path and propagates that to other internal routers. This means **internal traffic** to `1.1.1.1` goes to one specific POP — not the closest one.

This is usually fine, because anycast is meant for **external** traffic. Internal queries to your own anycast IPs from your own offices may all funnel through one POP. If you don't like that, you can configure more elaborate per-region BGP policies, or just don't anycast inside your network.

### Two-tier anycast: internal IGP plus external eBGP

Big anycast networks often split routing in two:

- **Internally** they use an IGP (OSPF, IS-IS, or EVPN) to share routes among their own routers. The anycast IP appears as an internal route from each POP.
- **Externally** they speak eBGP to upstream ISPs and peers, announcing the prefix outward.

Internal routing is fast (sub-second) and has fine-grained control. External routing is slower (seconds to minutes) and exposed to the outside internet.

A typical ISP/CDN architecture:

```
   POP-A ─── POP-B ─── POP-C ─── POP-D
     │         │         │         │
     │         │         │         │
   [IGP routes 1.1.1.0/24 internally as anycast]
     │         │         │         │
     │         │         │         │
   eBGP to    eBGP to    eBGP to   eBGP to
   upstream   upstream   upstream  upstream
   peer       peer       peer      peer
```

Each POP has IGP routes to every other POP's anycast IPs. So if a packet arrives at POP-A but the local instance can't serve it (some weird edge case), POP-A can forward via the internal mesh to another POP. This is the foundation for connection-sync mesh systems we discussed earlier.

## Anycast Withdrawal — the failure scenario

Withdrawal is the dual of announcement. It's how you take a POP out of service.

### Graceful withdrawal

The "graceful" sequence:

1. Operator decides to take POP-X offline (maintenance, software upgrade, hardware swap).
2. Operator instructs POP-X's BGP speaker to **withdraw** the anycast prefix from upstream peers. Internally, the BGP speaker stops sending UPDATE messages with the prefix in NLRI and instead sends withdrawals.
3. Upstream peers process the withdrawal. They remove POP-X from their best-path consideration for the prefix. They re-run best-path. They pick a different POP (or a different upstream entirely).
4. Upstream peers propagate the withdrawal to their own peers. The wave spreads outward.
5. Within minutes (often <1 minute for well-tuned BGP), the global internet has converged: traffic that was going to POP-X is now going elsewhere.
6. Operator confirms POP-X has zero traffic. Now safe to take down.
7. Operator does the maintenance.
8. Operator restarts services on POP-X.
9. Operator instructs POP-X's BGP speaker to **re-announce** the prefix.
10. Re-convergence happens. Traffic returns to POP-X.

### Forced withdrawal

If POP-X dies suddenly (power loss, network cut, kernel panic), its BGP sessions with upstreams **time out**. BGP's hold timer is typically 90–180 seconds, sometimes lower. Once the timer fires, upstreams declare the session dead and withdraw all prefixes that POP-X had announced.

This is slower than graceful withdrawal because of the hold timer. To speed it up, organizations use **BFD** (Bidirectional Forwarding Detection), which can detect link death in **sub-second** time. When BFD detects loss, it tears down the BGP session immediately, triggering withdrawals.

### Convergence math

Roughly:

- BGP keepalive timer: 30s default, 10s common in well-tuned networks.
- BGP hold timer: 90s default, 30s common.
- BFD interval: 50ms common, 1s default. Detection in 3x interval.
- Withdrawal propagation: a few seconds per AS hop.

A well-configured CDN with BFD on every peer detects failures in <1s and converges globally in 5–30s. A poorly-configured network might take minutes.

### A withdrawal storyboard

Let's walk through a single withdrawal in slow motion.

```
   T-1ms:   Operator sends "no network 1.1.1.0/24" to POP-X's BGP router.

   T+0ms:   POP-X's BGP daemon updates its local routing table.
            Marks 1.1.1.0/24 as "not announced."

   T+50ms:  POP-X's BGP daemon sends a BGP UPDATE message
            to each upstream peer:
              "WITHDRAW: 1.1.1.0/24"

   T+100ms: Each upstream peer's BGP daemon receives the WITHDRAW.
            Removes the route from its RIB-IN for this neighbor.
            Re-runs best-path. If POP-X was the best path, picks
            a new best path (maybe via another peer).

   T+200ms: Upstream peer's BGP daemon sends WITHDRAW (or alternate
            UPDATE) to its own peers downstream. Wave begins.

   T+1s:    Multiple AS hops have processed the withdrawal.
            The wave is spreading.

   T+30s:   Most of the internet has converged. Traffic for
            1.1.1.0/24 is no longer flowing toward POP-X.

   T+60s:   The slowest tier-2/3 networks finish converging.
            Operator confirms POP-X has zero new connections.

   T+5min:  Existing connections to POP-X have either completed,
            timed out, or been migrated. POP-X is fully drained.
```

If POP-X dies suddenly (no graceful withdrawal), the same wave happens — but starting from when the BGP session times out (90s default). That's why graceful shutdowns are 60s, but ungraceful failures are 2-3 minutes.

## Common Anycast Pitfalls

Things that bite anycast operators:

### 1. Mismatched server config

Each POP must serve **identical content** for stateless protocols, or **be able to look up state** for stateful protocols. If POP-A serves version 1 of `index.html` and POP-B serves version 2, users hit one or the other unpredictably and get inconsistent results.

This is especially annoying during deployment. If you push a new version to POP-A first, then POP-B 10 minutes later, users in different regions see different versions for 10 minutes.

Fix: deploy atomically across all POPs, or use feature flags so the rollout is decoupled from the deployment.

### 2. Time skew

If POP-A and POP-B disagree about what time it is, they might issue different timestamps in HTTP headers, different TLS not-before/not-after values, different signed cookies, etc. Anycast users see weird "this token is from the future" or "is this server lying about its clock" issues.

Fix: every POP runs strict NTP, sync'd to GPS or atomic time sources. Skew limited to a few milliseconds.

### 3. Certificate sharing

We covered this earlier: every POP needs the same TLS cert. If POP-A has a cert for `customerwebsite.com` valid for 90 days, and POP-B has the same cert but different expiry (because someone rotated the cert on B but not A), users get cert errors when they hit the un-rotated POP.

Fix: synchronize cert distribution. Use a centralized cert manager (like Cloudflare's API-driven cert system, or HashiCorp Vault + a sync agent). Treat cert rollouts as a global atomic operation.

### 4. The "cold cache" problem

A new POP comes online. Its cache is empty. Every request is a miss. The POP hammers the origin to fill its cache. The origin gets overloaded.

Fix: warm the cache before the POP starts taking traffic. Or use a small cache and let it heat up gradually under low load.

### 5. Traceroute lies

When you run `traceroute 1.1.1.1`, the route you see is **one path**, but the path the routing protocol uses for **production traffic** might be different (especially if routers do flow-based ECMP that hashes packets across multiple paths). You think you're tracing your packets, but you might be tracing a sibling path.

Tools like `paris-traceroute` and `mtr` are better than vanilla `traceroute` for anycast debugging.

### 6. `ping` doesn't tell you which node

A `ping` to `1.1.1.1` shows you the latency and that something is responding. It does **not** tell you which Cloudflare POP responded. To find that, you need to use Cloudflare's "trace" endpoint (`curl https://1.1.1.1/cdn-cgi/trace`) or inspect DNS responses for chaosnet records (`dig @1.1.1.1 chaos txt id.server.`).

### 7. Source address confusion

If a POP replies with a source address that doesn't match the destination address the client used (e.g., the POP replies from its unicast address instead of the anycast IP), the client's TCP stack will discard the reply because the 4-tuple doesn't match.

Fix: always reply from the anycast IP. Use `IP_TRANSPARENT` socket options (Linux) or BSD's `IP_BINDANY` to source-bind correctly.

### 8. Asymmetric routing

The forward path (client to server) might go through one set of ISPs. The reverse path (server to client) might go through a different set of ISPs. This is "asymmetric routing" and is **normal** on the internet. But it can break stateful middleboxes (firewalls) that expect symmetric flows.

Fix: middleboxes near anycast servers should be configured for asymmetric flows, or be stateless.

### Pitfall 9: monitoring blindness

Each POP can fail in different ways, and it's easy for one POP's failure to be invisible to monitoring that's hosted elsewhere. Imagine your monitoring is in Frankfurt and tests `1.1.1.1` from Frankfurt. Frankfurt sees Frankfurt-POP. Tokyo POP could be totally broken and Frankfurt monitoring would never notice.

Fix: monitor from multiple geographies. Run synthetic tests from probes scattered across the world. RIPE Atlas, Catchpoint, ThousandEyes, and homegrown probes-on-cloud-VMs are common solutions.

### Pitfall 10: dual-stack mismatch

You announce `1.1.1.0/24` for v4 and `2606:4700:4700::/48` for v6 anycast. But you only deploy v6 on some POPs. Now v4 users see one set of POPs and v6 users see another. Cert validation, content updates, and feature flags can be inconsistent across the v4/v6 split.

Fix: ensure every POP supports both v4 and v6, or consciously plan separate v4/v6 anycast tiers and verify they're synchronized.

### Pitfall 11: split-brain on origin fetches

Each POP fetches from the origin when it has a cache miss. If the origin is on the public internet (unicast), the POP-to-origin fetches take normal unicast paths. But if the origin is **also** anycast (some advanced setups do this), then POPs may end up fetching from different origin instances, which may have different content, leading to cache divergence.

Fix: keep the origin unicast. Anycast at the edge, unicast at the core.

### Pitfall 12: chained anycast across regions

If you stack multiple anycast services on top of each other (e.g., an anycast CDN fronting an anycast database), routing changes can cascade. A change in CDN-tier routing may cause CDN-tier connection drops, which retries land on a different CDN POP, which fetches from a different database POP, etc. Debugging the chain becomes hard.

Fix: limit anycast to one tier (the user-facing edge). Lower tiers should be unicast, with explicit, documented routing.

## Common Errors

Errors and warnings you'll see when working with anycast:

```
routing instability detected (anycast IP path flapping)
```
The same prefix is being announced and withdrawn rapidly. Usually caused by an unstable BGP session, a misconfigured route policy, or a script that's churning announcements. Stop the source of the churn.

```
BGP path-hunting on withdraw/announce
```
Routers are exploring multiple AS-paths in succession when a prefix is withdrawn. Each path is tried briefly before being rejected. This adds latency to convergence. Often caused by lack of BGP "soft reset" support or by aggressive path validation.

```
TCP RST after anycast hop
```
A TCP packet arrived at an anycast node that has no connection state for it. The node sent a RST. Usually means the routing changed mid-flow and the connection-sync system isn't (or wasn't) catching it.

```
TLS handshake timeout (different IP responded)
```
The client started a TLS handshake to one anycast node, then the routing changed, and the second handshake packet went to a different node. The new node has no handshake state, so it doesn't reply. The client times out.

```
DNS resolver loop on misconfigured anycast
```
A misconfigured anycast DNS server points to itself for upstream resolution, but anycast routes the upstream query right back to the same node. The query loops until TTL exhaustion.

```
certificate name mismatch
```
The TLS cert presented by the anycast node doesn't match the hostname in the request. Usually means one POP wasn't updated when the cert was rotated, or two POPs have different certs entirely.

```
UDP packet sequence reordering across anycast members
```
Different anycast nodes may answer different packets of a multi-packet UDP exchange. The client sees responses out of order. Most well-designed UDP protocols (like DNS) handle this, but custom UDP protocols may not.

```
IPv4 fragment racing
```
Fragments of one IP datagram go to different anycast nodes. Each node tries to reassemble but sees only some fragments. Reassembly fails. The application sees a lost packet. Common with large UDP DNS responses; mitigated by EDNS0 size advertising or DNS over TCP.

```
anycast-vs-unicast prefix overlap
```
Same IP advertised as anycast (`/24`) and a more-specific unicast (`/25` or `/26`). The more-specific wins routing because of longest-prefix match, defeating the anycast. Audit your announcements.

```
%BGP-3-NOTIFICATION: send to neighbor (anycast peer flapped)
```
Cisco-style log: BGP sent a notification to a neighbor, indicating a session-level event (often a hold-timer expiry). For anycast peers, this means the session bounced and the prefix may have flapped during the bounce.

```
routing-loop detected anycast
```
A router has two paths to the same anycast prefix that loop through each other. Usually caused by a misconfigured static route or a missing AS-path filter. Investigate the route-map and prefix-list configurations.

```
BGP UPDATE: AS-path malformed (prepend overflow)
```
You configured AS-path prepending and accidentally prepended too many copies of the AS, causing the path to be invalid. Reduce the prepend count.

```
glibc resolver round-robin tried all anycast IPs and failed
```
A DNS resolver was given multiple A records (treated as anycast siblings even if they aren't), tried each, and all failed. Suggests a deeper outage. Verify each IP individually.

```
TCP reset: stale connection (anycast peer changed)
```
The kernel marks the connection as dead because the peer's behavior shifted. Same as the RST-after-anycast-hop scenario, usually visible in `dmesg` on busy servers.

## Hands-On

Real commands you can run today. Most of these work on any Linux/Mac. Some need a router.

```bash
$ dig +short @1.1.1.1 google.com
142.250.190.46
```

A simple DNS query through Cloudflare's anycast resolver. Different POPs may give different answers because Google's `google.com` is also anycast and resolves differently per region.

```bash
$ dig +short @8.8.8.8 google.com
142.250.190.46
```

Same query through Google's public resolver. Compare answers; they're likely different per region.

```bash
$ ping -c 5 1.1.1.1
PING 1.1.1.1 (1.1.1.1): 56 data bytes
64 bytes from 1.1.1.1: icmp_seq=0 ttl=58 time=2.143 ms
64 bytes from 1.1.1.1: icmp_seq=1 ttl=58 time=2.301 ms
64 bytes from 1.1.1.1: icmp_seq=2 ttl=58 time=2.225 ms
64 bytes from 1.1.1.1: icmp_seq=3 ttl=58 time=2.198 ms
64 bytes from 1.1.1.1: icmp_seq=4 ttl=58 time=2.176 ms
```

A few-millisecond round trip means you're hitting a nearby Cloudflare POP. If you saw 200ms, you'd be hitting a far one.

```bash
$ traceroute -A 1.1.1.1
traceroute to 1.1.1.1 (1.1.1.1), 30 hops max, 60 byte packets
 1  192.168.1.1  [AS0]  0.812 ms
 2  isp-router.local  [AS22773]  3.221 ms
 3  edge01.cloudflare.com  [AS13335]  3.892 ms
 4  1.1.1.1  [AS13335]  3.945 ms
```

`-A` shows AS numbers. Cloudflare's AS is 13335. The fact that you reach AS13335 in only a few hops, and your latency is small, confirms you hit a local POP.

```bash
$ mtr -bz 1.1.1.1
HOST: laptop.local                 Loss%   Snt   Last   Avg  Best  Wrst StDev
  1. AS???    192.168.1.1          0.0%     5    0.7   0.7   0.6   0.9   0.1
  2. AS22773  isp-router.local     0.0%     5    3.1   3.2   3.0   3.5   0.2
  3. AS13335  edge01.cloudflare    0.0%     5    3.8   3.9   3.7   4.1   0.2
  4. AS13335  1.1.1.1              0.0%     5    3.9   3.9   3.7   4.0   0.1
```

`mtr` runs continuous traceroutes plus pings. `-b` shows IPs+names; `-z` shows AS numbers. Useful for spotting where latency or loss happens.

```bash
$ whois -h whois.cymru.com " -v 1.1.1.1"
AS      | IP               | BGP Prefix          | CC | Registry | Allocated  | AS Name
13335   | 1.1.1.1          | 1.1.1.0/24          | US | ARIN     | 2010-07-14 | CLOUDFLARENET, US
```

Team Cymru's whois service tells you the AS, the prefix, the country, and the AS name. Quick way to see who owns an anycast IP.

```bash
$ whois 1.1.1.1
NetRange:       1.1.1.0 - 1.1.1.255
CIDR:           1.1.1.0/24
NetName:        CLOUDFLARE-DNS
Organization:   Cloudflare, Inc. (CLOUD14)
```

Standard whois lookup. Doesn't tell you about anycast specifically, just who owns the prefix.

```bash
$ curl -v https://1.1.1.1/dns-query?name=google.com\&type=A
*   Trying 1.1.1.1:443...
* Connected to 1.1.1.1 (1.1.1.1) port 443
* TLSv1.3 (OUT), TLS handshake, Client hello (1):
* TLSv1.3 (IN), TLS handshake, Server hello (2):
< HTTP/2 200
< server: cloudflare
< content-type: application/dns-json
{"Status": 0, "Answer": [{"name": "google.com", "type": 1, "data": "142.250.190.46"}]}
```

DNS-over-HTTPS through Cloudflare's anycast. Fast and encrypted. Each request hits the nearest POP automatically.

```bash
$ host 1.1.1.1
1.1.1.1.in-addr.arpa domain name pointer one.one.one.one.
```

PTR lookup. Tells you the canonical name Cloudflare gave its anycast IP.

```bash
$ curl https://1.1.1.1/cdn-cgi/trace
fl=12abcd34
h=1.1.1.1
ip=203.0.113.42
ts=1762000000.123
visit_scheme=https
uag=curl/8.4.0
colo=SJC
sliver=none
http=http/2
loc=US
warp=off
gateway=off
rbi=off
kex=X25519
```

The magical `cdn-cgi/trace` endpoint reveals which Cloudflare POP you hit. `colo=SJC` means San Jose. Try this from different cities and you'll see different `colo` codes.

```bash
$ dig @1.1.1.1 chaos txt id.server.
;; ANSWER SECTION:
id.server.              0       CH      TXT     "SJC"
```

Same idea, but using DNS chaos-class queries. Cloudflare returns the POP's three-letter code in `id.server.`.

```bash
$ ip route get 1.1.1.1
1.1.1.1 via 192.168.1.1 dev wlan0 src 192.168.1.42 uid 1000
    cache
```

Linux command to see which interface and gateway your kernel will use to reach `1.1.1.1`. Useful for confirming routing.

```bash
$ netcat -uvz 1.1.1.1 53
Connection to 1.1.1.1 port 53 [udp/domain] succeeded!
```

Try a UDP connect to `1.1.1.1:53` (DNS). `nc -u -v -z` does a quick reachability check.

```bash
$ tcpdump -i any -n -e 'host 1.1.1.1'
listening on any, link-type LINUX_SLL2 (Linux cooked v2), capture size 262144 bytes
12:00:00.123456 In ethertype IPv4 (0x0800), length 75: 192.168.1.42.55502 > 1.1.1.1.53: 0+ A? google.com. (28)
12:00:00.124000 In ethertype IPv4 (0x0800), length 91: 1.1.1.1.53 > 192.168.1.42.55502: 0 1/0/0 A 142.250.190.46 (44)
```

Watch raw packets to/from `1.1.1.1`. Run `dig @1.1.1.1 google.com` in another terminal and see the query/response captured.

```bash
$ nslookup -type=A 1.1.1.1
Server:         8.8.8.8
Address:        8.8.8.8#53

** server can't find 1.1.1.1: NXDOMAIN
```

Reverse-lookup nslookup style. NXDOMAIN here just means there's no A record for the literal name "1.1.1.1" (not the IP).

Now some router-level commands. These are run on a Cisco/Juniper/FRR router that's running BGP. You won't run them on your laptop.

```cisco
router# show ip route 1.1.1.1
Routing entry for 1.1.1.0/24
  Known via "bgp 65000", distance 20, metric 0
  Tag 13335, type external
  Last update from 198.51.100.1 00:23:11 ago
  Routing Descriptor Blocks:
    198.51.100.1, from 198.51.100.1, 00:23:11 ago
      Route metric is 0, traffic share count is 1
      AS Hops 2
      Route tag 13335
```

Cisco IOS command to show how this router is reaching `1.1.1.0/24`. Note the AS-path length and the next hop.

```cisco
router# show ip bgp 1.1.1.1
BGP routing table entry for 1.1.1.0/24, version 12345
Paths: (3 available, best #1, table default)
  13335
    198.51.100.1 from 198.51.100.1 (172.16.0.1)
      Origin IGP, metric 0, localpref 100, valid, external, best
      Community: 65000:100
  6939 13335
    203.0.113.1 from 203.0.113.1 (172.16.0.2)
      Origin IGP, metric 0, localpref 100, valid, external
  174 6939 13335
    198.18.0.1 from 198.18.0.1 (172.16.0.3)
      Origin IGP, metric 0, localpref 100, valid, external
```

Three paths to the same prefix. The router picked path #1 because its AS-path is shortest. This is the heart of anycast — multiple paths, BGP picks the best.

```cisco
router# show bgp ipv4 unicast 1.1.1.1
[same kind of output, IPv4 unicast specifically]
```

```cisco
router# show bgp neighbors 198.51.100.1 advertised-routes | include 1.1.1.0
   1.1.1.0/24       198.51.100.1            0             0 13335 i
```

Confirm what we're announcing to a specific neighbor. Useful for verifying anycast announcements are actually going out.

To configure an anycast announcement on a Cisco router:

```cisco
router# configure terminal
router(config)# router bgp 65000
router(config-router)# network 1.1.1.0 mask 255.255.255.0
router(config-router)# !
router(config-router)# exit
router(config)# ip route 1.1.1.0 255.255.255.0 Null0
router(config)# end
```

The "null0" trick: BGP only announces a prefix if there's a matching IGP route. By creating a route to `Null0`, we satisfy that requirement. The actual traffic doesn't go to `Null0`; it gets BGP-routed to wherever the more-specific routes (or local interfaces) lead. This is a classic Cisco idiom for anycast announcements.

For FRR (the open-source BGP daemon):

```bash
$ vtysh -c 'configure terminal'
$ vtysh -c 'router bgp 65000'
$ vtysh -c 'address-family ipv4 unicast'
$ vtysh -c 'network 1.1.1.0/24'
$ vtysh -c 'exit-address-family'
$ vtysh -c 'exit'
```

Or via config file (`/etc/frr/frr.conf`):

```
router bgp 65000
 address-family ipv4 unicast
  network 1.1.1.0/24
 exit-address-family
exit
```

For GoBGP:

```bash
$ gobgp global rib add 1.1.1.0/24
```

For BIRD:

```bash
$ sudo birdc 'show route protocol bgp1'
$ sudo birdc 'show protocols all'
```

For BIRD, you'd add a static route in BIRD's config and a `protocol bgp` block exporting it. Minimal setup:

```
protocol static {
    ipv4;
    route 1.1.1.0/24 reject;
}
protocol bgp upstream1 {
    local as 65000;
    neighbor 198.51.100.1 as 174;
    ipv4 {
        export filter { if (net = 1.1.1.0/24) then accept; else reject; };
    };
}
```

Verify FRR's BGP state:

```bash
$ vtysh -c 'show ip bgp summary'
BGP router identifier 172.16.0.1, local AS number 65000 vrf-id 0
Neighbor   V  AS  MsgRcvd MsgSent TblVer InQ OutQ Up/Down  State/PfxRcd
198.51.100.1  4  174  12345    12340    1    0   0  03:21:00         845671
```

```bash
$ vtysh -c 'show ip bgp 1.1.1.1'
BGP routing table entry for 1.1.1.0/24
Paths: (1 available, best #1, table default)
  Advertised to non peer-group peers:
  198.51.100.1
  Local
    0.0.0.0 from 0.0.0.0 (172.16.0.1)
      Origin IGP, metric 0, localpref 100, weight 32768, valid, sourced, local, best (Network)
      Last update: ...
```

Show running BGP config:

```bash
$ vtysh -c 'show running-config' | sed -n '/router bgp/,/^!/p'
router bgp 65000
 address-family ipv4 unicast
  network 1.1.1.0/24
 exit-address-family
exit
```

(That `sed` snippet is a quick way to scope to the BGP section. Skip it if you have `show run | section bgp` available, like on Cisco.)

```bash
$ tcpdump -i any -nn -e 'host 1.1.1.1 and port 179'
```

Watch BGP packets (port 179) involving an anycast peer.

To watch a server **withdraw and re-announce** in test:

```bash
$ vtysh -c 'configure terminal' \
       -c 'router bgp 65000' \
       -c 'address-family ipv4 unicast' \
       -c 'no network 1.1.1.0/24' \
       -c 'exit-address-family' \
       -c 'exit'
# wait some seconds
$ vtysh -c 'configure terminal' \
       -c 'router bgp 65000' \
       -c 'address-family ipv4 unicast' \
       -c 'network 1.1.1.0/24' \
       -c 'exit-address-family' \
       -c 'exit'
```

In another terminal, run continuous pings to watch traffic shift away and come back.

## Common Confusions

People confuse a lot of things about anycast. Let's untangle:

### 1. "Anycast" vs "anycast with failover"

There's no separate "with failover" mode. **Every anycast deployment is failover by default** because BGP withdrawal naturally re-routes traffic. If a POP drops out, the next-best path is used. That's failover.

### 2. How does the routing converge?

When a POP withdraws, its upstream peers process the withdrawal in seconds. They re-run best-path. They forward the withdrawal to their peers. The wave propagates. Total convergence time depends on AS-path length: each AS hop adds a few seconds. A typical "shutdown of one POP" converges globally in 30s to 3 minutes.

### 3. What "selective withdraw" achieves

A POP can withdraw an announcement from **just one upstream peer** while continuing to announce to other peers. This shifts traffic away from that peer (because users behind that peer pick up an alternate path) while keeping the POP active overall. Useful for **partial maintenance**, peering link issues, or to push traffic away from a congested or expensive peer.

### 4. Can multiple anycast nodes share TLS certs?

Yes — every anycast node serving the same hostname must share the same cert (and matching private key). Otherwise users get cert errors when they hit different nodes. This is a key part of anycast design.

### 5. Anycast vs Geo-DNS

Anycast routes at the network layer (BGP). Geo-DNS routes at the DNS layer. Anycast is invisible to the application. Geo-DNS isn't — clients see different DNS answers.

Anycast is more elegant and faster (no extra DNS lookup time wasted). Geo-DNS is more flexible (you can route based on application-level signals like user-agent, time of day, etc.). Most large CDNs use both: anycast for the fast frontend, geo-DNS for routing to specific regional services.

### 6. What is "site-local anycast"?

Anycast within a single AS (one company's network). Used for things like internal services, where multiple replicas advertise the same IP and IGP routing (OSPF, IS-IS) sends each user to their nearest copy. Doesn't cross AS boundaries; just an internal optimization.

### 7. IPv6 reserved anycast addresses

RFC 2526 defines a few specific anycast addresses inside an IPv6 subnet (like the Subnet-Router address). Mostly for advanced routing tricks. Not used much in practice.

### 8. The "all-1s in last bits" subnet-router anycast in IPv6

Specifically, the Subnet-Router Anycast Address is the prefix followed by all zeros in the interface ID. So for `2001:db8:1::/64`, the address `2001:db8:1::` is anycast for "any router on this subnet."

### 9. How Spotify uses unicast not anycast

Spotify and other heavy media services often use **unicast** with DNS-based load balancing. Why? Because anycast's flow re-routing risk is dangerous for long media streams. Spotify deliberately picks one CDN node per session and stays there. They prioritize reliability over the small latency wins from anycast.

### 10. CDN anycast tier with backend unicast

Standard pattern: clients hit anycast CDN POPs. POPs talk to origin servers via unicast (over the public internet, or via private CDN backbone). This separates "user-facing global anycast" from "origin-facing reliable unicast."

### 11. What is TCP keepalive's role in anycast?

TCP keepalive sends occasional packets on idle connections to detect dead peers. In anycast, keepalive can **detect** when a connection has silently broken (because the peer node changed and won't reply). Without keepalive, an idle connection might appear up forever even though it's actually dead.

### 12. Mid-flight failure on stateless protocols

Stateless protocols (UDP DNS, NTP) don't care about mid-flight node changes. The next packet just gets a new node and a new answer. The "session" is one packet; the next packet is independent.

### 13. "Long flow" cookie-affinity-vs-anycast

Some applications use **session cookies** to identify a user. Anycast routing might bounce a user between POPs. If the application requires "all my requests go to the same POP" (because state is local), you have a problem. Solutions: use sticky cookies that include POP IDs, or build connection-sync infrastructure to migrate state.

### 14. How to debug "different responses from same IP"

When users see inconsistent responses from the same IP (`1.1.1.1`), it's almost always anycast pointing them to different POPs that have different state. Debug:

1. Use `cdn-cgi/trace` or `dig chaos txt id.server` to see which POP each user hit.
2. Check if the POPs disagree — software version, cert version, content version.
3. If they disagree, deploy a fix synchronously across all POPs.

### 15. Anycast vs ECMP

ECMP (Equal-Cost Multi-Path) is router-internal load balancing across multiple paths to the same destination. ECMP and anycast can interact: a router with ECMP to multiple anycast peers may hash-distribute its packets across them, sending some flows to one and some to another. This is fine for well-tuned anycast but can be problematic if flow-hashing is poorly configured.

### 16. Anycast vs DNS load balancing

DNS load balancing returns multiple A records for one name. The client picks one. This is "client-side load balancing" and is decoupled from anycast. You can do **both** at once: have multiple anycast IPs, return all of them via DNS, and let clients fall back if one is unreachable.

### 17. What about the source IP — is it anycast too?

Usually no. The source IP for the **server's** reply is the anycast IP (so the client receives a reply from the IP it queried). The source IP for the **server's** outgoing connections (e.g., to fetch from origin) is typically a unicast IP specific to that POP.

### 18. Anycast and IPv6 SLAAC

IPv6 SLAAC (Stateless Address Autoconfiguration) auto-generates host IPs from the prefix. Anycast addresses are deliberately assigned, not auto-generated. SLAAC and anycast don't overlap — the anycast address is set up by the operator, while SLAAC handles host addresses for normal traffic.

## Vocabulary

This list defines every weird word in this sheet. If something tripped you up, look here.

| Term | Plain English |
|---|---|
| anycast | One IP address shared by many machines; closest copy answers |
| unicast | Boring normal IP — one address, one machine |
| multicast | One address, many subscribed machines, all get a copy |
| broadcast | One address, every machine on this LAN gets a copy |
| anycast IP | The shared IP address used in anycast |
| anycast prefix | The IP range (e.g., `1.1.1.0/24`) advertised in anycast |
| anycast announcement | A BGP UPDATE message claiming reachability for an anycast prefix |
| BGP | Border Gateway Protocol — internet's main routing protocol |
| AS path | List of ASes a route passes through |
| AS_PATH attribute | The actual BGP attribute carrying the AS path |
| MED | Multi-Exit Discriminator — BGP attribute for picking between multiple paths from same neighbor AS |
| local preference | BGP attribute set by your AS to prefer some paths over others |
| BGP best-path algorithm | The series of tiebreaker rules BGP uses to pick one route from many |
| eBGP | External BGP — between two different ASes |
| iBGP | Internal BGP — within one AS |
| route reflector | An iBGP server that re-distributes routes to clients |
| BGP withdrawal | Telling neighbors "I no longer have a route for this prefix" |
| BGP UPDATE message | The BGP message that carries new prefixes or withdrawals |
| NLRI | Network Layer Reachability Information — list of prefixes in a BGP UPDATE |
| origin code | Where the route originated: IGP, EGP, or unknown |
| community attribute | Tags attached to BGP routes for traffic engineering |
| well-known communities | Standard tags like NO_EXPORT, NO_ADVERTISE |
| NO_EXPORT | "Don't tell anyone outside the AS about this route" |
| NO_ADVERTISE | "Don't advertise this route to any peer" |
| large communities | RFC 8092 expanded BGP communities — fits 32-bit ASNs |
| AS-set | Old way to summarize multiple ASes in one BGP attribute |
| AS-confederation | Splitting one AS into smaller logical ASes for scaling |
| BGP route flap dampening | Penalize prefixes that change too often, to limit churn |
| RPKI ROA validation | Verify that an AS is authorized to originate a prefix |
| route origin authorization | RPKI artifact that authorizes a specific AS to originate a prefix |
| prefix length | How many bits of the IP address are network bits (e.g., `/24`) |
| longest-prefix match | Routing rule: more specific prefixes win over less specific |
| AS-path prepending | Adding extra copies of your AS to a route to make the path longer |
| BFD | Bidirectional Forwarding Detection — sub-second link failure detection |
| fast convergence | BGP/IGP techniques for quick recovery after failures |
| graceful restart | Letting a router restart its BGP without dropping its routes |
| GTSM | Generalized TTL Security Mechanism — checks BGP packet TTL for security |
| MD5/TCP-AO authentication | Cryptographic auth on BGP TCP sessions |
| anycast cluster | A group of anycast nodes serving the same prefix |
| anycast pool | Same idea, often used for the set of physical IPs |
| anycast hash | Hash function used to pick which anycast node serves a given flow |
| anycast load balancer | Frontend load balancer fronted by anycast IPs |
| ECMP at the edge | Multiple paths from the edge router to the same destination |
| ECMP at the core | Multiple paths within the core network |
| AS_PATH manipulation | Doctoring the AS-path for traffic-engineering purposes |
| traffic engineering | Influencing where traffic flows in your network |
| route-map | Cisco/IOS construct for matching and modifying routes |
| prefix-list | List of prefixes used to filter or match routes |
| ip access-list | Generic packet filter on a Cisco router |
| ip prefix-list | More precise prefix-matching list than access-list |
| anycast cluster ID | Identifier for a group of anycast nodes |
| anycast hash function | Hash used to assign flows to nodes within a cluster |
| hashing 5-tuple | Hashing on (src_ip, src_port, dst_ip, dst_port, protocol) |
| hashing on inner-flow for VXLAN | Specialized hash for VXLAN-encapsulated traffic |
| NetFlow / IPFIX / sFlow | Sampling protocols for measuring traffic per anycast flow |
| ASN | Autonomous System Number — unique ID for each routing domain |
| IANA AS allocation | IANA delegates AS-number ranges to RIRs |
| RIR | Regional Internet Registry — gives out IPs/ASNs (ARIN, RIPE, APNIC, AFRINIC, LACNIC) |
| ARIN | RIR for North America |
| RIPE | RIR for Europe / Middle East / Central Asia |
| APNIC | RIR for Asia-Pacific |
| AFRINIC | RIR for Africa |
| LACNIC | RIR for Latin America and Caribbean |
| AS-Override | BGP feature to overwrite the customer's AS in the path with the SP's AS |
| allowas-in | Accept BGP routes that have your own AS in the path |
| ebgp-multihop | Allow eBGP sessions to neighbors more than 1 hop away |
| BGP Route Refresh | Ask neighbor to re-send all routes without dropping the session |
| BGP graceful shutdown | Withdrawing routes politely with a community signaling intent |
| BGP-LS | BGP Link-State extension — used by SDN controllers for visibility |
| BGP-FlowSpec | BGP-distributed flow filters for DDoS mitigation |
| BGP communities for traffic engineering | Use community tags to influence neighbor routing |
| RPKI-validated anycast | Anycast prefixes authorized by RPKI ROAs |
| BGPsec | RFC 8205 — cryptographic signing of AS_PATH segments |
| Resource Public Key Infrastructure | The PKI that backs RPKI |
| RPKI-to-Router protocol | How routers fetch RPKI validation data from validators |
| ASPA | Autonomous System Provider Authorization — RPKI-style authorization for upstream providers |
| DNSSEC and anycast | DNSSEC works fine with anycast as long as keys are sync'd |
| DoH | DNS-over-HTTPS — DNS over HTTPS, often anycast |
| DoT | DNS-over-TLS — DNS over TLS, also often anycast |
| ECS | EDNS0 Client Subnet — sends user's subnet to the auth server for geo-DNS |
| RIRs allocation policies | Each RIR's rules for IP/ASN delegation |
| IRR | Internet Routing Registry — public database of route announcements |
| RPSL | Routing Policy Specification Language — used by IRRs |
| MANRS | Mutually Agreed Norms for Routing Security — best practices for BGP |
| peering DB | peeringdb.com — public database of network peering points |
| IXP | Internet Exchange Point — physical place where networks peer |
| private peering | Direct fiber between two networks |
| public peering | Shared switch fabric at an IXP |
| route servers at IXPs | BGP-over-shared-fabric, simplifies multilateral peering |
| latency-based routing | Routing decisions based on measured latency |
| geographic routing | Routing decisions based on physical geography |
| geo-DNS | DNS that returns different answers based on requester's geography |
| GSLB | Global Server Load Balancing — DNS-driven multi-region balancing |
| F5 BIG-IP DNS | F5's GSLB product |
| AWS Route 53 latency-based | AWS's DNS-driven latency routing |
| Cloudflare global anycast | Cloudflare's planet-wide anycast network |
| Fastly anycast | Fastly's CDN anycast network |
| AWS Global Accelerator | AWS's anycast IP product (TCP+UDP front for AWS regions) |
| Google QUIC PSP | Google's privacy-sandbox protocol, uses QUIC |
| BBR | Bottleneck Bandwidth and RTT — modern TCP congestion control |
| ECMP polarization | When ECMP at multiple layers picks the same path, defeating the redundancy |
| 4-tuple | (src_ip, src_port, dst_ip, dst_port) — identifies a TCP/UDP flow |
| 5-tuple | 4-tuple plus protocol number |
| connection pinning | Keeping a TCP/UDP flow stuck to one node |
| connection sync | Sharing connection state between nodes for failover |
| connection migration | Moving a connection from one path/node to another (QUIC's superpower) |
| TCP keepalive | Periodic empty TCP packets to detect dead peers |
| TCP RST | Reset packet — abruptly closes a connection |
| TLS handshake | Encrypted-connection setup before app data flows |
| TLS session ticket | Reusable token to skip parts of the TLS handshake on repeat connections |
| QUIC | UDP-based transport protocol; HTTP/3 uses it |
| QUIC connection ID | Identifier in QUIC packet header, decouples from IP/port |
| HTTP/2 | TCP-based binary HTTP protocol |
| HTTP/3 | QUIC-based HTTP protocol |
| anycast convergence | Time for routing changes to propagate through BGP |
| graceful introduction | Adding back a returning POP slowly to avoid disruption |
| chaos class DNS | Special DNS class used for diagnostics (id.server.) |
| trace endpoint | Cloudflare's `cdn-cgi/trace` URL — reveals POP info |
| POP | Point of Presence — a physical site of a CDN/ISP |
| edge POP | Frontend POP that serves end users |
| origin server | The "real" backend server behind a CDN |
| cold cache | Empty cache on a freshly-started node |
| warm cache | Cache that's already populated with hot data |
| keyless SSL | Cloudflare's tech to do TLS without storing private keys at the POP |
| certificate sharing | Multiple anycast nodes serving the same cert |
| asymmetric routing | Forward and reverse paths use different routes |
| ECMP hash | Hash that picks one of several equal-cost paths |
| flow table | Per-flow state, used by stateful middleboxes |
| connection table | OS-level table of active TCP/UDP connections |
| graceful shutdown | Withdrawing a service in a way that minimizes user disruption |
| BFD echo mode | BFD running with self-echoes for sub-second detection |
| traffic share count | Cisco metric: how many flows are using each path |
| longest match wins | Routing rule: more-specific prefix beats less-specific |
| RPKI invalid | Route announcement that fails RPKI validation |
| RPKI valid | Route announcement that passes RPKI validation |
| RPKI not-found | No RPKI ROA covers the announcement (neither valid nor invalid) |
| AS path filter | Configuration that drops routes based on AS path patterns |
| BGP confederation member AS | Sub-AS within a BGP confederation |
| BGP next-hop | The IP of the immediate next router in the path |
| BGP peer-group | Configuration shortcut for groups of similar BGP peers |
| BGP community match | Filtering routes based on community tags |
| graceful restart helper | Neighbor that helps a restarting BGP speaker preserve state |
| graceful restart speaker | The router that's actually restarting |
| iBGP full mesh | Every iBGP router peers with every other; doesn't scale |
| route refresh capability | BGP capability to ask for a re-send of routes |
| anycast scaling | How anycast naturally scales as you add POPs |
| anycast probe | Active testing of anycast — ping each POP separately |
| BGP visibility tools | Looking glasses, route servers, RIPEstat, etc. |
| RIPEstat | RIPE's BGP visibility tool |
| Cymru WHOIS | Team Cymru's BGP-aware whois service |
| RouteViews | University of Oregon's BGP collection project |
| BGP looking glass | A web-accessible router from which to run BGP queries |
| AS_TRANS | Special AS used for backward compatibility with 16-bit ASNs |
| AS-set notation | Abbreviation for a set of ASes |
| AS-set in IRR | IRR object listing member ASes |
| inbound route filter | Filter on routes coming in from a peer |
| outbound route filter | Filter on routes going out to a peer |
| peering policy | Rules for who you peer with and how |
| peering ratio | Inbound/outbound traffic ratio between two networks |
| settlement-free peering | Peering with no money exchanged |
| transit | Paid service to reach the rest of the internet |
| flat-rate transit | Pay regardless of usage |
| 95th percentile billing | Pay based on 95th-percentile traffic over the month |
| metro fiber | Local-area fiber network in a city |
| dark fiber | Unused fiber that you can light up yourself |
| BGP add-path | BGP capability to advertise multiple paths to the same prefix |
| ECMP load distribution | Spreading traffic across equal-cost paths |
| Cloudflare colo | Cloudflare's term for a POP (e.g., `colo=SJC`) |
| Cloudflare 1.1.1.1 | Cloudflare's anycast public DNS resolver |
| Cloudflare 1.0.0.1 | Cloudflare's anycast secondary DNS resolver |
| Google 8.8.8.8 | Google's anycast public DNS resolver |
| Google 8.8.4.4 | Google's anycast secondary DNS resolver |
| Quad9 9.9.9.9 | Quad9's anycast public DNS resolver |
| OpenDNS 208.67.222.222 | OpenDNS/Cisco's anycast public DNS resolver |

### One more diagram: anycast at scale

Here's a sketch of how the entire ecosystem fits together. It's a lot, but it's the **whole picture** of why anycast works.

```
   ┌────────────────────────────────────────────────────────────┐
   │                       Global Internet                       │
   │                                                             │
   │    AS 174    AS 6939    AS 2914    AS 3356    AS 1299       │
   │  (Cogent)  (HE.net)  (NTT)     (Lumen)    (Telia)           │
   │     │         │         │         │         │               │
   │     └─────────┴─────────┴─────────┴─────────┘               │
   │                          │                                  │
   │                BGP transit/peering fabric                   │
   │                          │                                  │
   │  ┌───────────────────────┼───────────────────────┐          │
   │  │                       │                       │          │
   │  │            AS 13335 (Cloudflare)              │          │
   │  │  ┌─────────────────────────────────────────┐  │          │
   │  │  │ POPs around the world all announce      │  │          │
   │  │  │ 1.1.1.0/24 (and other anycast prefixes) │  │          │
   │  │  └─────────────────────────────────────────┘  │          │
   │  │                                               │          │
   │  │     POP-Tokyo    POP-LON    POP-NYC    ...    │          │
   │  └───────────────────────┬───────────────────────┘          │
   │                          │                                  │
   │  Each POP fetches from origin via standard unicast          │
   │  (or via Cloudflare's private mesh for cross-POP work)      │
   │                          │                                  │
   │  ┌───────────────────────┴────────────────────────┐         │
   │  │ Customer origin servers (regular unicast IPs)  │         │
   │  └────────────────────────────────────────────────┘         │
   └────────────────────────────────────────────────────────────┘
```

The transit network is unicast. Cloudflare's POPs are anycast. The origins are unicast. Anycast is a middle layer — a way to give one global IP to a globally-distributed service. It's a layer of abstraction that turns "many computers in many cities" into "one IP everyone uses." That abstraction lives entirely in BGP.

### A note on the future

Anycast is here to stay. If anything, it's getting more popular. New protocols (QUIC, MASQUE) are being designed with anycast-friendliness in mind. New CDN-like services (edge compute, edge databases, edge AI inference) all use anycast at the frontend.

The big areas of active development:

1. **Better connection migration** — letting connections survive routing changes.
2. **Per-flow steering** — letting individual flows be routed independently of the BGP best-path.
3. **Programmable anycast** — operators dynamically adjusting which POPs announce, based on real-time signals (load, latency, attacks).
4. **RPKI/BGPsec** — making anycast announcements provably authentic.
5. **Anycast for SD-WAN** — corporate networks using anycast for branch-to-branch traffic.

## Try This

Real experiments. Pick a few. Compare what you see across different network connections.

1. **POP comparison.** Run `curl https://1.1.1.1/cdn-cgi/trace` from your home network. Note the `colo=` value. Now SSH to a cloud VM in another region (or use a public looking glass like RIPEstat) and run the same command. Compare. You'll see different `colo` codes — confirming you hit different POPs.

2. **DNS chaos record.** Run `dig @1.1.1.1 chaos txt id.server.` from home, then again from a different network (mobile hotspot, work, friend's house). The TXT response is the POP code. Confirm anycast in action.

3. **Latency from multiple sources.** Use a tool like `globalping.io` (free, web-based) to ping `1.1.1.1` from 50 different locations worldwide. You'll see consistent low-millisecond latencies from every region — that's anycast giving everyone a local POP.

4. **Traceroute different anycast IPs.** Run `traceroute -A 1.1.1.1`, `traceroute -A 8.8.8.8`, and `traceroute -A 9.9.9.9` from the same machine. Note how the AS paths differ — each anycast network has its own routing.

5. **Route viewer.** Visit a public BGP looking glass (e.g., `lg.he.net` for Hurricane Electric's looking glass) and run `show ip bgp 1.1.1.1`. You'll see multiple paths advertised, with different AS-path lengths. The router picks the shortest.

6. **Withdraw test in a lab.** Set up a small lab with FRR or BIRD running on a VPS. Announce a `/24` of your own (e.g., from a public IP space). Add a second VPS in a different region. Announce the same `/24` from both. Watch traffic distribute. Then withdraw one — watch traffic shift to the other.

7. **DNS root timing.** Run `dig +short @a.root-servers.net . NS` and time it. Compare to running it from a different region. The roots are heavily anycast — both queries should come back fast.

8. **Compare same-name queries.** `dig @1.1.1.1 google.com` vs `dig @8.8.8.8 google.com`. The answers may differ because Google's `google.com` is also anycast and Google may return different IPs to different resolvers.

9. **HTTP from many anycast IPs.** Run `curl -v https://1.1.1.1/`, `curl -v https://1.0.0.1/`. Both Cloudflare anycast IPs. Note the TLS cert is the same and POP is the same.

10. **Watch BGP UPDATE messages.** If you have a router or BGP speaker, run a packet capture on TCP port 179 between you and your upstream. Filter for UPDATE messages. Watch the prefix appear and disappear when you withdraw.

11. **Curse of the cold cache.** Spin up a CDN POP from scratch (e.g., on a self-hosted Varnish). Add it to a small anycast pool. Watch its origin-fetch traffic spike. Then add another. The first POP's origin traffic drops as the second absorbs some load.

12. **Connection drops in flight.** Open a long-lived TCP connection (e.g., `nc 1.1.1.1 80` followed by an HTTP request that lingers). Then change your network (wifi to cellular). Watch the connection die — that's exactly what happens when anycast routing changes.

### Bonus experiments

13. **The withdraw-and-converge stopwatch.** From a remote VPS, run `mtr 1.1.1.1` continuously. Now (in another window, on a router you control) withdraw a prefix from your BGP. Watch the `mtr` output flicker as the route flaps. Time how long until the latency settles into a stable new value. That's your convergence time.

14. **AS-path inspection at scale.** Use the RIPEstat API: `curl 'https://stat.ripe.net/data/looking-glass/data.json?resource=1.1.1.1'`. The JSON shows AS paths from many observation points worldwide. You can see exactly which routes Cloudflare is announcing globally.

15. **Trace from many continents.** Use globalping.io to run `traceroute 1.1.1.1` from 5 continents simultaneously. Compare the AS paths. Each will be different, but each will land at Cloudflare in a few hops.

16. **DNS resolver distance.** Run `dig +short txt id.server. @1.1.1.1` and `dig +short txt id.server. @8.8.8.8` and `dig +short txt id.server. @9.9.9.9`. Each returns a different POP code in a different format. You're seeing three separate anycast resolvers' POP IDs side by side.

17. **Manual prefix-list inspection.** If you have a Cisco router, `show ip prefix-list` shows your filters. Add a prefix-list that only accepts your anycast prefix from your own AS. Now your BGP session won't accept anyone else's announcements. Useful for a hijack-defense check.

## Where to Go Next

You now understand anycast. Next steps:

- Read `ramp-up/dns-eli5` if you haven't yet — DNS is the most anycast-saturated protocol.
- Read `networking/bgp` for deeper BGP details (timers, attributes, configuration patterns).
- Read `networking/bgp-advanced` for traffic engineering, RPKI, BGPsec, and large-scale BGP design.
- Read `networking/quic` to understand the next-gen transport that handles anycast better.
- Read `networking/segment-routing` if you want to see how operators stitch anycast plus SR for path engineering.
- Read `security/network-defense` for DDoS mitigation patterns built on anycast.

## See Also

- `networking/bgp`
- `networking/bgp-advanced`
- `networking/dns`
- `networking/dig`
- `networking/coredns`
- `networking/doh-dot`
- `networking/quic`
- `networking/ecmp`
- `networking/segment-routing`
- `networking/ipv4`
- `networking/ipv6`
- `networking/ipv6-advanced`
- `security/network-defense`
- `ramp-up/bgp-eli5`
- `ramp-up/ip-eli5`
- `ramp-up/dns-eli5`
- `ramp-up/linux-kernel-eli5`

## References

- RFC 1546 — "Host Anycasting Service" (1993, informational; first formal mention of anycast)
- RFC 4786 — "Operation of Anycast Services" (2006; operational best practices)
- RFC 7094 — "Architectural Considerations of IP Anycast" (2014; modern thinking)
- RFC 6724 — "Default Address Selection for IPv6"
- RFC 4291 — "IPv6 Addressing Architecture" (defines anycast as first-class)
- RFC 2526 — "Reserved IPv6 Subnet Anycast Addresses"
- RFC 8092 — "BGP Large Communities Attribute"
- RFC 8205 — "BGPsec Protocol Specification"
- "How to Build a CDN" by Will Hawkins
- Cloudflare blog — "1.1.1.1 launch" (April 2018)
- Google's research papers on anycast for DNS
- Verisign technical reports on root server anycast deployments
- ICANN root server overview at root-servers.org
- BGP Best Path Selection — Cisco IOS documentation

### A final analogy: anycast and cell phone towers

Here's one more analogy because they're fun and they help.

When you walk around with a cell phone, you don't think about which cell tower you're connected to. You're just "on the network." As you walk, your phone quietly shifts between towers — a soft handoff. The towers are different physical hardware, but the network treats them as a unified service. Calls don't drop (unless you walk into a bad area). The system is robust because you're always on **a** tower, not **the** tower.

Anycast feels similar at internet scale. Your packets don't pick a server. They pick "the network." The network shifts your traffic between servers as conditions change. Most of the time, you don't notice. Anycast tries to behave like cell handoff: smooth, automatic, invisible.

The big difference is that cell phones have explicit handoff protocols (the phone and the network coordinate the switch). Anycast doesn't — packets just follow the routing-of-the-moment, and stateful applications above have to handle the consequences. Cell networks figured out a long time ago that smooth handoff requires real protocol effort. Anycast is still figuring that out (with QUIC migration, connection sync meshes, and so on).

But the goal is the same: make a giant distributed system feel like one service to the user.

### Version history

- 1993: RFC 1546 introduces the anycast concept (informational, never widely deployed as written).
- 2006: RFC 4786 documents the production-ready anycast operations conventions used by root servers and TLD operators.
- 2006: RFC 4291 makes IPv6 anycast a first-class concept in the IP addressing architecture.
- 2010-2015: CDNs (Cloudflare, Fastly) make anycast HTTP universal at the consumer edge.
- 2018: Cloudflare launches `1.1.1.1` as a public-facing anycast DNS resolver, dramatically popularizing anycast as a household name.
- 2020+: QUIC adoption brings anycast-friendly connection migration to HTTP/3, smoothing over many of TCP-anycast's issues.
- 2024+: ASPA (RFC 9509) and BGPsec deployment milestones strengthen anycast routing security.
