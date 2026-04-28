# Named Data Networking — ELI5

> NDN is a network where you ask for a thing by **name** ("/grandma/photos/birthday-cake.jpg") and the closest copy in the network ships back to you, no matter which computer is hosting it. The internet stops caring about WHERE data lives and starts caring about WHAT data you want.

## Prerequisites

- `ramp-up/ip-eli5` — what an IP address is and why every computer on today's internet has one.
- `ramp-up/dns-eli5` — how human names get resolved to addresses on today's internet (NDN basically eats DNS).
- `ramp-up/anycast-eli5` — how "the closest copy answers" works for IP addresses today (NDN does this for everything, by default).

This sheet uses words like "router," "packet," "DNS," "TCP," and "URL" without re-explaining them in full. If those words make you go "huh?" then go read the prereq sheets first. They are short. Come back here. We will wait.

If a word feels weird, look it up in the **Vocabulary** table near the bottom of this sheet. Every weird word has a one-line plain-English definition.

If you see a `$` at the start of a line in a code block, that means "type the rest of this line into your terminal." You do not type the `$`. The lines underneath that don't have a `$` are what your computer prints back. We call that "output."

## The Big Idea

### One sentence to carry around forever

**Named Data Networking is a future-internet design where every router along your path knows how to look up data by its name (like `/youtube/watch/dQw4w9WgXcQ/segment/42`) and ships back a signed, cacheable copy from wherever in the network it can find one — without ever caring about the *address* of the machine that originally produced it.**

That is the whole thing. That is NDN. Every other detail in this sheet is a footnote on that sentence. We are going to spend the rest of the sheet picking that sentence apart, because every word in it is doing real work and most of those words have surprises hiding inside.

The surprising words, in case you missed them:

- "**every router along your path**" — not just the endpoints. The boxes in the middle of the network do real work. They have memory. They have tables. They cache.
- "**knows how to look up data by its name**" — not by IP address. Not by port number. By a *path-like name* with slashes in it.
- "**ships back a signed copy**" — every chunk of data carries a cryptographic signature. The data is trustworthy on its own, anywhere you find it.
- "**cacheable**" — any router in the middle is allowed to keep a copy and answer future requests for the same name from cache.
- "**from wherever in the network it can find one**" — the closest copy wins. There is no canonical "origin server" the way there is on today's web.
- "**without ever caring about the address of the machine that originally produced it**" — the consumer doesn't know, doesn't care, and doesn't need to know which machine is hosting `/youtube/watch/...`.

If your gut just went "wait, isn't that what a CDN does?" — yes, sort of. NDN is what happens if you take the *idea* behind CDNs and bake it into the network layer itself, replacing IP. CDNs are the hack we built on top of an address-centric internet to make content-centric workloads bearable. NDN says: stop hacking, redesign the bottom turtle, make content-centric the default.

### The library checkout analogy

Pretend you walk into a public library and ask the librarian: "I want *Pride and Prejudice*, the 2008 Penguin Classics edition, chapter 4." That is your **Interest**. You did not say "give me the copy on shelf B-23, third from the left." You did not say "fetch me the copy at the Mountain View branch." You said: **what** you want. The librarian's job is to **find** a copy. They might:

- Hand you the one on the cart they're rolling around (cache hit at the desk).
- Walk over to the shelf and grab one (cache hit deeper in the building).
- Call another branch and ask them to interlibrary-loan a copy (forward the request upstream).
- Tell you it doesn't exist and you should ask the publisher (forward all the way to the producer).

Whatever they do, what comes back to you is **a book**. The book has its title, edition, and chapter on the cover. The book has the author's signature on it (you trust the author, not the librarian). Two different librarians might fetch you a different physical copy of the same chapter — but the *content* is identical, so you don't care.

That is NDN. The librarian is the **NDN forwarder** (a router that speaks NDN). Your request is an **Interest packet**. The book that comes back is a **Data packet**. The "title and edition and chapter" on the cover is the **Name**. The author's signature is the literal cryptographic signature on the Data packet. The "cart, shelf, and other branches" are tiers of cache: **Content Store** (in-router cache), **Pending Interest Table** (current requests), **Forwarding Information Base** (where to send requests we don't have answers for).

### The video-on-demand analogy

You start playing a Netflix episode. Internally Netflix has chunked that episode into hundreds of small video segments — say, 4 seconds of video each. Your player asks for them in order: segment 1, segment 2, segment 3, etc.

Today's internet way: your player opens a TCP connection to one of Netflix's CDN edge servers (an IP address picked by DNS), and asks for `/episode/season3/ep4/seg1.ts` over HTTP. That edge server might serve from its disk, or might fetch from a Netflix origin behind it. You re-do this for each segment. If a thousand neighbours on your ISP all ask for the same Netflix episode at the same time, your ISP's network carries a thousand copies of those segments across its backbone — one per viewer.

The NDN way: your player emits an **Interest** with the name `/netflix/show/strangerthings/s3/e4/seg1`. The first NDN-aware router (let's say in your home, or your ISP's headend) checks: do I already have this Data cached? If yes — boom, ships it back instantly, packet never crosses the ISP backbone. If no, the router forwards the Interest upstream (and remembers, in its **PIT**, that you asked). The Interest moves hop by hop until it hits a router that *does* have the chunk in cache, or all the way to a Netflix producer. The Data flows back along the **reverse path** of the Interest — and **every router on the way back caches the Data** — which means when your neighbour requests the same chunk three milliseconds later, the closest router answers from cache.

A thousand neighbours watching the same show? The chunk is fetched from Netflix once. It lives in a cache in the ISP and answers everybody. The backbone is no longer the bottleneck.

### The recipe analogy

You text grandma: "send me the recipe for chocolate-chip cookies." That is the **name** of the data you want: `/grandma/recipes/chocolate-chip-cookies`. Grandma might be the producer (she knows the recipe). But your sister might already have the recipe printed on the fridge. Your mom has it laminated. Your aunt has it in her email outbox.

In a content-centric world, you don't have to ask grandma specifically. You just *broadcast* "anyone who has `/grandma/recipes/chocolate-chip-cookies`, send it." Whoever has it sends it. The recipe is signed by grandma (she put her name at the bottom — you can verify it really came from her), so even if your aunt forwards it, you trust it because the signature is grandma's, not your aunt's.

That is NDN. The data is signed. The data is cacheable. The data is named. The producer doesn't have to be online for you to get the data. The data is the unit of trust, not the channel.

### Why does this matter?

Because today's internet was designed in 1973 for a world where you wanted to talk to a *specific computer*. ARPA wanted to give a researcher at MIT a way to log into a specific machine at Stanford. They invented IP addresses to identify machines. They invented TCP to give you a stable byte stream between two specific machines. The whole stack assumed the question was "where is that machine, and how do I open a session to it?"

That model worked beautifully for telnet and FTP. It started creaking under HTTP. It now spends most of its time cosplaying as a content-distribution system: **80%+ of internet traffic is video and software downloads**, and the way we make that traffic work is by lying about our address-centric assumptions. We invented CDNs (anycast IPs, geographic DNS, cache hierarchies). We invented BitTorrent. We invented HTTP/2 and HTTP/3 with their multiplexing. We invented QUIC. We invented tracker-style overlays. Every one of these is a workaround for the fact that the bottom turtle still thinks the question is "what's the address of the machine?"

NDN says: change the question. Ask "what's the *name* of the data?" Let routers natively answer that question. Now CDNs are no longer a giant overlay we have to build on top of the internet — they're just *how the internet works*. Caching, multicast, mobility, and trust become built-in instead of bolted-on.

That is the elevator pitch. The rest of the sheet is showing you how it actually works in machine-readable detail.

## Vocabulary

You will need most of these. Skim now. Refer back later.

| Word | Plain-English meaning |
| --- | --- |
| **NDN** | Named Data Networking. A clean-slate internet architecture where packets are addressed by name, not by IP address. |
| **ICN** | Information-Centric Networking. The umbrella term for the family of architectures NDN is part of. NDN is the most widely-known specific design. |
| **CCN / CCNx** | Content-Centric Networking. Earlier name for the same family of ideas; CCNx is the PARC-and-then-Cisco implementation. NDN forked from CCN circa 2010. |
| **Interest packet** | The "request" packet in NDN. Contains a Name. Sent by a Consumer. Routed hop-by-hop toward something that has the Data. |
| **Data packet** | The "reply" packet in NDN. Contains a Name, a payload, and a cryptographic signature. Sent by a Producer or by any cache that has it. |
| **Name** | A hierarchical, path-like, slash-separated identifier for a piece of data. Example: `/google/maps/v2/tile/zoom8/x123/y456`. |
| **Component** | One slash-separated piece of a Name. `/google/maps/v2` has three components: `google`, `maps`, `v2`. Components are TLV-encoded. |
| **TLV** | Type-Length-Value. The binary wire format both Interest and Data packets use. Every field is `<type byte(s)><length><bytes>`. |
| **Consumer** | The application asking for data. Sends Interests. (Today's IP equivalent: a TCP client.) |
| **Producer** | The application originating data. Receives Interests, replies with Data. (Today's IP equivalent: a TCP server.) |
| **Forwarder** | The NDN router. Speaks NDN. Has a Content Store, a PIT, and a FIB. Forwards Interests, returns Data, caches as it goes. |
| **NFD** | NDN Forwarding Daemon. The reference forwarder implementation. The thing you run on your laptop to participate in NDN. |
| **CS** | Content Store. The in-router cache where a forwarder stores Data packets it has seen, indexed by Name. |
| **PIT** | Pending Interest Table. The "I'm waiting on this" table in a forwarder. When an Interest comes in and isn't satisfied by the CS, the forwarder records it in the PIT before forwarding upstream. |
| **FIB** | Forwarding Information Base. The "where do I forward Interests for names I haven't seen?" table. Routed prefixes live here, like a routing table for names. |
| **Strategy** | A pluggable forwarding policy that decides *which* upstream face to send an Interest on when the FIB has multiple choices. NDN ships with several. |
| **Face** | An NDN abstraction for "a connection to another forwarder or to a local app." Could be a UDP tunnel, a TCP tunnel, an Ethernet interface, a Unix domain socket. Equivalent of "interface" in IP. |
| **Selector** | An optional field in older Interest packets that filtered candidate Data: `MustBeFresh`, `MinSuffixComponents`, `MaxSuffixComponents`, `Exclude`, `ChildSelector`. Many were deprecated in modern NDN. |
| **MustBeFresh** | Selector flag: "don't return stale cached Data; only return Data within its FreshnessPeriod." |
| **FreshnessPeriod** | A field in Data packets stating how long the Data is considered fresh in milliseconds. After it expires, MustBeFresh Interests skip it. |
| **Nonce** | A random number in the Interest packet used to detect loops (if a forwarder sees the same Name+Nonce twice, it's a loop). |
| **HopLimit** | TTL-equivalent in NDN. Decremented at each hop. Drops at 0. |
| **InterestLifetime** | How long the consumer is willing to wait for a Data response. Forwarders use it to time out PIT entries. |
| **ForwardingHint** | A list of *delegation names* attached to an Interest, telling forwarders "if you don't have a route for the actual name, try routing to one of these delegation names instead." Used for scalability of the global namespace. |
| **Link** | A small signed object that maps a name to one or more delegation names. Used to construct ForwardingHints. |
| **Trust schema** | A signed policy describing who is allowed to sign which names. The mechanism by which NDN networks decide which Data is authentic. |
| **Sync protocol** | A protocol that keeps multiple consumers in sync about a shared namespace (like ChronoSync, PSync, StateVectorSync). NDN's analog of pub/sub. |
| **Producer mobility** | The problem of a Producer changing location. In IP this is hard. In NDN it's mostly invisible to consumers. |
| **Consumer mobility** | The problem of a Consumer changing location. In IP this is also hard. In NDN it's free — consumers just resend Interests on the new face. |
| **NDN-DPDK** | A high-performance NDN forwarder using DPDK to bypass the kernel and reach 100 Gbps. Different from NFD. |
| **ndnSIM** | An NS-3-based NDN simulator. The standard tool for academic NDN research. |
| **ndn-cxx** | The C++ NDN library. NFD is built on it. Most native NDN apps use it. |
| **python-ndn** | The Python NDN library. Easier to prototype with. |
| **NDN testbed** | A global research network of universities running NDN forwarders connected over IP overlays, used as the public NDN reference network. |
| **DTN** | Delay-Tolerant Networking. A predecessor architecture to ICN, focused on store-and-forward in disconnected networks. |
| **PURSUIT** | An EU-funded ICN architecture using rendezvous-based pub/sub instead of name-routing. |
| **NetInf** | Another EU ICN project; a "network of information" architecture. |
| **MobilityFirst** | A US NSF-funded ICN-adjacent architecture using globally unique identifiers (GUIDs) and a name-resolution service. |
| **GUID** | Globally Unique Identifier. Used in MobilityFirst, not NDN. NDN uses hierarchical names instead. |
| **Self-certifying name** | A name where part of the name is a hash of the data or the producer's key. CCN/CCNx 1.0 used these heavily; NDN tends not to. |
| **Implicit digest** | The SHA-256 of a Data packet's full TLV-encoded form. Can be appended as a final name component to point at *that exact byte sequence* of Data. Useful for content addressing. |
| **Sync state** | The shared dataset two NDN apps want to keep in sync. Sync protocols converge on agreed sync state across members. |
| **PFX (prefix)** | A name prefix. `/google/maps` is a prefix; `/google/maps/v2/tile/...` is below it. FIB entries are keyed by prefix. |
| **LongestPrefixMatch** | The lookup rule used by both PIT and FIB: among all entries that are prefixes of the Interest's name, pick the longest. (Same idea as IP's longest-prefix-match on routes.) |
| **Aggregation** | When two consumers ask for the same Name, the second Interest is *suppressed* by the forwarder — it doesn't go upstream because the first one is already pending in the PIT. The Data, when it arrives, is sent to both. |
| **Multicast suppression** | The natural multicast-from-aggregation effect: 1,000 viewers, one Interest goes upstream. |
| **In-network caching** | The act of forwarders caching Data in their Content Store as it passes through. |
| **LCE** | Leave Copy Everywhere. The simplest in-network caching policy: every forwarder caches every Data it sees. |
| **LCD** | Leave Copy Down. Caching policy: only the forwarder one hop closer to the consumer caches. |
| **ProbCache** | A probabilistic caching policy that distributes cached copies across the path. |
| **Cache poisoning** | A security concern: a malicious forwarder injects bogus Data into a Content Store. Defended against by signature verification. |
| **VIP / Virtual Interest Packet** | A theoretical abstraction used in some forwarding strategies for analytical modeling. |
| **NLSR** | Named-data Link State Routing. A link-state routing protocol for NDN — populates FIBs the way OSPF populates IP routing tables. |
| **NDNCERT** | NDN's certificate management protocol. Issues certificates over NDN itself. |
| **KeyLocator** | A field in a Data packet pointing at the certificate that signed it. |
| **SignatureType** | A field in a Data packet identifying the signature algorithm: SHA-256, SHA-256-with-RSA, SHA-256-with-ECDSA, HMAC, etc. |
| **SignedInterest** | An Interest that itself carries a signature. Used for things like commands to a forwarder, or Interests that need authentication. |
| **CommandInterest** | An older specific kind of SignedInterest used to control NFD via `nfdc`. |
| **NFDC** | The CLI tool you use to talk to NFD: add faces, add routes, set strategies, dump tables. |
| **Face URI** | A string identifying a face: `udp4://198.51.100.1:6363`, `tcp4://...`, `unix:///run/nfd.sock`, `ether://...`, `dev://eth0`. |
| **DefaultRouteName** | `/`: the empty prefix. A FIB entry on `/` is a default route. |
| **Pull model** | The mode where consumers ask for data; producers respond. NDN's natural mode. |
| **Push model** | The mode where producers send data without being asked. NDN doesn't natively do this; sync protocols approximate it. |
| **PSync** | A specific NDN sync protocol using IBLTs (invertible Bloom lookup tables). |
| **ChronoSync** | An older NDN sync protocol; tracks per-producer sequence numbers. |
| **StateVectorSync** | Another NDN sync protocol using version vectors. |
| **CoLoC** | Concept of "co-located content" in some ICN designs. |
| **NRS / Name Resolution Service** | A service in MobilityFirst-style ICN that maps names to GUIDs/locators. NDN does *not* use a separate NRS — names *are* the routing keys. |
| **Pub/sub overlay** | A communication pattern. Most NDN sync protocols implement pub/sub semantics on top of NDN's pull model. |
| **Anycast (NDN-style)** | Built-in in NDN: any node can announce a prefix; the nearest copy answers. Same word as IP anycast, but applied to names. |
| **Multicast (NDN-style)** | Built-in in NDN: the PIT naturally aggregates duplicate Interests, so the Data flows back to multiple consumers without extra protocol. |
| **Federated namespace** | A namespace where different organizations control different prefixes — like DNS but for content names. |
| **Routable prefix** | A name prefix that has been announced to NDN's routing system (NLSR or static). Forwarders know how to reach it. |

There are more. We will introduce them as they show up. Refer to this table whenever something feels weird.

## Why NDN Exists

### The original sin: addresses, not content

When DARPA designed IP in the 70s, the question was: how do we get a packet from one *machine* to another *machine*? Vint Cerf and Bob Kahn wrote down a beautifully simple answer: every machine gets a number (an "address"), every packet has a source-address and a destination-address, and intermediate routers move packets toward their destinations using routing tables built by exchanging reachability info. This is the IP protocol you know.

It worked. It worked spectacularly. It scaled from four nodes to four billion. And then — somewhere around 2000 — something interesting started happening.

The traffic on the internet stopped being mostly "I want to talk to that specific computer." It started being mostly "I want a copy of that specific *thing*." A web page. A song. A movie. A software update. Increasingly: a video stream, a TikTok, a tweet, a GitHub repo, a Docker image. The thing the user wanted was *content*, not *a session with a specific machine*.

This was a problem because IP, the bottom of the stack, only knew how to ask "where is machine X?" To deliver content, applications had to bolt content-addressing on top:

- **DNS** lets you ask "where is `youtube.com`?" — but DNS only resolves to IP addresses, not to content.
- **HTTP** lets you ask "give me URL `/watch?v=foo`" — but HTTP needs a TCP connection to a specific machine, so you still need to pick *which* server to ask.
- **CDNs** put thousands of machines around the world that all answer for `youtube.com` — but DNS-and-IP-routing has to lie ("YouTube's IP" depends on where you are) to make it work.
- **BitTorrent** says "screw it, every viewer is also a server" — but it has to invent its own tracker overlays.
- **HTTP/2 multiplexing** says "let me at least re-use one TCP connection for many objects" — but that's fixing a TCP-era assumption that doesn't even apply if your protocol is content-centric.

Each of these is a clever workaround. None of them changes the fundamental fact that **the internet was designed to deliver packets to machines, and we are using it to deliver content to people**. There's an impedance mismatch.

### The Van Jacobson talk

In August 2006, Van Jacobson — yes, the guy who invented TCP congestion control, the guy on the "fathers of the modern internet" Mount Rushmore — gave a Google Tech Talk titled **"A New Way to Look at Networking"** that crystallized this argument. (Go watch it on YouTube. It's an hour long. It's the single best primer on why ICN exists.)

His thesis, paraphrased:

1. The first phone networks were about *who you wanted to talk to*: each call was a circuit between two named people.
2. Then we invented packet switching, and the internet was about *which machine you wanted to talk to*: each packet went to a numbered machine.
3. The next step is about *what content you want*: each packet should ask for a named piece of data.

He argued that:

- Caching is fundamental. The fact that we're hauling 80% of internet bytes across multi-hop paths when most of those bytes are *literally identical* to bytes already cached one hop away is insane.
- Multicast is fundamental. Lots of people want the same thing at the same time. A unicast-by-default network has to invent multicast as an afterthought, badly. A name-based network does multicast for free, by aggregating identical Interests.
- Mobility is fundamental. People walk around with computers in their pockets. An address-based network has to invent Mobile IP, which nobody uses. A name-based network doesn't care where you are.
- Security is fundamental. We secure connections (TLS) but should secure data (signatures on bytes). If the data is signed, you can store it anywhere, fetch it from anywhere, and still trust it.

He said all this at a Google Tech Talk in 2006. Then he went to PARC and started building it. The result was **CCN (Content-Centric Networking)**, which Cisco eventually picked up as **CCNx**. Around 2010 a research consortium spun off **NDN (Named Data Networking)** as a slightly different design — same big idea, different choices about names, signatures, and forwarding.

Today CCNx and NDN are sister projects. CCNx 1.0 went to the IRTF and became the basis of **RFC 8569 (CCNx semantics)** and **RFC 8609 (CCNx wire format)**. NDN has its own specifications hosted at named-data.net. Both implementations interoperate at the conceptual level even though their wire formats differ in details.

### What "redesign the bottom turtle" means

Here is the skeleton of today's IP stack:

```
Application       (HTTP, video, gRPC, etc.)
Transport         (TCP / UDP / QUIC)
Network           (IP)
Link              (Ethernet, Wi-Fi, etc.)
Physical          (copper, fiber, radio)
```

The Network layer asks: which **machine** owns this address? It's how we got from MIT to Stanford in the 70s.

NDN replaces the Network layer with a different question:

```
Application       (named-data apps: YouTube, but native NDN)
Transport         (sometimes empty — NDN handles it)
Network           (NDN: ask by name, get signed data)
Link              (Ethernet, Wi-Fi, NDN over UDP for transition)
Physical          (copper, fiber, radio)
```

That's it. That's the change. Everything above moves slightly to fit. Everything below is the same — NDN is happy to ride on Ethernet or Wi-Fi or 5G or fiber. Most modern NDN deployments run "NDN over UDP" because the world hasn't (yet) updated its Ethernet switches to speak NDN natively, and UDP-encapsulating NDN packets makes them compatible with today's IP-only links.

Or, in tabular form:

| Aspect | IP today | NDN |
| --- | --- | --- |
| Packet identifier | Source IP + dest IP | Name |
| Routing | By dest IP prefix | By name prefix |
| Trust unit | The connection (TLS) | The data (signature) |
| Caching | Bolted on (CDNs, browser cache) | Built-in (every router) |
| Multicast | Hard, separate protocol (PIM, IGMP) | Free, automatic |
| Mobility | Hard, Mobile IP (mostly unused) | Free for consumers; tractable for producers |
| Anycast | Hack via BGP | Free |

You can see why this is appealing in 2026: it directly maps to the workloads we actually have.

### Why we keep reinventing CDNs

A useful frame: every "we should put a cache there" idea on today's internet is, at heart, a small NDN. The history of the web is a slow re-invention of in-network caching:

- Browser caches (1993).
- HTTP proxy caches (Squid, 1996).
- ISP transparent proxies (late 90s, controversial).
- Akamai-style CDN edge servers (1998).
- BitTorrent peer-to-peer caching (2001).
- HTTP cache headers becoming a real thing (RFC 7234, 2014).
- Service workers in browsers (2014).
- Cloudflare global cache (2010s).
- HTTP-based "edge compute" (2018+).

Every one of these is application-layer machinery that exists because the network layer doesn't help with caching. NDN's pitch is: bake it into the network layer, lose the bolt-ons.

### Why NDN is research, not deployed

If NDN is so great, why aren't you running it right now?

Because: deploying a new network layer requires either replacing every router on Earth or running an overlay. NDN runs as an overlay (NFD-over-UDP) on the global testbed and inside research networks. Adoption in production is rare. Reasons:

- **Inertia.** IP works.
- **Identity-on-the-internet incentives.** Carriers, hyperscalers, and surveillance regimes are happy with address-centric.
- **Killer app problem.** No one application is so much better on NDN that it forces deployment.
- **Tooling gap.** Want a load balancer? An ACL? A monitoring tool? A DDoS scrubber? On IP they're commodity. On NDN you're rolling your own.
- **TCP congestion control taken for granted.** NDN re-opens these problems.

That said, NDN is *very* well suited to specific deployments — IoT mesh networks, scientific data dissemination (CMS at CERN, climate-model archives), in-vehicle networks, classroom video distribution. We'll cover those in the "When NDN is the right answer" section.

NDN is also being actively worked on as the basis for the IETF's ICNRG (Information-Centric Networking Research Group) drafts, NIST's FABRIC testbed, and the broader 6G / future-internet research community. So while it isn't carrying production internet traffic *today*, it is the leading clean-slate alternative architecture in the literature.

## The Three Tables

The heart of an NDN forwarder is three tables. Memorize their initials. They will be on the test.

- **CS** — Content Store. The cache of Data packets we've seen recently, indexed by Name.
- **PIT** — Pending Interest Table. The "I'm waiting on this" table of Interests we've forwarded upstream and are still expecting Data for.
- **FIB** — Forwarding Information Base. The "where do I forward Interests for this prefix?" table — a name-based routing table.

Here is the world's smallest ASCII diagram of an NDN forwarder:

```
          Interest ──────►┐
                          │
                          ▼
          ┌────────────────────────────┐
          │      NDN Forwarder         │
          │  ┌──────┐                  │
          │  │  CS  │  cache hit?      │──Data──► back to consumer
          │  └──────┘                  │
          │     │ miss                  │
          │     ▼                       │
          │  ┌──────┐                   │
          │  │ PIT  │  already pending? │──aggregate, no upstream
          │  └──────┘                   │
          │     │ no                    │
          │     ▼                       │
          │  ┌──────┐                   │
          │  │ FIB  │  pick face        │
          │  └──────┘                   │
          │     │                       │
          │     └─── Interest upstream──┼────►
          │                             │
          │  ◄──── Data downstream ─────┼─────
          │     │                       │
          │     ▼                       │
          │  insert into CS,            │
          │  consume PIT entries,       │
          │  forward to downstreams     │
          └────────────────────────────┘
```

In words:

1. An Interest arrives on some face.
2. The forwarder looks up the Interest's Name in the **CS**. If there's a hit, it ships the cached Data right back. Done. The Interest never goes upstream. The cache hit is the entire transaction.
3. If the CS misses, the forwarder looks up the Name in the **PIT**. If there's already a pending entry for this Name, it just records that this new face is also waiting (this is the *aggregation* trick — multiple consumers asking the same thing share one upstream Interest). The Interest still doesn't go upstream.
4. If the PIT also misses, the forwarder creates a new PIT entry, then looks up the Name's prefix in the **FIB** to decide which upstream face(s) to forward on. The Strategy is consulted to pick among multiple choices. The Interest is sent.
5. Eventually Data comes back on some upstream face. The forwarder:
   - Inserts the Data into the CS (so future requests can be served from cache).
   - Looks up the Name in the PIT to find which downstream faces were waiting.
   - Sends the Data to all of those waiting faces.
   - Removes the PIT entry.
6. If no Data ever comes back, the PIT entry expires after `InterestLifetime` and gets garbage-collected.

That's it. That's the whole forwarding plane. Three tables. Six bullet points.

### Content Store (CS)

The CS is a cache. It maps Name → Data. Forwarders insert into it whenever they see a Data packet, and look in it whenever they see an Interest.

CS lookup follows the Interest's matching rules:
- An Interest's Name matches a Data's Name if the Interest Name is a prefix of the Data Name (or equal). Selectors (deprecated in v0.3 spec) like `MustBeFresh`, `MaxSuffixComponents`, etc. further filter. Most modern apps just use exact Name match.
- If multiple Data in the CS match, one is picked deterministically (typically the most recent or the one with the longest matching suffix).

CS replacement policies (which entry to evict when full):
- **LRU** — Least Recently Used. Evict the entry that's been idle longest. Default in NFD.
- **LFU** — Least Frequently Used. Evict the entry hit fewest times.
- **FIFO** — First In First Out. Evict the oldest insert.
- **Priority** — App-defined.

Caching policies (which Data to insert in the first place — different from replacement):
- **LCE (Leave Copy Everywhere)** — every forwarder along the return path caches. This is the simplest, most aggressive, most popular default. Maximum cache redundancy.
- **LCD (Leave Copy Down)** — only the forwarder one hop closer to the consumer caches. The cache "moves down" toward the consumer over repeated requests. Less redundancy, more efficiency.
- **ProbCache** — probabilistic; each forwarder caches with probability `p` based on path position.
- **CL4M (Cache Less for More)** — only highly-central forwarders cache; spreads heat.
- **WAVE / Centrality / popularity-based** — cache only popular content; let cold content pass through.

NFD by default does LCE with LRU eviction. You can plug in custom policies in `nfd.conf`.

### Pending Interest Table (PIT)

The PIT is a register of Interests in flight. Each entry says: "I forwarded an Interest for Name N, on date D, on behalf of incoming faces F1, F2, F3, and I expect Data back any moment."

PIT entries expire after the Interest's `InterestLifetime` (default 4 seconds). On expiry the forwarder gives up — no Data response is generated to the consumer; the consumer's library notices the timeout and may retry.

The PIT is what gives you:

- **Aggregation:** if I have a PIT entry for `/foo/bar` from face A and I see another Interest for `/foo/bar` from face B, I just add face B to the existing PIT entry. I do not forward another Interest upstream. When Data eventually arrives, I send it to both A and B. **One upstream Interest serves multiple downstream consumers.**
- **State for the return path:** when Data arrives on the upstream, the PIT tells me which downstream faces wanted it. There's no "destination address" on the Data packet — there's no need for one — the PIT *is* the destination state.
- **Loop detection:** if I forward an Interest and it comes back to me with the same Name and Nonce, the PIT entry tells me "already saw this." I drop. The Nonce is what makes this work — without it, I'd happily forward the same Interest in circles.

### Forwarding Information Base (FIB)

The FIB is a name-prefix routing table. Each entry maps a prefix to a list of faces (with optional cost metrics).

Examples:

```
Prefix              Next-hop faces (cost)
─────────────────────────────────────────
/                   face=300 cost=0
/google             face=257 cost=10, face=258 cost=20
/google/maps        face=259 cost=5
/cmu/cs             face=260 cost=10
/uclalax/research   face=300 cost=50
```

When an Interest comes in for `/google/maps/v2/tile/123`, the forwarder does longest-prefix-match on the Name and finds `/google/maps` (longer match than `/google`, longer than `/`). It then asks the configured Strategy: among `face=259` and any other choices, which face do I send on?

FIB entries are populated in two ways:

- **Static** — operator runs `nfdc route add /prefix face_id cost=N`. Fine for small networks.
- **Dynamic via NLSR** — Named-data Link State Routing. Forwarders flood prefix announcements to neighbors, build a link-state database of the topology, and compute shortest paths to each prefix. Like OSPF for names.

There are also experimental BGP-style protocols for inter-domain NDN routing (NDN-BGP, hyperbolic routing in NDN, GREEDY routing). For now: NLSR within an island, static between islands.

### A note on PIT scaling

A criticism of NDN: the PIT must hold an entry for every in-flight Interest, and there are *a lot* of in-flight Interests at internet scale. Naive estimates put PIT size in the millions of entries on backbone forwarders. NFD on a laptop handles tens of thousands fine. Specialized forwarders (NDN-DPDK, the Cisco Vector PIT work) push this to millions. The general consensus in the literature is that PIT scaling is a real engineering problem but not a fundamental architectural blocker.

## How a Lookup Works

Time for a worked example. We'll trace one Interest from "Alice's laptop" to "Carol's media server" through two forwarders, and the Data back. Naming follows the convention `/<producer>/<topic>/<segment>`.

### Setup

- **Alice** runs an NDN consumer app on her laptop. She wants to watch a video.
- **Carol** runs an NDN producer app on a media server far away. She publishes the video at `/carol/movies/dune/seg42`.
- **Router R1** is Alice's home router, speaking NDN. It is connected to the WAN.
- **Router R2** is some intermediate ISP forwarder.
- **Router R3** is Carol's edge forwarder, connected to her producer.

Topology:

```
[Alice] ── face=A1 ──> [R1] ── face=R12 ──> [R2] ── face=R23 ──> [R3] ── face=R3C ──> [Carol]
            face=A2 <── R1 has FIB entry: /carol → face R12, cost 10
                         R2 has FIB entry: /carol → face R23, cost 10
                         R3 has FIB entry: /carol → face R3C, cost 5
```

All three forwarders start with empty CSes and PITs. All three have the FIB populated by NLSR. Time `t=0`.

### Step 1 — Alice emits an Interest

Alice's app calls `face.expressInterest(Interest("/carol/movies/dune/seg42"))`. The library wire-encodes a TLV packet:

```
Interest:
  Name: /carol/movies/dune/seg42
  Nonce: 0xF3A1B27E
  InterestLifetime: 4000 ms
  HopLimit: 32
```

The packet leaves Alice's laptop on her TCP/UDP face to R1. R1 receives it on face A1.

### Step 2 — R1 processes the Interest

```
R1: Interest arrived on face A1, name=/carol/movies/dune/seg42

  CS lookup → MISS (R1 has nothing cached yet).
  PIT lookup → MISS (no pending Interest for this name).
  PIT insert: name=/carol/movies/dune/seg42, in-faces={A1},
              nonce=0xF3A1B27E, expires at t+4s.
  FIB lookup → longest-prefix match on /carol → face=R12 (cost 10).
  Strategy: best-route → forward on face R12.
  HopLimit decrement: 32 → 31.
  Send Interest on face R12.
```

R1 forwards the Interest. Time so far: maybe 0.1 ms.

### Step 3 — R2 processes the Interest

Same dance:

```
R2: Interest arrived on face R12, name=/carol/movies/dune/seg42
  CS lookup → MISS.
  PIT lookup → MISS.
  PIT insert: name=/carol/movies/dune/seg42, in-faces={R12},
              nonce=0xF3A1B27E, expires at t+4s.
  FIB lookup → /carol → face=R23 (cost 10).
  Strategy: best-route.
  HopLimit decrement: 31 → 30.
  Send Interest on face R23.
```

### Step 4 — R3 processes the Interest

Again:

```
R3: Interest arrived on face R23, name=/carol/movies/dune/seg42
  CS lookup → MISS.
  PIT lookup → MISS.
  PIT insert: in-faces={R23}.
  FIB lookup → /carol → face=R3C.
  HopLimit decrement: 30 → 29.
  Send Interest on face R3C.
```

### Step 5 — Carol's producer responds

Carol's producer app has a handler registered for prefix `/carol/movies/dune`. The Interest matches. Carol's app produces a Data packet:

```
Data:
  Name: /carol/movies/dune/seg42
  MetaInfo:
    ContentType: BLOB
    FreshnessPeriod: 30000 ms
  Content: <188-byte MPEG-TS chunk of dune segment 42>
  SignatureInfo:
    SignatureType: SHA-256-with-ECDSA
    KeyLocator: /carol/keys/movies/2026
  SignatureValue: <64 bytes of ECDSA over Name+MetaInfo+Content+SignatureInfo>
```

Carol's app calls `face.put(data)`. The Data flows back on face R3C.

### Step 6 — R3 processes the Data

```
R3: Data arrived on face R3C, name=/carol/movies/dune/seg42
  PIT lookup → HIT, in-faces={R23}, expires at t+4s.
  CS insert: cache the Data, indexed by name. (LCE.)
  Send Data on each in-face: R23.
  PIT entry: removed.
```

### Step 7 — R2 processes the Data

```
R2: Data arrived on face R23, name=/carol/movies/dune/seg42
  PIT lookup → HIT, in-faces={R12}.
  CS insert: cached.
  Send Data on R12.
  PIT entry: removed.
```

### Step 8 — R1 processes the Data

```
R1: Data arrived on face R12, name=/carol/movies/dune/seg42
  PIT lookup → HIT, in-faces={A1}.
  CS insert: cached.
  Send Data on A1.
  PIT entry: removed.
```

### Step 9 — Alice receives the Data

Alice's app's callback fires with the Data. The library:

- Verifies the signature using the KeyLocator-pointed certificate (which it may need to fetch separately as a Data packet — same Interest/Data flow).
- Hands the Content to the consumer app.

Alice plays the dune chunk. Total path RTT: roughly the same as TCP would have been on the same path.

### Step 10 — Aggregation kicks in

Now Bob, who lives in the same house as Alice and connects to R1, asks for the same Interest. Time is `t+0.5s`:

```
R1: Interest arrived on face B1 (Bob's face), name=/carol/movies/dune/seg42
  CS lookup → HIT! (R1 cached the Data 0.5s ago.)
  Send Data on B1 directly. Never forwards upstream.
```

Bob's request is satisfied entirely from R1's cache. Carol's server has no idea Bob exists. R2 and R3 have no idea Bob exists. The neighbourhood ISP carried *zero* extra bytes for Bob's request.

This is **the moment NDN earns its keep**. On today's IP internet, Bob's player would have opened a fresh TCP connection to a Netflix CDN edge, fetched the chunk, and re-traversed all that path. On NDN: cache hit on the home router. The same trick happens for the next million viewers in the same neighbourhood, the same ISP, the same continent — wherever the cache hierarchy is.

### Multicast for free

Now imagine Bob and Carol's-neighbour Dave both ask at the *exact same time* `t=0` (Alice already gone). The Interests arrive at R1 within microseconds of each other:

```
R1: Interest from B1 → MISS in CS, MISS in PIT.
    PIT insert, in-faces={B1}, forward upstream on R12.
R1: Interest from D1 → MISS in CS, HIT in PIT.
    Add D1 to existing PIT in-faces. NO upstream forward.
```

Only one Interest goes from R1 toward R2. When the Data eventually returns, R1 ships it to *both* B1 and D1. From Carol's perspective, only one viewer asked. The network just did one-to-many distribution without any explicit multicast protocol.

This is the natural multicast property of NDN. It is one of the most-cited reasons to like the architecture.

### A second worked example — three consumers, two paths

Let's redo the earlier flow but with two redundant Producers behind the same name and three consumers along different downstream paths. This is where Strategy choices actually matter.

Topology:

```
[Consumer1]──┐
[Consumer2]──┤── face=A1, A2, A3 ──>[R1]── face=R12 ──>[R2]── face=R23a ──>[R3a]──>[ProducerA]
[Consumer3]──┘                                │                  
                                              │── face=R23b ──>[R3b]──>[ProducerB]
```

R2 has FIB:

```
/carol/movies → [R23a cost 10, R23b cost 20]
```

Now Consumer1 emits Interest `/carol/movies/dune/seg99`.

Case 1: Strategy = best-route at R2.
1. R2 forwards to R23a only (lowest cost).
2. If R3a returns Data within InterestLifetime, Data flows back. Done.
3. If R3a is dead, R2's strategy times out and re-issues on R23b.

Case 2: Strategy = multicast at R2.
1. R2 forwards on **both** R23a and R23b in parallel.
2. First Data response wins. Both R3a and R3b might respond if both have the chunk; R2 takes the first one and drops the second.
3. Faster failover at the cost of double upstream bandwidth.

Case 3: Strategy = ASF.
1. R2 maintains SRTT for both R23a and R23b.
2. New Interest goes to whichever face has lower current SRTT.
3. Periodic probes keep the alternative face's SRTT fresh.
4. If the chosen face has been silent: penalty, switch.

If you change the strategy from best-route to multicast, you trade upstream bandwidth for redundancy. If you change to ASF, you trade tracking-state-complexity for adaptivity. None of the strategy choice changes the *naming*, the *trust*, or the *caching* — just the upstream forwarding policy.

### Yet another walked-through case — a nasty cache poisoning attempt

Now consider an adversary on the path. Let's say a forwarder R-bad wants to inject fake content for `/alice/photos/badcake`. R-bad receives the legitimate Interest from a downstream consumer. Instead of forwarding upstream and waiting for Alice's signed Data, R-bad fabricates a Data:

```
Data:
  Name: /alice/photos/badcake
  Content: <bogus bytes>
  SignatureValue: <random bytes>
  KeyLocator: /alice/keys/photos/2026   (cargo-cult)
```

R-bad sends this to the consumer. Consumer's library:

1. Receives the Data.
2. Looks up the KeyLocator (`/alice/keys/photos/2026`) — which is itself an NDN name. To verify the signature, it issues an Interest for that cert.
3. Either fetches Alice's real cert (which means the *cert* is genuine) and tries to verify the bogus signature *against the real key* — which fails because the bytes are random.
4. Or R-bad spoofs the cert too — but the cert chain leads upward to a trust anchor the consumer has pinned. Eventually the chain fails to validate against the trust anchor.
5. Validator returns `ValidationError`. The Data is dropped. The consumer's app sees a fetch failure, retries — and hopefully the next path bypasses R-bad.

The architectural property: **a malicious forwarder cannot inject undetected bad data, *if* consumers verify signatures**. The cost is per-Data signature validation in the consumer. Apps that skip verification (some throughput-prioritized apps do) lose this guarantee.

### Yet another walked-through case — Interest aggregation under load

A flash crowd: 10,000 consumers in 10ms ask for `/news/breaking/seg=42`. They funnel through the same ISP edge router R1 (face A1...A10000). Watch what happens:

```
t=0     R1 receives Interest #1 from face A1.
        CS miss. PIT miss. Forward upstream on R12.
        PIT entry: in-faces={A1}.
t=1ms   R1 receives Interest #2 from face A2 (same Name).
        CS miss. PIT HIT — entry exists.
        Append A2 to PIT in-faces. Do not forward upstream.
t=2ms   R1 receives Interest #3 from face A3 (same Name).
        Same: append A3 to PIT.
   ...
t=10ms  R1 has accumulated 10000 in-faces in one PIT entry.
        Exactly one Interest went upstream.

t=20ms  R1 receives Data on R12 from upstream.
        PIT lookup: 10000 in-faces.
        Send Data to all 10000 of them.
        Insert into CS.
        PIT entry: removed.
```

The upstream link carried *one* Interest and *one* Data. The downstream links carried 10000 Data copies. This is the multicast effect of aggregation: incoming load is naturally fanned out.

This works even if consumers send Interests microseconds apart, with different Nonces, from different faces — as long as the Names match and the existing PIT entry is still pending, aggregation kicks in.

If R1's PIT had been full (DOS limits, memory pressure), the later Interests would be dropped. PIT capacity is one of the engineering constraints to size carefully.

## Names Are Hierarchical

NDN names look like file paths. They are hierarchical, slash-separated, human-readable (or machine-encoded — your choice), and they are the *only* thing that identifies a piece of data.

### Anatomy of a Name

```
/google/maps/v2/tile/zoom8/x123/y456
└──┬──┘└──┬─┘└──┬──┘└──┬──┘└──┬──┘└──┬──┘└──┬──┘
   1     2     3      4      5      6     7
```

That name has 7 components. Each component is one slash-separated piece. Components can be ASCII strings (`google`), URL-encoded strings (`tile%2Fzoom8`), or arbitrary binary blobs encoded as TLV.

Names are hierarchical the way DNS is hierarchical: by convention, the leftmost component is the most-administratively-scoped (the "TLD" or organization), and you go deeper rightward. A natural read: "google's maps service, version 2, tile API, zoom level 8, x=123, y=456."

But unlike DNS, NDN names are **not just human-readable strings**. They are TLV-encoded byte sequences. A component can contain binary data — a sequence number encoded as a 4-byte big-endian integer, a SHA-256 hash, a timestamp. The name is a sequence of typed components.

### Name encoding (TLV)

Every component has a type. The naming spec defines several:

| Type code | Name | Purpose |
| --- | --- | --- |
| 0x08 | GenericNameComponent | Default. Arbitrary bytes — typically UTF-8 string. |
| 0x01 | ImplicitSha256DigestComponent | SHA-256 of a Data packet. Used to point at *exactly* this Data. |
| 0x02 | ParametersSha256DigestComponent | SHA-256 of an Interest's parameters. Used in command Interests. |
| 0x21 | KeywordNameComponent | Reserved keywords (rarely used). |
| 0x32 | SegmentNameComponent | A segment number (specifically encoded). |
| 0x33 | ByteOffsetNameComponent | A byte offset (used for partial fetch). |
| 0x34 | VersionNameComponent | A version number. |
| 0x35 | TimestampNameComponent | A timestamp. |
| 0x36 | SequenceNumNameComponent | A sequence number. |

So the human-printable name `/foo/bar/v=42/seg=7` might encode as:

```
TLV: name
  TLV: GenericNameComponent "foo"   (type=0x08, length=3, value="foo")
  TLV: GenericNameComponent "bar"   (type=0x08, length=3, value="bar")
  TLV: VersionNameComponent  42     (type=0x34, length=1, value=0x2a)
  TLV: SegmentNameComponent  7      (type=0x32, length=1, value=0x07)
```

The text representation we type into commands and logs is a URI-style flattening:

```
/foo/bar/v=42/seg=7
   or
/foo/bar/54%3D42/50%3D07
```

Different tools render the typed components differently. NFD logs use the human-friendly form for known types and percent-encoding for unknown.

### Routable prefix vs application name

Not every name component is something a forwarder needs to know about. The convention is:

- The **routable prefix** is the leftmost few components. This is what FIB entries are keyed on. It's also what NLSR floods. It's roughly equivalent to a domain in DNS — every organization owns a prefix.
- The **application-specific suffix** is everything to the right of the routable prefix. The forwarder doesn't need to understand its structure; it just forwards toward the prefix.

Example:

```
Routable prefix    Application suffix
/google             /maps/v2/tile/zoom8/x123/y456
/cmu/cs             /research/papers/2024/foo.pdf
/youtube            /watch/dQw4w9WgXcQ/manifest
```

Carol's media server announces the prefix `/carol/movies` to NLSR. Routers in the testbed flood that announcement and end up with FIB entries for `/carol/movies → some_face`. Whatever's under `/carol/movies` is Carol's business; the network just routes Interests with that prefix toward Carol.

This is exactly like routing on IP prefixes — `8.8.8.0/24` is announced to BGP, and routers don't care about specific addresses inside that prefix until packets land on the right link.

### Versioning, segmentation, sequence numbers

A real application doesn't fetch one big object — it fetches many small ones, often in sequence, often versioned. NDN bakes this in:

- **Versioning:** name components like `v=NNN` represent versions. A producer publishes a new version by signing data under a name with a higher version. Consumers can ask for the latest version using the `RightmostChild` selector (deprecated) or by issuing an Interest for the prefix and letting the producer return the newest.
- **Segmentation:** a video, file, or large object gets chunked. Each chunk gets a name suffix `seg=N`. Consumers fetch them in order, possibly with a pipelining client like `ndncatchunks` or `ndn6-file-server`.
- **Sequence numbers:** a streaming app uses sequence numbers to identify each frame or update.

The convention for stream-like data:

```
/alice/sensor/temperature/v=2025-04-27T08:00:00/seg=0
/alice/sensor/temperature/v=2025-04-27T08:00:00/seg=1
/alice/sensor/temperature/v=2025-04-27T08:00:00/seg=2
   ...
```

A consumer interested in "the latest temperature" first asks the producer for the latest version (using the producer's catalog or by exclusion-based discovery), then segment-by-segment fetches the data.

### Implicit digest names

The implicit digest component lets you ask for "exactly this Data, no other." It's the SHA-256 of the full TLV-encoded Data packet, attached as a component:

```
/alice/sensor/temperature/v=42/seg=7/<sha256:abcd...1234>
```

If your Interest names this exact digest, only the byte-for-byte identical Data can satisfy it. Useful for self-certifying content addressing — basically the IPFS trick — without putting the hash front-and-center in the name.

### Naming conventions in the wild

Various NDN apps and testbed deployments have evolved naming conventions:

- **NDN Common Name Library** standardizes typed name components.
- **NDN Real Time Conferencing (NDN-RTC)** uses names like `/ndn/edu/<institution>/<user>/conference/<conference-id>/<media>/<frame>`.
- **NDNFS / repo-ng** uses `/repo/<file-id>/<segment>` or `/<owner>/<filesystem>/<path>/<segment>`.
- **CMS at CERN (NDN deployment)** uses `/ndn/eu/cern/cms/dataset/<dataset-id>/<file-id>/<segment>`.
- **NDN testbed root names** are `/ndn/<country>/<organization>/...`.

The big lesson: name design is *application design*. You think hard about your namespace upfront, the way HTTP API designers think hard about URL paths upfront. Bad name design hurts you in caching, security, and routing.

## Security: Sign the Data, Not the Channel

This is the section where NDN really diverges from today's TLS-everywhere mental model. Get a coffee. We'll go slow.

### TLS today: secure the pipe

When you visit `https://gmail.com` your browser:

1. Resolves `gmail.com` via DNS to an IP address.
2. Opens a TCP connection to that IP.
3. Performs a TLS handshake. Both sides authenticate using certificates (signed by a CA you trust).
4. Establishes encrypted symmetric session keys.
5. Sends HTTP requests through the encrypted tunnel.
6. Receives HTTP responses through the same tunnel.

The trust model: **the channel** is secured. You verified the server's certificate. Bytes that travel through this specific TCP connection from this specific server are trusted. If you cached a bytewise-identical copy of those bytes somewhere else, you'd have no way to know it's authentic — there's no signature on the bytes themselves; the signature was on the channel handshake.

This works for "I'm interacting with a specific service in real time." It's *terrible* for in-network caching. A cached HTTP response in your ISP's transparent proxy has no signature you can verify; the proxy could lie. CDNs solve this by putting a proxy under the *origin's* TLS umbrella (sharing keys, or terminating TLS at the edge with the origin's cert) — which works but couples cache deployment to credential management in painful ways.

### NDN's choice: secure the data

In NDN, **every Data packet is signed by its producer**. The signature covers the Name, the MetaInfo, the Content, and the SignatureInfo. The signature is verified by the consumer using the producer's public key.

Because the Data carries its signature, **it doesn't matter who hands it to you**. A forwarder, a USB drive, a tape backup, an email attachment — if the bytes are intact and the signature verifies against the producer's key, the Data is trustworthy.

This is the ICN / NDN security mantra: **secure the data, not the channel**.

### Anatomy of a signed Data packet

```
Data packet (TLV):
  Name:             /alice/photos/birthday/seg42
  MetaInfo:
    ContentType:    BLOB
    FreshnessPeriod: 60000ms
  Content:          <bytes>
  SignatureInfo:
    SignatureType:  SHA-256-with-ECDSA       (type code 0x03)
    KeyLocator:     /alice/keys/photos/2026  (Name pointing at her cert)
    ValidityPeriod: NotBefore=...t, NotAfter=...t
  SignatureValue:   <64 bytes ECDSA over Name||MetaInfo||Content||SignatureInfo>
```

The SignatureValue is computed over the canonical TLV bytes of Name, MetaInfo, Content, and SignatureInfo (in that order, sans the SignatureValue itself). The Consumer verifies it using the public key indicated by KeyLocator.

### KeyLocator and certificate fetching

The KeyLocator is itself an NDN Name. To verify the signature, the Consumer fetches the certificate at that name — which is just another Data packet (a certificate is a self-describing Data packet). That certificate may itself be signed by some authority's key, whose certificate is at *its* KeyLocator. You walk the chain until you hit a name you have already established trust in (a root key in your trust anchors).

The whole certificate chain fetch is just NDN Interests/Data flowing through the same forwarders. There is no special "TLS handshake." Authentication and data delivery use the same primitives.

### Trust schemas

A trust schema is a small policy that says: "the data under prefix `/alice/photos/...` must be signed by a key under prefix `/alice/keys/photos/...`, and that key must be signed by `/alice/keys/master`." Or: "the data under `/cmu/cs/students/<name>/grades/...` must be signed by a key under `/cmu/cs/registrar/...`."

These schemas are typically machine-checked by NDN libraries during signature verification. They give you fine-grained control over who can sign for which names. Trust schemas can be expressed in:

- **Hierarchical** form (most natural): name structure implies who can sign. `/cmu/cs/<x>` is signed by `/cmu/cs/admin`, etc.
- **Validator config** files: explicit rule lists matching name patterns to required signer name patterns.
- **NDNCERT 0.3 / NDN-Schematized Trust** policies.

Compared to CA-and-leaf-cert TLS, NDN trust schemas are dramatically more flexible — you can say "homework submissions must be signed by the student" and the network enforces it at the data layer.

### NDNCERT — issuing certs over NDN

NDN doesn't use X.509 directly. It has its own minimalistic certificate format (a Data packet whose Content is a public key with a few metadata fields). NDNCERT is the protocol that runs the Certificate Authority dance: a new identity makes Interests to a CA's prefix, gets challenged (e.g., by email), proves identity, and receives a signed certificate Data packet.

NDNCERT is itself an NDN application. The CA is just a Producer at a well-known prefix, like `/ndn/edu/cmu/cert-authority`.

### Encryption is separate from authentication

Signature ≠ confidentiality. Data is signed and visible to anyone who has the bytes. If you want confidentiality, you encrypt the Content portion separately, often using **NAC (Name-Based Access Control)**: a Producer encrypts content under a content key, encrypts the content key under per-consumer keys, and lets each consumer fetch the encrypted content key for their identity.

NAC is more complex than TLS encryption because it operates per-data-object rather than per-channel. The trade-off: encrypted-and-cacheable. Any forwarder can cache the ciphertext; only authorized consumers can decrypt.

### Cache poisoning concerns

A malicious or buggy forwarder could *try* to inject fake Data into its CS. Defenses:

- **Signatures.** A receiver always verifies signatures (or chooses to skip if it doesn't care). Fake Data fails signature verification and is dropped.
- **Trust schemas.** Even a valid signature from the wrong key fails the schema check.
- **Implicit digest names.** If you ask by SHA-256 digest, only the exact bytes can match.

Not all NDN apps verify signatures on every Data — ones that don't are vulnerable to in-cache poisoning. The convention is "verify what you trust."

### Walking a real cert chain

Here's what happens when a consumer fetches `/alice/photos/birthday/seg0` and validates it from a cold start.

1. Consumer issues Interest for `/alice/photos/birthday/seg0`.
2. Some forwarder satisfies it, returning Data signed with KeyLocator `/alice/keys/photos/2026`.
3. Consumer's validator: I don't have that cert cached. Issue Interest for `/alice/keys/photos/2026`.
4. Network returns Data — Alice's photos-key certificate. Its content is the public key. Its KeyLocator is `/alice/keys/master`.
5. Consumer's validator: I don't have *that* cert either. Issue Interest for `/alice/keys/master`.
6. Network returns Data — Alice's master cert. Its content is her master public key. Its KeyLocator is `/ndn/edu/somewhere/anchor` (the trust anchor).
7. Consumer's validator: I have `/ndn/edu/somewhere/anchor` pinned as a trust anchor. Stop fetching.
8. Validator now walks the chain *back down*: verify `/alice/keys/master` was signed by anchor; verify `/alice/keys/photos/2026` was signed by master; verify the original Data was signed by `/alice/keys/photos/2026`.
9. All three checks pass. Validator returns success. App receives Data.

If any check failed, validator returns ValidationError; app retries on a different face or gives up. Note: the cert-fetching Interests pass through the same forwarders, hitting their CSes; certs are highly cacheable. After the first fetch, subsequent consumers get the cert chain from cache without a round trip.

### Pinning and trust anchors

You configure trust anchors in `~/.ndn/config` or app-specific config:

```
trust-anchor
{
  type file
  file-name "/etc/ndn/trust-anchors/ndn-testbed.cert"
}
```

The file is the anchor's certificate (a binary TLV-encoded Data packet). All validations must root at one of your trust anchors.

You can also configure named anchors that are themselves NDN-fetched, allowing dynamic trust hierarchies.

### Validator config example

A validator config for "Data under /alice/photos must be signed by something under /alice/keys":

```ini
rule
{
  id "alice-photos-signed-by-alice"
  for data
  filter
  {
    type name
    regex ^/alice/photos/<>*$
  }
  checker
  {
    type customized
    sig-type ecdsa-sha256
    key-locator
    {
      type name
      regex ^/alice/keys/<>*$
    }
  }
}

trust-anchor
{
  type file
  file-name "alice-master.cert"
}
```

This is the schema-as-config style. Schemas can also be encoded in NDN-Schema language or generated from trust models. Either way, the validator enforces them automatically.

### Compromised key recovery

What happens if Alice's photos-key gets stolen?

1. Alice generates a new photos-key, signs a new cert under master.
2. She publishes the new cert under a new versioned name `/alice/keys/photos/2027`.
3. Future Data is signed with the new key, with KeyLocator `/alice/keys/photos/2027`.
4. Old Data signed with the compromised 2026 key remains "valid" — the network can't retract Data.
5. Alice can issue a revocation: a new Data packet under `/alice/revocations/keys/photos/2026` saying "this key is revoked as of timestamp T."
6. Validators that consult revocations will reject Data signed with revoked keys.

Revocation in NDN is conceptually the same as in PKI today: a separate channel, eventually-consistent, with all the same caveats.

### Comparison table

| Property | TLS-on-channel (today) | Sign-the-data (NDN) |
| --- | --- | --- |
| Authenticated by | Server cert during handshake | Per-Data signature |
| Trust unit | Connection | Data packet |
| Cacheable while signed | No (cache breaks trust) | Yes (signature travels with bytes) |
| Replay protection | Built into TLS | Per-app (Names are versioned/timestamped) |
| Confidentiality | Encrypted channel | Per-data encryption (NAC) |
| Identity | Single CA hierarchy (X.509) | Hierarchical NDN names + trust schemas |
| Forward secrecy | Built in (modern TLS) | App-level (less standard) |

## Caching Everywhere

Caching is so central to NDN that we already covered it briefly. Now let's get specific.

### What gets cached

Every Data packet that flows through a forwarder is a candidate. The forwarder asks two questions:

1. Does the producer want this cached? (FreshnessPeriod tells you for how long; some producers set FreshnessPeriod=0 to discourage caching.)
2. Does our caching policy say to cache it here? (LCE = always; LCD = only at the next hop down; ProbCache = with some probability.)

If yes to both, the forwarder inserts into its CS and may evict an older entry.

### CS structure

Conceptually a CS is a hash map: Name → Data (with metadata: insert time, hit count, expiry, etc.). NFD's CS is a name-prefix-tree-with-LRU implementation. NDN-DPDK uses a hash table with collision handling tuned for line-rate.

Lookup is exact-name-or-prefix, depending on the matching rules of the Interest. For most apps it's exact-name.

### Replacement policies

Plenty of academic work explores what to evict:

- **LRU** — evict least recently used. Cheap, predictable. Default.
- **LFU** — evict least frequently used. Better for stable popularity distributions.
- **2Q / ARC** — multi-segment LRU variants with frequency awareness.
- **LIRS** — locality-aware.
- **LFRU** — combined frequency + recency.

NFD lets you plug in your own replacement strategy via the C++ `cs::Policy` interface.

### Cache placement policies

Different from replacement: when Data flows through, *which* forwarders store it?

- **LCE (Leave Copy Everywhere)** — every router on the return path caches. Simple, redundant. Most popular cache placement choice.
- **LCD (Leave Copy Down)** — only the router one hop closer to the consumer caches. Over repeated requests, the cache "moves" toward the consumer, eventually settling at the network edge near demand.
- **LCP (Leave Copy at Producer's hop)** — only cache near the producer.
- **ProbCache** — cache with probability `p ∝ remaining_path/total_path`. Spreads copies along the path.
- **CL4M (Cache Less for More)** — based on betweenness centrality; only highly-connected routers cache.
- **WAVE** — popularity-driven; cache when the consumer-side request count exceeds a threshold.
- **Optimal Content Placement** — solve an LP problem with predicted demand. Theoretical bound.
- **Coding-aware caching** — store coded combinations of Data, recover by combining cache entries.

In practice: most production-ish deployments use LCE+LRU and call it a day. Researchers test their alternatives in ndnSIM.

### Cache size & eviction in NFD

NFD configures CS via `nfd.conf`:

```
cs_max_packets 65536    ; how many Data packets to keep
cs_policy lru           ; replacement policy: lru | priority_fifo
cs_unsolicited_policy drop-all  ; what to do with unsolicited Data
```

Set `cs_max_packets` to 0 to disable caching (a forwarder with no cache — useful for some edge boxes).

### Cache-related selectors and flags

- **MustBeFresh** — Interest flag. When set, cached Data within FreshnessPeriod is OK; Data past its FreshnessPeriod is *not* returned (the Interest goes upstream instead). When unset, stale Data is fine. Default in many apps: unset (let me have stale).
- **FreshnessPeriod** — Data field. The producer's hint about how long this Data is "fresh." After expiry, MustBeFresh Interests skip it. Note: **non-MustBeFresh Interests can still be served by it**. The Data isn't deleted from the cache by hitting its FreshnessPeriod; it just becomes ineligible for fresh-only requesters.
- **CanBePrefix** — Interest flag. When set, Data whose Name is *longer than* the Interest Name (i.e., the Interest's Name is a prefix of the Data's Name) can satisfy. Used for catalog discovery — "give me anything under `/alice/photos`."

### Cache hit example

Continue the Alice/Bob/Carol example. After Alice's request:

```
$ nfdc cs info
Content Store information:
  Capacity: 65536
  Admit: on
  Serve: on
  N entries: 1
  N hits: 0
  N misses: 4
```

After Bob's request (hits R1's cache):

```
$ nfdc cs info
  N entries: 1
  N hits: 1
  N misses: 4
```

The hit count went up. R1 served Bob from cache.

You inspect the actual entries:

```
$ nfdc cs erase /carol/movies/dune     # clear them all
Cache erased: 1 entries removed for prefix /carol/movies/dune
```

### When caching doesn't help

NDN's cache is only useful for **immutable, repeatedly-requested content**. It does little for:

- One-shot dynamic responses (e.g., a custom search query result).
- High-entropy or session-specific data.
- Encrypted-per-consumer content (the cache stores ciphertext, but each consumer gets a different ciphertext — the ciphertext is unique).

For those workloads, NDN behaves like an IP unicast network with extra protocol overhead. Caching brings real wins only for the multi-consumer-same-content case — which is, conveniently, the bulk of internet traffic.

## Forwarding Strategies

A Strategy is a plug-in policy that decides how the forwarder treats Interests at the FIB-and-face level. NDN ships several. You can write your own.

The FIB tells you "for this prefix, here are the candidate next-hop faces." The Strategy tells you which face(s) to use, with what timing, and how to react to Data and to NACKs (negative responses).

### Built-in strategies in NFD

| Strategy | What it does |
| --- | --- |
| **best-route** | Send to the lowest-cost FIB next-hop. Wait for Data. If timeout, retry on the next-best face. Default for most prefixes. |
| **multicast** | Forward the Interest on **all** FIB next-hops simultaneously. First Data wins. Useful for redundant replicas. |
| **access** | Aggressive caching + multicast for short-lifetime content access scenarios. |
| **asf (Adaptive SRTT-based Forwarding)** | Maintains a smoothed RTT estimate per face and prefers the lowest-RTT face. Falls back on timeout. Good for adapting to congestion. |
| **ncc (NCC strategy)** | The original CCN strategy. Uses prediction tables. Mostly historical. |
| **self-learning** | Learns paths by flooding the first Interest, observing which face Data comes back on, and remembering it. Used in ad-hoc / unknown-topology networks. |

You set the strategy per-prefix:

```bash
$ nfdc strategy set /carol/movies /localhost/nfd/strategy/multicast
strategy-set prefix=/carol/movies strategy=/localhost/nfd/strategy/multicast/v=4
```

### Best-route strategy in detail

For an Interest matching prefix P with FIB next-hops `[face1 cost=10, face2 cost=20, face3 cost=30]`:

1. Send on face1 (lowest cost).
2. If Data arrives within `InterestLifetime`, deliver to consumer. Strategy done.
3. If timeout (`InterestLifetime` elapses, or a NACK on face1 arrives), retry on face2.
4. If face2 also fails, try face3.
5. If all fail, propagate NACK to consumer (or just let it time out).

Best-route is what you want for "I have a primary path and a few backups." It's the most common default.

### Multicast strategy in detail

For the same FIB:

1. Send the Interest on **all** of face1, face2, face3 in parallel.
2. The PIT entry tracks which face it came from downstream and how many copies it sent upstream.
3. The first Data response that arrives is delivered to the consumer.
4. Subsequent (duplicate) Data responses on other faces are dropped.

Multicast is great for:

- **Redundant data sources.** Several caches all have the same content; race them.
- **Multi-homed consumers.** You want the fastest answer.
- **Fan-out producers.** Everyone in a multicast group should receive Interests.

The cost is bandwidth: every Interest is replicated. Use only where bandwidth is cheap or correctness depends on it.

### ASF strategy in detail

ASF (Adaptive SRTT Forwarding) is the "smart" default for cases where you want dynamic adaptation:

1. Each face maintains a smoothed RTT (SRTT), updated after each Data response.
2. New Interest: pick the face with lowest current SRTT.
3. Periodically, ASF probes other faces by sending a small fraction of Interests on each non-best face. This keeps the SRTT estimate fresh and detects when a previously-bad face has recovered.
4. On timeouts/NACKs: penalize the face's SRTT and retry on next-best.

ASF is genuinely useful in research deployments with multiple ISP uplinks where load conditions change.

### Self-learning strategy in detail

For an ad-hoc network where you don't have routing protocols, self-learning is your friend:

1. First Interest with no FIB entry: forwarder broadcasts on all faces (except incoming).
2. The Interest reaches a producer somewhere; Data flows back; on each hop the forwarder remembers which face the Data came in on.
3. Subsequent Interests for the same prefix get unicast on the learned face.
4. If the learned face stops returning Data: forget and re-broadcast.

Used in vehicular networks, mobile mesh, IoT. Doesn't scale to global routing but excellent for small-scale dynamic topologies.

### Strategy interface in NFD

Writing your own strategy means subclassing `nfd::fw::Strategy`:

```cpp
class MyStrategy : public Strategy {
public:
  MyStrategy(Forwarder& f, const Name& name) : Strategy(f) { ... }

  void afterReceiveInterest(const FaceEndpoint& ingress, const Interest& interest,
                            const shared_ptr<pit::Entry>& pitEntry) override;

  void beforeSatisfyInterest(const Data& data, const FaceEndpoint& ingress,
                             const shared_ptr<pit::Entry>& pitEntry) override;

  void afterReceiveNack(const FaceEndpoint& ingress, const lp::Nack& nack,
                        const shared_ptr<pit::Entry>& pitEntry) override;

  static const Name& getStrategyName();
};

NFD_REGISTER_STRATEGY(MyStrategy);
```

You override the three callbacks and decide what to send where. Compile, link as a shared library, point NFD at it via `nfd.conf`. Use `nfdc strategy set /myprefix /your/strategy/v=1`.

### Strategy NACKs

When something goes wrong, forwarders can issue **NACKs** (negative acknowledgments) instead of letting Interests time out silently. NACK reasons include:

- **Congestion** — downstream is queue-overloaded.
- **Duplicate** — the same Nonce was seen (loop).
- **NoRoute** — no FIB entry exists for this prefix.

Strategies decide what to do with received NACKs. Best-route retries on the next-best face. Multicast accumulates and forwards a NACK only if all faces NACKed. Custom strategies can implement any policy.

NACKs cost some bytes but provide much better failure semantics than silent timeouts.

### Per-prefix strategies

Different prefixes can have different strategies. This is configured by `nfdc strategy set`. Typical pattern:

```bash
nfdc strategy set /     /localhost/nfd/strategy/best-route
nfdc strategy set /ndn/multicast /localhost/nfd/strategy/multicast
nfdc strategy set /critical /localhost/nfd/strategy/asf
```

### Strategy = the closest IP analogue

If you've worked with IP, the rough analogues:

| IP concept | NDN concept |
| --- | --- |
| Routing protocol decisions | NLSR (populates FIB) |
| Per-prefix routing policy / route maps | Strategy |
| ECMP across multiple next-hops | Multicast strategy (all-paths) |
| Active-passive failover | Best-route strategy |
| IP TTL | HopLimit on Interest |

Strategies are pluggable code modules. Writing one means subclassing `nfd::fw::Strategy` in C++ (or implementing the equivalent interface in NDN-DPDK).

## Producer/Consumer Patterns

NDN is fundamentally a **pull** model. Consumers send Interests; Producers (or caches) reply with Data. Nothing in the architecture lets a Producer push unsolicited Data — the network has no way to deliver Data without a matching PIT entry. (A Data without a PIT entry on the receiving forwarder is "unsolicited" and dropped.)

But applications often need push-like semantics: pub/sub, real-time streams, notifications. NDN gets there through **sync protocols** — clever conventions where consumers periodically pull a "manifest" or "state vector" and notice when something new exists.

### Naive periodic polling

The simplest approach: every consumer asks every N milliseconds "is there new content under `/alice/news`?" Producer answers when there is; otherwise the Interest times out. This is `long polling`-style: consumer sends an Interest with a long InterestLifetime (say 30 seconds) and the producer holds it until it has new data, then responds.

This works but is fragile. PIT pressure builds up at every forwarder. Producers must keep state per Interest.

### Sync protocols

Better: have a "sync namespace" that producers update with sequence numbers, and consumers poll. The most common sync protocols:

- **ChronoSync** (2013) — every member of a group has a sequence number. The "digest" of all members' (name, seqnum) pairs is the sync state. When something changes, the digest changes. Consumers periodically fetch the latest digest (under `/group/sync`); producers respond with the digest of the new state, plus a list of (name, seqnum) updates.
- **PSync** (2017) — uses Invertible Bloom Lookup Tables (IBLTs) to compactly represent state. Consumers send Interests with their current IBLT; producers compute a difference IBLT and reply. Scales to thousands of members.
- **StateVectorSync (SVS)** — uses simple version vectors. More resilient to network partitions than PSync; less compact.

These protocols let an application implement multi-publisher pub/sub on top of NDN's pull model. Each publisher publishes Data under its own prefix; sync ensures everyone notices.

Conceptually, sync protocols are the "missing piece" of NDN. Without sync, NDN is just a request/response architecture. With sync, you can build chat, distributed file systems, IoT control planes, multi-party games.

### File transfer / segmentation patterns

For "fetch a large file," the convention is segmentation:

```
/alice/photos/birthday.jpg/v=1/seg=0
/alice/photos/birthday.jpg/v=1/seg=1
/alice/photos/birthday.jpg/v=1/seg=2
   ...
/alice/photos/birthday.jpg/v=1/seg=N    (final, marked with FinalBlockId in MetaInfo)
```

The consumer pipelines Interests for each segment. The library `ndncatchunks` does this. You set a window size — like TCP's congestion window — and adjust based on RTT and loss. AIMD-style window control adapts to congestion.

### Real-time streaming

Real-time data (video, audio, sensors) uses naming with high resolution:

```
/sensor/cam1/feed/v=2026-04-27T08:00/seg=0
/sensor/cam1/feed/v=2026-04-27T08:00/seg=1
   ...
```

Consumers request segments slightly ahead of when they expect them. Producers publish with low FreshnessPeriod (so caches don't serve stale segments). For interactive video conferencing, NDN-RTC uses fine-grained naming + sync to handle multi-party.

### Pub/sub on NDN: the cheats

You can fake push using:

- **Long-lived Interests.** Consumer sends Interest with InterestLifetime=60s. Producer sits on it. When something happens, producer responds. Repeat. Mostly works; PIT pressure is the cost.
- **Sync protocol.** As above.
- **Periodic poll + "tail" semantics.** Consumer asks for `/alice/news/latest`. Producer answers with the latest. Repeat every second. Caches help.

There is no "real" push. NDN's authors view this as a feature — push is what allows DDoS, spam, and unwanted notifications today. Pull means the consumer is always in control.

## Mobility for Free

Mobility is one of NDN's headline wins. Today's IP internet is allergic to mobility because IP addresses are tied to network attachment points: when you move, your address changes, and your TCP connections die. Mobile IP exists, was standardized, and is barely used because it's complex and slow.

### Why IP mobility is painful

Imagine your laptop is on Wi-Fi at home with IP `192.168.1.42`. You walk out of the house. Your laptop joins your phone's hotspot, which gives it IP `10.20.30.42`. Your IP just changed. Any TCP connections you had — to your email server, to a video stream — broke instantly, because the email server thinks it's talking to `192.168.1.42` and now your packets come from `10.20.30.42`.

Mobile IP's solution: your "home agent" pretends to still be you at the home address. Packets to your home address get tunneled to your current location. Performance, latency, and trust are all worse.

QUIC and HTTP/3 partially mitigate this (connection IDs survive IP changes), but only end-to-end and only if both endpoints support it. The architecture still assumes a stable host-to-host association.

### Why NDN mobility is free for consumers

In NDN, the consumer is just emitting Interests and waiting for Data. There is no "session" with a specific producer. There is no "address" the consumer needs to keep stable.

You walk out of the house. Your laptop's network attachment changes. Your NDN library notices and:

1. Re-emits any pending Interests (the ones whose previous Data hasn't arrived) on the new face.
2. Continues to emit new Interests on the new face.

Forwarders on the new path do their normal thing: look up FIB, forward, etc. Eventually Data flows back. The consumer doesn't even have to know which network it's on. It works.

### Why NDN producer mobility is harder (but still tractable)

Consumer mobility is easy. Producer mobility — when a producer-machine moves to a new location — is harder, because the FIB entries throughout the network used to point at one location and now must point at another. NDN has a few approaches:

- **Trust schemas + ForwardingHint.** A Producer has a "home prefix" used for trust (`/alice/...`) plus a list of one or more "delegation prefixes" for routing (`/cmu/students/alice/...`). The consumer sends an Interest with Name=`/alice/photos/...` and ForwardingHint=`/cmu/students/alice` (and maybe more). Forwarders that don't have a route for `/alice/photos` use the ForwardingHint instead. When Alice moves from CMU to MIT, her Link object updates her delegation prefixes. Trust still validates against `/alice/...`.
- **Mobile producer NACKs.** When a producer moves and its old prefix is no longer reachable, intermediate routers issue NACKs. The consumer or its closest forwarder retries with an updated ForwardingHint (after fetching an updated Link).
- **Anchor-based mobility.** A "rendezvous point" advertises the producer's prefix and proxies Interests/Data to the producer's current location. Some efficiency loss; some flexibility gain.

### Comparison

| Mobility scenario | IP today | NDN |
| --- | --- | --- |
| Consumer changes networks | TCP/QUIC connection breaks; TLS re-handshake; QUIC connection ID helps but not all apps support it | Trivial: re-issue Interests on the new face; signed Data verifies regardless of who delivers it |
| Producer moves | Mobile IP / DDNS — rare / painful / slow | ForwardingHint + Link object update; retry on NACK |
| Both move simultaneously | Almost intractable | Eventually converges as routing recomputes |
| Cache reuse across moves | Cache breaks (origin server identity matters) | Cache survives; signed Data is location-independent |

### Vehicular networks

A canonical NDN mobility win: cars on a highway. Cars are constantly moving relative to the roadside infrastructure. They want to fetch traffic data, map updates, sensor feeds. With IP this is a nightmare. With NDN, each car emits Interests for `/highway/segment/<region>/traffic/...` and the nearest cache (roadside unit, neighbor car, distant cloud) answers. As the car moves, the answers come from different caches. Nobody cares.

Several universities (UCLA, MIT, Aachen) have run NDN-on-cars test deployments. See `ndn-traffic` and the OpenBeacon work for examples.

## Real Implementations

NDN is a research architecture, but it has working code. Here's the landscape.

### NFD — NDN Forwarding Daemon

The reference forwarder. C++. Runs on Linux, macOS, FreeBSD. Built on `ndn-cxx`. Uses Boost.Asio for async I/O. CS, PIT, FIB, strategies, faces, congestion control, prefix announcements — all here.

Source: [https://github.com/named-data/NFD](https://github.com/named-data/NFD)
Docs: [https://named-data.net/doc/NFD/](https://named-data.net/doc/NFD/)

You install it on Ubuntu via PPA:

```bash
sudo add-apt-repository ppa:named-data/ppa
sudo apt-get update
sudo apt-get install nfd
sudo systemctl start nfd
```

Or build from source:

```bash
git clone https://github.com/named-data/ndn-cxx.git
cd ndn-cxx; ./waf configure; ./waf; sudo ./waf install
git clone https://github.com/named-data/NFD.git
cd NFD; ./waf configure; ./waf; sudo ./waf install
```

You control NFD via `nfdc`:

```bash
$ nfdc status report
General NFD status:
  version=0.9.1
  startTime=...
  uptime=...
Channels: ...
Faces: ...
FIB: ...
RIB: ...
Strategy choices: ...
Content Store: ...
```

### ndn-cxx

The core C++ library. Provides:

- `ndn::Face` — connection to a forwarder.
- `ndn::Interest` and `ndn::Data` — packet types.
- `ndn::Name` — TLV-encoded name.
- `ndn::KeyChain` — local key/cert storage and signing.
- `ndn::Validator` — signature verification with trust schemas.
- `ndn::Scheduler` — async scheduling.
- Encoding/decoding of TLV.

Most NDN apps in C++ are built on `ndn-cxx`. The library is well-documented on named-data.net.

### python-ndn

Pure-Python NDN library. Async-first. Easier for prototyping.

```bash
pip install python-ndn
```

Hello-world consumer:

```python
import asyncio
from ndn.app import NDNApp
from ndn.encoding import Name

app = NDNApp()

async def main():
    name = Name.from_str("/example/hello")
    _, _, content = await app.express_interest(name, must_be_fresh=True)
    print(bytes(content))
    app.shutdown()

app.run_forever(after_start=main())
```

### NDN-DPDK

A high-performance forwarder using Intel DPDK to bypass the kernel and reach line-rate (40-100 Gbps). Implemented in Go (control plane) + C (forwarding plane). Used in NIST's NDN benchmarks.

Source: [https://github.com/usnistgov/ndn-dpdk](https://github.com/usnistgov/ndn-dpdk)

Comparison:

| Forwarder | Throughput | CPU/HW |
| --- | --- | --- |
| NFD | ~1 Gbps | Single-thread, kernel sockets |
| NDN-DPDK | 40+ Gbps | DPDK, dedicated cores, SR-IOV NICs |

### ndnSIM

NS-3-based simulator. The standard tool for NDN academic research. You write a simulation in C++, define topology and traffic patterns, and ndnSIM runs the NDN protocol stack in simulation time. Outputs trace files for analysis.

Source: [https://ndnsim.net/](https://ndnsim.net/)

Key uses: evaluating new caching policies, new forwarding strategies, new trust schemas, scaling tests.

### Other forwarders

- **NDN-Lite** — a lightweight C library for IoT devices (Arduino, ESP32). Implements a minimal forwarder.
- **CCNx (RIOT-OS)** — the RIOT operating system has a CCN/NDN module for IoT.
- **Quagga-NDN** — experimental routing software.
- **NDN.JS** — JavaScript NDN library, runs in browsers and Node, talks to forwarders over WebSockets.
- **NDNgo** — Go bindings for NDN-DPDK.
- **jndn** — Java NDN library.
- **ndn-icp-download** — pipelined chunk fetcher with congestion control.

### Testbeds

- **NDN testbed** — a globally connected research network of universities and labs running NFD instances connected over UDP tunnels. About 30 nodes including UCLA, CMU, Tongji, Beijing Institute of Technology, Caida, BARC, etc. See [https://named-data.net/ndn-testbed/](https://named-data.net/ndn-testbed/).
- **NDN6 / NDN6-LSRP** — newer routing testbed.
- **FABRIC** — NSF's national-scale testbed; supports NDN experiments.
- **NDN over IP overlay** in CMS at CERN, in academic deployments at NIST, and in IoT testbeds.

### Reference apps

A non-exhaustive sampler:

- **ndn-tools** — CLI tools (ndn-ping, ndnpoke, ndnpeek, ndnputchunks, ndncatchunks, etc.).
- **ndn-traffic-generator** — load generator.
- **NDNFS / repo-ng** — file repository.
- **NDN-RTC** — real-time conferencing.
- **NLSR** — link-state routing for NDN.
- **NDN-DPDK** — high-performance forwarder.
- **NDNCERT** — certificate management.
- **ChronoChat / ChronoSync apps** — chat apps using sync.

## Paste-and-runnable

Time to get your hands dirty. We'll install NFD, start it, send Interests, set up a producer, watch the cache work. Everything below is verbatim — `$` lines you type, output lines you should expect (give or take small version differences).

### Install NFD on Ubuntu 22.04

```bash
$ sudo add-apt-repository -y ppa:named-data/ppa
$ sudo apt-get update
$ sudo apt-get install -y nfd ndn-tools
$ nfd --version
nfd version 0.9.1-2-g1234567 (compiled ...)
```

### Start NFD

```bash
$ sudo systemctl start nfd
$ sudo systemctl status nfd
● nfd.service - NDN Forwarding Daemon
     Loaded: loaded (/lib/systemd/system/nfd.service; disabled; vendor preset: enabled)
     Active: active (running) since ...
   Main PID: 12345 (nfd)
      Tasks: 5 (limit: 9494)
     Memory: 12.4M
        CPU: 0.034s
     CGroup: /system.slice/nfd.service
             └─12345 /usr/bin/nfd --config /etc/ndn/nfd.conf
```

Or run it in the foreground for debugging:

```bash
$ sudo nfd-start
nfd-start: starting NFD ...
nfd-start: started NFD (pid 12347)
```

(Your account needs to be in the `ndn` group to send Interests through it without sudo. `sudo usermod -a -G ndn $USER` and re-login.)

### Inspect status

```bash
$ nfdc status report
General NFD status:
  version=0.9.1
  startTime=20260427T080000.000000
  currentTime=20260427T080015.000000
  uptime=15 seconds
  nNameTreeEntries=2
  nFibEntries=1
  nPitEntries=0
  nMeasurementsEntries=0
  nCsEntries=0
  nInInterests=0
  nOutInterests=0
  nInData=0
  nOutData=0
  nInNacks=0
  nOutNacks=0
  nSatisfiedInterests=0
  nUnsatisfiedInterests=0

Channels:
  unix:///run/nfd.sock
  tcp4://0.0.0.0:6363
  tcp6://[::]:6363
  udp4://0.0.0.0:6363
  udp6://[::]:6363
  ws://0.0.0.0:9696

Faces:
  faceid=1 remote=internal:// local=internal://
    counters={in={0i 0d 0n 0B} out={0i 0d 0n 0B}}
    flags={local on-demand point-to-point}
  faceid=254 remote=contentstore:// local=contentstore://
    flags={local on-demand point-to-point}
  faceid=255 remote=null:// local=null://
    flags={local on-demand point-to-point}

FIB:
  / nexthops={faceid=1 cost=0}
```

### Run a ping server and ping it

In one terminal:

```bash
$ ndnpingserver /alice/ping
PING SERVER /alice/ping
Payload Size = 0
Freshness Period = 1000 milliseconds
```

In another terminal:

```bash
$ ndnping /alice/ping
=== Pinging /alice/ping ===
content from /alice/ping/ping/0: seq=0 time=2.43 ms
content from /alice/ping/ping/1: seq=1 time=1.97 ms
content from /alice/ping/ping/2: seq=2 time=1.84 ms
content from /alice/ping/ping/3: seq=3 time=1.79 ms
^C
=== /alice/ping ping statistics ===
4 packets transmitted, 4 received, 0% packet loss
rtt min/avg/max/mdev = 1.79/2.01/2.43/0.27 ms
```

### Add a face (UDP tunnel) to a remote NFD

```bash
$ nfdc face create udp4://198.51.100.10:6363
face-created id=257 local=udp4://192.0.2.1:54321 remote=udp4://198.51.100.10:6363 persistency=persistent reliability=off congestion-marking=off
```

### Add a route through that face

```bash
$ nfdc route add /carol/movies face=257 cost=10
route-add-accepted prefix=/carol/movies nexthop=257 origin=static cost=10 flags=child-inherit expires=never
```

### Inspect the FIB

```bash
$ nfdc fib list
FIB:
  / nexthops={faceid=1 cost=0}
  /alice/ping nexthops={faceid=259 (local app) cost=0}
  /carol/movies nexthops={faceid=257 cost=10}
  /localhost/nfd nexthops={faceid=1 cost=0}
```

### Inspect the routing information base (RIB)

```bash
$ nfdc rib list
/ origin=static cost=0 nexthops={faceid=1}
/carol/movies origin=static cost=10 nexthops={faceid=257}
```

### Set a strategy

```bash
$ nfdc strategy set /carol/movies /localhost/nfd/strategy/multicast
strategy-set prefix=/carol/movies strategy=/localhost/nfd/strategy/multicast/v=4
```

### Show strategy choices

```bash
$ nfdc strategy list
/ /localhost/nfd/strategy/best-route/v=5
/carol/movies /localhost/nfd/strategy/multicast/v=4
/localhost /localhost/nfd/strategy/best-route/v=5
/localhost/nfd /localhost/nfd/strategy/best-route/v=5
/ndn/broadcast /localhost/nfd/strategy/multicast/v=4
```

### Express an Interest manually

```bash
$ ndnpeek -p -f /carol/movies/dune/seg42
ndnpeek: data found
Name: /carol/movies/dune/seg42
ContentType: 0
FreshnessPeriod: 30000
SignatureType: 3 (SignatureSha256WithEcdsa)
KeyLocator: Name=/carol/keys/movies/2026
ContentSize: 188
```

(`-p` prints the full packet info; `-f` says the Interest's Name is fresh-required.)

### Publish a Data packet manually

```bash
$ echo "hello world" | ndnpoke /alice/greetings
DATA: /alice/greetings (5 bytes)
```

In another terminal you fetch:

```bash
$ ndnpeek /alice/greetings
hello world
```

### Inspect the Content Store

```bash
$ nfdc cs info
CS information:
  capacity=65536
  admit=on
  serve=on
  nEntries=3
  nHits=8
  nMisses=12
  policyName=lru
```

### Erase CS entries by prefix

```bash
$ nfdc cs erase /carol
cs-erased prefix=/carol count=2
```

### Watch packets on the wire

NFD includes a packet-trace tool:

```bash
$ ndndump
1714200000.001234 from=udp4://192.0.2.10:6363 INTEREST: /carol/movies/dune/seg42?MustBeFresh
1714200000.003421 to=udp4://192.0.2.10:6363 DATA: /carol/movies/dune/seg42 (188B)
1714200001.005678 from=udp4://192.0.2.10:6363 INTEREST: /carol/movies/dune/seg43
1714200001.007234 to=udp4://192.0.2.10:6363 DATA: /carol/movies/dune/seg43 (188B)
```

You can also use `tcpdump` on the underlying UDP/TCP traffic, but `ndndump` decodes the TLV and prints names.

### Connect to the NDN testbed

```bash
$ nfdc face create udp4://hobo.cs.arizona.edu:6363
face-created id=258 ...

$ nfdc route add /ndn face=258 cost=1
route-add-accepted prefix=/ndn nexthop=258 ...
```

You're now connected to the testbed. You can fetch any name announced anywhere on the global testbed:

```bash
$ ndnping /ndn/edu/ucla/ping
content from /ndn/edu/ucla/ping/ping/0: seq=0 time=87.32 ms
content from /ndn/edu/ucla/ping/ping/1: seq=1 time=86.91 ms
```

Latency reflects the actual round-trip across the testbed.

### Tell NFD to log verbosely

`/etc/ndn/nfd.conf`:

```
log
{
  default_level INFO
  Forwarder DEBUG
  Strategy   DEBUG
  Pit        TRACE
}
```

Restart NFD. Log messages now show every Interest forwarded, every PIT update, every Strategy decision. Useful for "why is my Interest going where I don't expect."

### Generate keys & sign Data

```bash
$ ndnsec key-gen /alice
... (creates /alice/KEY/...id...)

$ ndnsec list
/alice
+->* /alice/KEY/abcd/self
```

Now Data signed under `/alice` is signable. The library uses the KeyChain.

### Deploy a small file repository

```bash
$ ndn-repo-ng -c /etc/ndn/repo-ng.conf
[INFO] starting repo-ng
[INFO] reading config from /etc/ndn/repo-ng.conf

# In another terminal: insert a file
$ ndnputchunks /alice/files/manuscript.pdf < /home/me/manuscript.pdf
inserted /alice/files/manuscript.pdf [v=...] (3284 chunks)

# Fetch it back
$ ndncatchunks /alice/files/manuscript.pdf > /tmp/output.pdf
3284 chunks received
$ md5sum /home/me/manuscript.pdf /tmp/output.pdf
abcd1234... /home/me/manuscript.pdf
abcd1234... /tmp/output.pdf
```

(Hashes match — the file round-tripped through NDN, signed and chunked.)

### Common errors and their fixes

```text
ERROR: face-create failed: peer unreachable
```
Fix: `ping` the remote IP first; check firewall on UDP 6363; check NFD is running on the other side.

```text
ERROR: route-add failed: face does not exist
```
Fix: list faces (`nfdc face list`) — the face ID you passed doesn't exist or has been removed.

```text
ndnping: cannot get response (timeout)
```
Fix: is the producer running on this machine? `nfdc fib list` — is the prefix in the FIB? Did you start the producer *before* the ping? Did the InterestLifetime expire?

```text
ndn-cxx::Validator::ValidationError: Cannot fetch certificate
```
Fix: the trust anchor isn't installed in your keychain, or the cert KeyLocator points at a name your forwarder can't reach. Run `ndnsec list` to see what keys/certs you have. Run `ndnsec dump-cert <name>` to inspect specifics.

```text
ERROR: cannot connect to NFD
```
Fix: is NFD running? `systemctl status nfd`. Is the unix socket at `/run/nfd.sock` present? Are you in the `ndn` group?

```text
strategy-set failed: strategy not found
```
Fix: list strategies with `nfdc strategy list`. Use the exact strategy name (with version: `/localhost/nfd/strategy/best-route/v=5`).

## Hands-On — your first NDN app

Now you've installed NFD and seen it work. Let's write a tiny producer/consumer pair from scratch in Python. This solidifies the model.

### Setup

```bash
$ pip install python-ndn
$ python -c "import ndn; print(ndn.__version__)"
0.4.1
```

NFD must be running (`systemctl start nfd`).

### A producer (publishes /myapp/hello)

Save as `hello_producer.py`:

```python
import asyncio
from ndn.app import NDNApp
from ndn.encoding import Name

app = NDNApp()

@app.route("/myapp/hello")
def on_interest(name, _param, _app_param):
    content = b"Hello from NDN! You asked for " + Name.to_str(name).encode()
    print(f"Producer: got Interest for {Name.to_str(name)}")
    app.put_data(name, content=content, freshness_period=10000)

if __name__ == "__main__":
    print("Producer running. Listening on /myapp/hello.")
    app.run_forever()
```

Run it:

```bash
$ python hello_producer.py
Producer running. Listening on /myapp/hello.
```

### A consumer (asks /myapp/hello/world)

Save as `hello_consumer.py`:

```python
import asyncio
from ndn.app import NDNApp
from ndn.encoding import Name

app = NDNApp()

async def main():
    name = Name.from_str("/myapp/hello/world")
    print(f"Consumer: expressing Interest {Name.to_str(name)}")
    try:
        _data_name, _meta_info, content = await app.express_interest(
            name, must_be_fresh=True, can_be_prefix=False, lifetime=4000
        )
        print(f"Consumer: got {bytes(content)!r}")
    except Exception as e:
        print(f"Consumer: error {e}")
    app.shutdown()

if __name__ == "__main__":
    app.run_forever(after_start=main())
```

Run it (in a second terminal):

```bash
$ python hello_consumer.py
Consumer: expressing Interest /myapp/hello/world
Consumer: got b'Hello from NDN! You asked for /myapp/hello/world'
```

What happened:
1. The consumer connected to NFD over the unix socket.
2. The producer connected to NFD too, and registered the prefix `/myapp/hello` (NFD added a FIB entry pointing at the producer's face).
3. The consumer expressed an Interest for `/myapp/hello/world`.
4. NFD's CS missed; NFD's FIB matched `/myapp/hello`; NFD forwarded the Interest to the producer's face.
5. The producer's `on_interest` callback fired with name `/myapp/hello/world`.
6. The producer signed and emitted Data with that name and content.
7. NFD's PIT had recorded the consumer's face; NFD forwarded the Data back.
8. Consumer received it and printed.

### Inspect the FIB while running

```bash
$ nfdc fib list
FIB:
  / nexthops={faceid=1 cost=0}
  /localhost/nfd nexthops={faceid=1 cost=0}
  /myapp/hello nexthops={faceid=276 (from local app) cost=0}
```

The producer registered `/myapp/hello` automatically when it called `@app.route(...)`. NFD added a FIB entry pointing at faceid 276 (the producer's local face).

### Stop the producer; see what happens

Stop the producer with `Ctrl+C`. The face goes away; NFD removes the FIB entry. Run the consumer again:

```bash
$ python hello_consumer.py
Consumer: expressing Interest /myapp/hello/world
Consumer: error TimeoutError: Timeout
```

The Interest had nowhere to go. After 4 seconds (the InterestLifetime) the library raised TimeoutError. This is the NDN equivalent of "connection refused" — there's no producer, so no Data ever comes back.

### Add caching by triggering it twice

Restart the producer. Run the consumer twice:

```bash
$ python hello_consumer.py
Consumer: got b'...'   # Producer logs: got Interest

$ python hello_consumer.py
Consumer: got b'...'   # Producer logs: got Interest
```

Both runs hit the producer because the consumer set `must_be_fresh=True` and the cached Data's FreshnessPeriod is 10s. Hmm — but if the second run is within 10s, NFD's CS *should* serve it. Let's see:

```bash
$ nfdc cs info
N entries: 1
N hits: 1   <- the second consumer got served from cache!
N misses: 1
```

In fact, the second consumer was served from NFD's CS without going to the producer. The producer log only shows one "got Interest" line. The cache works.

### Make the second request must_be_fresh outside the freshness window

Wait 11 seconds. Run the consumer again. The producer log gets a fresh "got Interest" — the FreshnessPeriod expired, and `must_be_fresh=True` skipped the stale cache entry, so the Interest went to the producer.

If you set `must_be_fresh=False` in the consumer, the cached entry would be served *forever* (until evicted by LRU or capacity).

### A signed-Interest example

When the producer wants to validate that a particular consumer is allowed to ask, you use SignedInterest. python-ndn's high-level API hides this; here's the manual version:

```python
import asyncio
from ndn.app import NDNApp
from ndn.encoding import Name, NonStrictName, SignaturePtrs
from ndn.security import KeychainSqlite3, TpmFile

app = NDNApp()

async def main():
    keychain = KeychainSqlite3.create(...)
    signer = keychain.get_signer("/myidentity")
    name = Name.from_str("/myapp/admin/restart")
    interest = app.express_interest(name, signer=signer, must_be_fresh=True)
    _data_name, _, content = await interest
    print(content)

app.run_forever(after_start=main())
```

The producer's validator side checks that the SignedInterest's signature comes from a key it trusts.

### Build a tiny chat using PSync

Beyond hello-world, you can build pub/sub apps. The python-ndn examples directory has a `psync` example. Each chat participant runs a producer publishing messages under their own prefix; PSync keeps everyone in sync. This is the kind of pattern NDN really shines at — fan-out distribution without explicit servers.

### Run on the testbed

If you've connected your local NFD to the global NDN testbed (`nfdc face create udp4://hobo.cs.arizona.edu:6363`), you can fetch real names. Try:

```bash
$ ndnpeek -p /ndn/edu/ucla/ping/data
Name: /ndn/edu/ucla/ping/data
ContentType: 0
ContentSize: 16
```

Your local NFD forwarded the Interest to the testbed; UCLA's producer signed and returned Data; your forwarder cached it on the way back. Real-world NDN, end to end.

## Confusion pairs

Mistakes to learn the shape of, side-by-side.

### Interest vs Data

A common stumble: people conflate the two packet types. They are distinct.

Bad mental model:

> "I send an Interest, and I get an Interest back with the answer."

Right mental model:

> "I send an **Interest** packet (it's just a request — has a Name, no payload). I receive a **Data** packet (it's the response — has the same Name, plus a payload, plus a signature)."

Two different TLV packet types. Two different code paths. The Interest has TLV type 0x05; the Data has TLV type 0x06. (RFC 8609 / NDN-TLV spec.)

Quick reference:

| | Interest | Data |
| --- | --- | --- |
| Direction | Consumer → Producer (or cache) | Producer (or cache) → Consumer |
| Carries payload? | No, just a Name and selectors | Yes, the Content |
| Signed? | Optional (SignedInterest) | Always |
| Cached? | No (ephemeral, lives in PIT briefly) | Yes (in CS) |
| TLV type | 0x05 | 0x06 |

### CCNx vs NDN

Both are content-centric architectures. They share most concepts but differ in details.

Bad statement:

> "CCNx and NDN are the same thing."

Better statement:

> "CCNx and NDN are sister protocols in the same family. They share Interest/Data semantics, three-table forwarders, and signed-data trust. They differ on: wire format details, naming conventions, fixed-header layout, name component types, and some control-plane features. CCNx 1.0 is at the IRTF (RFC 8569 / RFC 8609). NDN is at named-data.net with its own TLV spec."

For 99% of conceptual discussion they're interchangeable. For implementation: pick one and stick with it. NFD speaks NDN-TLV. CCNx forwarders speak CCNx-TLV.

### Selector vs ForwardingHint

Both attach extra hints to an Interest. They do different things.

- **Selector** (deprecated in modern NDN, kept in CCNx): filters which Data is allowed to satisfy. `MustBeFresh` says "no stale Data." `MaxSuffixComponents` says "the Data Name has at most N components after my Interest Name." Selectors live entirely on the *matching* side: which cached Data is acceptable.
- **ForwardingHint**: a hint about *where to forward the Interest if the FIB doesn't have a direct route for the Name*. Lives entirely on the *forwarding* side. Used for scalable mobile producer support.

Bad mental model: "they're both just hints." Right mental model: "Selector filters Data; ForwardingHint helps routing."

### Name vs prefix

The terms "name" and "prefix" are sometimes used interchangeably; in NDN they have specific meanings.

- **Name**: the full identifier of a piece of data. `/alice/photos/2026/birthday/seg42`.
- **Prefix**: a left-anchored substring of a name. `/alice/photos` is a prefix of the above name. Prefixes are what you announce to the routing system; names are what individual Interests/Data carry.

A FIB entry is keyed by prefix. An Interest carries a full Name. Lookup uses longest-prefix-match against the FIB.

### CS hit vs PIT hit vs FIB hit

Every Interest goes through three table lookups in order. It might hit any of them.

- **CS hit**: there's a cached Data matching this Interest. The forwarder ships the Data back, no upstream action.
- **PIT hit**: there's already a pending Interest for this Name. The forwarder *aggregates* — adds the new face to the existing PIT entry. Still no upstream action.
- **FIB hit**: there's a route for the Name's prefix. The forwarder forwards the Interest upstream.

A successful Interest hits *one* of these and stops there. The forwarder doesn't keep going through the other two if the first one matched.

Common confusion: people think a "cache hit" means "FIB hit." It does not. A FIB hit means "I forwarded the Interest upstream." A CS hit means "I served from cache, never went upstream." Two completely different outcomes.

### Aggregation vs multicast (in NDN)

Both involve "multiple downstreams, one upstream."

- **Aggregation** is a *passive* effect of the PIT: when two consumers ask for the same Name in close succession, the PIT naturally merges them into one upstream Interest. The Data, when it returns, is delivered to both downstream faces. This is *automatic*, requires no protocol, and works regardless of strategy.
- **Multicast** in IP is a *protocol* (PIM, IGMP) that applications must opt into. In NDN, the Multicast *Strategy* explicitly tells the forwarder to send copies on multiple upstream faces (for redundancy or for fan-out). You only use Multicast Strategy when you want *parallel queries on multiple paths*.

When people say "NDN does multicast for free," they really mean "NDN's PIT does aggregation, which gives you 1-to-N delivery in the *downstream* direction without a separate protocol." Strategy-level multicast is the upstream side.

### Producer mobility vs consumer mobility

These are different problems with different solutions.

- **Consumer mobility** is *trivially handled*. The consumer simply re-issues Interests on its new face. No state in the network needs to change; the routing tables for Producer-side prefixes don't care which consumer asks.
- **Producer mobility** is *harder*. The Producer's prefix needs to be re-routable from elsewhere in the network. NDN handles this with Link objects + ForwardingHint, plus updates to NLSR routing. Not free, but tractable.

If someone says "NDN handles mobility for free" without qualification, they're being a little loose. Consumer mobility yes. Producer mobility takes work.

### NDN names ≠ filesystem paths

Both look like `/foo/bar/baz`. They mean different things.

- A filesystem path identifies a *file location on disk*. The path is opaque to the file's content.
- An NDN name identifies *a piece of data anywhere in the network*. The name is the data's identity.

A naive learner sees `/alice/photos/birthday.jpg` and thinks "Alice's home directory has a `photos/birthday.jpg` file." Maybe she does, maybe she doesn't. The NDN name is just *what to ask for*. There may be no actual file anywhere; the producer might generate the bytes on demand.

### NDN ≠ HTTP without TCP

A common pitch: "NDN is HTTP-but-on-the-network-layer." This is half right.

Half right: yes, NDN uses path-like names; yes, you GET-by-name; yes, caches are fundamental.

Half wrong: NDN's trust model is fundamentally different (sign the data, not the channel). NDN's forwarding model is fundamentally different (every router does Interest forwarding and caching). NDN has no "host" concept; HTTP has Host: headers. NDN's pull model is at the network layer; HTTP's request/response is application layer over TCP.

So: similar shape, different stack level. NDN is what you'd get if the IETF in 1995 had decided that the Web was so important they should put its primitives into the network layer rather than building HTTP on top of TCP/IP.

### NDN ≠ blockchain

Blockchains are decentralized append-only ledgers with consensus. NDN is a *naming and forwarding* architecture. They don't share an architectural goal.

There are intersection-explorations:
- Some research uses NDN to deliver blockchain blocks efficiently (multicast aggregation helps).
- Some research uses blockchain to maintain a distributed NDN trust schema.
- "Blockchain-of-content" speculations propose using a chain to register names.

But NDN is not a blockchain. NDN is not a distributed ledger. NDN is name-routed packet forwarding with signed data.

### NDN ≠ IPFS

Closer in spirit, still different.

- **IPFS** (InterPlanetary File System) is a content-addressed storage system. Data is identified by its SHA-256 hash (a CID). It runs over IP, on top of libp2p. Caches and DHT lookups locate data.
- **NDN** is content-routed networking. Data is identified by *name* (not hash, though hashes can appear). It replaces IP. Forwarders do native Interest routing.

IPFS keeps IP and adds an overlay. NDN replaces IP entirely. IPFS uses immutable hash-based names. NDN uses hierarchical mutable names with versions.

You can run IPFS on top of NDN (an active research area). You cannot run NDN on top of IPFS.

### Cache vs storage

A Content Store is a *cache*: small, opportunistic, evicts on pressure. It is not durable storage. If you insert a Data packet into NFD's CS by routing one through it, that Data may be evicted in seconds.

For durable storage you use a **repo** (NDN repository), like `repo-ng` or `ndn6-file-server`. Repos are explicit Producers that hold data persistently and respond to Interests. They look identical to consumers — same naming, same Data — but on the storage side, they're not relying on cache.

### Common gotchas — quick fire

- **Forgot to register the prefix.** Your producer is running but `nfdc fib list` doesn't show your prefix. You forgot to call `face.registerPrefix()` (or run `nfdc route add` for unmanaged prefixes). Interest goes to the default route or gets dropped.
- **Wrong strategy.** You assumed multicast but the default is best-route. Set with `nfdc strategy set`.
- **Trust anchor missing.** Your validator can't fetch the cert chain because you haven't installed a trust anchor. Use `ndnsec list` to see, `ndnsec install-cert` to add.
- **InterestLifetime too short.** Default is 4 seconds. Long-RTT paths through the testbed take >4s for the first hop. Set explicitly: `Interest::setInterestLifetime(time::seconds(30))`.
- **MustBeFresh and FreshnessPeriod=0.** Producer publishes Data with FreshnessPeriod=0, consumer asks with MustBeFresh. The Data is *immediately stale*; consumer keeps fetching from the producer; cache never serves.
- **Signed Interest with bad time.** SignedInterests use a timestamp; if your clock drifts the producer rejects them. Use NTP.
- **Forgot HopLimit.** Default is 32; long testbed paths can exceed this. Set HopLimit higher for long paths.
- **Face on wrong scheme.** UDP multicast face vs UDP unicast face — different syntax. `udp4://224.0.23.170:56363` for multicast; `udp4://198.51.100.1:6363` for unicast.

## DTN, ICN, CCN, PURSUIT

NDN doesn't exist alone. It's part of a research family that has been working on "the future of networking" for two decades. Knowing the family helps you understand NDN's specific position.

### Information-Centric Networking (ICN)

The umbrella term. ICN is a network architecture where the primary primitive is *named, signed information* rather than *machine-to-machine connectivity*. Different ICN proposals make different concrete choices, but they share these properties:

- Data is named.
- Data is signed (or otherwise verifiable in itself).
- Forwarders cache.
- Consumers ask by name; producers respond.

The IRTF's ICN Research Group (ICNRG) is the standards body coordinating ICN drafts. They have produced informational RFCs and architectural drafts. Key documents:

- **RFC 7476** — ICN baseline scenarios.
- **RFC 7927** — ICN research challenges.
- **RFC 7945** — ICN evaluation methodology.
- **RFC 8569** — CCNx semantics.
- **RFC 8609** — CCNx wire format.
- **draft-irtf-icnrg-** various — naming, security, wire format.

### CCN / CCNx

The earliest concrete ICN design (PARC, 2009) by Van Jacobson and team. Originally called CCN. Later evolved into CCNx by Cisco / a later research effort. CCNx 1.0 is what RFC 8569 describes.

Differences from NDN (a partial list):

- **Wire format header.** CCNx uses a fixed-header preamble (8 bytes) followed by TLV; NDN uses pure TLV.
- **HashGroup signatures.** CCNx allows hash-based content addressing more directly.
- **Manifests.** CCNx has a "Manifest" concept used for chunking large objects, distinct from how NDN does segmentation.
- **Registry of TLV types.** CCNx and NDN have parallel but non-identical type-code registries.

The two camps interoperate at the conceptual level but not at the wire-format level. A CCNx forwarder can't directly process an NDN packet.

### NDN

A 2010-onward research project led initially by Lixia Zhang at UCLA, with collaborators across major U.S. universities (UCLA, Caida, Memphis, Arizona, CSU, Tongji, Beijing Jiaotong, Cornell, etc.) and partly funded by NSF as part of the Future Internet Architecture program. The project produced NFD, ndn-cxx, the testbed, and a long list of demonstration apps. It's the most active and most-cited ICN effort today.

NDN focused on:

- Hierarchical, mutable, app-meaningful names (not self-certifying hashes).
- Trust schemas on top of signed data.
- A practical forwarder (NFD) and library (ndn-cxx) that researchers can run on commodity hardware.
- A testbed.

If "ICN" is the genus, NDN is the species you're most likely to actually run code with.

### PURSUIT / PSIRP

A European Union FP7 project (2008-2013). Took a different architectural approach: **rendezvous-based pub/sub** instead of name-routing.

In PURSUIT:
- A producer announces "I have content with this RID (rendezvous identifier)."
- A consumer subscribes to "I want content with this SID (scope identifier)."
- A separate **Rendezvous Network** matches publications to subscriptions and creates a forwarding path.
- Data flows over a **Topology Manager** path, often using Bloom-filter-based source routing.

PURSUIT is more theoretical, less implemented than NDN. The names are flat (RIDs/SIDs), not hierarchical. The architecture has three planes (rendezvous, topology, forwarding) instead of NDN's one-plane forwarder.

### NetInf

Another EU project (4WARD, SAIL). Proposed a "Network of Information" architecture with **named information objects** identified by hashes. Less hierarchical, more like a content-addressed object store.

NetInf's contributions:
- Self-certifying flat names (hash-based).
- Named Information ("ni://") URI scheme — RFC 6920.
- Routing via various mechanisms including DHT-based name resolution.

NetInf is largely subsumed by NDN/CCNx for practical work, but its naming ideas (especially ni:// URIs) live on in some IoT and content-addressing standards.

### MobilityFirst

A US NSF FIA project (Rutgers et al., 2010-2016). Different architectural choice: **flat globally-unique identifiers (GUIDs)** plus a **Global Name Resolution Service (GNRS)**.

Approach:
- Each "thing" (host, content, sensor, group) gets a GUID.
- A globally distributed GNRS maps GUIDs to network attachment points.
- Routing uses GUIDs and dynamic late binding to current locations.
- Inherently focused on mobility and intermittent connectivity.

MobilityFirst is more host-centric than NDN, but acknowledges content as a first-class entity. It chose a different abstraction (flat GUIDs + lookup service) versus NDN's (hierarchical names + native routing).

### Delay-Tolerant Networking (DTN)

A predecessor architecture, dating back to the late 1990s, originally targeted at **interplanetary internet** (Cerf et al., motivated by Mars rovers).

DTN assumes:
- The path between two endpoints may be intermittent — link cuts, contact opportunities, hours-long propagation delay.
- Store-and-forward at every node, with data persisting on disk between contacts.
- "Bundles" — large units of data with persistent metadata.

DTN shares ICN's "the network has memory" philosophy but came at it from a different angle: not "content-centric to enable caching for performance" but "store-and-forward to deal with disconnection."

DTN protocols (RFC 5050 / Bundle Protocol) are deployed in NASA missions, in remote-area networks, in disaster-response rigs. They overlap conceptually with NDN — both put state in the network — but they're optimized for different workloads.

### LISP — a related-but-different idea

Locator/Identifier Separation Protocol (RFC 6830/9300, `networking/lisp`) is *not* an ICN architecture. It's a host-centric routing architecture that splits the IP address into "where you are" (locator) and "who you are" (identifier).

LISP and NDN share the *insight* that current IP conflates two things and we should separate them. LISP separates locator from identifier *for hosts*. NDN goes further and replaces both with names *for content*.

If LISP is "let's add a level of indirection so hosts can move," NDN is "let's get rid of the host as the routing target entirely."

### Putting them on a map

| Architecture | Naming style | Network role | Status (2026) |
| --- | --- | --- | --- |
| **IP** | Address-based (host-centric) | Per-machine packet delivery | Deployed everywhere |
| **DTN** | App-defined names in bundles | Store-and-forward across intermittent links | Niche deployments (space, remote) |
| **CCNx** | Hierarchical name (with optional hashes) | Content forwarding + cache | RFC stack at IRTF; active CCN-Lite |
| **NDN** | Hierarchical name | Content forwarding + cache | Active testbed, research/labs |
| **PURSUIT** | Flat RIDs/SIDs | Rendezvous-based pub/sub | Mostly historical (research) |
| **NetInf** | Hash-based ni:// URIs | Content-addressed object store | Mostly historical |
| **MobilityFirst** | Flat GUIDs + lookup service | Mobility-first IP-like | Inactive after 2017 |
| **LISP** | Identifier + locator (still IP-host-centric) | Host indirection | Deployed in some enterprise |

NDN is the one to learn first. The others are useful context.

## Russ White's Take (Ch 30, "Computer Networking Problems and Solutions")

In the 2018 Pearson textbook *Computer Networking Problems and Solutions* by Russ White and Ethan Banks, Chapter 30 ("Information-Centric Networking") is one of the better short explainers of NDN written for working network engineers (rather than for academics). A summary of the chapter's framing:

### Russ's lens: solve a problem

White takes the position that NDN is interesting *because of the problem it solves*, not because it's a clean redesign. The problem is:

> Most internet traffic today is content delivery, not host-to-host conversation. We've optimized for the latter and bolted on workarounds for the former. ICN flips the design priority.

The book sets up a thought experiment: imagine you're building a network from scratch in 2018. The dominant traffic mix is video-on-demand, software updates, and content distribution. Would you build IP and bolt on CDNs? Or would you build something content-centric from the start?

White argues you'd at least *consider* the second option. Then he walks through the implications.

### The five-part argument

Russ structures the case for ICN around five claims:

1. **Names are richer than addresses.** A name can carry semantic meaning ("the latest weather report for ZIP 90210") that an address cannot. This makes routing decisions more flexible.
2. **Caching is fundamental.** When most of your traffic is the same bytes repeated, a network without native caching is fighting its own bandwidth. ICN caches at every hop; this is not a feature, it's the architecture.
3. **Trust is in the data.** Once data is signed, you don't need to trust the channel. This decouples security from delivery — anyone can deliver, only the producer needs to be trusted.
4. **Multicast is automatic.** Aggregation in PIT-equivalent tables turns N parallel requests into 1 upstream Interest. Multicast becomes the default rather than a separate protocol.
5. **Mobility is decoupled from addressing.** When the routing target is content rather than a host, the host's location stops mattering for routing.

### What White says it costs

The book is honest about trade-offs. The chapter lists:

- **PIT scaling.** A backbone forwarder might have millions of in-flight Interests. PIT memory and lookup performance is a real engineering problem.
- **Routing scaling.** The global namespace is huge — possibly larger than the BGP IPv4 table. Whether NLSR-like protocols scale to this is open.
- **Cache management.** Effective caching across heterogeneous demand patterns is hard.
- **Privacy.** Names reveal content interests. A surveilling forwarder learns a lot about what consumers want. Encryption layers (NAC) help but add complexity.
- **Deployment economics.** Carriers don't get paid for caching; routers cost more if they have CS hardware.

### Russ's bottom line

The chapter reads as cautiously optimistic. White doesn't predict NDN will replace IP. He suggests instead:

- ICN principles are likely to influence future designs (he was right — see CDN evolution, edge compute, content-aware service meshes).
- Specific deployments (IoT meshes, scientific data, video distribution, vehicular networks, classroom video) will adopt ICN-style architectures *internally* even if the global internet doesn't.
- Whatever comes after IPv6 will probably borrow heavily from ICN ideas — content-addressed semantics, signed data, in-network caching.

He stops short of saying "deploy NDN tomorrow." He doesn't have to. The chapter's value is teaching you to recognize ICN-shaped problems (and ICN-shaped solutions) when they show up in real networks.

### Beyond the chapter: related Russ takes

White's other writing on this:
- *The Art of Network Architecture* (Cisco Press, 2014) discusses content-aware networking as a design pattern.
- His blog posts at rule11.tech occasionally cover ICN/NDN.
- The "On the 'Net" podcast frequently features ICN-adjacent topics.

The takeaway across his work: think about what's actually moving through the network. If it's content, design for content. If it's conversation, design for conversation. IP optimizes for conversation. ICN/NDN optimize for content.

### Named Function Networking — the next step

A research extension White briefly cites: **Named Function Networking (NFN)**. Instead of asking for *named data*, you ask for *named functions applied to named data*. The network executes computation on the path.

Example name: `/lambda/grayscale(/alice/photos/birthday/seg42)`.

A forwarder that understands NFN could:
1. Look up the function `grayscale` (an executable Data packet).
2. Look up the input `/alice/photos/birthday/seg42`.
3. Run the function.
4. Cache the *result* for future identical requests.

NFN sits at the intersection of NDN, serverless compute, and content addressing. It's an active research direction. As of 2026 there's no production deployment, but academic work continues.

### "Blockchain-of-content"

Another speculative direction Russ touches on: a global registry of named content secured by a blockchain. Each piece of content is registered with its name, hash, signature, and producer identity in a public ledger. NDN forwarders consult the ledger to verify signatures and detect misbehaving producers.

This is highly speculative. No current deployment uses it. But it's an example of how NDN's signed-data architecture combines naturally with append-only-ledger technology.

## When NDN is the right answer

Don't deploy NDN because it's neat. Deploy it because it solves a problem you have. Specific scenarios where NDN is genuinely the right tool:

### IoT mesh networks

Big sensor deployments (smart farms, environmental monitoring, industrial control) often have:
- Many sensors, low data rates each.
- Intermittent connectivity (some sensors sleep; some links are flaky).
- Bandwidth-constrained backhaul.
- Heterogeneous hardware.

NDN-Lite and similar minimal NDN stacks fit this. Sensors publish data with hierarchical names (`/farm5/north/sensor7/temp/v=...`). Consumers (gateways, edge analytics, cloud aggregators) pull data by name. Caching at gateway level reduces backhaul traffic when multiple consumers want the same data. Sleep cycles don't matter — Interests can wait, or be served from cache.

### Scientific data dissemination

Big-science projects (CMS at CERN, LIGO, climate modeling, genomics consortia) generate enormous datasets that hundreds of researchers want to access. Today they run elaborate distribution networks (XRootD, dCache, XCache) to handle this. These are essentially custom CDNs for science data.

NDN's content-routing model is a natural fit: physicists ask for `/cern/cms/dataset/2025/run123/event456` and the network delivers from the closest cache. CERN has actually run NDN deployments in the CMS context as a proof of concept.

### Vehicular networks (V2X)

Cars want map updates, traffic data, weather, sensor feeds, software patches. They're moving constantly. Cars near each other often want the same data. Roadside infrastructure can cache. This is exactly what NDN's mobility-and-caching combo is good at.

UCLA's NDN-on-cars work, the OpenSim vehicular extensions, and various V2X testbeds use NDN-style content distribution. Real-world deployment is rare but the architectural fit is excellent.

### Classroom and campus video

A university lecture broadcast to 500 students all on campus: today, that's 500 unicast HTTP streams. With NDN, the campus router caches the stream once and aggregates Interests for all 500 viewers. Bandwidth on the campus core drops to (1+ε) streams.

ASR's classroom deployments and similar campus testbeds have shown order-of-magnitude bandwidth wins for live video on NDN.

### Content distribution inside data centers

Microservices increasingly look like content distribution: pulling configs, fetching ML model weights, distributing compiled artifacts. Within a data center, these can be NDN-routed for free caching. Some research uses NDN as the inter-pod transport in Kubernetes-like clusters (`orchestration/kubernetes`).

### Disconnected / disaster networks

DTN-meets-NDN: when connectivity is intermittent (after a hurricane, in an arctic camp, in a war zone), store-and-forward in a content-centric architecture works much better than IP. Bundles get cached by intermediate hops, requests get satisfied opportunistically when paths come up.

### Anywhere multicast-by-default helps

If your traffic is fundamentally one-to-many — live video, news feeds, software updates, sensor firehose — NDN gives you multicast for free. IP-level multicast is hard and rarely deployed in the wide area. NDN bypasses the problem.

## When NDN is overkill

Equally important: knowing when *not* to reach for NDN.

### Interactive 1-1 communication

A phone call, a chat session, a single SSH connection. You and one specific other endpoint want to exchange data. Caching doesn't help (you're producing brand-new data each second). Multicast doesn't help (only two parties). Mobility might or might not matter. NDN works fine here but offers no improvement over IP+QUIC; the protocol overhead (signing every packet, name lookup at each hop) costs more than it saves.

Use IP/UDP/TCP/QUIC for interactive 1-1.

### Sub-second updates with high entropy

A real-time stock ticker. A high-frequency sensor network with new readings every millisecond. Each piece of data is unique, requested by only one or two consumers, and stale a few hundred milliseconds after birth. Caching doesn't help; signing per-packet is expensive overhead.

Use a tight UDP-based custom protocol for sub-millisecond high-entropy data.

### Classic OLTP

Database transactions: bank transfers, e-commerce checkouts, ledger updates. These need:
- ACID semantics (not NDN's job).
- Two-phase commit / consensus (`cs-theory/distributed-consensus`).
- Strong consistency, not eventual consistency.

NDN does *deliver* these workloads — your bank app could fetch its API responses over NDN — but the heavy lifting (consensus, distributed transactions, ledger replication) happens above the network layer. NDN doesn't provide consensus primitives. CAP-theorem-aware system design (`cs-theory/cap-theorem`) doesn't change just because the network changed.

### Single-tenant point-to-point overlays

If you're building a VPN connecting two specific sites and just want a fat encrypted tunnel between them, IPsec or WireGuard is simpler and more deployed. NDN doesn't pay off for endpoint-to-endpoint tunnels.

### Latency-sensitive, low-jitter, single-path

Real-time voice calling, professional audio streaming, networked control loops where you need a single tight path with predictable latency. NDN's path-adaptive forwarding can introduce variability. Stick with deterministic networking (TSN, DetNet) for these.

### Where you can't (or won't) replace IP

Most ISPs. Most data centers in production. Most enterprises. NDN as an overlay over UDP is *fine for research* but adds latency and complexity that production environments aren't paying for unless they have specific gains in mind.

### Rule of thumb

Ask yourself:

1. Is most of my traffic the same bytes asked for by many consumers? → NDN may help.
2. Is my workload sensitive to mobility or intermittent connectivity? → NDN may help.
3. Do I want trust in the data rather than the channel? → NDN may help.
4. Otherwise → IP works fine.

### Hybrid deployments — the realistic case

In practice, most NDN you see in production is *hybrid*: a chunk of an enterprise or research network runs NDN internally, with gateways translating to/from IP at the edges. This makes sense because:

- You can deploy gradually, one site at a time.
- You don't need every router on Earth to upgrade.
- You get NDN's wins where they matter (caching, multicast, mobility) without forcing the rest of the world to switch.
- IP gateways handle the "talk to the rest of the internet" use case.

A hybrid pattern looks like:

```
[NDN island A] <==> [IP backbone] <==> [NDN island B]
       \                                   /
        +-- Producer 1                    +-- Consumer 1
        +-- Producer 2                    +-- Consumer 2
```

NDN islands have full forwarders; the IP backbone just hauls UDP-encapsulated NDN packets between them. Each island has its own NLSR, its own caches, its own trust. Federation between islands happens at the routing-prefix level (each island announces a different `/ndn/edu/<institution>` prefix).

This is roughly how the global NDN testbed works today: ~30 university islands, each with a few NFD instances, connected over UDP tunnels through the public IP internet.

### When the answer is "ICN ideas without NDN itself"

Sometimes you don't need NDN per se but you do need ICN-shaped thinking:

- **HTTP/2 push and HTTP/3 datagrams** brought some content-multicast ideas into IP transport.
- **Service workers in browsers** cache HTTP responses on the client; this is small-scale in-browser CS.
- **Edge compute platforms** (Cloudflare Workers, Lambda@Edge, Fastly Compute) are essentially a managed CDN with content-aware execution — exactly what Named Function Networking imagines.
- **Content-addressed storage** (IPFS, Git, Docker registries) borrows the "name = hash of content" idea.
- **CDNs themselves** are massive ICN-style overlays.

If you can't deploy NDN, you can still adopt the architectural lessons: cache aggressively, sign data not channels, design names with structure, plan for one-to-many fan-out.

## Try This

Concrete exercises to lock the concepts in. Do them in order.

### Exercise 1 — install NFD and ping localhost

Install NFD per the Paste-and-runnable section. Start it. Run `ndnpingserver /me/ping` in one terminal and `ndnping /me/ping` in another. Observe the RTT (probably <2ms — it's loopback). Stop the producer; observe the consumer report timeouts. Restart the producer; observe the consumer recover.

What you've learned: faces, prefix registration, Interest/Data round trip, timeouts.

### Exercise 2 — write a Python producer

Implement the `hello_producer.py` from the Hands-On section. Add a counter to it that increments each time it's hit. Make the producer return Data whose Content includes the counter. Run consumers repeatedly; observe whether the counter increments (it does the first time, then NFD's CS serves cached Data and the counter freezes — until the FreshnessPeriod expires).

Vary FreshnessPeriod from 1s to 30s. Observe how often the counter actually increments.

What you've learned: caching policy in action; FreshnessPeriod versus MustBeFresh.

### Exercise 3 — set up two NFDs

Run two NFD instances on the same machine using different config files (different unix sockets, different log files). Set up faces between them using `nfdc face create udp4://localhost:6363` (and the other instance on a different port). Add routes. Run a producer on one side and a consumer on the other. Watch the cache populate on both sides as Data flows through.

What you've learned: faces between forwarders, multi-hop NDN.

### Exercise 4 — measure aggregation

Run one producer + N consumers (say N=100, using a shell script to launch them in parallel). Use `ndndump` on the producer-side NFD to see how many Interests actually reach the producer. With high N and tight timing, the aggregation effect should drop the Interests-per-Data ratio dramatically.

What you've learned: PIT aggregation in practice.

### Exercise 5 — break it on purpose

While a consumer is fetching, kill the producer. Watch the consumer time out. Restart the producer; watch the consumer succeed if you re-run. Now do the opposite: kill the consumer mid-fetch. The producer doesn't notice — it published Data, NFD held it in cache, no PIT entry existed for an absent consumer.

What you've learned: where state lives in NDN (PIT for in-flight, CS for completed) and what each side does/doesn't observe.

### Exercise 6 — connect to the testbed

Connect your local NFD to the global NDN testbed via UDP face. Fetch `/ndn/edu/ucla/ping/data`. Time it. Disconnect; re-fetch — observe a cache hit on your local NFD if you run within FreshnessPeriod.

What you've learned: the global NDN namespace; how cache distance translates to RTT.

### Exercise 7 — write a trust schema

Generate two identities with `ndnsec key-gen`. Sign Data under one. Configure a validator to require signatures from the other. Watch validation fail. Switch the validator to accept the first. Watch it succeed.

What you've learned: trust schemas in practice; producer-key vs validator-policy.

### Exercise 8 — read NFD logs

Set NFD's log level to TRACE in `nfd.conf`. Reproduce one of the earlier exercises. Read the log. Map every log line to a step in the forwarding plane: face accept, Interest receipt, CS lookup, PIT lookup, FIB lookup, strategy decision, Interest send, Data receipt, PIT consume, CS insert, Data send.

What you've learned: how NFD does what it does, with literal evidence.

### Exercise 9 — try ndnSIM

Install ndnSIM (NS-3 + NDN module). Run the example simulation `ndn-simple.cc`. Modify it to use multicast strategy. Re-run; compare the topology-level behavior. Plot the cache hit ratio across time.

What you've learned: ndnSIM workflow; how to measure architectural choices.

### Exercise 10 — compare to HTTP

For one chosen workload (say, fetching 100 chunks of a video), instrument both an HTTP-on-IP and an NDN implementation. Measure: bytes on the wire, total latency, cache hits at intermediate nodes if any. NDN should win on the bytes-on-the-wire metric for repeated fetches; HTTP may win on raw latency for one-shot fetches.

What you've learned: empirically when NDN pays off and when it doesn't.

## See Also

- `ramp-up/dns-eli5` — DNS as a name-resolution analogy; NDN basically replaces DNS by putting names directly into routing.
- `ramp-up/anycast-eli5` — closest-replica IP addressing; NDN does this for content names by default.
- `ramp-up/ip-eli5` — the address-centric layer NDN replaces.
- `ramp-up/bgp-eli5` — global routing today; NLSR is the NDN equivalent.
- `ramp-up/kubernetes-eli5` — service discovery analogy and microservice content distribution.
- `networking/dns` — engineer-grade DNS reference.
- `networking/multicast-routing` — IP-layer multicast (PIM, IGMP) as the protocol NDN displaces with PIT aggregation.
- `networking/lisp` — locator/identifier separation; the same architectural impulse one step back from NDN.
- `networking/quic` — modern transport on IP; reduces some of the address-centric pain NDN solves at the architecture level.
- `networking/http3` — HTTP over QUIC; the closest IP-stack-equivalent to native NDN-content workloads.
- `networking/grpc` — RPC framework illustrating service-meshy patterns NDN can replace.
- `networking/coredns` — DNS server you'd run today; the namespace-management problem NDN inherits.
- `networking/dpdk` — kernel-bypass packet I/O; the substrate NDN-DPDK uses for line-rate forwarding.
- `networking/ipv4` — what IPv4 actually is (the layer NDN replaces).
- `networking/ipv6` — the still-host-centric IP successor NDN goes further than.
- `networking/tcp` — the connection-oriented transport NDN doesn't need.
- `networking/udp` — the datagram protocol NDN-over-UDP rides on.
- `networking/bgp` — inter-domain routing today; NDN's inter-domain story remains open research.
- `networking/ecmp` — equal-cost multipath; analogous to multicast strategy in NDN.
- `cs-theory/distributed-systems` — the broader theory NDN sits inside.
- `cs-theory/distributed-consensus` — consensus primitives NDN does *not* provide; you build them on top.
- `cs-theory/cap-theorem` — applies to NDN-built distributed systems exactly as it does to IP-built ones.
- `cs-theory/graph-theory` — NLSR and forwarding strategy reasoning relies on graph algorithms.
- `cs-theory/crdt` — eventual consistency primitives that work well with NDN sync protocols.
- `orchestration/kubernetes` — service discovery and content distribution patterns inside a cluster.

## References

- **RFC 8569** — *Content-Centric Networking (CCNx) Semantics.* Mosko, Solis, Wood. IRTF, July 2019. The protocol semantics document for CCNx 1.0. NDN's design is a near sibling. <https://www.rfc-editor.org/rfc/rfc8569.html>
- **RFC 8609** — *Content-Centric Networking (CCNx) Messages in TLV Format.* Mosko, Solis, Wood. IRTF, July 2019. The wire format. <https://www.rfc-editor.org/rfc/rfc8609.html>
- **RFC 7476** — *Information-Centric Networking: Baseline Scenarios.* Pentikousis, Ohlman, et al. IRTF, March 2015. Defines the canonical scenarios used to evaluate ICN proposals.
- **RFC 7927** — *Information-Centric Networking (ICN) Research Challenges.* Kutscher, et al. IRTF, July 2016. The state-of-the-art problem statement.
- **RFC 7945** — *Information-Centric Networking (ICN) Evaluation Methodology.* Pentikousis, Ohlman, et al. IRTF, October 2016. How to compare ICN designs apples-to-apples.
- **RFC 6920** — *Naming Things with Hashes.* Farrell, Kutscher, et al. IETF, April 2013. The `ni://` URI scheme used in NetInf and content-addressing generally.
- **NDN Project Technical Report 0001 onward.** Hosted at <https://named-data.net/publications/techreports/>. The canonical NDN spec series — naming, packet format, forwarding, security.
- **Named Data Networking** (project home page). <https://named-data.net/>. Includes NFD docs, ndn-cxx docs, testbed status.
- **NFD Developer's Guide.** <https://named-data.net/doc/NFD/current/>. The forwarder's official documentation.
- **ndn-cxx Documentation.** <https://named-data.net/doc/ndn-cxx/current/>. C++ library reference.
- **python-ndn Documentation.** <https://python-ndn.readthedocs.io/>. Python library reference.
- **NDN-DPDK at NIST.** <https://github.com/usnistgov/ndn-dpdk>. High-performance forwarder.
- **ndnSIM Documentation.** <https://ndnsim.net/current/>. NS-3-based simulator.
- **Networking Named Content.** Jacobson, Smetters, Thornton, Plass, Briggs, Braynard. CoNEXT 2009. The foundational paper. <https://dl.acm.org/doi/10.1145/1658939.1658941>
- **A Case for Stateful Forwarding Plane.** Yi, Afanasyev, Moiseenko, Wang, Zhang, Zhang. NDN Project, 2012. Justifies the stateful PIT-driven forwarder design.
- **NDN, Technical Report.** UCLA, NDN-0001 onward. The collection of project tech reports.
- **Van Jacobson, "A New Way to Look at Networking," Google Tech Talk, August 2006.** YouTube. The talk that started the modern push.
- **Computer Networking Problems and Solutions.** Russ White, Ethan Banks. Pearson, 2018. ISBN 978-1587145049. Chapter 30 covers ICN/NDN. The accessible-engineer-grade overview cited extensively above.
- **Information-Centric Networking: Patterns and Anti-Patterns.** Westphal et al., IEEE Communications Magazine, 2017. Pragmatic survey.
- **A Brief Survey of Name Data Networking.** Saxena et al., 2016. Concise overview.
- **CCN Lite Project.** <https://github.com/cn-uofbasel/ccn-lite>. Lightweight CCNx implementation, useful comparison code.
- **PURSUIT Project Final Report.** EU FP7 Final, 2013. The canonical PURSUIT writeup.
- **MobilityFirst Architecture Final Report.** Rutgers / NSF FIA, 2016.
- **DTN: Bundle Protocol.** RFC 5050 (and successors RFC 9171). The DTN core.
- **NDN Testbed.** <https://named-data.net/ndn-testbed/>. The global research testbed status page.
- **IRTF ICNRG (Information-Centric Networking Research Group).** <https://datatracker.ietf.org/group/icnrg/about/>. Standards drafts and meeting minutes.
- **NDNCERT Specification.** <https://docs.named-data.net/ndncert/>. Certificate management protocol.
- **Trust Schemas in NDN.** Yu, Afanasyev, Clark, Claffy, Jacobson, Zhang. ACM ICN 2015. The schema-based trust paper.
- **Schematizing Trust in Named Data Networking.** Yu et al. CCS 2015. More schema work.
- **NLSR: Named-data Link State Routing Protocol.** Hoque, Amin, Alyyan, Zhang, Wang, Zhang. ACM SIGCOMM ICN 2013. The OSPF-equivalent for NDN.
- **NDN Tutorial.** Various ACM SIGCOMM tutorials, 2013-2024. Slides at <https://named-data.net/tutorials/>.
- **NDN Common Name Library.** <https://github.com/named-data/ndn-cnl>. Standard typed-component conventions.
- **Wireshark NDN Dissector.** Lets you decode NDN packets in Wireshark traces. Bundled with modern Wireshark.
