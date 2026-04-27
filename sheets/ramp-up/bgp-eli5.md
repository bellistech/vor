# BGP — ELI5 (The Internet's GPS System)

> BGP is the giant address book the internet uses so that every neighborhood (network) knows how to send your packets to every other neighborhood, even though no single neighborhood knows the whole map.

## Prerequisites

(none — this is a self-contained ramp-up sheet)

If you want a little extra context, the only thing that helps is knowing that "the internet" is not one thing — it's lots of separate networks glued together. That's literally the entire mental model you need to start. Everything else we'll build from scratch.

## What Even Is BGP?

### The internet is not one network

You probably picture "the internet" as one big magical cloud. It isn't. The internet is **thousands of separate networks** all stitched together — like a huge sprawling city made of thousands of neighborhoods, each one run by a different company, each with its own little postal service.

Some of those neighborhoods are tiny:

- Your home WiFi is one tiny network. You run it. There's basically just you, your laptop, your phone, maybe a smart bulb pretending it isn't spying on you.
- Your apartment building or your office probably has a slightly bigger network — a few dozen people sharing one router.

Some are huge:

- Your **ISP** (Internet Service Provider — Comcast, AT&T, Verizon, BT, Deutsche Telekom, etc.) is a much bigger neighborhood. They have thousands of customers and lots of routers.
- Google has its own gigantic network. So does Amazon, Microsoft, Apple, Facebook/Meta, Cloudflare, Netflix.
- Your school, your bank, your government, your favorite weird forum all have networks too.

Each of these separate networks is called an **Autonomous System** — usually shortened to **AS**. Think of "AS" as just a fancy word for "neighborhood with its own little post office and its own postal rules."

> There are over 75,000 of these AS neighborhoods in the world. Every single one of them is a separately managed chunk of the internet.

### The problem: how do all these neighborhoods talk to each other?

Imagine you live in neighborhood A and want to mail a letter to your friend in neighborhood Z. Your local post office knows about everyone in your neighborhood, and it knows the few neighboring post offices it can hand mail off to — but it doesn't know how to reach Z directly.

So how does the letter ever get there?

The post offices need a way of **telling each other which addresses they can reach, and the path mail should take to get there**. They don't all have to know the *whole* map — they just need to know "if it's headed to addresses 1-1000, hand it to the post office down the street; they'll know what to do next."

That's exactly what computers on the internet have to do, and that's exactly what BGP solves.

### BGP = the internet's GPS

**BGP** stands for **Border Gateway Protocol**. The "border" is the part you should remember: BGP is the language spoken at the *border* of each neighborhood, between the routers that talk to other networks.

Think of BGP as the **GPS navigation system for the entire internet**. The comparison goes pretty deep:

| GPS | BGP |
|---|---|
| Knows all the roads | Knows all the network paths |
| Calculates the best route to your destination | Calculates the best path to a destination network |
| Updates when roads close or open | Updates when network links go down or come up |
| Shared between all cars | Shared between all networks |
| Run by satellites | Run by routers at the border of each network |

A regular GPS in your car has a map of roads. BGP has a "map" of how to get from one network to another. When a road closes, your GPS reroutes you. When a network link drops, BGP reroutes everyone's traffic. When a new highway opens, your GPS finds a new shortcut. When a new internet link is built, BGP starts using it.

The big difference is: BGP isn't run by satellites or by one company. It's run by every router at the border of every network on Earth, and they're all gossiping with each other constantly to keep the map up to date.

### How BGP works in one paragraph

Every network tells its direct neighbors *"hey, I can reach these addresses, and here is the path through me to get there."* The neighbors tell **their** neighbors. They tell theirs. And so on. Eventually every network on the internet has heard at least one path to every other network. When something breaks, the routers gossip about that too — *"I can't reach those addresses any more, scratch that path"* — and everyone updates their map.

That's it. That's the whole protocol, in one sentence. The rest of this sheet is "but how exactly?"

### A tiny example

Picture three networks:

```
Network A (AT&T)         Network B (Comcast)       Network C (Google)
   AS 7018       <----->     AS 7922      <----->     AS 15169
       \                                                /
        \                                              /
         +------>     Network D (Level3)       <------+
                          AS 3356
```

You're at home and you're connected to AT&T. You ask for `google.com` (which lives at addresses owned by Google's AS 15169). BGP has discovered two possible paths:

- **Path 1:** AT&T → Comcast → Google (2 hops)
- **Path 2:** AT&T → Level3 → Google (2 hops)

They have the same length. BGP will pick *one* of them, based on rules we'll meet in a few sections. It might pick the cheaper one (Comcast might be a cheaper deal for AT&T) or the lower-latency one or the one with bigger pipes. The whole point is: AT&T has *options*, and BGP is the machinery that lets it pick.

### Tier 1, tier 2, tier 3 ISPs (highways, city streets, driveways)

Not all networks are the same size, and not all of them connect to each other the same way. People talk about "tiers" of ISPs:

- **Tier 1** is a network so big it doesn't have to *pay* anyone for transit. They're the giant intercontinental backbones — the **highway operators** of the internet. Examples: Lumen (formerly CenturyLink/Level3), Telia, NTT, Cogent, Tata, Zayo. They peer with each other for free because they each carry so much traffic that it's mutually beneficial.
- **Tier 2** is a regional or national ISP that pays a tier-1 to reach the parts of the world they don't connect to directly. They're the **city-street networks** — large enough to handle a city or a country, but they still rent highway access from somebody bigger. Examples: most national consumer ISPs.
- **Tier 3** is an end-customer ISP, often very local, that buys all of its transit from upstreams. They're the **last mile, the driveway** — the company that runs the cable into your house.

Your packet's journey from your laptop to a server might look like:

> **driveway (your home ISP, tier 3)** → city street (regional ISP, tier 2) → highway (tier 1 backbone) → city street (destination's tier 2) → driveway (destination's tier 3) → server.

BGP is what stitches all those tiers together. Every "→" arrow above is a BGP-speaking border router making a decision about where the packet goes next.

### So what does BGP actually *do* for me?

Honestly: nothing, directly. You'll never type a BGP command yourself unless you operate a network. But every single packet you send to a server outside your ISP gets where it's going because thousands of BGP routers cooperated to route it. **If BGP breaks, the internet breaks.** When Facebook went dark for six hours in October 2021, it was a BGP misconfiguration. Their routes got withdrawn from the global internet, and just like that, nobody could find them. Their employees couldn't even badge into the office because the badge system needed DNS, which needed BGP.

That's the kind of importance we're talking about.

### BGP runs on trust (mostly)

The single weirdest thing about BGP is: **it runs on trust.** When a network says "I can reach this prefix," its neighbors just kind of... believe it. They might filter the announcement, they might drop ones that look obviously wrong, but the protocol itself doesn't have built-in cryptographic proof. That's why bad BGP announcements (accidental or malicious) can briefly black-hole big chunks of the internet. We have layered fixes on top now (RPKI, ROAs, BGPsec) but the underlying conversation is still very much "I'm gonna take your word for it."

Hold onto that fact — it explains a lot of weird BGP stories.

## How BGP Actually Talks

### Routers, not people

The conversation we're describing isn't between humans. It's between two **routers** — specifically, the border routers at each network. We call them **peers** or **neighbors** when they have a BGP session with each other.

Two BGP routers form a **session** with each other. A session is just a long-running TCP connection over which they exchange messages. As long as the session is up, they keep each other informed. When the session goes down, all the routes they learned from each other are pulled out of use.

### TCP port 179

BGP runs on **TCP port 179**. Every BGP session is a TCP connection from one router to the other on port 179. That's important for a few reasons:

- It's a real TCP connection, so it gets reliable delivery, congestion control, all the good stuff TCP gives you.
- If a firewall in between blocks port 179, BGP can't form. (This is a common cause of stuck sessions.)
- Because it's TCP, you can capture it with tools like `tcpdump` and `wireshark`.

A session is **always between exactly two routers**. There's no broadcasting, no multicast — just a one-on-one private conversation between two configured neighbors.

### Two neighbors agreeing to share their address books

The simplest way to picture a BGP session is: imagine two next-door neighbors meeting at the fence. They've decided to swap address books. Each one has a list of addresses they know how to reach (people they know, or people their other friends know). They tell each other.

After they swap, each neighbor knows not just their own addresses, but every address their friend knows too. And they tag each one with "you can reach this person *through me*."

That's a BGP session in a sentence.

### Adjacency states (the sleepy handshake walkthrough)

Two routers don't go from "have never spoken" to "fully sharing routes" in one step. They go through a state machine first. Here's the official sequence:

```
Idle → Connect → Active → OpenSent → OpenConfirm → Established
```

Let's walk through it like two sleepy friends agreeing to be pen pals.

**1. Idle** — "I'm thinking about it." The router has the neighbor configured but isn't actively trying to talk yet. It might be waiting for the network to come up, or for a hold-down timer to expire, or it just got told to be quiet.

**2. Connect** — "Trying to dial." The router is opening a TCP connection to the neighbor on port 179. If the TCP three-way handshake completes, it moves to the next state. If it fails, it might fall back to **Active** to try again.

**3. Active** — "I'm actively trying to reach you, but you're not answering." This is the state most people misunderstand: **Active is bad**, not good. It means "I keep trying and not getting through." Common causes: firewall in between, neighbor isn't configured to expect this router yet, wrong IP address.

**4. OpenSent** — "I sent my hello." Once TCP is up, BGP sends an **OPEN** message containing its **Autonomous System Number** (ASN), its **router ID**, the version of BGP, the **hold timer** it would like, and the **capabilities** it supports.

**5. OpenConfirm** — "I got your hello, and I sent you a KEEPALIVE to confirm." Both sides like what they saw in the OPEN messages, so they each send a **KEEPALIVE** to confirm they're sticking with this session.

**6. Established** — "We're talking." Routes can now flow. UPDATE messages start being exchanged. KEEPALIVEs continue every 60 seconds (by default) so each side knows the other is still alive.

If anything goes wrong at any step, you get a **NOTIFICATION** message describing the problem and the session drops back to Idle, usually with a hold-down before it tries again.

```
   ┌──────┐
   │ Idle │◄──────────────┐ Hold time, NOTIFICATION,
   └──┬───┘                │ TCP close, manual reset
      │ start
      ▼
   ┌─────────┐    TCP up   ┌─────────┐
   │ Connect │────────────►│OpenSent │
   └─┬───────┘             └────┬────┘
     │ TCP fails                │ recv OPEN, OK
     ▼                          ▼
   ┌────────┐              ┌────────────┐
   │ Active │              │OpenConfirm │
   └────┬───┘              └────┬───────┘
        ▲ retry                 │ recv KEEPALIVE
        │                       ▼
        │                  ┌────────────┐
        └──────────────────┤ Established│  ← UPDATE, KEEPALIVE flow here
                           └────────────┘
```

### KEEPALIVE: the "still here?" tap on the shoulder

Once a session is **Established**, the two routers send each other a tiny **KEEPALIVE** message every 60 seconds (by default — it's actually 1/3 of the **hold timer**, which defaults to 180 seconds). If a router doesn't hear *anything* from the other side for a full hold timer, it assumes the neighbor is dead, sends a NOTIFICATION ("Hold time expired"), and tears the session down.

This is why bad TCP behavior (lots of retransmits, big RTT changes, MTU problems on the path) can cause BGP sessions to flap. If you can't get a KEEPALIVE through in 180 seconds, you're done.

### Why TCP and not UDP?

BGP needs reliable delivery. If a route announcement gets lost, the neighbors will have inconsistent maps. TCP gives you ordered, reliable, retransmitted, congestion-controlled delivery for free. UDP would force BGP to reinvent all of that in the protocol itself. So: TCP, port 179, that's the deal.

## What BGP Says

### The four message types

BGP only has **four** kinds of messages. That's it. The whole protocol fits on a postcard:

| Message | Purpose | When Sent |
|---|---|---|
| **OPEN** | Hello! Here's my ASN, router ID, and capabilities. | Once, at session start |
| **UPDATE** | Here are new routes / these routes are withdrawn. | Whenever routing changes |
| **KEEPALIVE** | I'm still alive! | Every ~60s while idle |
| **NOTIFICATION** | Something is wrong; I'm closing this session. | On error, then session tears down |

The interesting one is **UPDATE**. That's where all the actual routing information lives. The other three are basically session bookkeeping.

### The UPDATE message — "here's a parcel, with labels"

An UPDATE message is BGP saying *"hey, neighbor, here are some prefixes I can reach, and here's a bunch of metadata about them."* You can also stuff withdrawals into it: *"by the way, those prefixes I told you about yesterday? Forget those, I can't reach them any more."*

Imagine each route announcement is a parcel handed across the fence. The parcel has stickers and tags on it — those are the **path attributes** — and the contents of the parcel are the actual addresses being announced (the **NLRI**, "network layer reachability information," which is a fancy way of saying "the IP prefixes").

### Anatomy of an UPDATE

```
┌───────────────────────────────────────────────────────────┐
│ Marker (16 bytes, all 1s) + Length + Type=2 (UPDATE)      │
├───────────────────────────────────────────────────────────┤
│ Withdrawn Routes Length + Withdrawn Routes (prefixes)     │
│   "These routes? Forget them. Can't reach them anymore."  │
├───────────────────────────────────────────────────────────┤
│ Total Path Attributes Length                              │
├───────────────────────────────────────────────────────────┤
│ Path Attributes (the parcel labels):                      │
│   • ORIGIN (where did this route come from?)              │
│   • AS_PATH (which neighborhoods has this passed through?)│
│   • NEXT_HOP (where do I send the traffic?)               │
│   • LOCAL_PREF (how badly do *I* want this route?)        │
│   • MED (how badly does the *neighbor* want this?)        │
│   • COMMUNITIES (any tags/instructions?)                  │
│   • AGGREGATOR, ATOMIC_AGGREGATE, ORIGINATOR_ID, ...      │
├───────────────────────────────────────────────────────────┤
│ NLRI = Network Layer Reachability Information             │
│   = the actual prefixes being announced                   │
│   e.g., 8.8.8.0/24, 8.8.4.0/24                            │
└───────────────────────────────────────────────────────────┘
```

Let's go through the path attributes one by one. These are the parcel labels.

### AS_PATH — the breadcrumb trail

The **AS_PATH** is the most important attribute. It's the list of ASes the announcement has passed through, in order, with the **most recent first**.

When Google originates `8.8.8.0/24`:

> AS_PATH: `15169` (just Google)

When Comcast (AS 7922) receives that and re-announces it to AT&T:

> AS_PATH: `7922 15169` (Comcast prepends itself)

When AT&T (AS 7018) receives that and re-announces it further:

> AS_PATH: `7018 7922 15169`

It's literally a breadcrumb trail. By the time a route reaches you across half the internet, the AS_PATH might be 4–7 ASes long.

**AS_PATH does two huge jobs at once:**

1. **Loop prevention.** If a router sees its own ASN already in the AS_PATH of an announcement, it rejects the announcement. Done. No loops are possible. Compare to OSPF, which has to compute Dijkstra over the full link-state database to avoid loops. AS_PATH is way simpler.
2. **Path length.** Shorter AS_PATH = "fewer neighborhoods to cross" = (usually) preferred. We'll see this in the decision algorithm.

### NEXT_HOP — "where do I actually send the packet?"

The **NEXT_HOP** attribute says *"the IP address you should forward traffic to in order to reach this prefix."* This is the IP of the router that will *take it from here.*

Sometimes NEXT_HOP is your direct neighbor's interface IP. Sometimes (for iBGP) it's some far-away router inside the same AS. Either way, your routing table needs to know how to reach that NEXT_HOP IP itself, or the route is unusable.

**Common confusion:** "Why can't I reach this prefix even though BGP shows it as best?" Answer: usually the NEXT_HOP isn't reachable via your IGP. The route shows up but stays unusable.

### LOCAL_PREF — "how badly *I* want this route"

**LOCAL_PREF** is a number you set inside your AS to say *"I prefer this route over other routes for the same prefix."* Higher = more preferred. It's strictly local — never leaves your AS.

Use it to express business decisions. "Always prefer routes through customers over routes through transit providers." That's two route-maps and a local-pref bump. Boom.

Default: 100. Most operators move customer routes to 200, peer routes to 150, transit routes to 100.

### MED — "how badly *I* want *you* to use a particular entry point"

**MED** stands for "Multi-Exit Discriminator." It's how a neighbor *suggests* which of several entry points into their network you should use.

If AS A and AS B have two different links between them (say in New York and in San Francisco), AS A can attach a low MED to the routes it advertises out of NY and a high MED to the routes it advertises out of SF, telling AS B *"please prefer my NY entry."*

Lower MED is preferred (the opposite of LOCAL_PREF). It's a *suggestion* — by default it only matters when comparing routes from the *same neighbor AS*, not across different neighbors.

### COMMUNITIES — "tags on the parcel"

**Communities** are little tags you stick on a route. They don't *mean* anything to the protocol itself — they're just numeric labels that your network operator (and yours, and yours) agree on. They're how policy travels.

Examples:

- `64500:1000` might mean "this is a customer route" inside your AS.
- `64500:666` might mean "blackhole this prefix on my edge" (DDoS scrub).
- `NO_EXPORT` (a well-known one): "don't re-advertise this to eBGP peers."
- `NO_ADVERTISE`: "don't advertise this to anyone."
- `BLACKHOLE` (RFC 7999): "drop traffic to this prefix; I'm under attack."

A route can carry many communities at once. This is how complex inter-domain policy gets expressed.

### ORIGIN — "where did this route originally come from?"

**ORIGIN** has three values:

- `IGP` — the originator learned it from inside its own network (most common).
- `EGP` — learned from EGP, the dinosaur ancestor of BGP. You will never see this in the wild.
- `Incomplete` — learned by some other means (usually static routes redistributed in).

It's almost a vestigial attribute. The decision process uses it as a tiebreaker between equally-good routes. IGP > EGP > Incomplete.

### A worked example

Google originates `8.8.8.0/24`. The first announcement, leaving Google, looks (roughly) like:

```
UPDATE
  NLRI:        8.8.8.0/24
  AS_PATH:     15169
  NEXT_HOP:    <google's edge router IP>
  ORIGIN:      IGP
  COMMUNITIES: 15169:1234 (some internal tag)
  MED:         50
```

Comcast receives that, prepends itself, swaps the NEXT_HOP to its own router (because eBGP), and ships it out:

```
UPDATE
  NLRI:        8.8.8.0/24
  AS_PATH:     7922 15169
  NEXT_HOP:    <comcast edge router IP>
  ORIGIN:      IGP
  COMMUNITIES: 7922:200 15169:1234 (added a community)
  MED:         (typically reset / not propagated)
```

AT&T sees both Comcast's announcement and Level3's announcement. Now AT&T has to pick one. Time for the decision algorithm.

## How Routers Pick the Best Path

### "GPS picking the fastest route"

Your car GPS, when it sees multiple ways to get to a destination, ranks them by some criteria — fastest, shortest, fewest tolls, no highways, whatever. BGP does exactly the same thing, but the criteria are different and they're applied **in strict order**. The first criterion that produces a clear winner wins. Ties fall through to the next criterion.

This is called the **BGP best-path selection algorithm** (or "the decision process"). It's the same on every router that speaks BGP, though some vendors add a couple of vendor-specific steps near the top.

### The decision process, in order

| # | Step | Prefer | Plain English |
|---|---|---|---|
| 1 | **Weight** (Cisco-only) | Highest | "I personally really want this one." Local to one router; never leaves it. |
| 2 | **LOCAL_PREF** | Highest | "My whole AS prefers this." |
| 3 | **Locally originated** | Yes > No | "I myself originated this; I trust it most." |
| 4 | **AS_PATH length** | Shortest | "Fewer neighborhoods to cross." |
| 5 | **ORIGIN** | IGP > EGP > Incomplete | Where the route originally came from. |
| 6 | **MED** | Lowest | The neighbor's suggestion about which entry point. |
| 7 | **eBGP over iBGP** | eBGP | "External neighbor knows better than internal neighbor." |
| 8 | **IGP cost to NEXT_HOP** | Lowest | "Closest exit door." (a.k.a. hot potato) |
| 9 | **Oldest route** | Yes | "This route's been stable longest." (some vendors) |
| 10 | **Lowest router ID** | Lowest | Coin flip. |
| 11 | **Lowest neighbor IP** | Lowest | Final coin flip. |

If two routes tie all the way down, you get one of them deterministically (and increasingly, both of them with multipath enabled).

### How it actually feels

99% of the time, the decision comes down to **steps 2 and 4**. LOCAL_PREF is how operators express *policy* (business decisions, security, "I never want to use that ISP for video"), and AS_PATH length is how the protocol naturally finds short paths. Almost everything below MED is a tiebreaker for routes that already look identical.

### A concrete example with our three networks

You're AT&T, you have two routes to Google's `8.8.8.0/24`:

```
Path A: via Comcast (AS_PATH 7922 15169) -- learned from eBGP peer 1.2.3.4
Path B: via Level3  (AS_PATH 3356 15169) -- learned from eBGP peer 5.6.7.8
```

- Step 1 (Weight): both default. Tie.
- Step 2 (LOCAL_PREF): assume both default to 100. Tie.
- Step 3 (Locally originated): no, neither. Tie.
- Step 4 (AS_PATH length): both are 2. Tie.
- Step 5 (ORIGIN): both IGP. Tie.
- Step 6 (MED): suppose Comcast advertises MED 50, Level3 advertises MED 100. Comcast wins.

Done. AT&T installs the Comcast path as best. If Comcast's route gets withdrawn (link fails), the Level3 path takes over and *that* becomes best.

### What if I want to force a different decision?

You bend the higher-priority knobs. If you want Level3's path to be preferred, the cheapest way is to bump LOCAL_PREF on it via inbound policy: "any route I receive from Level3 gets LOCAL_PREF 200." Now step 2 produces a clear winner before we ever get to MED.

That's why operators talk about LOCAL_PREF a lot — it's the strongest knob you have over inbound traffic decisions.

### "Hot potato" routing

When step 8 kicks in (IGP cost to NEXT_HOP), what happens is: when two paths look equal otherwise, the router prefers the one whose NEXT_HOP is *closest inside its own network*. That means it wants to dump the packet on a peering link as soon as possible — like a hot potato. Cheap for the router, but maybe not optimal for the user (the packet might cross more of the *destination's* network).

The opposite is **cold-potato** routing: an AS prefers to carry traffic on its own network as far as possible before handing it off, to give users a better experience. Cold-potato is more cooperative, more expensive, but generally yields lower latency for end users. Tier-1s and CDNs often play cold-potato.

## iBGP vs eBGP

### "Roommate vs neighbor down the street"

There are exactly two flavors of BGP session, depending on whether the two routers are in the **same** AS or **different** ASes.

- **eBGP** (external BGP) — between *different* ASes. Like talking to your neighbor across the fence. This is what you imagine when you think "BGP."
- **iBGP** (internal BGP) — between routers *inside the same* AS. Like talking to your roommate. Same household, just need to keep each other in the loop about what mail came in.

They're the same protocol, just configured differently and with different rules.

### Differences

| | eBGP | iBGP |
|---|---|---|
| Between | Different AS numbers | Same AS number |
| AS_PATH | Prepends own ASN before advertising | Does NOT change AS_PATH |
| NEXT_HOP | Set to advertising router's IP | Preserved (must be reachable via IGP) |
| Default TTL | 1 (directly connected) | 255 (multihop OK) |
| Topology | Peer with the neighbors you have | Logically full-mesh required (or RR) |
| Usual use | Talk to other organizations | Distribute external routes inside your own AS |

### Why iBGP needs a full mesh

Here's the gotcha: **iBGP does not re-advertise routes learned from one iBGP peer to another iBGP peer.** This is a loop-prevention rule (because AS_PATH doesn't grow inside an AS, you can't use it to detect loops there). The consequence is that for every iBGP router to know about every external route, every iBGP router must peer directly with every other.

That's a **full mesh**. Every router talks to every other.

If you have N iBGP routers, that's N × (N-1) / 2 sessions:

- 5 routers → 10 sessions
- 10 routers → 45 sessions
- 100 routers → 4,950 sessions
- 500 routers → 124,750 sessions

This obviously doesn't scale. There are two well-known scaling tricks: **route reflectors** and **confederations**.

## Route Reflectors and Confederations

### Route reflectors (RRs) — "the office gossip"

A **route reflector** is an iBGP router that *bends the rules.* Specifically, it's allowed to **re-advertise** routes learned from one iBGP peer to another iBGP peer, *as long as* it's been configured as an RR.

So instead of every router talking to every router, every router talks to one (or two, for redundancy) **reflectors**, and the reflectors gossip everything they hear to everyone else.

```
       ┌──────────┐
       │   RR1    │            ┌──────────┐
       └────┬─────┘            │   RR2    │
            │                  └────┬─────┘
   ┌────────┼──────────┐            │
   │        │          │   ┌────────┼──────────┐
   ▼        ▼          ▼   ▼        ▼          ▼
 ┌────┐  ┌────┐    ┌────┐  ┌────┐  ┌────┐    ┌────┐
 │ R1 │  │ R2 │    │ R3 │  │ R1 │  │ R2 │    │ R3 │
 └────┘  └────┘    └────┘  └────┘  └────┘    └────┘
   (R1, R2, R3 are RR clients; they peer ONLY with the reflectors)
```

100 routers + 2 RRs → 200 sessions instead of 4,950. Massive scale win.

The "office gossip" analogy is dead-on. Everyone tells the gossip what they heard. The gossip tells everybody else. Nobody has to talk directly to anyone but the gossip.

To prevent loops between *reflectors* (yes, this can happen), BGP adds two attributes:

- **ORIGINATOR_ID** — the router ID of the original iBGP source.
- **CLUSTER_LIST** — the chain of RR clusters the route has crossed.

If a router sees its own ORIGINATOR_ID or a CLUSTER_ID it's already part of, it drops the route. Loop avoided.

### Confederations — "the city splits into HOA districts"

A **confederation** is a way to chop a big AS into a few smaller "sub-ASes," each of which runs full-mesh iBGP internally. The sub-ASes use a slightly modified eBGP between each other inside the confederation.

To the outside world, the whole confederation still looks like one AS (the "AS confederation identifier"). But internally, you've divided the work.

```
  ┌── Confederation: AS 65001 (public-facing) ────────────────┐
  │                                                            │
  │   Sub-AS 65111            Sub-AS 65112      Sub-AS 65113   │
  │   ┌──┬──┬──┐              ┌──┬──┬──┐         ┌──┬──┬──┐   │
  │   │R1│R2│R3│ ◄─intra-conf►│R4│R5│R6│ ◄─intra-conf► │R7│R8│R9│   │
  │   └──┴──┴──┘              └──┴──┴──┘         └──┴──┴──┘   │
  │                                                            │
  └────────────────────────────────────────────────────────────┘
                                    ↕  eBGP
                               (real outside world)
```

Each sub-AS only has to full-mesh internally. The sub-ASes peer with each other using "intra-confederation eBGP" sessions. The big advantage: you keep all the eBGP-style policy controls (LOCAL_PREF, communities, route maps) at sub-AS boundaries.

Confederations are less fashionable now than RRs because RRs are simpler to operate. Big networks tend to use RRs (often two layers of them); some legacy networks still use confederations. Many large operators end up using both: confederations to chop the network, RRs inside each sub-AS.

## Communities and Policy

### Communities = secret tags on a parcel

A **community** is just a numeric label you can attach to a route. The protocol does nothing with it directly; it's a way for operators to coordinate. Examples:

- *"Don't ship this to Asia."* You configure the community `65000:65521` to mean "don't export to Asian peers." Your edge routers honor it.
- *"This is a discount class — don't use expensive transit for it."* Tag `65000:42` means "drop priority on this." Your queueing knows to deprioritize.
- *"Customer-originated. LOCAL_PREF 200."* Tag `65000:100`. Your iBGP imports map it.

There are three kinds of communities you'll meet:

- **Standard communities**: `ASN:value`, like `65000:100`. Two 16-bit halves. Most-used.
- **Extended communities**: structured tags, like Route Targets used in MPLS L3VPN and EVPN (`rt:65000:1`).
- **Large communities** (RFC 8092): `ASN:value:value`, three 32-bit halves. Required for 4-byte ASNs to be expressible cleanly.

### Well-known communities

A handful of communities have agreed-upon meanings:

- `NO_EXPORT` — "Do not advertise this beyond my AS." Used to keep a route inside the local provider only.
- `NO_ADVERTISE` — "Do not advertise this to *any* peer." Effectively a private-only route.
- `LOCAL_AS` — "Do not advertise outside my confederation." (RFC 1997)
- `BLACKHOLE` (RFC 7999) — "Drop all traffic to this prefix." Used during DDoS attacks: announce your attacked /32 with this community to your upstream and they'll null-route it for you. You lose access to that one IP, but you save the rest.

### RPKI / ROA — "the notary service"

When a network announces a prefix, how do you know they're allowed to? Historically: you don't, you just trust them. That's how BGP hijacks work. Some bad actor (or fat-fingered operator) announces someone else's prefix, and traffic for it suddenly flows the wrong way.

**RPKI** (Resource Public Key Infrastructure) and **ROAs** (Route Origin Authorizations) fix this with a notary service. Roughly:

- The legitimate holder of an IP prefix signs a **ROA** that says "AS 65001 is authorized to originate `2001:db8::/48`." This signed object is published in the global RPKI repository.
- Routers around the world run an **RPKI validator** that downloads ROAs and checks BGP announcements against them.
- When an announcement comes in, the validator marks it **Valid**, **Invalid**, or **NotFound** (no ROA exists for the prefix at all).
- Operators write policy: drop Invalid, prefer Valid, accept NotFound (for now, while the world catches up).

```
   ┌──────────┐    publish    ┌────────────────┐
   │ Prefix   │──────────────►│ RPKI Repository│
   │ Holder   │               └───────┬────────┘
   └──────────┘                       │ rsync/RRDP
                                      ▼
                              ┌─────────────┐
                              │  Validator  │
                              │ (Routinator,│
                              │  Fort, etc.)│
                              └──────┬──────┘
                                     │ RTR (port 323)
                                     ▼
                              ┌─────────────┐
                              │   Router    │
                              │ (drops      │
                              │  RPKI       │
                              │  invalids)  │
                              └─────────────┘
```

This is one of the few real defenses against accidental and malicious BGP hijacks. RPKI adoption has grown massively since 2019; as of the mid-2020s, roughly half of all announcements are covered.

A related, lighter-weight scheme is **ASPA** (RFC 9582), which validates *provider-customer relationships* rather than per-hop signatures. It's gaining traction because BGPsec turned out to be too heavy.

### BGPsec — the heavyweight cousin

**BGPsec** (RFC 8205) extends RPKI further: each AS along the path *cryptographically signs* its hop in the AS_PATH. Receivers can verify the entire chain. This catches *path manipulation* attacks where someone forges an AS_PATH.

It works in theory. In practice it's barely deployed (<1% of ASes) because:

- You can't aggregate signed paths.
- Signature verification is expensive (~1000/sec in software).
- One unsigned hop breaks the chain — useless if your peers don't deploy.

The industry has mostly moved on to **ASPA** for path validation as a more pragmatic alternative.

## When BGP Goes Wrong

### Route flap

A **route flap** is when a route is repeatedly going up, down, up, down. Each flap triggers a fresh round of UPDATEs across the internet. If lots of routes are flapping, the world's routers spend their CPU re-running the decision process instead of forwarding packets.

Historically, BGP had **route flap dampening**: when a route flapped too much, your router would *suppress* it for a while, refusing to use it even if it came back up. The math is exponential decay — each flap adds "penalty," and the route is re-introduced once the penalty decays below a threshold.

Modern best practice: dampening is mostly *off* for IPv4 unicast (caused too many problems). Use BFD (sub-second failure detection) instead of dampening.

### BGP hijacks (the YouTube/Pakistan 2008 story)

In February 2008, the government of Pakistan ordered ISPs to block YouTube. Pakistan Telecom (AS 17557), trying to do this, configured their routers to announce a more-specific of YouTube's prefix (`208.65.153.0/24`) into BGP — intending to black-hole it inside their own network.

But they didn't filter the announcement on egress. So they announced `208.65.153.0/24` to their upstream provider, **PCCW Global (AS 3491)**. PCCW didn't filter either, and the announcement leaked to the global internet.

`208.65.153.0/24` is *more specific* than YouTube's actual `208.65.152.0/22`. Most-specific match wins. Within minutes, every ISP on the planet thought the way to reach YouTube was through Pakistan Telecom — which then black-holed it. YouTube went dark globally for about two hours.

That's a textbook **sub-prefix hijack**. The fix: better egress filtering, RPKI-based filtering by upstreams, max-prefix-length checks in ROAs. RPKI would have rejected the bogus announcement immediately because no ROA authorized AS 17557 to originate any part of YouTube's space.

### Route leaks

A **route leak** is when an AS re-advertises routes to the wrong neighbor. Classic example: a customer (small ISP) re-advertises routes from one transit provider to another transit provider. Suddenly that customer becomes the apparent path between two huge backbones — and gets *all* the traffic. Usually their links melt.

The August 2017 incident where AS 7018 (Verizon) accidentally accepted leaked routes from AS 6724 (Hathway) routed Verizon's US east-coast traffic through India for a few hours. Same shape as the YouTube hijack, but with leaks instead of straight hijacks.

Mitigation: **RFC 9234 BGP Roles** and the **Only-to-Customer (OTC)** attribute. Sessions are tagged with their commercial role (provider/customer/peer) and the protocol enforces "valley-free routing" automatically.

### Black-holing (good kind and bad kind)

- **Bad kind:** misconfigured prefix announced as "go nowhere" by mistake.
- **Good kind:** you're being DDoSed at IP `203.0.113.5`. You announce `203.0.113.5/32` to your upstream with the **BLACKHOLE** community (RFC 7999). Your upstream null-routes that one IP, the attack stops, the rest of your network keeps working. You sacrificed one IP to save 65,535 others.

### Convergence is slow

When a link fails, BGP can take a *long time* to converge — minutes, not seconds. The reason is the **MRAI timer** (Minimum Route Advertisement Interval, default 30 seconds for eBGP) plus path-exploration: routers test all possible paths before settling on the new winner.

Mitigations to make convergence faster:

- **BFD** (Bidirectional Forwarding Detection): sub-second link failure detection. The BGP session goes down 150ms after the link does, instead of waiting up to 180s for the hold timer.
- **Reduced MRAI**: 5s or even 0 for some scenarios.
- **Add-Path** (RFC 7911): advertise multiple paths per prefix so backups are pre-installed.
- **BGP PIC (Prefix Independent Convergence)**: pre-install backup next-hops in the FIB so failover is instant.
- **Graceful Restart** (RFC 4724): keep forwarding while the BGP daemon restarts.

### "It says I have a route, but traffic doesn't go that way."

Common, painful. Usually one of:

- **NEXT_HOP unreachable.** The route is in the BGP table but not in the FIB.
- **Route map filtering on egress.** You see it locally, but you're not advertising it to the neighbor you think you are.
- **iBGP without NEXT_HOP-self.** Your border learned it eBGP, with NEXT_HOP being the eBGP peer's IP. iBGP propagates that NEXT_HOP unchanged. None of your other routers can reach that IP. They drop the route.

The fix to the last one: `next-hop-self` on the iBGP session, or an IGP route to the eBGP peer's network.

## Stories: Famous BGP Outages

### The Facebook outage of October 2021

On October 4, 2021, at around 15:39 UTC, Facebook (Meta), Instagram, WhatsApp, Oculus, and several other Meta products *vanished from the internet* for about six hours. Not "slow." Not "intermittent." Gone. DNS resolution for `facebook.com` returned `SERVFAIL`. Connection attempts failed instantly. Even Facebook's internal systems went dark — engineers couldn't badge into the office because the badging system depended on DNS, which depended on BGP, which had just yeeted Facebook off the internet.

The cause: a routine maintenance change. An engineer at Facebook ran a command intended to assess available capacity on the global backbone. A bug in the audit tool didn't catch a faulty parameter, and the command effectively withdrew **all** of Facebook's BGP announcements from the global routing table. Including the prefixes hosting Facebook's authoritative DNS servers.

Once the DNS servers' prefixes were withdrawn, no one could resolve `facebook.com`. Without resolution, no one could reach anything Facebook-related. Without DNS, even Facebook's *internal* tools — including the ones used to fix BGP — couldn't talk to each other. Engineers had to physically drive to data centers and log into routers via console. Six hours of downtime later, BGP was reconfigured and announcements came back.

The lesson: **BGP is fundamental.** Even Facebook, with thousands of network engineers and gold-plated automation, can take itself off the internet with one bad command. Anything you build at the edge of an AS needs validation, dry runs, and an out-of-band fallback that doesn't rely on the very thing you're touching.

### The Pakistan/YouTube hijack of 2008

We mentioned this one above but it's worth telling in full. On Sunday, February 24, 2008, the Pakistan Telecommunications Authority issued a directive to ISPs in Pakistan: block YouTube. The reason was a video they considered offensive. The classic, low-tech way to "block" a website is to install a more-specific route in your network pointing to nowhere — a black hole.

Pakistan Telecom, AS 17557, dutifully pushed a route for `208.65.153.0/24` (a more-specific of YouTube's `208.65.152.0/22`) into their internal network, with the next-hop being `Null0` — drop. Inside Pakistan, this worked: YouTube was blocked.

But Pakistan Telecom didn't filter their announcements on egress. So they advertised `208.65.153.0/24` to their upstream provider, **PCCW Global (AS 3491)**. PCCW didn't filter incoming announcements from their customer either. So PCCW propagated the bogus announcement to its peers and customers — and within minutes, the more-specific `/24` was being heard everywhere on the global internet.

Routers prefer the more-specific match. So instead of sending YouTube traffic to `AS 36561` (YouTube's actual ASN), the world's routers started sending it to `AS 17557` (Pakistan). Where it was promptly black-holed. YouTube went dark globally for about two hours.

Total time-to-detect: minutes. Total time-to-fix: about an hour while engineers at YouTube announced more-specific /25s to win the longest-prefix-match battle, and PCCW was poked into filtering their downstream.

The fixes that came out of this:

- Strict egress filtering at every level: customers should never advertise routes that aren't theirs.
- IRR-based prefix filtering: upstream providers should accept only the prefixes their customer has registered.
- RPKI: cryptographic verification that an AS is allowed to originate a prefix.
- Maximum-length attributes in ROAs: stop sub-prefix hijacks even from the legitimate origin AS.

Today, RPKI alone would have rejected the announcement. There's no ROA authorizing AS 17557 to originate any part of YouTube's space. A router with `match rpki invalid` policy in its imports would have dropped it on sight.

### The Verizon route leak of 2019

In June 2019, a small Pennsylvania ISP (DQE Communications, AS 33154) leaked tens of thousands of routes from one of their transits (Verizon, AS 701) back to *another* of their transits (Verizon... again, accidentally). Verizon accepted the leak. For a couple of hours, a chunk of the internet thought the way to reach Cloudflare, Amazon, Linode, and many others was through the small Pennsylvania ISP.

That ISP did not have the capacity. Their links saturated, packet loss spiked, and a noticeable portion of the global internet had a bad afternoon.

The fix here is RFC 9234 BGP Roles + the OTC attribute. With session roles configured, the leak from a customer to a peer would have been automatically marked invalid and dropped. Operational adoption of OTC has been slow but steady; we're still living with route leaks until coverage is widespread.

### The Mai 2024 Ukrainian fiber cut and the convergence reality check

In early May 2024, several major fiber cuts in Ukraine, combined with simultaneous DDoS attacks, caused a wave of BGP withdrawals. Cloudflare and other CDNs reported convergence times in the order of seconds — because they run BFD, Add-Path, and tight MRAI. Smaller networks without those mitigations saw multi-minute traffic loss. The lesson: BGP convergence is a *configurable* thing. Defaults are slow; tuned BGP is fast.

## ASCII Field Guide

### A typical eBGP/iBGP topology

```
                     The Internet (other ASes)
                              │
                              │  eBGP
                              ▼
                  ┌──────────────────────┐
                  │  Edge Routers (E1)   │  ← peers with upstreams via eBGP
                  └─────┬────────────┬───┘
                        │ iBGP       │ iBGP
                        ▼            ▼
                ┌────────────┐  ┌────────────┐
                │ Core (C1)  │◄►│ Core (C2)  │ ← full-mesh iBGP
                └─────┬──────┘  └──────┬─────┘
                      │ iBGP           │ iBGP
                      ▼                ▼
         ┌────────────┐    ┌────────────┐    ┌────────────┐
         │ Aggreg A1  │    │ Aggreg A2  │    │ Aggreg A3  │
         └────┬───────┘    └─────┬──────┘    └─────┬──────┘
              │                  │                 │
              ▼                  ▼                 ▼
          customer            customer          customer
```

The edge routers speak eBGP to upstream providers. Internally, every router speaks iBGP to every other (or, more practically, all of them peer with two route reflectors). Outside-world routes flow in via eBGP, get propagated through iBGP, and reach every router in the AS.

### A simplified BGP UPDATE on the wire

```
+-------------------+
| Marker (16 bytes) |   = all 1s, legacy auth field
+-------------------+
| Length (2 bytes)  |   = total UPDATE length
+-------------------+
| Type (1 byte) = 2 |   = UPDATE message type
+-------------------+
| Withdrawn Routes  |   length + list of prefixes
| Length (2 bytes)  |
| Withdrawn Routes  |
+-------------------+
| Total Path Attr   |   length of the attributes block
| Length (2 bytes)  |
+-------------------+
| Path Attributes   |
|                   |
|   Each attribute: |
|   +-------------+ |
|   | Flags  (1B) | |   well-known/optional, transitive,
|   +-------------+ |   partial, extended length
|   | Type   (1B) | |
|   +-------------+ |
|   | Length(1-2) | |
|   +-------------+ |
|   | Value       | |
|   +-------------+ |
+-------------------+
| NLRI              |   the actual prefixes being announced
| (variable)        |
+-------------------+
```

### Decision algorithm flowchart

```
     Receive UPDATE for prefix P
                │
                ▼
       Pass inbound policy?
        │              │
       no             yes
        │              │
   discard       Apply attributes
                       │
                       ▼
   Multiple paths to P? ──── no ──► install as best
        │
       yes
        │
        ▼
  +-----------------+
  | Step 1: Weight  |  highest wins
  +-----------------+
        │ tie
        ▼
  +-----------------+
  | Step 2: LOCAL_  |  highest wins
  | PREF            |
  +-----------------+
        │ tie
        ▼
  +-----------------+
  | Step 3: locally |  yes wins
  | originated      |
  +-----------------+
        │ tie
        ▼
  +-----------------+
  | Step 4: AS_PATH |  shortest wins
  | length          |
  +-----------------+
        │ tie
        ▼
  +-----------------+
  | Step 5: ORIGIN  |  IGP > EGP > Incomplete
  +-----------------+
        │ tie
        ▼
  +-----------------+
  | Step 6: MED     |  lowest wins (only if from same AS)
  +-----------------+
        │ tie
        ▼
  +-----------------+
  | Step 7: eBGP    |  eBGP > iBGP
  | over iBGP       |
  +-----------------+
        │ tie
        ▼
  +-----------------+
  | Step 8: IGP     |  lowest IGP cost wins (hot potato)
  | cost to NEXT_HOP|
  +-----------------+
        │ tie
        ▼
  +-----------------+
  | Step 9: oldest  |  most stable wins
  +-----------------+
        │ tie
        ▼
  +-----------------+
  | Step 10: lowest |  router ID, then neighbor IP
  | router ID       |
  +-----------------+
        │
        ▼
   install best path
```

### Route reflector cluster

```
                    ┌──────────────┐
                    │ RR1 (cluster │
                    │   ID: 1.1.1) │
                    └────┬────┬────┘
                         │    │
        ┌────────────────┘    └─────────────────┐
        │                                       │
        ▼                                       ▼
  ┌───────────┐  ┌───────────┐  ┌───────────┐  ┌───────────┐
  │ Client R1 │  │ Client R2 │  │ Client R3 │  │ Client R4 │
  └───────────┘  └───────────┘  └───────────┘  └───────────┘
                          │
                          │ optionally peers also with...
                          ▼
                    ┌──────────────┐
                    │ RR2 (cluster │
                    │   ID: 2.2.2) │
                    └──────────────┘

  • RR1 and RR2 in different clusters peer with each other (non-client iBGP).
  • Each client peers with both RRs for redundancy.
  • RRs propagate routes among themselves and to all clients.
  • CLUSTER_LIST attribute prevents RR-to-RR loops.
  • ORIGINATOR_ID attribute prevents same-cluster loops.
```

### RPKI validation flow

```
   ┌──────────────┐ creates/signs ┌──────────────┐
   │ Prefix holder│──────────────►│   ROA        │  AS X may originate
   │ (resource    │               │  (signed)    │  prefix P, max-length L
   │  cert)       │               └──────┬───────┘
   └──────────────┘                      │ publishes
                                         ▼
                              ┌────────────────────┐
                              │  RPKI Repository   │  rsync / RRDP
                              │  (CA hierarchy)    │
                              └─────────┬──────────┘
                                        │ pulled by
                                        ▼
                              ┌────────────────────┐
                              │   Validator        │  Routinator,
                              │ (verifies signing, │  Fort, OctoRPKI
                              │  produces VRPs)    │
                              └─────────┬──────────┘
                                        │ RTR (port 323)
                                        ▼
                              ┌────────────────────┐
                              │      Router        │
                              │  evaluates UPDATEs │
                              │  Valid/Invalid/    │
                              │  NotFound          │
                              └────────────────────┘
                                        │
                                        ▼
                              import policy:
                                set local-pref by validity
                                drop invalids
```

### State machine (full)

```
                  ┌──────┐
                  │ Idle │
                  └──┬───┘
              start │
                    ▼
                ┌─────────┐
                │ Connect │ ← TCP not yet up
                └────┬────┘
       TCP fail │       │ TCP up
                ▼       ▼
            ┌────────┐ ┌────────┐
            │ Active │ │OpenSent│
            └───┬────┘ └────┬───┘
                │ retry     │ recv valid OPEN
                ▼           ▼
            (back to    ┌────────────┐
             Connect)   │OpenConfirm │
                        └────┬───────┘
                             │ recv KEEPALIVE
                             ▼
                        ┌────────────┐
                        │ Established│ ← UPDATE / KEEPALIVE flow
                        └────────────┘
                             │
                             │ NOTIFICATION received,
                             │ TCP reset, hold timer expired,
                             │ manual reset
                             ▼
                          (Idle, possibly with hold-down before retry)
```

## Verbatim Errors and Their Fixes

### `%BGP-3-NOTIFICATION: sent to neighbor X.X.X.X 4/0 (hold time expired)`

**What it means:** You stopped hearing from your neighbor for the full hold timer (default 180s). You sent them a NOTIFICATION (code 4 = Hold Timer Expired, subcode 0) and dropped the session.

**Fix path:**

1. Is the link itself up? `ping X.X.X.X`. If you can't ping, fix the layer 1/2/3 issue.
2. Is TCP getting through? `ss -i | grep :179` — look at retrans counters; if they're climbing, you've got a TCP problem (MTU, packet loss).
3. Is BFD running? If yes, did BFD also drop? If only BGP dropped but BFD is fine, look at CPU on either router; the BGP daemon may be too busy to send KEEPALIVEs.
4. Is the path MTU too small? Try `ping X.X.X.X -M do -s 1472` to test.

### `%BGP-5-ADJCHANGE: neighbor X.X.X.X Down BGP Notification received`

**What it means:** The neighbor sent you a NOTIFICATION explaining why they're hanging up.

**Fix path:** Look at the next log line; it'll usually have the code/subcode. Common ones:

- `2/2` — peer reset (administrative)
- `2/6` — other configuration change
- `4/0` — hold time expired (their side)
- `6/2` — bad BGP identifier
- `6/4` — unsupported optional parameter

### `%BGP-3-NOTIFICATION: received from neighbor X.X.X.X 2/2 (peer in wrong AS)`

**What it means:** You configured the neighbor as remote-as `Y` but they identified themselves as remote-as `Z` in their OPEN message.

**Fix path:** Confirm the actual ASN of the neighbor (`whois -h whois.cymru.com " -v X.X.X.X"`). Update one side or the other. Both sides must agree.

### `%BGP-4-MAXPFX: No. of unicast prefix received from X.X.X.X has reached 80% of the configured limit`

**What it means:** You set a max-prefix limit (good!) and the neighbor is approaching it.

**Fix path:** Either it's normal growth (raise the limit), or the neighbor is leaking routes to you (filter or accept and investigate). Don't disable the limit — it's a critical safety net.

### `%BGP-5-ADJCHANGE: neighbor X.X.X.X Down Peer closed the session`

**What it means:** Their TCP closed cleanly (FIN, not RST).

**Fix path:** Probably a maintenance event on their side, or their daemon restarted. Watch for it to come back. If it doesn't, check whether the underlay link is actually up.

### `Neighbor capability has changed; resetting`

**What it means:** Either you or they renegotiated capabilities (e.g., enabled an additional address family). Capability change forces a session reset.

**Fix path:** Schedule capability changes for maintenance windows; expect a brief outage. `route refresh` (without dropping the session) avoids this for some changes.

### `Active`

(In `show bgp summary` output, where you'd usually see a number.)

**What it means:** The session is *not* up. Active means the BGP process is actively trying TCP but not getting through. Most common in: firewall blocks port 179; remote side hasn't configured this neighbor yet; routing problem so SYNs don't return.

**Fix path:** `tcpdump -nn port 179 host X.X.X.X` on both sides. If you see SYN from one side and no SYN-ACK back, network or firewall problem. If you see SYN both directions and RST, one side rejects the connection (usually wrong remote-as or no neighbor configured).

### `Idle (Admin)` or just `Idle`

**What it means:** Session is administratively shut down. Or the hold-down after a previous failure hasn't expired.

**Fix path:** `no neighbor X.X.X.X shutdown` or wait for the hold-down. If it's repeatedly returning to Idle right after Connect, that's a deeper problem; look at logs.

### `%BGP-3-NOTIFICATION: ... 6/2 (bad BGP identifier)`

**What it means:** Both BGP routers must have *unique* router IDs (a 32-bit number, by convention an IPv4 address). If you have two routers with the same router ID, they can't form a session.

**Fix path:** Use a unique loopback IP per router, set `bgp router-id <unique>` explicitly. Don't let it auto-pick from interface IPs in environments where IPs may collide.

### `%BGP-3-NOTIFICATION: ... 6/4 (unsupported optional parameter)`

**What it means:** You advertised a capability the neighbor doesn't understand (e.g., 4-byte ASN, MP-BGP for an address family they don't speak).

**Fix path:** Disable the unsupported capability for this neighbor, or upgrade the neighbor.

### `Hold time expired` shown in NOTIFICATION exchange

**What it means:** The classic "you didn't keepalive in time."

**Fix path:** As above for hold-time-expired logs. Also: consider running BFD with sub-second timers so you don't depend on the BGP hold timer for failure detection.

### `Connection collision` or `connection-collision`

**What it means:** Both routers tried to initiate a TCP connection to each other simultaneously and need to choose one.

**Fix path:** This is normal. The higher router-ID wins per RFC 4271. If you see it constantly, you might have a configuration that has both sides actively initiating and somehow not settling.

### `BGP_OPEN_DENIED`

**What it means:** Your router rejected an incoming OPEN, usually because of MD5 password mismatch, capability mismatch, or a deny in your ACL.

**Fix path:** Check `tcpdump` to see whether the OPEN is even arriving; look at `ip access-list` if you have one bound to BGP; double-check the MD5 password on both sides (passwords with whitespace get botched).

### `Bad TCP MD5` (or kernel: `MD5 hash mismatch`)

**What it means:** Either the TCP MD5 password is wrong on one side, or one side has a password and the other doesn't.

**Fix path:** Confirm both sides have the same `password X` configured for this neighbor. Linux kernel versions matter here too — TCP-MD5 wasn't always solid.

## Version Notes (Recent BGP Changes)

- **4-byte ASNs (RFC 6793, 2007):** ASNs are now 32-bit. Old routers see them as `23456` (the AS_TRANS placeholder). Modern routers reconstruct them from the AS4_PATH attribute. As of the mid-2020s, basically every public router supports them. If you see `23456` in real traffic today, something is *very* old and needs upgrading.
- **Add-Path (RFC 7911, 2016):** Lets a router advertise multiple paths per prefix. Critical for fast failover and load distribution. Requires negotiation in OPEN capabilities.
- **BGP Roles & OTC (RFC 9234, 2022):** Tags sessions with their commercial role and adds the Only-to-Customer attribute. Prevents route leaks at the protocol level. Adoption is still growing.
- **ASPA (RFC 9582, 2024):** Lighter-weight path validation than BGPsec. Tracks provider-customer relationships. Considered the practical replacement for BGPsec.
- **BGP Optimal Route Reflection (RFC 9107, 2021):** Lets RRs consider the IGP cost from each *client* to compute best path, so clients see closest-exit instead of RR-relative-best.
- **Large communities (RFC 8092, 2017):** Three 32-bit fields. Required for clean policy expression in 4-byte-ASN networks. You'll see them everywhere modern.
- **BLACKHOLE community (RFC 7999, 2016):** The well-known number is `65535:666`. Almost all transit providers honor it now for DDoS scrubbing.
- **BMP (RFC 7854, 2016):** BGP Monitoring Protocol — out-of-band streaming telemetry of BGP state. Most modern monitoring is BMP-based instead of polling `show bgp summary`.

## Hands-On

You can poke at BGP from a regular Linux box without operating any router yourself. Several of these commands need internet access or `sudo`. Expected output is what you'd see on a typical Linux system, real outputs vary.

**See your local IPv4 routing table:**

```
$ ip route show | head -10
default via 192.168.1.1 dev wlan0 proto dhcp src 192.168.1.42 metric 600
169.254.0.0/16 dev wlan0 scope link metric 1000
192.168.1.0/24 dev wlan0 proto kernel scope link src 192.168.1.42 metric 600
```

The first line is your default route — "if you don't know where to send it, send it to the gateway." That gateway is usually your home router; from there your traffic follows BGP-derived paths through your ISP and out into the world.

**See your IPv6 routing table:**

```
$ ip -6 route show | head -10
::1 dev lo proto kernel metric 256 pref medium
2600:1700:abc::/64 dev wlan0 proto ra metric 600 pref medium
fe80::/64 dev wlan0 proto kernel metric 1024 pref medium
default via fe80::1 dev wlan0 proto ra metric 600 pref medium
```

**See the same data via /proc on Linux:**

```
$ cat /proc/net/route | head -5
Iface   Destination     Gateway         Flags   RefCnt  Use     Metric  Mask            MTU     Window  IRTT
wlan0   00000000        0101A8C0        0003    0       0       600     00000000        0       0       0
wlan0   00000000        00000000        0001    0       0       1000    0000FFFF        0       0       0
```

(Hex numbers are little-endian — `0101A8C0` = `192.168.1.1`.)

**Trace where your packets go (asks each hop in turn for AS info, with `-A`):**

```
$ traceroute -A -n 8.8.8.8 | head -15
traceroute to 8.8.8.8 (8.8.8.8), 30 hops max, 60 byte packets
 1  192.168.1.1 [*]  0.42 ms  0.39 ms  0.38 ms
 2  10.0.0.1 [AS7922]  9.21 ms  9.19 ms  9.18 ms
 3  68.86.x.x [AS7922]  10.5 ms  10.4 ms  10.3 ms
 4  96.110.x.x [AS7922]  12.1 ms  12.0 ms  11.9 ms
 5  72.14.x.x [AS15169]  13.2 ms  13.1 ms  13.0 ms
 6  108.170.x.x [AS15169]  13.3 ms  13.2 ms  13.1 ms
 7  8.8.8.8 [AS15169]  13.4 ms  13.3 ms  13.2 ms
```

You can literally watch the AS numbers change as you cross network boundaries.

**Trace with running statistics (`mtr` is `traceroute` + `ping` in one):**

```
$ mtr -rwbz -c 5 google.com | head -20
Start: 2026-04-27T10:00:00+0000
HOST: my-laptop                        Loss%   Snt   Last   Avg  Best  Wrst StDev
  1. AS???    192.168.1.1               0.0%     5    0.4   0.4   0.3   0.5   0.1
  2. AS7922   10.0.0.1                  0.0%     5    9.2   9.4   9.1  10.0   0.4
  3. AS7922   68.86.x.x                 0.0%     5   10.5  10.6  10.4  11.0   0.3
  4. AS7922   96.110.x.x                0.0%     5   12.1  12.3  12.0  12.7   0.3
  5. AS15169  72.14.x.x                 0.0%     5   13.2  13.4  13.1  13.8   0.3
  6. AS15169  108.170.x.x               0.0%     5   13.3  13.5  13.2  13.9   0.3
```

The `-z` flag adds AS lookups, `-b` shows IPs and hostnames, `-w` is wide format.

**Look up an ASN (the company behind a network):**

```
$ whois -h whois.radb.net AS15169 | head -20
aut-num:    AS15169
as-name:    GOOGLE
descr:      Google LLC
admin-c:    ...
tech-c:     ...
mnt-by:     MAINT-AS15169
source:     RADB
```

**Look up the AS that owns a given IP (Team Cymru's whois service):**

```
$ whois -h whois.cymru.com " -v 8.8.8.8"
AS      | IP               | BGP Prefix          | CC | Registry | Allocated   | AS Name
15169   | 8.8.8.8          | 8.8.8.0/24          | US | arin     | 1992-12-01  | GOOGLE, US
```

**Look up the origin AS via DNS (Team Cymru's DNS service):**

```
$ dig +short -t TXT 8.8.8.8.origin.asn.cymru.com
"15169 | 8.8.8.0/24 | US | arin | 1992-12-01"
```

(Note the IP has to be reversed for the older `origin.asn.cymru.com` style: `dig +short -t TXT 3.3.3.3.origin.asn.cymru.com` for `3.3.3.3`.)

**Find your own public IP and AS:**

```
$ MY_IP=$(curl -s ifconfig.me); whois -h whois.cymru.com " -v $MY_IP"
AS      | IP               | BGP Prefix          | CC | Registry | Allocated   | AS Name
7922    | 73.x.y.z         | 73.0.0.0/8          | US | arin     | 2005-11-15  | COMCAST-7922, US
```

**Generate a prefix list for a known AS (handy for filter configs):**

```
$ bgpq4 -h whois.radb.net AS15169 | head -20
no ip prefix-list NN
ip prefix-list NN permit 8.8.4.0/24
ip prefix-list NN permit 8.8.8.0/24
ip prefix-list NN permit 8.34.208.0/20
ip prefix-list NN permit 8.35.192.0/20
ip prefix-list NN permit 23.236.48.0/20
...
```

`bgpq4` queries IRR databases (or RPKI) to enumerate the prefixes an AS announces; you turn that into a prefix-list for filtering.

**Look up an AS-set / AS macro:**

```
$ bgpq4 -h whois.radb.net -l GOOGLE-PREFIXES AS-GOOGLE | head -10
no ip prefix-list GOOGLE-PREFIXES
ip prefix-list GOOGLE-PREFIXES permit 8.8.4.0/24
ip prefix-list GOOGLE-PREFIXES permit 8.8.8.0/24
...
```

**Capture BGP traffic (needs `sudo`; watch your own session if you have one):**

```
$ sudo tcpdump -i any -n port 179
tcpdump: data link type LINUX_SLL2
listening on any, link-type LINUX_SLL2 (Linux cooked v2), capture size 262144 bytes
14:30:01.12345 IP 192.0.2.1.179 > 192.0.2.2.12345: Flags [P.], length 19: BGP, length: 19
14:30:01.23456 IP 192.0.2.2.12345 > 192.0.2.1.179: Flags [.], ack 19, win 1234, length 0
```

**See routes on a Cisco/Quagga/FRR-style router (if you have one):**

```
router# show ip bgp summary
BGP router identifier 10.0.0.1, local AS number 65001
BGP table version is 12345, main routing table version 12345
934567 BGP NLRI entries, 1234567 path entries

Neighbor     V    AS  MsgRcvd  MsgSent  TblVer  InQ  OutQ  Up/Down  State/PfxRcd
192.0.2.1    4   100  9876543  1234567   12345    0     0  3w2d     934567
192.0.2.5    4   200  8765432  1234890   12345    0     0  1d4h     834120
```

The `State/PfxRcd` column is critical: a number = it's Established and that many prefixes were received; a word like `Active` or `Idle` means the session isn't up.

**Look at one specific neighbor on FRR/Cisco:**

```
router# show bgp ipv6 unicast neighbors 2001:db8::2
BGP neighbor is 2001:db8::2, remote AS 100, external link
  Description: Transit Provider A
  BGP version 4, remote router ID 192.0.2.1
  BGP state = Established, up for 3w2d
  Last read 00:00:23, Last write 00:00:11
  Hold time is 90, keepalive interval is 30 seconds
  ...
```

**See the routes a neighbor advertised to you:**

```
router# show bgp ipv6 unicast neighbors 2001:db8::2 received-routes | include 2001:db8:abcd
*> 2001:db8:abcd::/48     2001:db8::2  100   0 100 65001 i
```

**See what you advertised to that neighbor:**

```
router# show bgp ipv6 unicast neighbors 2001:db8::2 advertised-routes | head -20
```

**Look up a specific prefix in your BGP table (all paths):**

```
router# show bgp ipv6 unicast 2001:db8:abcd::/48
BGP routing table entry for 2001:db8:abcd::/48
Paths: (2 available, best #1, table default)
  Advertised to non peer-group peers:
    2001:db8::2
  100 65001
    2001:db8::2 (used)
      Origin IGP, metric 0, localpref 200, valid, external, best
  300 65001
    2001:db8::6
      Origin IGP, metric 0, localpref 100, valid, external
```

You can see *both* paths and which one won (`best`).

**See RPKI status (FRR):**

```
router# show rpki prefix-table | head -10
RPKI/RTR prefix table
Prefix                          Prefix Length    Origin-AS
2001:db8::/32                   32 - 48          65000
8.8.8.0/24                      24 - 24          15169
```

**Check whether a specific route was RPKI-valid:**

```
router# show bgp ipv6 unicast 2001:db8:abcd::/48 bestpath
...
  rpki validation-state: valid
```

**Real-time pre-deployment view (Hurricane Electric's BGP tool, web-only — note this leaves the terminal):**

```
$ open https://bgp.he.net/AS15169
# (or just use whois/bgpq4 above to stay in-terminal)
```

**Look up a prefix from a public looking glass via API:**

```
$ curl -s "https://stat.ripe.net/data/routing-status/data.json?resource=8.8.8.0/24" | head -30
{
  "messages": [],
  "see_also": [],
  "version": "1.4",
  "data_call_name": "routing-status",
  "data_call_status": "supported - production",
  ...
  "data": {
    "first_seen": {...},
    "visibility": {"v4": 0.99, ...},
    "announced_space": {...},
    "less_specifics": [...],
    "more_specifics": [...]
  }
}
```

The `visibility` field tells you what fraction of the world's routers see this prefix.

**Use `ip-r` to view kernel route changes in real time:**

```
$ ip monitor route
[ROUTE]2001:db8::/32 dev eth0 proto bgp metric 20 pref medium
```

(Useful when you have FRR/Bird/GoBGP injecting into the kernel via Netlink.)

**See what `bird` or `gobgp` see if you run a soft router:**

```
$ birdc show route
BIRD 2.13.1 ready.
Table master4:
0.0.0.0/0           via 192.0.2.1 on eth0 [bgp1 11:23:45 from 192.0.2.1] * (100) [AS65000i]
8.8.8.0/24          via 192.0.2.1 on eth0 [bgp1 11:23:45 from 192.0.2.1] * (100) [AS65000 15169i]

$ gobgp neighbor
Peer        AS  Up/Down State       |#Received  Accepted
192.0.2.1   65000  03:14:15 Established | 934567    934567
```

**Save a BGP table to disk for offline poking (FRR):**

```
router# show ipv6 bgp | redirect /tmp/bgp.txt
```

**Check kernel routing decisions:**

```
$ ip route get 8.8.8.8
8.8.8.8 via 192.168.1.1 dev wlan0 src 192.168.1.42 uid 1000
    cache
```

The kernel tells you exactly which route it would use for a given destination.

**Watch link state in real time (BFD-relevant):**

```
$ ip monitor link
1: lo: <LOOPBACK,UP,LOWER_UP> ...
2: eth0: <BROADCAST,MULTICAST,UP,LOWER_UP> ...
```

If you're running BFD, link-state changes trigger BGP teardown immediately.

**Inspect a `.pcap` of BGP traffic:**

```
$ tcpdump -r bgp.pcap -nn -v port 179 | head -50
14:30:01 IP 192.0.2.1.179 > 192.0.2.2.12345: BGP Open Message
  Version 4, length: 71, my AS 65001, Holdtime 90s, id 192.0.2.1
  Optional parameters length: 42
    Option Capabilities Advertisement (2), length: 40
      Multiprotocol Extensions (1), length: 4: AFI IPv6 (2), SAFI Unicast (1)
      Route Refresh (2), length: 0
      4-octet AS Number (65), length: 4: 65001
      ...
```

**What the message types look like in tcpdump:**

```
$ sudo tcpdump -i any -nn -v port 179 -c 20
... BGP, OPEN ...
... BGP, KEEPALIVE ...
... BGP, UPDATE, attribute(s) [LP=200, AS_PATH=15169, NH=192.0.2.1] NLRI=8.8.8.0/24 ...
... BGP, KEEPALIVE ...
... BGP, NOTIFICATION, error: Hold Timer Expired ...
```

That last NOTIFICATION is the death rattle. Hold timer expired.

## Common Confusions

**"Is BGP only for the big ISPs?"**
> **Broken:** "I'm not Google. I don't need BGP."
> **Fixed:** Anyone with **two or more uplinks** that they want to fail over between runs BGP. That includes mid-sized companies, content providers, anyone with their own ASN. You also need it any time you have your own provider-independent (PI) IP space and want it routed.

**"Why does my BGP session keep flapping between Active and Connect?"**
> **Broken:** "I see Active in the state column, must be active and good!"
> **Fixed:** Active is *bad*. It means TCP isn't completing. Most common reasons: firewall blocking port 179 between the two routers, mismatched neighbor IP on one side, asymmetric routing meaning return packets don't make it back, ACL on a transit interface. Run `tcpdump -nn port 179` on both ends, see whether SYN goes out and ACK comes back.

**"Why does my BGP session bounce every 3 minutes?"**
> **Broken:** "Hold timer keeps expiring. The protocol must be broken."
> **Fixed:** That's the symptom, not the cause. The cause is usually one of: TCP MD5 password mismatch (configured on one side, not the other), MTU problem somewhere on the path (large UPDATEs get fragmented or dropped), interface flapping (BFD would catch this faster), or unicast RPF filtering dropping the BGP packets. Look at TCP retransmits: `ss -i | grep :179` will show retrans counts.

**"Why isn't my route being advertised to my upstream?"**
> **Broken:** "I see it in my BGP table; surely my upstream sees it."
> **Fixed:** Routes go through *outbound* policy before being advertised. Common causes: `network` statement missing for that prefix in the BGP config; route-map deny on egress; the route is iBGP-learned and you haven't configured `next-hop-self` or proper policy; soft-reconfiguration outbound disabled, so old policy is being used. Check `show bgp neighbor X advertised-routes` to see exactly what you're sending.

**"Why isn't my route being received by my neighbor?"**
> **Broken:** "It must be a propagation delay. Wait an hour."
> **Fixed:** Probably an inbound policy issue on *their* side. Check `show bgp neighbor X received-routes` to see what you actually accepted; look at inbound route-map; check RPKI validation state (a misconfigured ROA = invalid = dropped); check max-prefix limits.

**"What's the difference between BGP and OSPF?"**
> **Broken:** "They're both routing protocols; I'll just pick one."
> **Fixed:** OSPF (and IS-IS) are **interior** gateway protocols (IGPs). They use link-state and Dijkstra to compute shortest paths *inside* a single AS. BGP is an **exterior** gateway protocol (EGP). It's a path-vector protocol designed for *policy*, not just shortest path. You almost always run *both*: an IGP for fast intra-AS convergence and BGP for inter-AS reachability. They serve different problems.

**"Why does BGP convergence feel so slow?"**
> **Broken:** "BGP is just slow. Nothing I can do."
> **Fixed:** It's slow by default because of the **MRAI** timer (30s) and path-exploration. You speed it up with: BFD (sub-second link failure detection), reduced MRAI (5s or 0 in datacenter), Add-Path (RFC 7911) for pre-installed backups, BGP PIC (FIB-level pre-programming of backup next-hops), and Graceful Restart for daemon upgrades. A datacenter fabric running BGP today converges in tens of milliseconds, not minutes.

**"Why is my prefix not visible globally even though it's in my BGP table?"**
> **Broken:** "I added the network statement; it should propagate."
> **Fixed:** Possibilities: (1) your upstream is filtering you (max prefix, prefix-list, IRR mismatch); (2) your prefix is RPKI-invalid (your ROA's max-length is too short); (3) the prefix length is too long (most operators filter out anything more specific than /24 in IPv4 or /48 in IPv6); (4) it's being dropped as a bogon. Check it with a looking glass: `https://bgp.he.net/net/<prefix>` and `https://stat.ripe.net/data/routing-status/data.json?resource=<prefix>`.

**"Why is traffic taking the wrong path? My LOCAL_PREF should fix it."**
> **Broken:** "I set LOCAL_PREF 200 on path A; it's still going via B."
> **Fixed:** LOCAL_PREF only controls **inbound** path decisions inside *your* AS. It does not influence which path *the other guy* uses to send traffic *to you*. To control inbound traffic to you, you have to influence *their* decision: AS-PATH prepending, MED hints, or community-based policy at their side. To control outbound traffic, LOCAL_PREF is the right tool — but check that the import route-map is actually being applied (`show bgp neighbor X received-routes` and look at `localpref`).

**"Why am I getting the same prefix from two neighbors but only one is best?"**
> **Broken:** "Both look identical. The router is broken."
> **Fixed:** Walk the decision algorithm. Even when two routes look identical, *something* is different — router IDs, neighbor IPs, IGP cost to NEXT_HOP, MED. The protocol is deterministic; it always picks one. If you want to use *both*, enable **BGP multipath** (`maximum-paths`) and tune the relevant attribute thresholds.

**"Why does my private ASN cause problems with my upstream?"**
> **Broken:** "I'm using AS 65001 and my upstream rejects my routes."
> **Fixed:** ASNs 64512–65534 (private ASNs) and 4200000000–4294967294 (private 4-byte) are *not allowed* on the public internet. If you announce routes from a private ASN to an upstream, they should strip the private ASN with `remove-private-AS` on egress. If they don't, your routes look weird (private ASN in the path) and other operators may reject them. Either get a real ASN or make sure your upstream strips them.

**"Why does my router show 4-byte ASNs as `23456`?"**
> **Broken:** "All my big ASNs got renamed to 23456!"
> **Fixed:** AS 23456 is the special "AS_TRANS" placeholder used for backwards compatibility with old routers that don't understand 4-byte ASNs (RFC 6793). When a path with a 4-byte ASN crosses an old router, the 4-byte ASN gets temporarily replaced with 23456. Modern routers reconstruct the real ASN from the AS4_PATH attribute. If you only see 23456 everywhere, something on the path is downgrading. Update the routers in question; 4-byte ASNs have been standard since 2007.

**"Is BGP a control plane or a data plane protocol?"**
> **Broken:** "BGP carries packets, right?"
> **Fixed:** BGP is **control plane only**. It only decides *which way packets should go*; it doesn't carry the packets themselves. The actual packets follow the FIB (forwarding information base) the BGP RIB feeds into. The data plane is what's running between hops; BGP is the table that decides who's next.

**"What's the difference between BGP and a default route?"**
> **Broken:** "I just have a default route to my ISP, that should be enough."
> **Fixed:** A default route is "send everything you don't know to this gateway." That works fine if you're a stub network with one upstream. But if you have *two* upstreams and want to fail over, or if you want to receive selective routes (full table or partial), or if you want to *announce* your own prefixes, you need BGP. Default routes alone can't do load-balancing across providers or failover with sub-second detection.

## Vocabulary

| Term | Meaning |
|---|---|
| AS (Autonomous System) | A separately managed network; a "neighborhood" with its own routing policy. |
| ASN | Autonomous System Number. The unique numeric ID for an AS. |
| 4-byte ASN | A 32-bit ASN (RFC 6793). Standard since 2007. Old routers see them as `23456`. |
| Private ASN | 64512–65534 (16-bit) or 4200000000–4294967294 (32-bit). Not allowed on the public internet. |
| Peer | A router with which you have a BGP session. Same as neighbor. |
| Neighbor | Same as peer. The other end of a BGP session. |
| eBGP | External BGP. Between two different ASes. |
| iBGP | Internal BGP. Between two routers inside the same AS. |
| Prefix | An IP network/length pair, e.g. `192.0.2.0/24`. |
| NLRI | Network Layer Reachability Information. The prefixes inside an UPDATE. |
| UPDATE | The BGP message that adds or withdraws routes and carries path attributes. |
| OPEN | First BGP message in a session: "hello, my ASN is X, my router ID is Y, here are my capabilities." |
| KEEPALIVE | "I'm still here." Sent every ~60s. |
| NOTIFICATION | "Something is wrong; closing the session." |
| Withdraw | Telling a peer "this prefix I told you about? Forget it. I can't reach it any more." |
| Hold timer | Max time you'll wait for *anything* from a peer before declaring them dead. Default 180s. |
| KEEPALIVE timer | How often you send a KEEPALIVE. Default 60s (1/3 of hold timer). |
| MRAI | Minimum Route Advertisement Interval. Smoothing timer. Default 30s for eBGP. |
| AS_PATH | Ordered list of ASes the route has crossed; used for loop detection and length comparison. |
| AS_PATH prepending | Adding your ASN multiple times to make the path look longer (and thus less preferred). |
| NEXT_HOP | The IP address to forward traffic to in order to reach a prefix. |
| LOCAL_PREF | "How much *I* want this route." Higher is better. Local to your AS. |
| MED | Multi-Exit Discriminator. Neighbor's hint about which entry point to prefer. Lower is better. |
| ORIGIN | How the route was first learned: IGP, EGP, or Incomplete. IGP > EGP > Incomplete. |
| Weight | Cisco-specific local preference. Highest priority in the decision algorithm. Local to a single router. |
| COMMUNITIES | Numeric tags attached to routes for policy coordination. |
| Standard community | 16-bit ASN : 16-bit value. The classic format. |
| Extended community | Structured tags like Route Targets. |
| Large community | RFC 8092 format ASN:value:value. Required for clean 4-byte ASN expression. |
| NO_EXPORT | Well-known community: "don't advertise this beyond my AS." |
| NO_ADVERTISE | Well-known community: "don't advertise this to any peer at all." |
| BLACKHOLE | RFC 7999 community: "drop traffic to this prefix" (DDoS scrub). |
| Aggregate | A summarized prefix that covers several smaller ones. |
| Deaggregate | Announcing many small prefixes instead of one large one. Generally rude. |
| Route reflector (RR) | An iBGP router that re-advertises routes to its clients, breaking the full-mesh requirement. |
| RR client | An iBGP router that peers with a route reflector instead of with all other iBGP routers. |
| Confederation | A way to chop a big AS into sub-ASes for scaling iBGP. |
| Transit | One AS pays another to carry all its traffic, including to/from other transit. |
| Peering | Two ASes carry each other's traffic for free. |
| Tier-1 | A network so big it doesn't pay for transit; peers with all other tier-1s. |
| Tier-2 | A regional/national network that pays tier-1s for global reach. |
| Tier-3 | A local end-customer ISP, mostly downstream-only. |
| IXP | Internet Exchange Point. A physical location where many ASes peer. |
| Looking Glass | A web tool that shows the BGP table from one or more router's view. |
| Route flap | A route that goes up-down-up-down quickly. |
| Dampening | Suppression of a route that has flapped too much. Mostly turned off today. |
| RIB | Routing Information Base. The full BGP table on a router. |
| FIB | Forwarding Information Base. The actual data-plane forwarding table installed in hardware. |
| Adj-RIB-In | Per-neighbor input table (raw routes received from that neighbor). |
| Adj-RIB-Out | Per-neighbor output table (routes you'll advertise to that neighbor). |
| Loc-RIB | Your local best-path table after applying inbound policy. |
| Soft reconfiguration | Re-applying inbound policy without resetting the BGP session. Costs memory. |
| Route refresh | A capability where a router can ask a neighbor for a fresh send of all routes. |
| RPKI | Resource Public Key Infrastructure. Cryptographic system tying prefixes to authorized ASNs. |
| ROA | Route Origin Authorization. The signed object: "AS X may originate prefix P up to length L." |
| Validator | A piece of software that downloads ROAs and feeds validation results to your routers (Routinator, Fort, OctoRPKI). |
| RTR | RPKI to Router. The protocol the validator uses to ship results to routers (port 323). |
| Valid | RPKI state: ROA exists and the announcement matches. |
| Invalid | RPKI state: ROA exists, announcement does not match (wrong ASN or too long). |
| NotFound | RPKI state: no ROA covers this prefix. |
| BGPsec | RFC 8205 cryptographic AS_PATH validation. Barely deployed. |
| ASPA | RFC 9582 Autonomous System Provider Authorization. Lighter-weight path validation. |
| Full-mesh | Every iBGP router peers with every other iBGP router. |
| Default route | `0.0.0.0/0` (or `::/0`) — "anything I don't have a more specific route for, send here." |
| Blackhole | A null route. Traffic for that prefix gets dropped. |
| Graceful Restart | RFC 4724. Continue forwarding while the BGP daemon restarts. |
| BFD | Bidirectional Forwarding Detection. Sub-second link/path failure detection. |
| Multihop eBGP | An eBGP session whose neighbor is more than one hop away. Requires raising the TTL. |
| Update-source | The local interface IP your BGP session uses (often loopback for iBGP). |
| Route map | A policy expression: "if route matches X, do Y (set attributes / accept / deny)." |
| Filter-list | Filter on AS_PATH regex. |
| Prefix-list | Filter on prefix and length range. |
| Distribute-list | Older filter on access-list. Mostly superseded by prefix-list. |
| AS_PATH access list | Regex-based filter on the AS_PATH attribute. |
| Bogon | A prefix that should never appear in the global routing table (RFC 1918 space, documentation prefixes, link-local, etc). |
| Add-Path | RFC 7911. Advertise multiple paths per prefix to one peer. |
| Multipath | Use multiple equal-cost paths simultaneously. |
| Optimal Route Reflection | RFC 9107. RR considers IGP cost in path selection so clients get the closest exit. |
| Convergence | The time it takes for the network to settle on a stable set of routes after a change. |
| Hot potato | Send traffic out the closest exit; minimize what *you* carry. |
| Cold potato | Carry traffic on your own network as long as possible; better for the user, more expensive. |
| Gao-Rexford | Conditions on policy that guarantee BGP converges to a unique stable state. |
| Dispute wheel | A cycle of conflicting policies that prevents BGP convergence. |
| OTC (Only-to-Customer) | RFC 9234 attribute that prevents route leaks by tagging session roles. |
| BMP | BGP Monitoring Protocol. Streaming telemetry from a router about its BGP state. |
| Session reset | Tearing down and re-establishing a BGP session. Heavy. |
| Capability | A feature negotiated in OPEN: 4-byte ASN, MP-BGP for non-IPv4-unicast, route refresh, Add-Path. |

## Try This

Five to ten safe, low-risk experiments you can run from any Linux box without operating a router.

1. **Trace a path with AS lookup.** `traceroute -A -n 8.8.8.8` and watch the AS column change. You're seeing BGP-decided paths in real time. Try it again to `1.1.1.1`, `9.9.9.9`, `cloudflare.com`. Compare paths.

2. **See your own AS.** `whois -h whois.cymru.com " -v $(curl -s ifconfig.me)"`. Find your ISP's ASN. Then plug it into `bgpq4 -h whois.radb.net <ASN>` to enumerate the prefixes they announce.

3. **Find the closest CDN node to you.** `traceroute -A -n www.cloudflare.com`. Look at the second-to-last hop; that's the AS Cloudflare is reaching you through. The CDN deliberately put a node nearby; you'll usually see <20ms RTT.

4. **Check the visibility of a prefix worldwide.** `curl -s "https://stat.ripe.net/data/routing-status/data.json?resource=8.8.8.0/24" | python3 -m json.tool | head -40`. The `visibility` field tells you what fraction of global routers see this prefix. For a "famous" prefix it'll be ~0.99.

5. **Validate your own prefix's RPKI status.** Visit `https://stat.ripe.net/data/rpki-validation/data.json?resource=AS<your-asn>&prefix=<your-prefix>` (web). Or use `dig -t CHAOS TXT _origin.<prefix>.routinator.example.com` if you run a validator.

6. **Watch BGP packets fly past your interface.** `sudo tcpdump -i any -nn port 179`. You won't see anything unless you're running a router or have a session, but if you do, you'll see OPEN/UPDATE/KEEPALIVE messages live.

7. **Compare paths from different vantage points.** Use a public looking glass like RIPE RIS Live: `wss://ris-live.ripe.net/v1/`. You can subscribe to UPDATE messages from real BGP collectors around the world. (This crosses out of the terminal, but it's an option.)

8. **Generate a prefix-list for an upstream's customers.** `bgpq4 -h whois.radb.net AS-AMAZON | head -30`. Read the output; it's exactly what you'd paste into a router config to filter that AS.

9. **Look at historical BGP data.** `curl -s "https://stat.ripe.net/data/whats-my-as/data.json?resource=$(curl -s ifconfig.me)" | python3 -m json.tool`. RIPE NCC has years of archived BGP data; you can ask "what AS owned this IP at this time?"

10. **Build a tiny iBGP lab.** Run `containerlab` with FRR images. Two nodes, an iBGP session over a link, one announces a /48, the other receives. You go from zero BGP to working session in about 15 minutes. (This is the single best way to understand BGP — type `show bgp summary` after every step and watch the counters move.)

## A Day in the Life of a Packet

Let's trace one packet from your laptop to a server, paying attention to BGP at each step. Pretend you typed `curl https://example.com/` and hit enter.

**Step 1 — DNS first.** Your laptop asks DNS for `example.com`. DNS resolution itself uses UDP (or TCP, or DoH/DoT), but every DNS query packet is itself routed by BGP-learned paths. The first packet of your day already depends on BGP. Your local resolver returns `93.184.216.34`.

**Step 2 — your laptop builds a TCP SYN.** Source: `192.168.1.42:54321`. Destination: `93.184.216.34:443`. The kernel looks at its routing table:

```
default via 192.168.1.1 dev wlan0
```

The destination doesn't match any specific route, so it falls through to the default. Send to gateway `192.168.1.1`. Off goes the SYN.

**Step 3 — your home router.** It receives the SYN. Its routing table also says "default via the ISP." It NATs the source IP from `192.168.1.42` to whatever public IP you have, and forwards.

**Step 4 — your ISP's CPE-facing router.** This box has more routes. It might have a default route to its core, or it might have a partial BGP table. It forwards toward the upstream.

**Step 5 — your ISP's BGP-speaking border router.** *Now* BGP starts mattering for your packet. This router has the full BGP table (hundreds of thousands of IPv4 routes, tens of thousands of IPv6). It does a longest-prefix match for `93.184.216.34`. It finds:

```
93.184.216.0/24 via 198.51.100.1, AS_PATH 174 15133, LOCAL_PREF 100
```

(`AS 15133` is Edgecast/Verizon Media; `AS 174` is Cogent.) The packet is forwarded to `198.51.100.1`, a router inside Cogent's AS.

**Step 6 — Cogent's network.** Cogent has its own iBGP, distributing the AS 15133 routes throughout. Whichever Cogent router has the closest exit toward Edgecast (hot potato) will hand off.

**Step 7 — handoff to Edgecast.** Edgecast's edge router accepts the packet, looks up `93.184.216.34` in its own internal routing, finds the specific server, forwards.

**Step 8 — the server replies.** It builds a SYN-ACK. Same process in reverse, except *Edgecast's* BGP table is consulted first. Their best path back to your home IP might *not* be the same path the SYN took. Welcome to **asymmetric routing**.

**Step 9 — the SYN-ACK arrives.** TCP completes the handshake. TLS does its negotiation. HTTP request goes. Response comes back. Your `curl` finishes.

For this single round-trip, dozens of BGP-speaking routers consulted their RIBs and forwarded along BGP-determined paths. Every single one of them picked "best" based on inbound policy, AS_PATH length, and economic preference.

That's BGP doing its job, invisibly, billions of times a second worldwide.

## Frequent BGP CLI cheats

A grab bag of one-liners you'll want when you start operating BGP yourself.

```bash
# FRR / Quagga — show summary
vtysh -c 'show bgp summary'
vtysh -c 'show bgp ipv6 unicast summary'

# FRR — see one specific neighbor
vtysh -c 'show bgp neighbors 2001:db8::2'

# FRR — see what the neighbor advertised to you
vtysh -c 'show bgp ipv6 unicast neighbor 2001:db8::2 received-routes'

# FRR — see what you advertised to the neighbor
vtysh -c 'show bgp ipv6 unicast neighbor 2001:db8::2 advertised-routes'

# FRR — clear a single neighbor (soft, no session reset)
vtysh -c 'clear bgp ipv6 unicast 2001:db8::2 soft in'
vtysh -c 'clear bgp ipv6 unicast 2001:db8::2 soft out'

# FRR — hard reset
vtysh -c 'clear bgp ipv6 unicast 2001:db8::2'

# Cisco IOS-XE / IOS-XR variants
show ip bgp summary
show bgp ipv6 unicast summary
show ip bgp neighbors 192.0.2.1 received-routes  # IOS
show bgp ipv4 unicast neighbors 192.0.2.1 received routes  # IOS-XR
clear ip bgp 192.0.2.1 soft in

# Junos
show bgp summary
show bgp neighbor 192.0.2.1
show route receive-protocol bgp 192.0.2.1
show route advertising-protocol bgp 192.0.2.1
clear bgp neighbor 192.0.2.1 soft in

# Bird
birdc show protocols
birdc show route
birdc show route protocol bgp1 all
birdc reload bgp1

# GoBGP
gobgp neighbor
gobgp neighbor 192.0.2.1
gobgp global rib
gobgp global rib add 10.0.0.0/24 -a ipv4
```

## Mental Models That Help

### "BGP is gossip, not authority"

A frequently-missed truth about BGP: it has *no central authority*. There is no master map. There is no DNS-style root server for routing. There is only the gossip among neighbors. The "global routing table" is an emergent, eventually-consistent thing — it doesn't exist anywhere as a single object.

Each router builds its own view, based on what it heard from its neighbors. If two routers' views disagree, traffic might take asymmetric paths. If many routers' views are wrong (because of a hijack, leak, or outage), traffic goes the wrong place. The internet is an emergent agreement.

This is why BGP is so robust *and* so fragile: robust because there's no single point of failure to attack; fragile because there's no single point of truth to verify against. RPKI is one attempt to add a thin layer of authority on top of the gossip.

### "Policy is everything"

If you only learn one thing about real-world BGP, learn this: **the protocol does what you tell it, and operators tell it different things based on business decisions.**

The decision algorithm has 10+ steps, but in practice 90% of routing is decided by step 2 (LOCAL_PREF, set by inbound policy) and step 4 (AS_PATH length, also tunable via prepending). The protocol's job is to provide the *machinery* — attribute encoding, propagation, decision — and the operator's job is to express *what they want* via policy.

That's why the same prefix can take wildly different paths for different observers. Two ISPs may have very different views of "best path" depending on which customer/peer/provider relationships are active and how each operator has configured their LOCAL_PREF maps.

### "iBGP is a parallel control plane"

Inside an AS, you have two routing planes happening simultaneously:

- The **IGP** (OSPF, IS-IS, etc.) tells you how to reach every IP *inside* the AS.
- **iBGP** tells you how to reach every IP *outside* the AS.

iBGP relies on the IGP to make NEXT_HOPs reachable. If the IGP doesn't know how to get to a NEXT_HOP, the iBGP-learned route is unusable. That's why the most common iBGP failure is "route is in BGP table but not in FIB" — the NEXT_HOP isn't reachable via IGP.

The fix: `next-hop-self` on iBGP sessions originating eBGP-learned routes, or making sure your IGP includes the eBGP peering subnets.

### "The internet is a graph, BGP walks it"

Mathematically, the internet is a labeled directed graph: each AS is a node, each peering session is a labeled edge. BGP is the algorithm that *walks* this graph from your AS to every other AS, choosing one path per (prefix, source) pair.

If you've ever played with graph traversal — Dijkstra, Bellman-Ford, A* — BGP feels like a Bellman-Ford-flavored variant where the "cost" function is replaced with operator policy. The major insight from Griffin/Shepherd/Wilfong's work in 2002 is that BGP, in general, is *not* guaranteed to converge — operator policies can create cycles ("dispute wheels") that prevent stable assignments. Gao and Rexford's 2001 paper showed that *if* operators follow the customer/peer/transit hierarchy naturally, convergence is guaranteed.

This is why "the internet works at all": the commercial structure of inter-AS relationships happens to satisfy the conditions for convergence.

## Quick Mental Test

If you've absorbed this sheet, you should be able to answer:

1. What does AS stand for? *Autonomous System.*
2. What TCP port is BGP on? *179.*
3. What's the difference between eBGP and iBGP? *eBGP is between different ASes; iBGP is inside one AS. eBGP prepends ASN, iBGP doesn't.*
4. What's the most important step in the decision algorithm in practice? *LOCAL_PREF (step 2) for policy, AS_PATH length (step 4) for shortest path.*
5. Why does iBGP need a full mesh? *Because iBGP doesn't re-advertise routes between iBGP peers. Either full-mesh, or use route reflectors.*
6. What's a route reflector for? *To break the iBGP full-mesh requirement; an RR re-advertises iBGP routes to its clients.*
7. What's a community? *A numeric tag attached to a route for policy coordination, with no meaning to the protocol itself.*
8. What does NO_EXPORT do? *Tells receivers not to advertise this route beyond their own AS.*
9. What is RPKI? *A cryptographic system that ties IP prefixes to authorized origin ASNs, with ROAs as the signed objects.*
10. What's the BLACKHOLE community for? *DDoS scrubbing; tells your upstream to drop traffic to this prefix.*
11. Why is BGP convergence slow? *MRAI timer (30s default), path exploration on withdrawal, large RIB sizes.*
12. What makes it fast? *BFD, reduced MRAI, Add-Path, BGP PIC, Graceful Restart.*

If a few of these were tricky, scroll back up. If they all clicked, you have working knowledge of BGP.

## Where to Go Next

- `cs networking bgp` — the dense reference sheet on BGP commands, attributes, and configurations.
- `cs networking bgp-advanced` — multipath, Add-Path, optimal-route-reflection, advanced policy.
- `cs detail networking/bgp-advanced` — the math: convergence, dampening half-life, Gao-Rexford derivation.
- `cs networking ospf` — the most common interior gateway protocol that runs alongside BGP.
- `cs networking is-is` — the other major IGP, common in service-provider networks.
- `cs juniper junos-bgp-advanced` — Junos-specific BGP configuration patterns.
- `cs juniper junos-routing-policy` — how to write inbound and outbound policy on Junos.
- `cs networking mpls` — MPLS, often paired with BGP for L3VPNs and EVPN.
- `cs networking tcp` — BGP rides on TCP; understanding TCP timers, MD5, and retrans helps debug session flaps.
- `cs networking dns` — DNS often resolves first, then BGP routes the resulting packets.
- `cs ramp-up linux-kernel-eli5` — what's running underneath all those routers.
- `cs fundamentals how-the-internet-works` — the wider picture BGP fits into.
- `cs fundamentals how-networking-works` — the layers below BGP.

## See Also

- `networking/bgp`
- `networking/bgp-advanced`
- `networking/ospf`
- `networking/is-is`
- `networking/mpls`
- `networking/tcp`
- `networking/dns`
- `juniper/junos-bgp-advanced`
- `juniper/junos-routing-policy`
- `fundamentals/how-the-internet-works`
- `fundamentals/how-networking-works`
- `ramp-up/linux-kernel-eli5`

## References

- RFC 4271 — A Border Gateway Protocol 4 (BGP-4)
- RFC 4760 — Multiprotocol Extensions for BGP-4
- RFC 1997 — BGP Communities Attribute
- RFC 7999 — BLACKHOLE Community
- RFC 8092 — BGP Large Communities
- RFC 6793 — BGP Support for 4-Byte AS Number
- RFC 7911 — Advertisement of Multiple Paths in BGP (Add-Path)
- RFC 7432 — BGP MPLS-Based Ethernet VPN (EVPN)
- RFC 4724 — Graceful Restart Mechanism for BGP
- RFC 5880 — Bidirectional Forwarding Detection (BFD)
- RFC 6480 — RPKI: An Infrastructure to Support Secure Internet Routing
- RFC 6811 — BGP Prefix Origin Validation
- RFC 8205 — BGPsec Protocol Specification
- RFC 9234 — Route Leak Prevention and Detection (BGP Roles)
- RFC 9582 — Autonomous System Provider Authorization (ASPA)
- RFC 9107 — BGP Optimal Route Reflection
- "Internet Routing Architectures" by Sam Halabi (Cisco Press)
- "BGP" by Iljitsch van Beijnum (O'Reilly)
- "Day One: Deploying BGP Routing Protocol" — Juniper Networks
- man `frr.conf`, man `bird`, man `gobgpd`
- bgp.he.net — Hurricane Electric BGP toolkit (view-only)
- bgp.tools — modern BGP visibility tool
- stat.ripe.net — RIPE NCC's BGP data API
- routeviews.org — University of Oregon BGP collector
- bgpstream.caida.org — real-time BGP event feed
