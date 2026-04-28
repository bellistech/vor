# Spine-Leaf — ELI5

> Spine-Leaf is a way to wire up a data center so every computer can talk to every other computer in exactly two hops, with no traffic jam at the top, no matter how busy it gets.

## Prerequisites

- `ramp-up/ip-eli5` — you should know what an IP address is, what a packet is, and what "routing" means in plain English.
- `ramp-up/bgp-eli5` — you should know that BGP is the protocol routers use to tell each other "hey, I can reach this address, send packets to me."

If those two sheets feel fuzzy, read them first. This sheet will use words from both. We will still define every word again here in plain English, but it is much easier if you have already met them once.

If a word feels weird, look it up in the **Vocabulary** table near the bottom. Every weird word in this sheet is in that table with a one-line plain-English definition.

If you see a `$` at the start of a line in a code block, that means "type the rest of this line into your terminal." You do not type the `$`. The lines underneath that don't have a `$` are what your computer prints back at you. We call that "output."

## What Even Is Spine-Leaf

### Imagine the world's fairest shopping mall

Picture a giant shopping mall. The mall has hundreds of shops. Every shop is on the ground floor. Above the ground floor there is a balcony walkway. The balcony goes around the whole mall. On the balcony there are special staircases. Every shop on the ground floor has its own staircase up to the balcony, and there are several staircases (let us say four) going up from each shop.

If a person in shop A wants to visit a person in shop B, they do not walk along the ground floor. The ground floor between shops is closed off. Instead they go up one staircase to the balcony, walk along the balcony, then go down a staircase to shop B. That is exactly two stair trips: one up, one down. No matter which two shops you pick, the trip is two stair trips.

Now here is the magic. Because every shop has four staircases up, when shop A wants to send a delivery to shop B, the mall can pick any of the four staircases. If staircase one is busy, use staircase two. If two is busy, use three. The mall load-balances deliveries across all four staircases. There is never a single bottleneck. There is no "main escalator" that everybody has to use.

That is **Spine-Leaf**. The shops are **leaves**. The balcony walkways are **spines**. Every leaf connects to every spine. Never leaf-to-leaf. Never spine-to-spine. Two hops, every time. Many parallel paths, so traffic spreads out evenly.

### Why we call it Spine-Leaf

Imagine a tree leaf. The thick middle stem is the **spine**. The flat green parts coming off the stem are the **leaves**. In the diagram, the leaves are at the bottom (where the servers plug in), and the spines are at the top (the big fat fabric that connects everything). It looks a bit like the underside of a leaf. Some people draw it the other way up. The shape is the same either way.

Some books call it a **two-tier Clos** or a **folded Clos** or a **fat tree**. Those are all words for the same idea. We will explain every one of those words in this sheet.

### The one-sentence summary

Spine-Leaf is two layers of switches: a row of **leaves** at the bottom (servers plug into these) and a row of **spines** at the top (these only connect leaves together). Every leaf has a wire to every spine. Leaves never connect to other leaves. Spines never connect to other spines. Servers plug into leaves only.

That is it. That is the whole topology. Everything else in this sheet is consequences of that one rule.

## The Old World: 3-Tier Tree

### The classic data center of the 1990s and 2000s

Before Spine-Leaf, data centers used a **three-tier tree**. Picture a real tree. There is a thick trunk at the top (the **core**). The trunk splits into a few big branches (the **aggregation** or **distribution** layer). Each big branch splits into many small branches (the **access** layer). The leaves of the real tree are the servers, plugged into the access layer.

Drawn out:

```
                   [ Core 1 ]   [ Core 2 ]
                       \\         //
                  +-----+----+----+-----+
                  |        / \         |
              [ Agg 1 ]  [ Agg 2 ]  [ Agg 3 ]  [ Agg 4 ]
                |  |       |  |       |  |       |  |
            [Acc1][Acc2][Acc3][Acc4][Acc5][Acc6][Acc7][Acc8]
              |    |    |    |    |    |    |    |
           servers servers servers servers servers servers
```

The idea was simple. Local traffic between two servers under the same access switch never had to go up. Traffic between two servers under the same aggregation switch only had to go up two levels. Only the rarest traffic had to climb all the way to the core.

That assumption was true in 1995. It is not true today.

### Why three-tier broke

In 1995, a server mostly talked to a human sitting at a PC. The human typed a request, the server answered, the human read the answer. Most traffic went **north-south**: in from the internet, down to a server, back out to the internet.

By 2010, servers talked to other servers all day long. A web search talks to a hundred index servers. A page load talks to a database, then a cache, then an ad server, then a recommendation engine, then a logging service. That traffic stays inside the data center. It goes **east-west**: server to server, not in and out.

In a three-tier tree, east-west traffic between two servers in different aggregation pods has to climb to the core, cross the core, and come back down. The core was always the most expensive box. There were always only one or two of them. Suddenly the core had to carry far more traffic than its designers expected. The core became the bottleneck. Adding more servers made the bottleneck worse, not better. The whole tree had to be rebuilt every few years with bigger, more expensive core boxes.

That is what hyperscalers (Google, Facebook, Amazon, Microsoft) noticed first. They needed a topology that did not get worse as it got bigger.

### What the cracks looked like in real life

- A single core failure took out half the data center, because the second core was already at capacity and could not soak up the load.
- Adding a new pod required rewiring the core, often during a maintenance window, often at 2 AM on a Sunday.
- Spanning Tree Protocol (STP) ran across the whole fabric. STP turns half the links off to stop loops. Half your wires sat unused.
- Hot spots formed. One agg pod would run hot while another sat at 5%, because traffic patterns are not uniform.
- Upgrading the core meant buying the biggest, most expensive switch on Earth. There was always one vendor selling it for a fortune.

Spine-Leaf fixes all of these. We will see how.

## The Clos Theorem (1953)

### A 70-year-old phone-system idea

In 1953, a man named Charles Clos was working on telephone exchanges at Bell Labs. The problem he was solving was: how do you build a phone switch that can connect any caller to any callee, without needing a separate wire from every phone to every other phone?

Charles Clos wrote a paper called *A Study of Non-Blocking Switching Networks*. In it he proved a theorem. Stated in plain English: if you build a switching fabric in three stages — input, middle, output — and you wire every input switch to every middle switch, and every middle switch to every output switch, then for any number of phones N, you can build a switch that is **non-blocking** with a number of crosspoints much smaller than N times N.

**Non-blocking** means: if Alice picks up the phone and Bob picks up the phone, the switch can always connect them, no matter who else is already on a call. The switch never says "sorry, all lines busy" because of internal limits.

That is the **Clos network**. That is the math behind Spine-Leaf.

### Folding the Clos in half

Charles Clos's original drawing had inputs on the left, middle stages in the middle, and outputs on the right. Three stages, left to right.

In a data center, a server both sends and receives. So the inputs and outputs are the same. We can fold the Clos network in half, like folding a piece of paper. The left side meets the right side. Now we have:

- **Top of fold (the spines)** — the middle stage of the original Clos.
- **Bottom of fold (the leaves)** — the input and output stages, merged into one.

This folded shape is called a **folded Clos**. It has two visible tiers (leaves and spines) but it is mathematically a three-stage Clos. That is why some books say "two-tier" and others say "three-stage." They are talking about the same thing from different angles.

### The non-blocking proof, sketched

Here is the back-of-a-napkin version. Suppose every leaf has K downlinks (to servers) and K uplinks (to spines). Suppose there are K spines. Suppose every leaf connects to every spine with one link.

Total uplink capacity from one leaf = K spines × link speed = K × link.
Total downlink capacity from one leaf to its servers = K servers × link speed = K × link.

Up = down. The fabric can absorb every server's full bandwidth at the same time. No traffic can be blocked by lack of capacity inside the fabric. That is **1:1 non-blocking**.

If you want to save money, you can give each leaf only K/2 spine uplinks. Then you have **2:1 oversubscription**: two units of server bandwidth for every one unit of fabric bandwidth. If your servers never all talk at full speed at once (most don't), you can get away with this and save half the spine cost.

```
Non-blocking fabric (1:1):
  Server bandwidth total      ===  Spine bandwidth total

Oversubscribed (3:1):
  Server bandwidth total      ===  3 × Spine bandwidth total

(more servers than fabric can handle if they all go at once)
```

## Spine-Leaf Topology

### The wiring rules

There are exactly three rules. Memorize them.

1. **Every leaf connects to every spine.** No exceptions. If you have 4 spines, every leaf has 4 uplinks. If you have 16 spines, every leaf has 16 uplinks.
2. **Leaves never connect to other leaves.** A wire from leaf-1 to leaf-2 is forbidden in a pure Spine-Leaf design. (We will see one almost-exception later: MC-LAG peer links.)
3. **Spines never connect to other spines.** No wire from spine-1 to spine-2. The spines are siblings. They do not talk to each other.

That is the whole topology. Look at this drawing.

```
       [Spine-1]   [Spine-2]   [Spine-3]   [Spine-4]
        / | \ \      / | \ \    / | \ \    / | \ \
       /  |  \  \   /  |  \ \  /  |  \ \  /  |  \ \
      /   |   \   X    |   \ X    |   \X    |   \ \
     /    |    \ / \   |    X     |    X    |    \ \
    /     |     X    \ |   /  \   |   /  \  |     \ \
   /      |    / \    \|  /    \  |  /    \ |      \ \
  [Leaf-1] [Leaf-2] [Leaf-3] [Leaf-4] [Leaf-5] [Leaf-6]
   | | |    | | |    | | |    | | |    | | |    | | |
   srv srv  srv srv  srv srv  srv srv  srv srv  srv srv
```

Every leaf has four wires going up — one to each spine. No leaf has a wire to another leaf. No spine has a wire to another spine.

### Why exactly two hops?

A packet from server-A on Leaf-1 to server-B on Leaf-3:

1. Leaves Leaf-1 (hop 1).
2. Goes up to one of the spines (some-spine).
3. Spine forwards down to Leaf-3 (hop 2).
4. Leaf-3 hands it to server-B.

Two switch hops. Every single time. Pick any two servers in the fabric, the packet path is exactly two switch hops. This is the most important property of Spine-Leaf. Every flow has the same latency. There are no "fast" paths and "slow" paths. The fabric is **predictable**.

If the two servers are on the same leaf, that is one hop (just the leaf). That is the only exception. Some designs place chatty server pairs deliberately on the same leaf to get one-hop latency.

### Why is this better than a tree?

In a three-tier tree, the path from server to server depends on where the two servers are. Same-pod is two hops. Cross-pod is four hops. This is **non-uniform**. Application performance depends on where the scheduler put your VM.

In Spine-Leaf, every flow gets the same number of hops. The application does not need to care where it lives. The fabric is **uniform**.

## East/West vs North/South Traffic

### The two compass directions

In a data center diagram, **north** is "out toward the internet" and **south** is "down toward servers." A packet going from a server out to the internet is going **north**. A packet coming in from the internet to a server is going **south**.

If you only care about north and south, a tree was fine. Tree topologies have one fat trunk pointing north (to the internet) and many small branches pointing south (to servers). That matches the traffic.

But a packet going from one server to another server inside the same data center is going neither north nor south. It is going **sideways**. In a diagram, that looks like east or west. So we call it **east-west traffic**.

### Why east-west blew up

Modern applications are split into many small services. A single user click might trigger:

- Frontend talks to API gateway.
- API gateway talks to authentication service.
- Authentication talks to a session cache.
- API gateway talks to a recommendation service.
- Recommendation talks to a feature store, then a model server.
- API gateway talks to a database.
- Database talks to a replica for read scaling.
- API gateway talks to a logging service.
- Logging service talks to a queue.
- Queue talks to an analytics warehouse.

That is one human click. It produced ten east-west flows inside the data center for every one north-south flow with the user. East-west to north-south ratios of 4:1, 10:1, even 30:1 are normal in modern data centers.

A topology that bottlenecks east-west traffic at a single core box is no good. Spine-Leaf makes east-west the **default fast path**. Any leaf to any other leaf is two hops, with as many parallel paths as there are spines.

### Real numbers

A 32-port spine and 32-port leaf design with 4 spines and 32 leaves can move:

- 32 servers per leaf × 32 leaves = 1024 server ports.
- 4 spine uplinks per leaf × 32 leaves = 128 fabric ports.
- If server links are 25 Gbps and spine links are 100 Gbps: server total = 1024 × 25 = 25.6 Tbps. Spine total = 128 × 100 = 12.8 Tbps. That is **2:1 oversubscription**.

Hyperscalers run designs with thousands of leaves and hundreds of spines. The math scales.

## Equal Cost Multipath (ECMP)

### What if there are several equally-good paths?

A leaf wants to send a packet to another leaf. There are 4 spines. All 4 spines are equally close (one hop). All 4 paths are equally good. Which path does the leaf pick?

Old answer: pick one favorite, send everything down that one. Other three sit idle. That wastes 75% of your fabric.

New answer: **Equal Cost Multipath**, or **ECMP**. The leaf has a list of 4 next-hops, all equal cost. For each new flow, the leaf computes a hash, takes the hash modulo 4, and picks one of the 4 spines. Different flows get different hashes. They get spread across all 4 spines roughly evenly. All 4 spines stay busy. No path is idle.

### What is "a flow"?

A **flow** is a specific conversation: from this source IP and source port, to this destination IP and destination port, using this protocol. Five fields. We call this the **5-tuple**:

1. Source IP
2. Destination IP
3. Source port
4. Destination port
5. Protocol (TCP, UDP, ICMP, etc.)

Every packet in the same TCP conversation has the same 5-tuple. ECMP hashes the 5-tuple and picks a path. Because all packets in the same flow have the same hash, they all take the same path. Same path means in-order arrival. TCP loves in-order arrival.

### ECMP path distribution diagram

```
                 (4 equally-good paths to Leaf-3)
                        |
                        v
   +-------- hash(srcIP, dstIP, srcPort, dstPort, proto) % 4 -------+
   |                                                                |
   v                                                                v
 path 0 = Spine-A     path 1 = Spine-B    path 2 = Spine-C    path 3 = Spine-D

 Flow X (hash=15) -> 15 % 4 = 3  -> Spine-D
 Flow Y (hash=42) -> 42 % 4 = 2  -> Spine-C
 Flow Z (hash=7)  -> 7 % 4  = 3  -> Spine-D
 Flow W (hash=8)  -> 8 % 4  = 0  -> Spine-A
```

Even spread. Every spine pulls its weight.

### When ECMP goes wrong

ECMP is statistical. Sometimes flows hash to the same spine by bad luck. **Elephant flows** (huge long-running flows like a backup or a model training shuffle) can pin themselves to one path and saturate it while other paths are mostly idle. That is a **hash collision** problem. Solutions:

- **Adaptive routing** — the switch watches link load and steers new flows to lighter paths.
- **Flowlet switching** — when there is a tiny pause in a flow (TCP back-off), treat the next burst as a new flow and re-hash it.
- **Inner-flow hashing for VXLAN** — see next section.

## ECMP Hash for VXLAN/Geneve

### The encapsulation problem

We will explain VXLAN properly in a few sections. For now, just know that in modern fabrics, the original packet from the server gets wrapped in an outer UDP packet before it crosses the spines. The 5-tuple the spines see is the **outer** 5-tuple, not the **inner** one.

If two leaves are tunneling traffic between each other, every packet has the same outer source IP (one leaf's loopback) and same outer destination IP (the other leaf's loopback) and same outer destination UDP port (4789 for VXLAN). The only thing that varies is the outer source UDP port.

Modern leaves know this trick. When they encapsulate, they compute a hash of the **inner** flow (the real 5-tuple of the original packet) and stuff that hash into the outer source UDP port. Now the outer 5-tuple varies per inner flow. The spine, hashing the outer 5-tuple, gets the same effect as if it had hashed the inner. ECMP works again.

This trick goes by names like **entropy in the source port**, **inner-flow hashing**, **flow-aware encap**. Most modern switches do it automatically. If you ever see all your VXLAN traffic taking one spine path, your encap is dropping the inner-flow entropy and you have a bug.

## Underlay vs Overlay

### Two networks running on top of each other

Modern Spine-Leaf has two logical networks layered on top of each other:

- **Underlay** — the physical IP network. Routes between leaf loopbacks. Boring, simple, fast. Built once, mostly never changes.
- **Overlay** — the virtual network the tenants see. Their VLANs, their subnets, their MAC tables. Lives inside encapsulation tunnels (VXLAN, Geneve, NVGRE) that ride on top of the underlay.

### Why two layers?

The underlay needs to be **stable, scalable, and fast**. It only carries IP packets between leaf loopbacks. There are no broadcast domains. There is no MAC learning to worry about. The underlay scales to many thousands of switches because it is just plain old IP routing.

The overlay needs to be **flexible and programmable**. Tenants want to spin up new VLANs, new subnets, new VRFs, on demand. The overlay can change a hundred times an hour without touching the underlay at all. Imagine drawing on a whiteboard that is sitting on top of a desk. You can erase and re-draw the whiteboard all day. You never touch the desk.

### Underlay = roads. Overlay = delivery routes.

Picture a city. The streets are the underlay. They were built once and rarely change. On top of those streets, a delivery company runs a thousand trucks. Each truck has its own route, its own list of stops. The delivery company can change routes hourly without anybody ripping up the streets.

That is underlay versus overlay. The underlay is the roads. The overlay is the delivery routes.

### Encapsulation in one picture

```
Original packet from VM A to VM B:
  +-------------------------------------+
  | Eth | IP | TCP | App data           |
  +-------------------------------------+

After Leaf-1 encapsulates for VXLAN transit:
  +-------------------------------------------------------------+
  | Outer Eth | Outer IP | UDP | VXLAN hdr | (original packet)  |
  +-------------------------------------------------------------+
                                          ^
                                          |
                          The original packet is the payload
```

Outer Eth/IP/UDP get stripped at Leaf-2 (the destination VTEP). The original packet is delivered to VM B with no clue it ever rode in a tunnel.

## BGP as Underlay

### Why BGP and not OSPF or IS-IS?

In the old world, the underlay used an Interior Gateway Protocol (IGP) like **OSPF** or **IS-IS**. Those are link-state protocols. They flood every change to every router. They were designed for trees with tens of routers.

In a hyperscale Spine-Leaf with 5,000 switches, link-state floods become a problem. Every link flap causes all 5,000 switches to recompute. Convergence storms. And IGPs were not designed to express tenant or path policy easily.

**BGP** was designed for the internet, where there are hundreds of thousands of routers and constant change. BGP is a path-vector protocol. Routes are advertised hop-by-hop. Each switch only knows about its immediate neighbors. There is no flood. Convergence is local. Policy is rich.

So modern data centers run BGP **inside** the data center as the underlay routing protocol. This was popularized by Dinesh Dutt's book *BGP in the Data Center*, by the Cumulus / NVIDIA folks, and by a Facebook paper called "Introducing data center fabric, the next-generation Facebook data center network."

### AS-per-leaf design

In the simplest design, **each leaf gets its own private AS number**. The spines also each get an AS number (sometimes one AS for all spines, sometimes one per spine). Leaves run **eBGP** sessions to every spine. eBGP is BGP between different ASNs.

This means:

- Loops are broken automatically by AS_PATH. A spine will never accept a route that already has its own ASN in the path.
- No IGP needed. BGP carries everything.
- Load balancing across spines is just "BGP multipath" with equal-cost.
- Convergence is fast because every leaf only peers with N spines, not with all leaves.

```
   [Spine-1 AS 65100]   [Spine-2 AS 65100]
       |       |             |       |
      eBGP    eBGP           eBGP    eBGP
       |       |             |       |
   [Leaf-1 AS 65001]    [Leaf-2 AS 65002]
   [Leaf-3 AS 65003]    [Leaf-4 AS 65004]
```

(Some designs use one AS for all leaves and disable AS_PATH loop checks. Some use four-byte ASNs so they don't run out. Both approaches are common.)

### What does the underlay actually carry?

Just leaf loopbacks. Each leaf has a /32 loopback address (e.g. 10.0.0.1/32, 10.0.0.2/32). Those /32s are advertised to all spines via eBGP. The spines reflect them to all other leaves. Now every leaf knows how to reach every other leaf's loopback in two hops, with ECMP across all spines.

That is the entire underlay. Plain IP routes between leaf loopbacks. Nothing else.

## BGP-EVPN as Overlay Control Plane

### The overlay needs a brain

VXLAN, the encapsulation, only handles **data plane**. It says "wrap this packet, send it across, unwrap it." It does **not** answer the question "which leaf is VM B sitting behind right now?" Some other system has to answer that.

Old VXLAN designs used **multicast** in the underlay to flood discovery. Every leaf would send broadcast/unknown-unicast/multicast (BUM) traffic out a multicast group. Every leaf with that VNI would receive it. MAC learning happened by flooding, just like an old Ethernet bridge. This worked but it was slow, wasteful, and hated multicast in the underlay.

The fix is **BGP-EVPN**. EVPN stands for **Ethernet VPN**. It is a BGP address family (specifically, the **L2VPN/EVPN** address family) that lets BGP carry MAC addresses, IP addresses, and tenant identifiers. Now the control plane that distributes "where is VM B" is the same BGP we already use for the underlay. No multicast needed. Discovery is exact and fast.

### The five EVPN route types you will meet

EVPN routes come in numbered types. The most common ones:

- **Type 1 — Ethernet Auto-Discovery (A-D) route.** Used for multi-homing.
- **Type 2 — MAC/IP advertisement.** "VM with this MAC and this IP lives behind me." This is the bread and butter.
- **Type 3 — Inclusive multicast.** "I have this VNI; send me your BUM traffic for it."
- **Type 4 — Ethernet segment route.** Used for designated forwarder election among multi-homing leaves.
- **Type 5 — IP prefix route.** Used for inter-VRF routing and external prefix advertisement.

For this ELI5, only Type 2 matters: every leaf tells every other leaf about every MAC and IP behind it. Like a giant, fabric-wide ARP table.

### EVPN type-2 in one diagram

```
VM-A (mac=aa:bb:cc:dd:ee:01, ip=10.10.10.5) is on Leaf-1.

Leaf-1 advertises EVPN type-2 via BGP:
  +-----------------------------------------------------------------+
  | RD: Leaf-1-loopback:1                                           |
  | Ethernet Tag: 0                                                 |
  | MAC address: aa:bb:cc:dd:ee:01                                  |
  | IP address: 10.10.10.5                                          |
  | MPLS Label / VNI: 50100                                         |
  | Next-hop: Leaf-1 loopback (10.0.0.1)                            |
  | Route Target: 1:50100                                           |
  +-----------------------------------------------------------------+

Every other leaf with RT 1:50100 imports this route.
Now they know: "to reach aa:bb:cc:dd:ee:01, encap to 10.0.0.1 with VNI 50100."
```

### Why this rocks

- No flooding for known unicast. ARP responses can be locally generated by each leaf using the EVPN-learned IP-MAC binding (this is **ARP suppression**).
- Failover is fast. When VM-A migrates to Leaf-2, Leaf-2 advertises the type-2 route and Leaf-1 withdraws it. All leaves update their forwarding in a fraction of a second.
- Multi-tenant works. Each tenant gets a different RT (route target). Leaves only import routes whose RT they care about.

## VTEP on Every Leaf

### What is a VTEP?

A **VTEP** is a **VXLAN Tunnel End Point**. It is the thing that wraps and unwraps VXLAN packets. It needs an IP address (the **VTEP IP**) and the ability to encapsulate.

In Spine-Leaf, every leaf is a VTEP. The spines are **not** VTEPs. The spines just route plain IP between leaf loopbacks. The spines do not know or care that the IP packets they are forwarding contain VXLAN inside them.

This is a key design choice. **The intelligence lives at the leaves. The spines are dumb fast pipes.** Spines can be cheaper, simpler, more boringly stable. New features ship at the leaves where they are easier to test and roll back.

### Hardware vs software VTEP

A VTEP can be:

- **Hardware**: a switch ASIC (Tomahawk, Spectrum, Silicon One, Trident) does the encap at line rate, no CPU help.
- **Software**: a server's CPU runs the VTEP, usually via OVS, VPP, or the kernel's built-in VXLAN driver.

Hardware is faster (line-rate at hundreds of Gbps). Software is more flexible (can do crazy custom encaps). Many fabrics use both: hardware VTEPs at the leaves, software VTEPs on hypervisors. They speak the same VXLAN/EVPN, so they interop.

## Multi-Tenant Isolation

### Tenants who must not see each other

A data center often hosts many tenants. Tenant A and Tenant B are separate customers, separate companies, possibly competitors. Their traffic must never mix. Tenant A's packets must never be visible to tenant B, even if both are on the same physical wire.

We need **isolation primitives**. Spine-Leaf gives us three layered ones:

### VRF — Virtual Routing and Forwarding

A **VRF** is a separate routing table on the same router. Imagine your router has 50 routing tables instead of one, each named after a tenant. A packet arriving on a tenant-A interface gets looked up in the tenant-A table. It cannot leak into the tenant-B table because they are separate tables.

VRFs work at Layer 3. They isolate IP routing between tenants. Two tenants can both use 10.0.0.0/8 internally and never collide, because each tenant's 10.0.0.0/8 lives in a different VRF.

### EVPN VNI — Virtual Network Identifier

A **VNI** (also called a VXLAN Network Identifier) is a 24-bit number stuck in every VXLAN header. It says which virtual network this packet belongs to. There are 16 million possible VNIs (2^24). Compare to VLANs, which only have 4,096 (2^12). VNIs let you have far more tenants than VLANs ever could.

VNIs work at Layer 2. They isolate Ethernet broadcast domains.

Each VNI is its own **broadcast domain**. A broadcast frame in VNI 50100 only reaches leaves that have that VNI provisioned. Other leaves never see it.

### RD and RT — Route Distinguisher and Route Target

When EVPN routes are advertised in BGP, they need to be unique even across overlapping address spaces. The **Route Distinguisher** (RD) is a string prepended to every route to make it globally unique. Typically it is `router-id:VNI` or `router-id:VRF-id`.

The **Route Target** (RT) is a tag on every route saying "this route belongs to tenant X." Each leaf is configured with a list of RTs it imports (cares about) and exports (advertises with). Leaf-1 in tenant-A only imports RT 1:50100, so it never sees tenant-B's routes even though they are on the same BGP session.

### Three layers of isolation, working together

```
+-------------------------------------------------------+
| Tenant A's packet                                     |
|                                                       |
|   travels in: VRF "Customer_A"                        |
|   carried by: VNI 50100                               |
|   tagged with RT 1:50100 in BGP-EVPN                  |
+-------------------------------------------------------+

+-------------------------------------------------------+
| Tenant B's packet                                     |
|                                                       |
|   travels in: VRF "Customer_B"                        |
|   carried by: VNI 50200                               |
|   tagged with RT 1:50200 in BGP-EVPN                  |
+-------------------------------------------------------+
```

Three independent walls. Even if one wall has a bug, two others stand.

## Border Leaf, Service Leaf, Compute Leaf

Not every leaf has the same job. Pure Spine-Leaf still has named roles for leaves with special connections.

### Compute Leaf

The default. A leaf where servers plug in. Most leaves in a data center are compute leaves.

### Border Leaf

A **border leaf** has uplinks to the outside world: WAN routers, internet edge routers, a campus core, another data center. Border leaves do **route leaking** between tenant VRFs and the global routing table. They also do BGP peering with external routers.

Border leaves are where north-south traffic enters and leaves the fabric. They tend to be slightly beefier than compute leaves to handle the asymmetric throughput.

### Service Leaf

A **service leaf** connects to L4-L7 service appliances: firewalls, load balancers, IDS, WAF, DDoS scrubbers. Services tend to be expensive boxes that you can't easily virtualize, so they sit on dedicated leaves and are stitched into traffic via service chaining or anycast.

Some designs collapse Border + Service into one combined "border-services leaf." Either is fine.

## Multi-Site / DC-Interconnect (DCI)

### One fabric is great. Two fabrics is geography.

Most companies have more than one data center. Maybe a primary in Virginia and a backup in Oregon. Maybe a fabric in every metro. They want servers in DC1 to reach servers in DC2 without losing tenant isolation.

**DCI** (Data Center Interconnect) is the thing that stitches multiple Spine-Leaf fabrics together. The most common modern way is **EVPN multi-site**.

### EVPN multi-site, simplified

Each fabric has a pair of **border gateways** (BGW). The BGWs in DC1 peer EVPN with the BGWs in DC2. Local fabric EVPN routes get re-advertised with a BGW next-hop. From the local fabric's perspective, the BGW is "where the other DC lives." From the remote fabric's perspective, the same BGW (or its remote twin) is "where the other DC lives."

```
       Fabric-A (Virginia)              Fabric-B (Oregon)
   +---------------------+         +---------------------+
   |  Spines             |         |  Spines             |
   |   |                 |         |   |                 |
   |  Leaves             |         |  Leaves             |
   |   |                 |         |   |                 |
   |  BGW-A1   BGW-A2 ===========  BGW-B1   BGW-B2       |
   +---------------------+ DCI    +---------------------+
                              EVPN
```

VXLAN tunnels stitch end-to-end through the BGWs. Tenant isolation is preserved across sites.

### Other DCI options

- **OTV** — Cisco's older overlay transport.
- **VPLS / EoMPLS** — service-provider style L2VPN.
- **Direct VXLAN** — leaf-to-leaf VXLAN across the WAN, skipping a BGW. Usually only viable for small site counts.

The EVPN multi-site model (RFC 7432 and friends) is the modern default.

## Scale-Out

### Two axes, both linear

A Spine-Leaf scales in two directions independently:

- **More leaves = more server ports.** Want more servers? Add more leaves. The fabric absorbs them with no architectural change. The only constraint is that every new leaf needs one uplink to every spine.
- **More spines = more fabric bandwidth.** Want more east-west capacity? Add more spines. Every leaf grows another uplink. Each new spine is another ECMP path.

This is **scale-out**. You add small, identical boxes. Compare to **scale-up**, where you replace one big box with a bigger box. Scale-out is what hyperscalers do. Scale-up is what 1995 did.

### What about more than two tiers?

When you fill all the ports on the spines and still need more leaves, you add a third tier: **super-spines** (or **spine-spines** or **fabric switches**). The super-spines connect groups of spines together. This is a **5-stage Clos** (input leaf, spine, super-spine, spine, output leaf — three middle stages).

Hyperscalers run 5-stage and even 7-stage Clos. Facebook's F4 fabric and Google's Jupiter fabric are extreme cases.

```
         [SuperSpine] [SuperSpine] [SuperSpine] [SuperSpine]
          /  |  \  |   /  |  \  |  ...
   +-----+   |   +-----+
   |     +---+---+
   v             v
 [Spine][Spine][Spine][Spine]   (one pod's spines)
   |  |  |  |  |  |  |  |
   v  v  v  v  v  v  v  v
 [Leaves]               (one pod's leaves)
```

Each pod is its own folded Clos. Pods connect via super-spines.

## Oversubscription Ratios

### What does "1:1" mean?

The **oversubscription ratio** compares server bandwidth on a leaf to fabric bandwidth from that leaf:

- **1:1** — every server's full bandwidth fits in the fabric uplinks. **Non-blocking.**
- **3:1** — three units of server bandwidth for every one unit of fabric. The fabric can handle one third of all servers running flat-out at once. The most common setting.
- **6:1** — six units of server bandwidth per unit of fabric. Dense, cheap, but careful.

A typical compute leaf: 48 × 25 GbE downlinks (1.2 Tbps) and 8 × 100 GbE uplinks (800 Gbps). 1.2 / 0.8 = 1.5:1. Very lightly oversubscribed.

A storage leaf carrying RoCE traffic (storage hates loss): 1:1 because the workload demands it.

A general-purpose leaf running web servers: 3:1 is fine because web workloads are bursty.

### When oversubscription bites

If you over-pick (3:1 when you really needed 1:1), you will see latency spikes during peak. ECN will mark packets. Senders will throttle. Tail latency goes up. Pageviews per second goes down.

Always sketch your worst-case east-west pattern before you commit to a ratio. **Distributed training jobs** and **distributed databases** routinely break naive ratios.

## Active-Active First-Hop

### The default gateway problem

Every server has a default gateway. The default gateway lives on a leaf. Servers send their first-hop IP traffic to that gateway.

What if the leaf dies? Half your servers lose their gateway. They cannot send anything until the gateway comes back. This is **single point of failure** at the first hop.

Old fix: **HSRP** or **VRRP**. Two leaves share a virtual IP. One is active, one is standby. If active dies, standby takes over. But standby was idle the whole time. Half your fabric capacity wasted.

New fix: **active-active first-hop**. Both leaves are active at the same time. Both answer the same gateway IP. A server's traffic can go to either one. We have two ways to do this:

### MC-LAG / vPC / MLAG

**MC-LAG** stands for Multi-Chassis Link Aggregation Group. Two leaves act as one logical switch from the server's point of view. The server runs LACP to bond two links, one to each leaf. Both links are active. Both leaves have the same MAC for the gateway.

Vendor names for the same idea:

- **vPC** — Cisco "Virtual Port Channel."
- **MLAG** — Arista, Cumulus.
- **MC-LAG** — generic / Juniper.

This requires the two leaves to have a peer-link between them. Yes, that breaks the "no leaf to leaf" rule slightly. The peer-link is a deliberate exception, used only for state sync between the MC-LAG pair.

### EVPN Anycast Gateway

The modern, vendor-neutral, EVPN-native fix. **All leaves with that subnet share the same gateway IP and gateway MAC.** Every leaf is the gateway. A server's ARP for its gateway gets answered by whichever leaf the server is connected to. The server's traffic hits its local leaf first, which routes the packet locally if possible. No cross-leaf hop just to reach a gateway.

This requires EVPN type-2 with router-MAC extended community. All EVPN-capable platforms support it now.

```
Server-A on Leaf-1 sends to its gateway (10.10.10.1).
  -> Leaf-1 itself is 10.10.10.1. Leaf-1 routes the packet locally.

Server-B on Leaf-2 sends to its gateway (10.10.10.1).
  -> Leaf-2 itself is also 10.10.10.1. Leaf-2 routes the packet locally.

The same IP lives on every leaf. Each leaf is a local copy of the gateway.
```

Anycast gateway is the modern default. MC-LAG is fading except for legacy and specific dual-attached server cases.

## Hyperscaler Variants

### Facebook F4 / F16

Facebook published a paper in 2014 introducing their data center fabric. It was a 5-stage Clos with **48 leaves per pod** and **48 fabric switches per pod**. Each pod connected to a higher tier of **spine planes**, with **4 independent planes**, each with 48 spine switches. Total: hundreds of thousands of 10 GbE ports per data center.

Around 2019 they upgraded to **F16**, with 16 planes instead of 4 and 400 GbE links. Same idea, more parallelism.

### Google Jupiter

Google published *Jupiter Rising* at SIGCOMM 2015. Their fabric is a hierarchy of small "Centauri" switches (24-port 40 GbE) wired into a 5-stage Clos. The fabric scales to **40 Tbps per cluster** and **1.3 Pbps total**. They also pioneered **OpenFlow-style control planes** where the fabric is centrally orchestrated rather than relying on traditional distributed routing protocols.

### AWS

AWS publishes less detail but their public talks describe a similarly massive multi-stage Clos with custom silicon and a heavy reliance on simple, high-radix switches.

The takeaway: **everybody who has to scale beyond one room ends up with the same answer.** Folded Clos. Many small switches. ECMP. BGP-ish underlay. Overlay on top. Any company running a serious data center now starts here.

## Common Errors (Verbatim)

These are the real strings you will see in your logs and screen output. Memorize the ones for the platform you use.

```
% Invalid input detected at '^' marker.
```

(Cisco. You typed a command in the wrong mode or with a typo. Check `?` for completion.)

```
ERROR: cannot apply running-config (some lines may have been ignored).
```

(Cisco. Pushed a config but a couple of lines failed. Check `show run | grep <feature>` for what got accepted.)

```
% BGP: neighbor 10.0.0.2 Active
```

(Cisco. Your BGP neighbor is in the Active state, which actually means you are trying to connect and it is **not** working. This name is famously confusing.)

```
%BGP-3-NOTIFICATION: received from neighbor 10.0.0.2 4/0 (hold time expired)
```

(Hold timer expired. The peer stopped sending keepalives. Check the link, the peer, and timers.)

```
EVPN-MH: ESI-MH: ESI 00:00:00:00:00:00:00:00:00:00 is invalid
```

(EVPN multi-homing needs a non-zero ESI. You forgot to set one on the ESI-LAG.)

```
%VPC-2-VPC_KEEPALIVE_DOWN: vPC keepalive link down
```

(Cisco vPC peer-keepalive is down. Peer-link probably also degraded. Check both peer-link and peer-keepalive ASAP.)

```
duplicate IP detected: 10.10.10.5 on Leaf-1 and Leaf-2
```

(Two VTEPs both claim the same host. Check EVPN type-2 routes; one is stale or there is a real duplicate.)

```
TCAM exhausted: cannot install route 10.20.0.0/16
```

(Hardware forwarding table is full. Either reduce route count, summarize, or scale up to a deeper TCAM platform.)

```
VXLAN: encap MTU exceeded by 50 bytes
```

(Underlay MTU is too small for VXLAN payload. Bump MTU on the underlay by 50 bytes (50 = VXLAN+UDP+outer-IP+outer-Eth overhead). 9000 / jumbo is standard.)

```
LACP: link <eth1/49> is in suspended state
```

(LACP is configured but the peer is not sending LACPDUs. Check peer config and cable.)

```
PFC storm detected on interface eth1/3, pausing
```

(Storm of PFC PAUSE frames. Often a misbehaving end-host NIC. Disable PFC on that link or fix the host driver.)

```
ECN-CE bit count exceeded threshold for queue 3
```

(Congestion. ECN is doing its job. If CE marks are constant, you are oversubscribed and need more spines or less traffic.)

```
LLDP: TLV "System Name" too long, truncated
```

(LLDP frames have length limits per TLV. Cosmetic.)

```
spanning-tree: BPDU received on port nve1
```

(Someone is sending STP BPDUs into your VXLAN tunnel interface. That should never happen. Trace the source.)

```
BFD: session 192.0.2.1 down: control detection time expired
```

(BFD declared the peer down because no Hello in N intervals. Check the link and the peer.)

## Hands-On

These are real commands. Try them on a switch you own. Outputs vary slightly by version. You will recognize the shape.

### Show BGP underlay summary (Cisco NX-OS or FRR)

```
$ show ip bgp summary
BGP summary information for VRF default, address family IPv4 Unicast
BGP router identifier 10.0.0.1, local AS number 65001
BGP table version is 42, IPv4 Unicast config peers 4, capable peers 4
14 network entries and 28 paths using 4576 bytes of memory

Neighbor        V    AS MsgRcvd MsgSent   TblVer  InQ OutQ Up/Down State/PfxRcd
10.0.1.1        4 65100   12345   12345       42    0    0 1d04h00m  7
10.0.1.2        4 65100   12300   12350       42    0    0 1d04h00m  7
10.0.1.3        4 65100   12400   12400       42    0    0 1d04h00m  7
10.0.1.4        4 65100   12200   12200       42    0    0 1d04h00m  7
```

The `Up/Down` column should show real time. `State/PfxRcd` should be a number, not a state name like Active or Idle.

### Show BGP EVPN summary

```
$ show bgp l2vpn evpn summary
BGP summary information for VRF default, address family L2VPN EVPN
Neighbor        V    AS MsgRcvd MsgSent   TblVer  InQ OutQ Up/Down State/PfxRcd
10.0.1.1        4 65100   54321   54300      109    0    0 1d04h00m  256
10.0.1.2        4 65100   54300   54310      109    0    0 1d04h00m  256
10.0.1.3        4 65100   54400   54400      109    0    0 1d04h00m  256
10.0.1.4        4 65100   54350   54350      109    0    0 1d04h00m  256
```

### List EVPN routes for one VNI

```
$ show bgp l2vpn evpn vni 50100
BGP routing table information for VRF default, address family L2VPN EVPN
Route Distinguisher: 10.0.0.1:50100   (L2VNI 50100)

* i[2]:[0]:[0]:[48]:[aa:bb:cc:dd:ee:01]:[0]:[0.0.0.0]/216
                      10.0.0.2                                  0    100      0 i
*>i[2]:[0]:[0]:[48]:[aa:bb:cc:dd:ee:01]:[32]:[10.10.10.5]/248
                      10.0.0.2                                  0    100      0 i
```

Each `[2]:` is a type-2 route. The MAC and IP at the end is the host.

### Show all VXLAN VNIs

```
$ show vxlan vni
Codes: * - L3-VNI

Interface    VNI         Multicast-group   State    Mode      Type [BD/VRF]      Flags
nve1         50100       UnicastBGP        Up       CP        L2 [10]
nve1         50200       UnicastBGP        Up       CP        L2 [20]
nve1       * 50000       n/a               Up       CP        L3 [VRF tenantA]
```

### Show the VXLAN MAC/address table

```
$ show vxlan address-table
* - primary entry, G - Gateway MAC, (R) - Routed MAC

   VLAN     MAC Address      Type      age     Secure NTFY Ports
* 10       aa:bb:cc:dd:ee:01 dynamic   00:01:23  F      F   nve1(10.0.0.2)
* 10       aa:bb:cc:dd:ee:02 dynamic   00:00:42  F      F   eth1/3
* 10       0001.aabb.0001    static    -         F      F   sup-eth1(R)
```

Anything pointing at `nve1(<remote-IP>)` is learned via EVPN from a remote leaf.

### Show the IP routing table (kernel / FRR)

```
$ show ip route
Codes: K - kernel, C - connected, S - static, R - RIP, O - OSPF,
       I - IS-IS, B - BGP, ...

C>* 10.0.0.1/32 is directly connected, lo
B>* 10.0.0.2/32 [20/0] via 10.0.1.1, eth1/49, 1d04h
B>* 10.0.0.3/32 [20/0] via 10.0.1.1, eth1/49, 1d04h
B>* 10.0.0.4/32 [20/0] via 10.0.1.1, eth1/49, 1d04h
                       via 10.0.1.2, eth1/50, 1d04h
                       via 10.0.1.3, eth1/51, 1d04h
                       via 10.0.1.4, eth1/52, 1d04h
```

Multiple `via` lines on one route = ECMP. The kernel will hash flows across all of them.

### Linux ip route show, on a server using EVPN VTEP

```
$ ip route show
default via 10.10.10.1 dev eth0
10.10.10.0/24 dev eth0 proto kernel scope link src 10.10.10.5
10.0.0.0/8 nhid 12 proto bgp metric 20
        nexthop via 10.0.1.1 dev eth0 weight 1
        nexthop via 10.0.1.2 dev eth0 weight 1
        nexthop via 10.0.1.3 dev eth0 weight 1
        nexthop via 10.0.1.4 dev eth0 weight 1
```

### Show port-channel (LACP / MC-LAG)

```
$ show port-channel summary
Flags: D - Down P - Up in port-channel (members)
       I - Individual  H - Hot-standby (LACP only)
-------------------------------------------------------------------------------
Group  Port-       Type     Protocol  Member Ports
       Channel
-------------------------------------------------------------------------------
1     Po1(SU)     Eth      LACP      Eth1/1(P)    Eth1/2(P)
```

`Po1(SU)` = Switched-port, Up. Both members `(P)` = in the port-channel.

### Show LLDP neighbors

```
$ show lldp neighbors
Capability codes:
   (R) Router, (B) Bridge, (T) Telephone, (W) WLAN
   (S) Station, (O) Other
Local Intf       Chassis ID         Port ID         Hold-time  Capability
Eth1/49          spine1.dc1         Eth1/1          120        BR
Eth1/50          spine2.dc1         Eth1/1          120        BR
Eth1/51          spine3.dc1         Eth1/1          120        BR
Eth1/52          spine4.dc1         Eth1/1          120        BR
```

If you see only one or two spines, you are missing wiring. Every leaf should see every spine via LLDP.

### Linux: lldpctl on a host

```
$ lldpctl
-------------------------------------------------------------------------------
Interface:    eno1, via: LLDP, RID: 1, Time: 0 day, 00:42:18
  Chassis:
    ChassisID:    mac aa:bb:cc:dd:00:01
    SysName:      leaf1.dc1
    SysDescr:     Cisco Nexus 9000 Series ...
  Port:
    PortID:       ifname Ethernet1/3
    PortDescr:    server-rack-1-host-5
```

Fast way to check that the cable to the server matches what you think it is.

### Open vSwitch: ovs-appctl for software VTEP

```
$ ovs-appctl ofproto/trace br-int 'in_port=1,dl_src=...,dl_dst=...,...'
Flow: in_port=1, ...

bridge("br-int")
-------------
0. priority=0, NORMAL
   Resubmit to bridge "br-vxlan"

bridge("br-vxlan")
-------------
0. tunnel: set_vni:50100, set_dst:10.0.0.2, output:vxlan0
```

### kubectl: see CNI / overlay state

```
$ kubectl get pods -n kube-system
NAME                                  READY   STATUS    RESTARTS   AGE
calico-node-xxxxx                     1/1     Running   0          1d
calico-kube-controllers-xxxxxxxx-xxx  1/1     Running   0          1d
```

```
$ kubectl get bgppeers.crd.projectcalico.org
NAME       PEERIP            NODE        ASN
spine-1    10.0.1.1          (all)       65100
spine-2    10.0.1.2          (all)       65100
spine-3    10.0.1.3          (all)       65100
spine-4    10.0.1.4          (all)       65100
```

### calicoctl: BGP peer state

```
$ calicoctl get bgppeer -o wide
NAME      PEERIP        NODE      ASN
spine-1   10.0.1.1      (all)     65100
spine-2   10.0.1.2      (all)     65100

$ calicoctl node status
Calico process is running.
IPv4 BGP status
+---------------+-------------------+-------+----------+-------------+
| PEER ADDRESS  |     PEER TYPE     | STATE |  SINCE   |    INFO     |
+---------------+-------------------+-------+----------+-------------+
| 10.0.1.1      | node specific     | up    | 22:35:01 | Established |
| 10.0.1.2      | node specific     | up    | 22:35:01 | Established |
+---------------+-------------------+-------+----------+-------------+
```

### gobgp: a Go BGP daemon for testing

```
$ gobgp neighbor
Peer        AS  Up/Down State       |#Received  Accepted
10.0.1.1 65100 04:12:33 Establ      |       42        42

$ gobgp global rib summary -a evpn
Table evpn
Destination: 12, Path: 12
```

### gnmic: gNMI streaming telemetry

```
$ gnmic -a leaf1.dc1:6030 -u admin --insecure subscribe --path "/interfaces/interface/state/counters"
{
  "name": "Ethernet1/49",
  "in-octets": 12345678901,
  "out-octets": 23456789012
}
```

### vtysh / FRR shell

```
$ sudo vtysh
leaf1#
leaf1# show bgp evpn summary
...
leaf1# show ip route 10.0.0.2
Routing entry for 10.0.0.2/32
  Known via "bgp", distance 20, metric 0, best
  Last update 1d04h00m ago
  * 10.0.1.1, via eth1/49 (recursive resolution)
  * 10.0.1.2, via eth1/50 (recursive resolution)
  * 10.0.1.3, via eth1/51 (recursive resolution)
  * 10.0.1.4, via eth1/52 (recursive resolution)
```

### Linux ip neigh — see ARP entries from EVPN ARP suppression

```
$ ip neigh
10.10.10.5 dev vlan10 lladdr aa:bb:cc:dd:ee:01 REACHABLE
10.10.10.6 dev vlan10 lladdr aa:bb:cc:dd:ee:02 REACHABLE
```

If the entry shows `extern_learn` or `nud noarp`, it came from the EVPN control plane.

### bridge fdb — VXLAN forwarding table on Linux

```
$ bridge fdb show dev vxlan100
00:00:00:00:00:00 dst 10.0.0.2 self permanent
aa:bb:cc:dd:ee:01 dst 10.0.0.2 self
aa:bb:cc:dd:ee:02 dst 10.0.0.3 self
```

### ip vrf — list VRFs on Linux

```
$ ip vrf
Name              Table
-----------------------
tenantA           1001
tenantB           1002
```

### show vrf detail on a switch

```
$ show vrf detail
VRF tenantA; default RD 10.0.0.1:1001; default VPNID <not set>
  Description: Tenant A production
  VRF Table ID = 1001
  Address family ipv4 unicast (Table ID = 0x3E9):
    Connected addresses are not in global
    Export VPN route-target communities
      RT:1:1001
    Import VPN route-target communities
      RT:1:1001
```

### show running-config interface nve1

```
$ show running-config interface nve1
interface nve1
  no shutdown
  host-reachability protocol bgp
  source-interface loopback0
  member vni 50100
    suppress-arp
    mcast-group 0.0.0.0
  member vni 50000 associate-vrf
```

### show interface counters errors

```
$ show interface counters errors
Port           Align-Err  FCS-Err   Xmit-Err  Rcv-Err  UnderSize  OutDiscards
Eth1/1                 0        0          0        0          0            0
Eth1/2                 0        0          0        0          0            0
Eth1/49                0        2          0       11          0          124
```

`Rcv-Err` and `OutDiscards` going up = bad cable or congestion.

### show queuing interface

```
$ show queuing interface ethernet 1/49
slot  1
=======

Egress Queuing for Ethernet1/49 [Interface]
Queue 0: q0  scheduling: WRR weight: 25
Queue 1: q1  scheduling: WRR weight: 25  ECN: enabled threshold 80%
Queue 2: q2  scheduling: WRR weight: 25  PFC: enabled
Queue 3: q3  scheduling: WRR weight: 25
Drops: 0
ECN-marked: 12345
```

### Linux tc qdisc show — queueing on a host

```
$ tc qdisc show dev eno1
qdisc fq 8001: root refcnt 2 limit 10000p flow_limit 100p buckets 1024 ...
```

### ping with very large packet to test underlay MTU

```
$ ping -M do -s 8950 10.0.0.2
PING 10.0.0.2 (10.0.0.2) 8950(8978) bytes of data.
8958 bytes from 10.0.0.2: icmp_seq=1 ttl=64 time=0.187 ms
```

If you get `Frag needed and DF set (mtu = 1500)`, your underlay MTU is too small for VXLAN. Bump it.

### traceroute to confirm two-hop path

```
$ traceroute 10.10.20.5
traceroute to 10.10.20.5 (10.10.20.5), 30 hops max, 60 byte packets
 1  10.10.10.1   0.412 ms  0.398 ms  0.387 ms
 2  10.10.20.5   0.501 ms  0.498 ms  0.495 ms
```

Two hops. Spine-Leaf, doing what it says on the tin. (The spines are usually invisible in trace because they don't decrement the inner TTL; the outer encap hides them.)

## Common Confusions

### Spine-Leaf vs Clos vs Fat-Tree

They are mostly the same thing. **Clos** is the math (1953 telephone paper). **Fat-tree** is a specific Clos with all-equal-radius switches (Charles Leiserson, 1985). **Spine-Leaf** is the marketing name for a folded 3-stage Clos in a data center. If somebody insists they are different, ask them to draw it; you'll usually find they are arguing about the diagram, not the topology.

### Spine vs Super-Spine

A **spine** sits one tier above leaves. A **super-spine** sits one tier above spines. In a 5-stage Clos, you have leaf, spine, super-spine, spine, leaf. In a 3-stage Clos, no super-spine. If your fabric only has two tiers, you do not have super-spines.

### Underlay vs Overlay (one more time)

**Underlay** = physical IP network. Routes between leaf loopbacks. **Overlay** = virtual network on top of the underlay. Tenants live in the overlay. The underlay knows nothing of tenants.

### VLAN vs VNI

A **VLAN** is the old, 12-bit (4,096 max), L2-on-the-wire tag. A **VNI** is the new, 24-bit (16,777,216 max), inside-VXLAN identifier. Many leaves map a local VLAN to a global VNI: VLAN 10 on leaf-1 might map to VNI 50100 in the fabric.

### VXLAN vs Geneve vs NVGRE

All three encapsulate Ethernet inside IP for transport. **VXLAN** is the most popular (RFC 7348, 2014). **Geneve** is the modern replacement with extensible TLVs (RFC 8926, 2020). **NVGRE** is largely Microsoft and rare today. New designs default to VXLAN; new control planes are starting to add Geneve.

### EVPN vs VXLAN

**VXLAN** is data plane: encap. **EVPN** is control plane: who lives where. You want both. Pure VXLAN with multicast flooding is old and bad; EVPN with VXLAN data plane is the modern default.

### eBGP vs iBGP in the data center

**eBGP** is BGP between different ASNs. **iBGP** is BGP within the same ASN. Modern Spine-Leaf almost always uses eBGP, with one ASN per leaf (or per leaf pair). iBGP requires either a full mesh or route reflectors and is older.

### ECMP vs LACP

**ECMP** = Layer 3 load balancing across multiple equal-cost IP paths. **LACP** = Layer 2 link aggregation across multiple parallel links into one bundle. Both spread load. They live at different layers and solve different problems.

### MC-LAG vs vPC vs MLAG

These are vendor names for the same idea: two switches sharing a port-channel from a server's perspective. **vPC** = Cisco. **MLAG** = Arista, Cumulus. **MC-LAG** = generic. Same protocol, different marketing.

### Anycast Gateway vs HSRP/VRRP

**Anycast gateway** = every leaf is the gateway. **HSRP/VRRP** = one of two leaves is the active gateway. Anycast wins on simplicity and fairness.

### RD vs RT

**RD** (Route Distinguisher) makes routes globally unique. Each tenant's routes get a different RD so they don't collide in BGP tables. **RT** (Route Target) controls who imports the route. Different concept, both used.

### IRB vs SVI

**IRB** = Integrated Routing and Bridging. The router-MAC inside a bridge. **SVI** = Switched Virtual Interface. A logical L3 interface on top of a VLAN. They are essentially the same thing under different vendor names. EVPN docs lean on "IRB."

### 1:1 vs 3:1 oversubscription

**1:1** = non-blocking, expensive. **3:1** = oversubscribed, cheaper, fine for typical workloads. Pick based on application demand.

### North-South vs East-West

**North-south** = traffic in/out of the data center. **East-west** = server-to-server inside. Modern apps are mostly east-west.

### Spine-Leaf vs Cisco ACI

**Cisco ACI** is a specific commercial product that runs on a Spine-Leaf topology with Cisco's APIC controller. ACI uses a proprietary encapsulation and policy model on top. You can do plain Spine-Leaf without ACI; ACI is one way to operate it.

## Vocabulary

| Word | What it means in plain English |
|------|---------------------------------|
| **spine** | The top tier of switches in a Spine-Leaf fabric. Connects leaves to each other. |
| **leaf** | The bottom tier of switches. Servers plug into these. |
| **super-spine** | A third tier above spines, used when one pod's spines run out of ports. |
| **border leaf** | A leaf with uplinks to the outside world (WAN, internet). |
| **service leaf** | A leaf with firewalls, load balancers, or other L4-L7 devices attached. |
| **compute leaf** | A regular leaf where compute servers plug in. |
| **ToR** | Top-of-Rack switch. The leaf at the top of a server rack. |
| **EoR** | End-of-Row switch. An older design where a switch at the end of a row aggregates many racks. Mostly replaced by ToR. |
| **MoR** | Middle-of-Row. Same idea as EoR but in the middle. |
| **Clos network** | A type of multi-stage switching network from 1953 that is non-blocking with fewer crosspoints than a full mesh. |
| **3-stage Clos** | Original Clos with input, middle, output stages. |
| **5-stage Clos** | Extended Clos with leaf, spine, super-spine, spine, leaf. Used by hyperscalers. |
| **folded Clos** | A 3-stage Clos with input and output sides merged, used in data centers. |
| **fat tree** | A specific Clos with all switches the same size. Charles Leiserson, 1985. |
| **non-blocking** | The fabric never refuses a connection because of internal capacity limits. |
| **oversubscription** | Ratio of server bandwidth to fabric bandwidth. 1:1 = non-blocking. 3:1 = three times more server than fabric. |
| **bisection bandwidth** | The bandwidth you'd lose if you cut the fabric in half. Bigger is better. |
| **ECMP** | Equal Cost Multipath. Spreading flows across several equal-cost paths. |
| **5-tuple** | The five header fields (src IP, dst IP, src port, dst port, proto) used to identify a flow. |
| **flow** | A specific conversation between two endpoints, identified by the 5-tuple. |
| **elephant flow** | A long-running, high-bandwidth flow that can saturate one ECMP path. |
| **mouse flow** | A short, low-bandwidth flow. |
| **adaptive routing** | ECMP that watches link load and steers flows away from busy paths. |
| **flowlet switching** | Treating bursts in a flow as separate flows for ECMP re-hash. |
| **EVPN** | Ethernet VPN. A BGP address family for carrying MAC and IP info. |
| **EVPN-VXLAN** | EVPN as control plane plus VXLAN as data plane. The modern default. |
| **BGP-EVPN** | Synonym for EVPN — BGP carrying L2VPN/EVPN routes. |
| **MP-BGP** | Multiprotocol BGP. BGP with non-IPv4 address families. EVPN runs on top of MP-BGP. |
| **L2VPN** | Layer 2 VPN. EVPN is one. VPLS is another, older. |
| **L2VPN/EVPN address family** | The specific address family code in BGP for EVPN routes. |
| **RD** | Route Distinguisher. Prefix added to a route to make it globally unique in BGP. |
| **RT** | Route Target. Tag on a route saying which tenants/leaves should import it. |
| **ESI** | Ethernet Segment Identifier. Identifies a multi-homed link in EVPN. |
| **DF election** | Designated Forwarder election among multi-homing leaves to decide who forwards BUM. |
| **MC-LAG** | Multi-Chassis Link Aggregation Group. Two switches as one logical switch from a server's view. |
| **vPC** | Cisco's name for MC-LAG. Virtual Port Channel. |
| **MLAG** | Arista/Cumulus name for MC-LAG. |
| **VXLAN** | Virtual Extensible LAN. Encap that wraps Ethernet in UDP/IP. RFC 7348, 2014. |
| **VTEP** | VXLAN Tunnel End Point. The device that encaps and decaps VXLAN. Lives on every leaf. |
| **VNI** | VXLAN Network Identifier. 24-bit number identifying the virtual network. |
| **Geneve** | Generic Network Virtualization Encapsulation. Replacement for VXLAN with TLV extensibility. RFC 8926, 2020. |
| **NVGRE** | Network Virtualization using GRE. Microsoft's original encap, mostly historical. |
| **IRB** | Integrated Routing and Bridging. The L3 interface on top of a bridge. |
| **SVI** | Switched Virtual Interface. Vendor synonym for IRB. |
| **anycast gateway** | All leaves answer the same gateway IP. Modern default for first-hop redundancy. |
| **FHRP** | First-Hop Redundancy Protocol. HSRP, VRRP, GLBP. Older active/standby gateway redundancy. |
| **HSRP** | Hot Standby Router Protocol. Cisco FHRP. |
| **VRRP** | Virtual Router Redundancy Protocol. Open standard FHRP. |
| **GLBP** | Gateway Load Balancing Protocol. Cisco FHRP with built-in load sharing. |
| **VRF** | Virtual Routing and Forwarding. Multiple routing tables on one router. |
| **VRF-Lite** | A simple VRF without MPLS. Just separate routing tables, no fancy transport. |
| **route reflector** | A BGP speaker that reflects iBGP routes between peers, replacing full mesh. |
| **two-byte AS** | An AS number that fits in 16 bits (0-65535). |
| **four-byte AS** | An AS number that uses 32 bits. Needed when the world ran out of two-byte. |
| **private AS** | AS numbers reserved for internal use (64512-65534, 4200000000-4294967294). |
| **ASN** | Autonomous System Number. Unique ID for a routing domain. |
| **autonomous system** | A group of routers under one administrative control sharing one ASN. |
| **eBGP** | External BGP. BGP between different ASNs. |
| **iBGP** | Internal BGP. BGP within one ASN. |
| **AS_PATH** | The list of ASNs a BGP route traversed. Used for loop prevention. |
| **next-hop** | The router IP a packet should be sent to as the next step. |
| **loopback** | A virtual interface on a router with a stable IP, used as the router's address. |
| **TCAM** | Ternary Content-Addressable Memory. Hardware table that holds routes/ACLs. |
| **FIB** | Forwarding Information Base. The table the data plane uses for forwarding. |
| **RIB** | Routing Information Base. The control plane's table of all known routes. |
| **NDFC** | Cisco's Nexus Dashboard Fabric Controller. GUI orchestrator for VXLAN-EVPN. |
| **APIC** | Cisco's Application Policy Infrastructure Controller. Brain of ACI fabrics. |
| **Apstra** | A vendor-neutral fabric automation platform, now owned by Juniper. |
| **Ansible** | A configuration management tool. Often used to push fabric configs. |
| **ZTP** | Zero-Touch Provisioning. New switches boot up, find a server, get config and OS. No human typing. |
| **ONIE** | Open Network Install Environment. A bootloader on white-box switches that installs the NOS. |
| **NOS** | Network Operating System. The OS that runs on a switch. SONiC, Cumulus, EOS, NX-OS, Junos. |
| **SONiC** | Microsoft's open-source NOS. Runs on many vendors' hardware. |
| **Cumulus** | An open NOS based on Debian, now owned by NVIDIA. |
| **NVIDIA Cumulus** | The current name after acquisition. |
| **Arista EOS** | Arista's NOS. Linux underneath, custom CLI on top. |
| **Cisco NX-OS** | Cisco's data center NOS. Used on Nexus switches. |
| **Junos** | Juniper's NOS. Used on QFX, MX, and others. |
| **OpenConfig** | A vendor-neutral YANG model for switch config. |
| **gNMI** | gRPC Network Management Interface. Streaming telemetry and config. |
| **gNOI** | gRPC Network Operations Interface. Operations like reboot, package install. |
| **NETCONF** | The older XML/SSH-based config protocol. Predates gNMI. |
| **YANG** | A data modeling language used by NETCONF and OpenConfig. |
| **RoCE** | RDMA over Converged Ethernet. Storage and HPC traffic that needs no loss. |
| **RoCEv2** | The IP-routable version of RoCE. Most common today. |
| **RDMA** | Remote Direct Memory Access. Move data into a remote machine's RAM with no CPU help. |
| **PFC** | Priority Flow Control. 802.1Qbb. Per-class pause to make Ethernet lossless. |
| **ECN** | Explicit Congestion Notification. Marks IP packets to signal congestion instead of dropping. |
| **DCQCN** | Data Center Quantized Congestion Notification. Combines PFC and ECN for RoCEv2. |
| **DCBX** | Data Center Bridging Exchange. Negotiates PFC and ETS settings between switches. |
| **ETS** | Enhanced Transmission Selection. 802.1Qaz. Bandwidth share per class. |
| **lossless Ethernet** | Ethernet that never drops because PFC pauses upstream when buffers fill. |
| **NPU** | Network Processing Unit. The chip that forwards packets at line rate. |
| **switch ASIC** | Application-Specific Integrated Circuit for switching. The silicon brain of a switch. |
| **Tomahawk** | Broadcom's high-radix switch ASIC family. Common in spines. |
| **Trident** | Broadcom's lower-radix, more feature-rich ASIC family. Common in leaves. |
| **Spectrum** | NVIDIA/Mellanox's switch ASIC family. |
| **Silicon One** | Cisco's modern unified ASIC. |
| **Jericho** | Broadcom's deep-buffer ASIC for service-provider edges and DCI. |
| **broadcast domain** | A set of devices that all see each other's broadcasts. One VLAN or one VNI. |
| **BUM** | Broadcast, Unknown unicast, Multicast. The three things that traditionally flooded. |
| **ARP suppression** | Leaf answers ARP locally using EVPN-learned bindings. No flood. |
| **GARP** | Gratuitous ARP. A device announces its MAC unsolicited. |
| **L2 stretch** | Extending an L2 broadcast domain across multiple sites. Usually via EVPN. |
| **L3 fabric** | A fabric where every link is L3 routed. The default for modern Spine-Leaf. |
| **STP** | Spanning Tree Protocol. Old loop-prevention protocol. Mostly avoided in modern Spine-Leaf. |
| **BPDU** | Bridge Protocol Data Unit. STP control packet. |
| **MTU** | Maximum Transmission Unit. Largest packet a link can carry. Bumped to 9000+ for VXLAN. |
| **jumbo frame** | An Ethernet frame larger than 1500 bytes. Common at 9000. |
| **BFD** | Bidirectional Forwarding Detection. Sub-second link liveness checks. |
| **LACP** | Link Aggregation Control Protocol. Negotiates bonded links. |
| **LLDP** | Link Layer Discovery Protocol. Tells you who is plugged into a port. |
| **DCI** | Data Center Interconnect. The wires/protocols that join two fabrics. |
| **BGW** | Border Gateway. The leaf that interconnects fabrics in EVPN multi-site. |
| **OTV** | Cisco's Overlay Transport Virtualization. Older DCI. |
| **VPLS** | Virtual Private LAN Service. MPLS-based L2VPN. |
| **EoMPLS** | Ethernet over MPLS. A pseudo-wire. |
| **MPLS** | Multiprotocol Label Switching. Label-based forwarding, common in SP networks. |
| **segment routing** | Source-routing variant of MPLS using prepended segment IDs. |
| **EVPN type-1** | Ethernet auto-discovery route. Multi-homing. |
| **EVPN type-2** | MAC/IP advertisement route. The most common. |
| **EVPN type-3** | Inclusive multicast Ethernet tag route. BUM replication signaling. |
| **EVPN type-4** | Ethernet segment route. Designated forwarder election. |
| **EVPN type-5** | IP prefix route. Inter-VRF and external prefixes. |
| **route leak** | Deliberately importing a route from one VRF into another. |
| **fabric** | The whole switch network of a data center taken as one logical thing. |
| **pod** | One Spine-Leaf unit. Hyperscalers connect many pods. |
| **room** | One physical building or hall. Sometimes synonym for pod. |
| **rack** | A vertical unit of servers, with one (or two) ToR leaf at the top. |
| **U** | Rack unit. 1U = 1.75 inches of vertical rack space. |
| **OS upgrade** | A new firmware on a switch. Usually rolling, leaf-by-leaf. |
| **ISSU** | In-Service Software Upgrade. Upgrade firmware without dropping traffic. |
| **NSF** | Non-Stop Forwarding. Data plane keeps running while control plane reboots. |
| **GR** | Graceful Restart. Peers wait for you to come back instead of withdrawing routes. |
| **EVPN-MH** | EVPN Multi-Homing. The standards-based replacement for MC-LAG. |
| **A/A** | Active-Active. Both peers carry traffic. |
| **A/S** | Active-Standby. Only one peer carries traffic. |
| **broadcast** | A frame for everybody on the segment. |
| **multicast** | A frame for a specific group of receivers. |
| **unicast** | A frame for one specific receiver. |
| **flooding** | Sending a frame everywhere because we don't know where it should go. |
| **MAC learning** | Watching source MACs of incoming frames to remember where each MAC lives. |

## More Pictures: A Few Other Ways To See It

Different mental pictures help different people. Here are a few more, all of the same topology.

### The shopping mall, again, with details

Picture the mall again. Each shop is a leaf. Each balcony is a spine. The escalators between are the wires. Now imagine somebody adds a new shop to the mall. They hire a contractor to build the shop, then they install **one escalator from the new shop to every existing balcony**. They never run an escalator from the new shop to another shop. Customers go from any shop to any other shop in two escalator trips.

Now imagine the mall gets so popular that the four balconies are themselves overloaded. Add a fifth balcony. Run one escalator from every existing shop to the new balcony. Now there are five balconies and five parallel escalators per shop. Capacity grows in a straight line with the number of balconies. No shop has to be torn down for the upgrade.

That is exactly how a Spine-Leaf grows. Add a leaf, wire it to every spine. Add a spine, wire it to every leaf. Each operation touches only the new device and the wires from it. The rest of the fabric carries on.

### The honeycomb

Picture a honeycomb. The bottom row of cells is the leaves. The top row is the spines. Lines connect every bottom cell to every top cell. Bees fly from a bottom cell to another bottom cell by going up to a top cell, then back down. The path is always two segments of flight.

Honeycombs scale. You can keep adding cells without rebuilding the structure. The whole comb is uniform. Bees never starve waiting for a route.

### The airport hub

Picture a flat city of small airports (leaves) and a row of big hub airports (spines). To fly from one small airport to another small airport, you fly to a hub, then onward to the destination. Every small airport has flights to every hub. There are several hubs, so flights can spread across hubs.

If one hub is closed for snow, your trip routes through a different hub automatically. If a small airport opens, it joins the network the moment it has a flight to every hub.

### The square dance

Picture a square dance with two rows of dancers. The bottom row is leaves; the top row is spines. The caller (the protocol) tells partners how to pair up. Dancers in the bottom row only dance with dancers in the top row. They never dance with their neighbours in the same row. Across many beats, every bottom dancer ends up partnering with every top dancer in turn.

Same idea: leaves only talk to spines, never to other leaves. The pattern is regular. Nobody sits out.

## A Day in the Life of a Packet

Let us follow one TCP packet from VM-A on Leaf-1 to VM-B on Leaf-3, end to end. This is the most-asked question by people learning Spine-Leaf for the first time, so it is worth doing carefully.

### Step 1 — VM-A creates the packet

The application on VM-A calls `connect()` to a destination IP. The kernel decides this address is not on the same subnet, so it picks the default gateway. ARPs the gateway MAC. Builds the Ethernet frame: source MAC = VM-A's NIC MAC, destination MAC = gateway MAC, IP source = VM-A IP, IP destination = VM-B IP, TCP source port = a random ephemeral, TCP destination port = whatever the app asked for, payload = some bytes.

### Step 2 — Frame arrives at Leaf-1

The frame hits the leaf's downlink port. Leaf-1's silicon recognizes its own MAC as the destination — it is the gateway. It does an L3 lookup on the destination IP (VM-B's IP). It finds the IP/MAC binding in its EVPN-learned host table. The next-hop is the remote VTEP — Leaf-3's loopback (10.0.0.3).

### Step 3 — Encapsulation

Leaf-1 wraps the original frame in a VXLAN header, then a UDP header (dest port 4789, source port = a hash of the inner 5-tuple for ECMP entropy), then an outer IP header (source = Leaf-1's loopback, destination = Leaf-3's loopback), then an outer Ethernet header.

### Step 4 — Outer IP lookup

Leaf-1 looks up Leaf-3's loopback in its routing table. It finds 4 ECMP next-hops (one per spine). It picks one based on the outer 5-tuple hash. Off the packet goes onto the chosen uplink port.

### Step 5 — Spine forwards

The spine sees the outer IP packet. Destination = Leaf-3's loopback. Looks up the route. Sends out the port that goes to Leaf-3. The spine has no idea this is VXLAN; it never decapsulates. It is just an IP router.

### Step 6 — Frame arrives at Leaf-3

Outer IP destination = Leaf-3's own loopback. Leaf-3 looks at the outer headers, sees this is a VXLAN packet, and de-encapsulates. Inner frame is now exposed.

### Step 7 — Inner forwarding

Leaf-3 sees the inner destination MAC, the inner destination IP, the VNI. It looks up the local interface for VM-B in this VNI. Sends the original frame out the right downlink port. Decrements the inner TTL or not, depending on whether routing happened.

### Step 8 — VM-B receives

VM-B's NIC sees the frame. Strips Ethernet. Hands the IP packet to the kernel. Kernel sees TCP, hands to the right socket. App receives the bytes.

### How long did all this take?

Two switch hops. End-to-end latency on a modern fabric: 5-15 microseconds for the round trip, depending on cable lengths and switch ASIC. That is faster than the ping you'd get to a server in the same rack a decade ago.

## Diagram: Leaf-Spine Wiring Reality

A picture of a real 4-spine, 6-leaf fabric, with every wire drawn:

```
     +-------+   +-------+   +-------+   +-------+
     |Spine-1|   |Spine-2|   |Spine-3|   |Spine-4|
     +---+---+   +---+---+   +---+---+   +---+---+
         |           |           |           |
   wires|wires|wires|wires|wires|wires|wires|wires
         |           |           |           |
   ------+-----+-----+-----+-----+-----+-----+-----
   |     |     |     |     |     |     |     |    |
+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
|     |     |     |     |     |     |     |       |
| L1  | L2  | L3  | L4  | L5  | L6  | ... | total |
| 4up | 4up | 4up | 4up | 4up | 4up |     | 24 up |
+-----+-----+-----+-----+-----+-----+-----+-------+

Every leaf has FOUR uplinks (one to each spine).
Every spine has SIX downlinks (one to each leaf).
Total wires in the fabric = 4 * 6 = 24.
```

Server ports are on the bottom of each leaf, not shown.

## Diagram: The Same Fabric Drawn As ECMP Paths

Pick Leaf-1 trying to reach Leaf-4. There are 4 paths, one per spine. ECMP hashes flows across all 4:

```
                  spine1   spine2   spine3   spine4
                    A        B        C        D
                   / \      / \      / \      / \
                  /   \    /   \    /   \    /   \
                 /     \  /     \  /     \  /     \
       Leaf-1 --+      v +      v +      v +      v
                |      |  |     |  |     |  |     |
                |      |  |     |  |     |  |     |
       Leaf-4 --+------+--+-----+--+-----+--+-----+
                  via       via       via       via
                  spine1    spine2    spine3    spine4

       hash(flowX) -> spine2
       hash(flowY) -> spine3
       hash(flowZ) -> spine1
       hash(flowW) -> spine4
```

Four equal-cost paths. All four spines busy. Latency uniform.

## Diagram: VXLAN Header Math (MTU Implications)

Why your underlay MTU has to be larger than your tenant MTU:

```
+-----------------------------------------------------------------+
| Outer Eth (14)| Outer IP (20) | UDP (8) | VXLAN (8) | Inner ... |
+-----------------------------------------------------------------+
       14            20             8         8        = 50 bytes
                                                          overhead

If tenant wants 1500-byte payload:
   underlay MTU must be at least 1550 bytes.

If you run jumbo frames (9000) for tenants:
   underlay MTU must be at least 9050 bytes.

A common safe value: 9216 (covers 9000 tenant + ample slack).
```

If your underlay MTU is too small, VXLAN-encapsulated packets get fragmented or dropped. You will see "Frag needed and DF set" or just silent latency spikes. Always size underlay MTU correctly. **Always.**

## Diagram: The AS-Per-Leaf Layout

A simple BGP layout for the underlay. Only the most relevant ASNs are shown:

```
                   AS 65100 (all spines)
       +-----------+-----------+-----------+
       |           |           |           |
   Spine-1     Spine-2     Spine-3     Spine-4
       |           |           |           |
       +-----------+-----------+-----------+
       |           |           |
   Leaf-1     Leaf-2     Leaf-3
   AS 65001   AS 65002   AS 65003

   eBGP sessions between every leaf and every spine.
   AS_PATH prevents loops automatically.
```

If a route advertised by Leaf-2 (AS 65002) somehow tried to come back into Leaf-2, the AS_PATH would already contain 65002 and Leaf-2 would reject it. Loop free by construction.

## Diagram: Multi-Site DCI (EVPN Multi-Site)

Two fabrics, each with its own underlay, glued by border gateways speaking EVPN to each other:

```
            Fabric DC1                              Fabric DC2
   +----------------------+                +----------------------+
   |                      |                |                      |
   |  spines     spines   |                |  spines     spines   |
   |    |          |      |                |    |          |      |
   |  leaves    leaves    |                |  leaves    leaves    |
   |    |          |      |                |    |          |      |
   |  BGW-A1   BGW-A2 +============+  BGW-B1   BGW-B2     |
   |                      |   DCI    |                       |
   +----------------------+   IPSEC  +----------------------+
                                or MPLS
                                or dark fiber

   EVPN-VXLAN tunnels stitch end-to-end through the BGWs.
   Tenant VNIs preserved across sites.
   Tenant VRFs preserved across sites.
   ARP suppression preserved across sites.
```

If one BGW fails, the other in the pair takes over seamlessly. If both BGWs in a site fail, that site is islanded but functions internally.

## Diagram: EVPN Type-2 Advertisement Flow

A new VM boots on Leaf-1. The fabric learns about it without flooding. Here is the dance:

```
    Step 1: Server-A boots, sends GARP (or normal traffic).
       VM-A (mac=aa:bb:cc:dd:ee:01, ip=10.10.10.5)
                       |
                       v
   +---------------+
   |    Leaf-1     |  Local MAC learn on port eth1/3.
   +-------+-------+  Local ARP/ND snoops the IP-MAC binding.
           |
           | (BGP-EVPN type-2 route)
           |
           v
   +---------------+
   | Spine (route- |  Spine route-reflects type-2 to all peer leaves.
   |   reflector)  |
   +-------+-------+
           |
       fanout to all leaves
           |
           v
   +---------------+   +---------------+   +---------------+
   |    Leaf-2     |   |    Leaf-3     |   |    Leaf-N     |
   +---------------+   +---------------+   +---------------+
   Each leaf installs:
     - MAC aa:bb:cc:dd:ee:01 -> nve1 next-hop 10.0.0.1, VNI 50100
     - ARP entry 10.10.10.5 -> aa:bb:cc:dd:ee:01 (in EVPN-imported VRF)
```

Now if any other leaf has a server in VNI 50100 that needs to talk to VM-A, it knows exactly where to go. No flood.

When VM-A migrates to a different leaf (say a vMotion event), Leaf-1 withdraws the type-2 route and the new leaf advertises a fresh one. All other leaves update their forwarding entries within milliseconds.

## A Quick Note on Failure Modes

Spine-Leaf fails gracefully but you should know the modes.

### Single spine fails

You lose 1/N of fabric capacity. ECMP detects via BFD and re-converges. Packets in flight may be dropped for the duration of one BFD detection interval (commonly 150 ms). Then traffic continues across the remaining spines. Application-level performance might dip briefly under sustained heavy load.

### Single leaf fails

Servers attached to that leaf go offline (unless dual-attached via MC-LAG/anycast gateway to a peer). The fabric itself is unaffected. Routes for hosts behind that leaf get withdrawn and the rest of the fabric stops sending to it.

### Two spines fail simultaneously

Now you have lost 2/N of fabric capacity. If your fabric was sized at 4 spines and 1:1, you are now 1:2 oversubscribed. Tail latency rises. SLOs may bend. Add capacity or fix the spines fast.

### Spine-leaf wire fails

ECMP shrinks. The leaf has N-1 paths to that spine via the remaining spines, sort of — wait, no. The wire that died was a direct leaf-to-spine link. If only that link died, the leaf simply removes that ECMP next-hop. All other paths still work. ECMP shrinks from N to N-1.

### Underlay route flaps

A leaf is announcing and withdrawing a route over and over. BGP's dampening kicks in, suppressing the noisy route. Some leaves stop hearing about it. This is bad. Diagnose root cause; do not just disable dampening.

### EVPN type-2 stale

A VM moved but the old leaf is still advertising it. Two leaves both claim VM-A. Traffic split-brains. The fix is normally MAC mobility tracking (the EVPN sequence number extended community), which BGP-EVPN handles automatically when the new leaf wins by sequence.

## Performance Tuning Checklist

If your fabric is under heavy load and you want to tune:

1. **MTU.** Underlay 9216 minimum. Tenant 9000 if hosts can use it.
2. **ECMP polarization.** Make sure every leaf's hash includes inner-flow entropy for VXLAN. Symptom: one spine carries 80% of traffic.
3. **Buffer tuning.** Leaves with shallow buffers under bursty incast workloads will drop. Consider deep-buffer leaves in storage and HPC pods.
4. **Cut-through vs store-and-forward.** Cut-through saves a few microseconds per hop. Store-and-forward catches errors. Most modern leaves do cut-through unless you tell them otherwise.
5. **PFC and ECN tuning for RoCEv2.** Get the thresholds right or you get incast collapse or PFC storms.
6. **BGP timers.** Hold time of 9 seconds, keepalive of 3 seconds is common. With BFD enabled, you can be much more aggressive.
7. **BFD intervals.** 50 ms x 3 missed = 150 ms detect time. Aggressive enough for fast failover; not so aggressive you trigger false positives during normal CPU load.
8. **Anycast gateway placement.** Always at the leaf, never at the spine.
9. **Route summarization.** Summarize tenant subnets where possible so leaf TCAM does not explode at scale.
10. **Multicast.** Default to ingress-replication for BUM in VXLAN-EVPN. Don't run multicast in the underlay unless you have a reason.

## Try This

If you have a switch lab or a virtual fabric (FRR + Cumulus VX + GNS3, or Containerlab + SONiC, or EVE-NG + Nexus 9000v):

1. Build a 2-spine, 4-leaf topology in your simulator. Wire every leaf to every spine. Do **not** wire any leaf to any other leaf.
2. Configure each leaf with its own private ASN and each spine with one shared ASN (say 65100).
3. Bring up eBGP between every leaf and every spine. Use unnumbered BGP (peering to interface, not IP) if your platform supports it — modern FRR does, and it makes life much simpler.
4. Verify the underlay: from leaf-1, ping leaf-2's loopback. Confirm there are 2 ECMP paths in `show ip route`.
5. Configure VXLAN and EVPN on every leaf. Pick a VNI (say 50100), bind it to a VLAN, put two server interfaces in that VLAN.
6. Bring up an EVPN BGP session for the L2VPN/EVPN address family.
7. Plug a "server" (a Linux container with an IP in the VNI's subnet) into leaf-1 and another into leaf-3.
8. Ping between the two servers. They are in different racks. Their packet rides VXLAN across the spines.
9. Run `tcpdump -i any -nn vxlan` on a spine. You will see UDP/4789 traffic.
10. Take down one spine. Watch ECMP shrink to one path. Confirm the ping keeps flowing.
11. Bring the spine back. Watch ECMP grow back to two paths. Confirm convergence in well under a second with BFD enabled.
12. Add a third tenant with a new VNI and a new VRF. Confirm they do not see each other.
13. Read your `show bgp evpn` carefully. Find a type-2 route. Identify the MAC, IP, RD, RT, and next-hop. That is the heartbeat of EVPN.

You now understand Spine-Leaf better than most network engineers who got into the field before 2015.

## Where to Go Next

- Read the deep-dive theory page for this topic if it exists.
- Read `networking/data-center-design` for design considerations beyond pure topology.
- Read `networking/evpn-advanced` for the type-1/3/4/5 routes and multi-homing.
- Read `networking/vxlan` for the data-plane details (header layout, UDP port choice, MTU math).
- Read `networking/geneve` for the modern replacement encap.
- Read `networking/ecmp` for the load-balancing math, hash variants, and flow tracking.
- Read `networking/segment-routing` for the SR/SR-MPLS world that is starting to merge with EVPN at the WAN edge.
- Read `networking/cisco-aci` for the proprietary policy model that runs on Spine-Leaf.
- Read `ramp-up/bgp-eli5` again, now that you have seen BGP in this context.
- Build a Containerlab topology and break things on purpose. Pull a cable mid-flow. Reboot a spine. Misconfigure an RT. The pain is the lesson.

## See Also

- `networking/data-center-design`
- `networking/evpn-advanced`
- `networking/vxlan`
- `networking/geneve`
- `networking/lacp`
- `networking/ecmp`
- `networking/segment-routing`
- `networking/mpls`
- `networking/bgp-advanced`
- `networking/sp-qos`
- `networking/cisco-aci`
- `networking/cisco-dna-center`
- `ramp-up/bgp-eli5`
- `ramp-up/ip-eli5`
- `ramp-up/linux-kernel-eli5`

## References

- Charles Clos, *A Study of Non-Blocking Switching Networks*, Bell System Technical Journal, March 1953. The original paper. Surprisingly readable for a 70-year-old math paper.
- Charles Leiserson, *Fat-Trees: Universal Networks for Hardware-Efficient Supercomputing*, IEEE Transactions on Computers, 1985. The supercomputing-network ancestor of modern Spine-Leaf.
- Dinesh Dutt, *BGP in the Data Center*, O'Reilly, 2017. The standard reference for using BGP as a data center underlay. Free PDF from Cumulus/NVIDIA at the time of writing.
- NVIDIA / Cumulus whitepapers on EVPN-VXLAN. Several well-written guides on Cumulus Linux configuration for the modern fabric.
- Alexey Andreyev, *Introducing data center fabric, the next-generation Facebook data center network*, Facebook Engineering blog, November 2014. The F4 paper.
- Arjun Singh et al., *Jupiter Rising: A Decade of Clos Topologies and Centralized Control in Google's Datacenter Network*, SIGCOMM 2015. Google's fabric history and architecture.
- RFC 7348, *Virtual eXtensible Local Area Network (VXLAN)*, August 2014. The data-plane encapsulation.
- RFC 7432, *BGP MPLS-Based Ethernet VPN*, February 2015. EVPN over MPLS — the foundational EVPN spec.
- RFC 8365, *A Network Virtualization Overlay Solution Using Ethernet VPN (EVPN)*, March 2018. EVPN with VXLAN/Geneve/NVGRE data planes.
- RFC 8926, *Geneve: Generic Network Virtualization Encapsulation*, November 2020. The modern replacement for VXLAN.
- RFC 7637, *NVGRE: Network Virtualization Using Generic Routing Encapsulation*, September 2015. Microsoft's original encap, mostly historical now.
- IEEE 802.1Qbb, *Priority-based Flow Control*, the spec for PFC.
- IEEE 802.1Qaz, *Enhanced Transmission Selection*, the spec for ETS and DCBX.
- *Network Algorithmics* by George Varghese — covers the underlying packet-processing techniques (TCAM, hashing, ECMP) that make all of this work in hardware.
