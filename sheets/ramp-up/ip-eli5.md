# IP — ELI5 (The Internet's Postal Address System)

> IP is the postal addressing system of the internet. Every device has an address, every packet has a "from" and a "to," and routers are the sorting centers that hand the package on toward where it needs to go.

## Prerequisites

(none — but `cs ramp-up binary-numbering-eli5` is a great companion if the address arithmetic feels weird)

This sheet is the very first stop for understanding IP. You do not need to know what a "packet" is. You do not need to know what a "router" is. You do not need to know what binary or hex are. By the end of this sheet you will know all of those things in plain English, and you will have typed real commands into a real terminal and watched the network actually answer back.

If a word feels weird, look it up in the **Vocabulary** table near the bottom. Every weird word in this sheet is in that table with a one-line plain-English definition.

If you see a `$` at the start of a line in a code block, that means "type the rest of this line into your terminal." You do not type the `$`. The lines underneath that don't have a `$` are what your computer prints back at you. We call that "output."

## What Even Is IP?

### The post office story

Imagine you want to send a birthday card to your cousin in another city. You write the card. You put it in an envelope. On the envelope, you write two addresses. The first address is yours: **From: 123 Maple Street, Springfield**. The second address is your cousin's: **To: 47 Oak Avenue, Riverdale**. You drop the envelope in a mailbox.

What happens next? You don't know. You don't care. The post office takes care of everything. A truck picks the envelope up and drives it to the local post office. The local post office looks at the destination ("Riverdale") and says, "this isn't local, this needs to go on the long-distance truck." The long-distance truck drives the envelope to a regional sorting center. The regional sorting center looks at the destination again ("Oak Avenue, Riverdale") and says, "this needs to go on the truck headed to Riverdale." Another truck. Another stop. Eventually a mailman in Riverdale walks down Oak Avenue with your envelope in hand, finds number 47, and drops it in the mailbox.

You did not draw a map. You did not tell the post office which roads to drive. You did not coordinate any trucks. You just wrote two addresses on the envelope and let the post office figure it out. This is the magic of postal mail: the addressing is enough.

**IP is the post office for the internet.**

Every device on the internet has an address. Every chunk of data — we call those chunks **packets** — has a from-address and a to-address written on it. The data devices we call **routers** are the sorting centers. They look at the destination address, decide which direction the packet needs to go next, and forward it along. Hop by hop, the packet gets closer to where it's going. Eventually the destination device opens up the packet and reads the data inside.

You did not draw a map. You did not coordinate routers. You just gave the destination address and let IP figure it out.

That's it. That's IP. The whole internet runs on this idea.

### A second picture: pneumatic tubes in an old department store

Imagine you have walked into one of those old-fashioned department stores. The store is huge. There are dozens of departments on different floors: shoes, hats, kitchenware, toys, books. Every department needs to send little messages to the central office: "we need more red shoes," "a customer wants to return this hat," "what's the price on this teapot?"

Instead of running messages by hand, the store has **pneumatic tubes**. Each department has a station. You write your message, you stuff it into a small canister, you slide the canister into the tube, and a hiss of compressed air whooshes the canister through a network of tubes to its destination.

But how does the canister know where to go? Easy. You write the destination on the outside. "To: Central Office." Or "To: Shoe Department." There is a switching room in the basement where all the tubes meet. When a canister arrives at the switching room, an operator looks at the destination and pushes the canister into the right tube to send it on its way. Sometimes a canister has to pass through several switching rooms before it finally reaches the destination.

The pneumatic tube network is IP. The canisters are packets. The switching rooms are routers. The destination address is the IP address. The hiss of air is the link layer (Wi-Fi, Ethernet, fiber). The whole system works because every canister carries its own address.

### What IP is not

IP is sometimes confusing because people mix it up with other things. Here is what IP is not.

- **IP is not a guarantee.** Just like the post office can lose a letter, IP can lose a packet. IP says "I will try my best." It does not say "I promise this will arrive." That is why we say IP is **best-effort**. If you need a guarantee, you use TCP on top of IP. TCP keeps track of which packets made it and asks for the lost ones again. IP itself doesn't bother.
- **IP is not the cable.** IP is the addressing system. The actual moving of bits down a cable is a different layer below IP, called the **link layer** (Ethernet, Wi-Fi, fiber, etc.). IP is the address you write on the package. The link layer is the truck.
- **IP is not a name.** "google.com" is a name, not an IP address. Names are turned into IP addresses by another system called DNS. IP only deals with numbers.
- **IP is not security.** IP carries packets but does nothing to protect them from being read or tampered with. That is why we wrap things in TLS for security. IP just carries the box; TLS makes the box opaque.

### Layers — where IP fits

People talk about "layers" of networking. Here is a simple stack from bottom to top.

```
+---------------------------------------------+
|  Application (your browser, email, etc.)    |
+---------------------------------------------+
|  Transport (TCP / UDP)                      |
+---------------------------------------------+
|  Network    (IP)              <-- you are here
+---------------------------------------------+
|  Link       (Ethernet / Wi-Fi / fiber)      |
+---------------------------------------------+
|  Physical   (electrons / photons / radio)   |
+---------------------------------------------+
```

You can think of these like the layers of a wedding cake. Each layer talks to the layer right above and right below it. The IP layer talks down to the link layer (which actually moves bits) and up to the transport layer (which decides whether it's TCP or UDP that wants to use IP).

We call IP **layer 3**. The link layer is **layer 2**. The transport layer is **layer 4**. These numbers come from a chart called the OSI model. You will see them everywhere. Just remember: **IP = layer 3 = the network layer**.

### Every device has an address

Your laptop has an IP address. Your phone has an IP address. Your printer has an IP address. Your Wi-Fi router has multiple IP addresses (one for the network inside your house and one for the rest of the internet). Servers on the internet (like the one running google.com) have IP addresses. Even smart light bulbs have IP addresses.

If a device wants to talk over the internet, it needs an IP address. Period.

Some devices have **one** IP address. Some have **many** (one per network connection — Ethernet plus Wi-Fi plus VPN, etc.). Big servers might have dozens. The kernel manages all of this for you. You usually don't have to think about it.

### Every packet has from and to

Every packet on the internet looks roughly like this on the wire.

```
+----------------------------------+
| FROM: 203.0.113.42               |   <- source IP address
| TO:   198.51.100.7               |   <- destination IP address
| ... other header fields ...      |
| ... a chunk of actual data ...   |   <- payload
+----------------------------------+
```

That's an IP packet. The address part is at the top (we call it the **header**), and the actual data is below it (we call it the **payload**). The whole thing might be a tiny 60-byte ping or a chunky 1500-byte chunk of a web page. Either way, the structure is the same: header on top, payload below, source and destination addresses always in the header.

When the destination device receives the packet, it strips off the IP header (it has done its job — gotten the packet here) and hands the payload up to the next layer. If the payload is a TCP segment, it goes to the TCP code. If it's a UDP datagram, it goes to UDP. If it's a ping, it goes to ICMP. All of these are layer-4 protocols. IP doesn't care which one — it just carries them.

### Hop by hop

A packet doesn't go from your computer directly to a server in some data center across the world. It hops through many routers along the way.

Your packet leaves your laptop. It goes to your home router. Your home router forwards it to your **internet service provider** (ISP — the company you pay for internet). The ISP has many routers. Your packet bounces through several of them. Then it crosses to another ISP, perhaps under the ocean through a fiber cable. It hops through more routers in another country. Eventually it reaches the data center where the server lives. A router there forwards it to the right server. The server reads it.

Every hop, a router looks at the destination address and decides where to send the packet next. The router has a table called a **routing table** that says "if the destination starts with these numbers, forward through that interface." Like the post office sorting bins. Bin one is for letters going north. Bin two is for letters going south. The router does the same thing with packets.

A typical packet might travel through **10 to 20 routers** between source and destination. Each hop adds a tiny bit of delay (because the packet has to be processed). The total delay we call **latency**, usually measured in milliseconds.

### Death by TTL

Sometimes packets get lost in loops. Maybe two routers each think the destination is "the other way" and bounce a packet back and forth forever. To prevent this, every IP packet has a **TTL** field — Time To Live.

TTL starts at some number (usually 64 or 128). Every router that handles the packet subtracts 1 from the TTL. If a router subtracts 1 and gets 0, it throws the packet away (and usually tells the sender about it with an ICMP message). This stops loops dead. A packet can never live forever.

TTL is measured in **hops**, not in time. Even though it's called "Time To Live," it really counts router hops, not seconds. (In IPv6, the same field is more honestly called the **Hop Limit**.)

This is also how `traceroute` works. It sends packets with increasing TTL: first a packet with TTL=1 (which the first router kills, telling traceroute "I killed it"), then TTL=2 (which the second router kills), and so on. Each "I killed it" message tells traceroute about one hop in the path. After enough probes, traceroute has a list of every router between you and the destination.

## IPv4 Addresses

The version of IP that's been running the internet since 1981 is called **IPv4**. (There is also IPv6, which we'll get to.) IPv4 addresses are 32 bits long.

### What is "32 bits"?

A bit is a single 0 or 1. 32 bits means a string of 32 zeros and ones. For example: `11000000 10101000 00000001 00101010`. That's 32 bits — a valid IPv4 address — but humans cannot read those.

So we break the 32 bits into 4 chunks of 8 bits. Each chunk is called an **octet** (Latin for "group of eight"). We write each octet as a regular decimal number from 0 to 255. We separate the octets with dots. Like this:

```
11000000.10101000.00000001.00101010
   192   .   168  .    1   .    42
```

So that 32-bit address becomes `192.168.1.42`. We call this format **dotted-quad** or **dotted-decimal**. It is the universal way humans write IPv4 addresses.

A small reminder: each octet ranges from 0 to 255. So `192.168.1.42` is fine. `999.0.0.1` is **not** a valid IPv4 address — 999 is too big for one octet.

### How many addresses is that?

32 bits gives 2^32 = **4,294,967,296** addresses. About 4.3 billion. That sounds like a lot. It isn't. There are over 8 billion humans on Earth. There are tens of billions of devices that want to connect to the internet (phones, watches, cars, smart fridges, doorbells, etc.). 4.3 billion addresses simply isn't enough.

Worse, not all 4.3 billion are usable. Big chunks are reserved for special purposes (loopback, multicast, broadcast, private use, future use). The actually-routable public IPv4 space is something like 3.7 billion addresses.

The world ran out of fresh IPv4 addresses in **2011**. The five regional internet registries (RIRs — the organizations that hand out blocks of addresses) drained their pools one by one between 2011 and 2019. There are now no more fresh IPv4 blocks to give out. Companies trade them on a secondary market. A single IPv4 address can cost $40 to $60 each, and getting a new block of 1024 addresses can cost tens of thousands of dollars.

### How did we cope with running out?

Three big tricks kept IPv4 alive past its expiration date.

**Trick 1: CIDR.** Old IPv4 had only three sizes of network blocks (called classes A, B, and C — see below). With CIDR, networks can be any power-of-two size. This lets ISPs hand out exactly the right size to each customer, instead of wasting big chunks. CIDR is so important it has its own section below.

**Trick 2: NAT.** Network Address Translation lets many private addresses share one public address. Almost every home Wi-Fi router does this. Your laptop, your phone, your TV — all of them have private addresses inside your house, but they all appear to the rest of the internet as a single public address. NAT is the entire reason your home Wi-Fi works. It also gets its own section below.

**Trick 3: IPv6.** A whole new version of IP with a vastly larger address space. The proper long-term fix. Slow rollout, but climbing every year. Also gets its own section below.

### IPv4 address classes (mostly historical)

Long ago, IPv4 was divided into "classes" based on the first few bits of the address. You will sometimes still hear these names:

- **Class A** — addresses from `1.0.0.0` to `127.255.255.255`. The first octet identifies the network; the rest are hosts. About 16 million hosts per network. There were 128 of these.
- **Class B** — addresses from `128.0.0.0` to `191.255.255.255`. First two octets are the network. About 65 thousand hosts per network. There were about 16 thousand of these.
- **Class C** — addresses from `192.0.0.0` to `223.255.255.255`. First three octets are the network. 256 hosts per network. There were about 2 million of these.
- **Class D** — addresses from `224.0.0.0` to `239.255.255.255`. Multicast.
- **Class E** — addresses from `240.0.0.0` to `255.255.255.255`. Reserved/experimental.

The class system was wasteful. A company that needed 5,000 hosts would get a class B (65,000 addresses) — 60,000 of them wasted. Or it would have to glue together a bunch of class C blocks, making routing tables complicated. So in 1993 we replaced classes with CIDR. The old class boundaries don't really matter anymore. But the names linger in casual speech: people still say "a class C" to mean "a /24-sized network."

### A small map of the IPv4 address space

```
0.0.0.0/8        — "this network" (special)
10.0.0.0/8       — private (RFC 1918)
127.0.0.0/8      — loopback
169.254.0.0/16   — link-local
172.16.0.0/12    — private (RFC 1918)
192.168.0.0/16   — private (RFC 1918)
198.18.0.0/15    — benchmark testing (RFC 2544)
224.0.0.0/4      — multicast
240.0.0.0/4      — class E (reserved)
255.255.255.255  — broadcast
... everything else: unicast public addresses ...
```

We will go through each of these in the **Public vs Private vs Reserved** section.

## IPv6 Addresses

Because IPv4 ran out, the internet community designed a new version. They skipped 5 (it was a draft for an experimental protocol) and called it **IPv6**. IPv6 was first published in 1995 and finalized in 1998. It became a full standard in 2017.

### What is "128 bits"?

IPv6 addresses are **128 bits** long. Four times as wide as IPv4.

How many addresses is that? 2^128. Let me write that out:

```
2^128 = 340,282,366,920,938,463,463,374,607,431,768,211,456
```

That is **340 undecillion** addresses. Or about **3.4 × 10^38**.

Numbers that big don't really make sense to humans, so here's an analogy. There are about 10^49 atoms in the Earth. So IPv6 has roughly enough addresses to give every atom in your fingernail its own IP. There are estimated to be 10^21 grains of sand on Earth. IPv6 has way more addresses than that. Way, way more.

It is so much address space that nobody seriously expects we'll run out. Even if we colonize the solar system and every grain of dust gets its own address, we'd still have addresses left over.

### Notation

IPv4 had four octets in dotted-decimal. IPv6 has 16 bytes — too many for dotted decimal. So we use a different notation: **eight groups of 16 bits, written in hex, separated by colons**. Each 16-bit group is sometimes called a **hextet**. Like this:

```
2001:0db8:85a3:0000:0000:8a2e:0370:7334
```

That is one full IPv6 address. Eight hextets. Each hextet is up to 4 hex digits.

That is hard to read, so the standard allows two shortcuts.

**Shortcut 1: drop leading zeros in each hextet.**

```
2001:0db8:85a3:0000:0000:8a2e:0370:7334
2001:db8:85a3:0:0:8a2e:370:7334
```

Both lines mean exactly the same address. The second is easier to read.

**Shortcut 2: replace one run of consecutive zero hextets with `::`.**

```
2001:db8:85a3:0:0:8a2e:370:7334
2001:db8:85a3::8a2e:370:7334
```

The `::` swallows up the two zero hextets. This shortcut can only be used **once** per address. Otherwise, you'd lose track of how many zero hextets it stands for.

A few common addresses in this short form:

- `::1` — the IPv6 loopback (equivalent to IPv4 `127.0.0.1`).
- `::` — the all-zeros address (equivalent to IPv4 `0.0.0.0`).
- `2001:db8::` — a "documentation" prefix used in books and tutorials.
- `fe80::1` — the most common link-local address.

There are formal rules about how to write IPv6 addresses (RFC 5952). The big ones are: always lowercase the hex letters, drop leading zeros, and use `::` once on the longest run of zeros (and don't use it for a single zero hextet).

### Address types — global, link-local, etc.

Not every IPv6 address can go anywhere. There are different scopes.

- **Global unicast** — routable across the whole internet. Anything in `2000::/3` is global. (Like a street address that mail trucks across the world know how to deliver to.)
- **Link-local** — only valid on the local link. Anything in `fe80::/10` is link-local. (Like an apartment number — only meaningful inside one building.)
- **Unique local** (ULA) — like IPv4 private. Anything in `fc00::/7` (in practice `fd00::/8`). Routable inside an organization, not across the public internet.
- **Multicast** — one-to-many. Anything in `ff00::/8`.
- **Loopback** — `::1`. Talking to yourself.
- **Unspecified** — `::`. Used as "I don't have an address yet" during configuration.

### Why we needed it

The big reason is address space. IPv4 ran out. IPv6 has effectively unlimited room.

But there are smaller upsides too.

- **Simpler header.** IPv6 dropped a bunch of rarely-used IPv4 fields. The fixed header is exactly 40 bytes. Routers can process it faster.
- **No router-side fragmentation.** In IPv4, any router on the path can break a packet up if it's too big for the next link. That's complicated and slow. IPv6 says only the original sender can fragment, with the help of ICMPv6 messages. Routers just pass packets through.
- **Built-in multicast.** IPv6 has multicast as a first-class feature. Lots of important protocols (NDP, DHCPv6) use multicast instead of broadcast.
- **No more NAT (mostly).** Because every device can have a real public address, NAT isn't strictly needed. Some networks still use it for policy reasons, but the protocol doesn't push you toward it.
- **Stateless auto-config (SLAAC).** A device can come up on a network and configure its own IPv6 address with no DHCP server, just by listening for a router advertisement. Plug-and-play addressing.

### Why isn't IPv6 everywhere yet?

Inertia. Most networks already had IPv4 working; IPv6 felt like a lot of work for not much gain. NAT papered over the address shortage well enough. Training engineers in IPv6 takes time. Some old gear doesn't speak IPv6.

But adoption has steadily climbed. As of 2026, around **45-50%** of internet traffic to large content providers (Google, Facebook, etc.) is over IPv6. Mobile networks have been particularly aggressive — most modern phones use IPv6 for everything by default. Major content providers run **dual-stack** (both IPv4 and IPv6 simultaneously). Eventually IPv4-only networks will become the exception.

If you're reading this in 2026 or later, your phone is almost certainly using IPv6 right now without you noticing.

## Public vs Private vs Reserved

Not every IPv4 address is routable across the public internet. The IETF (the body that designs internet protocols) reserves chunks for special uses. Let's walk through them.

### RFC 1918 — private use

These three blocks were carved out by **RFC 1918** in 1996 for use inside private networks (homes, offices, etc.). They are **not routable** across the public internet — every backbone router actively discards traffic to or from these addresses.

```
10.0.0.0/8       — about 16.7 million addresses (huge networks)
172.16.0.0/12    — about 1 million addresses (medium networks)
192.168.0.0/16   — about 65 thousand addresses (small networks)
```

`192.168.x.x` is the most famous because almost every consumer Wi-Fi router uses something in there for the home network. `192.168.1.x` and `192.168.0.x` are by far the most common defaults.

The whole point of RFC 1918 is: many private networks can use the same private addresses without conflict, because their addresses never escape into the public internet. Your home network has `192.168.1.42` and so does your neighbor's. You don't conflict because the address never leaves your house.

If you want to send a packet from `192.168.1.42` to a server out on the internet, NAT is what makes it work. (See **NAT** section below.)

### Loopback — `127.0.0.0/8`

The whole `127.0.0.0/8` block (16.7 million addresses!) is reserved for **loopback** — the computer talking to itself. By far the most common one is `127.0.0.1`, which has the nickname `localhost`.

When a program on your computer connects to `127.0.0.1`, the packet never goes near a network card. The kernel just hands it back to itself. It's the absolute fastest possible network connection (no wires, no Wi-Fi, no anything).

People run local servers (databases, web apps in development) on `localhost` so that nobody else can reach them.

In IPv6, the loopback is `::1`.

### Link-local — `169.254.0.0/16` and `fe80::/10`

If a device cannot get an address from DHCP, it gives itself a random address in `169.254.0.0/16` (IPv4) or `fe80::/10` (IPv6). This is so it can at least talk to other devices on the same wire.

You will sometimes see `169.254.x.x` in `ip addr` output when DHCP failed. It's almost always a sign of trouble — your computer wanted a real address but couldn't get one.

In IPv6, link-local is way more important. Every IPv6 interface always has a link-local address. Many IPv6 things (like Neighbor Discovery, see below) use the link-local address even when the interface also has a global address.

### Multicast — `224.0.0.0/4` and `ff00::/8`

A multicast address is for one-to-many delivery. A packet sent to `224.0.0.5` will be delivered to every device on the local network that has subscribed to that multicast group.

Common multicast addresses you'll see:

- `224.0.0.1` — all hosts on the link.
- `224.0.0.2` — all routers on the link.
- `224.0.0.5`, `224.0.0.6` — OSPF (a routing protocol).
- `224.0.0.9` — RIP (another routing protocol).

In IPv6:

- `ff02::1` — all nodes on the link.
- `ff02::2` — all routers on the link.
- `ff02::5`, `ff02::6` — OSPFv3.

Multicast is how a lot of "everybody pay attention" messages on a local network work. ARP doesn't use multicast (it uses broadcast), but its IPv6 replacement NDP does.

### Broadcast — `255.255.255.255`

Sending to `255.255.255.255` means "everyone on the local link." Every device on the network gets the packet. Used for things like DHCP DISCOVER (a brand-new device shouting "is there a DHCP server out there?").

Each subnet also has its own **directed broadcast** address — the all-ones host portion. For `192.168.1.0/24`, that's `192.168.1.255`. Sending to `192.168.1.255` from outside the subnet would (theoretically) reach all hosts on that subnet, but most routers block this for security reasons (it was abused in old denial-of-service attacks called "smurf attacks").

IPv6 has no broadcast — multicast covers that role.

### Class E — `240.0.0.0/4`

Reserved long ago for "future use." The future never came. It's still reserved. Don't use it. Some software won't even let you (it considers `240.0.0.0/4` invalid).

### Documentation — `192.0.2.0/24`, `198.51.100.0/24`, `203.0.113.0/24`

These three blocks are reserved for use in books, tutorials, and example configurations. They will never be assigned to a real network. So when you see `192.0.2.42` in a guide somewhere, you can be sure the author isn't accidentally referring to a real device.

In IPv6, the documentation prefix is `2001:db8::/32`.

### CGNAT — `100.64.0.0/10`

Reserved for **Carrier-Grade NAT**, where ISPs run a giant NAT in the middle of their network to hide many customers behind few public addresses. You won't usually see this on a home network, but mobile carriers and some cable ISPs use it heavily. About 4 million addresses. Routable internally inside an ISP, not on the public internet.

### IPv6 reserved blocks

In IPv6, the equivalent reservations are:

```
::/128             — unspecified
::1/128            — loopback
fc00::/7           — unique local (private)  (fd00::/8 in practice)
fe80::/10          — link-local
ff00::/8           — multicast
2001:db8::/32      — documentation
2002::/16          — 6to4 (transition tech, mostly historical)
2000::/3           — global unicast
```

## CIDR Notation

CIDR (pronounced "cider") stands for **Classless Inter-Domain Routing**. It's the way we write IP networks today, and it replaced the old "classes A, B, C" system in 1993.

### The slash

A CIDR notation looks like this:

```
192.168.1.0/24
```

There's an address, a slash, and a number. The number is called the **prefix length** or **subnet size**. It says: "the first this-many bits identify the network, and the rest identify the host."

So `192.168.1.0/24` means: the first 24 bits (`192.168.1.`) are the network identifier, and the last 8 bits are the host part. There are 8 host bits, which means 2^8 = 256 host addresses in this network: `192.168.1.0` through `192.168.1.255`.

A diagram of the bits:

```
 192      168       1     .  host
 11000000 10101000 00000001 00000000
 |---- network (24 bits) ----|--host (8 bits)--|
```

The `/24` is just saying: the divider line falls 24 bits in. Anything before the divider is the network. Anything after is the host within that network.

### Common prefix lengths

```
/8   — 16,777,216 addresses  (a "class A" sized network)
/16  — 65,536 addresses       (a "class B" sized network)
/24  — 256 addresses          (a "class C" sized network)
/25  — 128 addresses
/26  — 64 addresses
/27  — 32 addresses
/28  — 16 addresses
/29  — 8 addresses
/30  — 4 addresses (typical for a small router-to-router link)
/31  — 2 addresses (modern point-to-point links — RFC 3021)
/32  — 1 address (a single host — used in routing as "this exact host")
```

Each step in the prefix length doubles or halves the network size. `/24` has twice as many addresses as `/25`. `/23` has twice as many as `/24`. It's all powers of two.

### Usable hosts

In IPv4, two addresses in every subnet are special:

- The **network address** (all host bits zero — first address) — identifies the network itself, not a host.
- The **broadcast address** (all host bits one — last address) — for the directed broadcast.

So in `192.168.1.0/24`:

- `192.168.1.0` — the network address (not assignable to a host).
- `192.168.1.1` through `192.168.1.254` — usable host addresses.
- `192.168.1.255` — the broadcast address (not assignable to a host).

That's 254 usable hosts out of 256 total addresses. The formula is: usable hosts = 2^(host bits) − 2.

For a `/30` network (4 total addresses, usually used for a point-to-point link), there are 2 usable hosts. For example, `10.0.0.0/30`:

- `10.0.0.0` — network.
- `10.0.0.1` — usable.
- `10.0.0.2` — usable.
- `10.0.0.3` — broadcast.

The two usable ones go on each end of the link.

A `/31` is special. Modern point-to-point links can use a `/31`, treating both addresses as usable. RFC 3021 explained how. With only 2 addresses total, there's no room for separate network and broadcast — but on a point-to-point link, neither concept applies anyway.

A `/32` is a single host. You'll see `/32` masks attached to specific routes ("this exact host's traffic goes here").

### IPv6 CIDR

IPv6 uses CIDR exactly the same way. The prefix length goes from 0 to 128.

Common IPv6 prefix lengths:

- `/32` — what an ISP gets from a regional registry. Lots of room.
- `/48` — typical for an organization or business. Lots of room for many internal subnets.
- `/56` — sometimes given to a residential customer.
- `/64` — the standard size of a single subnet. Important: most IPv6 features (like SLAAC) assume `/64`.
- `/127` — point-to-point link (RFC 6164).
- `/128` — single host.

Notice `/64` is the standard subnet size in IPv6 — that's 18 quintillion addresses per subnet. You will never run out of host addresses inside a single subnet.

### Wildcard masks (Cisco-speak)

Some Cisco gear uses **wildcard masks** instead of regular subnet masks. A wildcard mask is the bit-flipped version of a subnet mask. For `255.255.255.0` (mask for /24), the wildcard mask is `0.0.0.255`. Same network, different way of writing it. You'll see it in Cisco access lists.

## Subnets and Masks

A **subnet** is a chunk of an IP network. The **subnet mask** says where the network boundary is. CIDR notation is one way to write this. The dotted-decimal subnet mask is another.

### The mask

The subnet mask has 1-bits in the network portion and 0-bits in the host portion. For `/24`:

```
binary:  11111111 11111111 11111111 00000000
decimal: 255      .255     .255     .0
```

The 24 ones at the front say "those bits are network." The 8 zeros at the end say "those bits are host."

Common dotted-decimal masks:

- `255.0.0.0` = `/8`
- `255.255.0.0` = `/16`
- `255.255.255.0` = `/24`
- `255.255.255.128` = `/25` (split a /24 in half)
- `255.255.255.192` = `/26`
- `255.255.255.224` = `/27`
- `255.255.255.240` = `/28`
- `255.255.255.248` = `/29`
- `255.255.255.252` = `/30`
- `255.255.255.255` = `/32`

You will see both styles in the wild. CIDR (`/24`) is more compact. The dotted-decimal mask is older but still common.

### The "is this local or remote?" decision

Every time your computer wants to send a packet, the kernel asks: **is the destination in my subnet, or do I send to the gateway?**

The answer involves the subnet mask. Take your IP address. Take the destination IP. Apply the subnet mask to both (this is just zeroing out the host bits). If the masked addresses are the same, the destination is in your subnet — send the packet directly to the destination's MAC address. If they're different, the destination is somewhere else — send the packet to your **default gateway**, which is your router.

Concrete example. Your computer:

- IP: `192.168.1.42`
- Mask: `255.255.255.0` (`/24`)
- Default gateway: `192.168.1.1`

Now you want to talk to `192.168.1.99`. Apply the mask:

- Your network: `192.168.1.42` & `255.255.255.0` = `192.168.1.0`
- Their network: `192.168.1.99` & `255.255.255.0` = `192.168.1.0`

Same. Local delivery. Find their MAC address with ARP, send the packet directly.

Now you want to talk to `8.8.8.8`. Apply the mask:

- Your network: `192.168.1.42` & `255.255.255.0` = `192.168.1.0`
- Their network: `8.8.8.8` & `255.255.255.0` = `8.8.8.0`

Different. Remote delivery. Send the packet to your default gateway (`192.168.1.1`) and let it figure out the rest.

This local-vs-remote decision happens for every single outgoing packet. It's the most fundamental question in IP routing. The mask tells you the answer.

### The dual-stack ASCII diagram

A real interface in 2026 typically has both an IPv4 and an IPv6 address at the same time. We call this **dual-stack**. Here's what one looks like.

```
                      eth0 (one physical NIC)
              +-----------------------------------+
              |                                   |
   IPv4 stack |   192.168.1.42/24                 |
              |   gateway: 192.168.1.1            |
              |   ARP for layer-2                 |
              |                                   |
              +-----------------------------------+
              |                                   |
   IPv6 stack |   2001:db8:abcd::1234/64          |  <- global
              |   fe80::aaaa:bbbb:cccc:dddd/64    |  <- link-local
              |   gateway: fe80::1 (the router)   |
              |   NDP for layer-2                 |
              |                                   |
              +-----------------------------------+
                            |
                            v
              [ packets on the wire ]
```

The kernel treats the two stacks pretty independently. Each has its own routing table, its own neighbor cache, its own gateway. Programs can use either one. Modern programs prefer IPv6 (per RFC 6724) and fall back to IPv4 if IPv6 doesn't work.

## NAT (Network Address Translation)

NAT solves the IPv4 shortage by letting many private addresses share a single public address. It is the invisible magic that makes home Wi-Fi work.

### The office building analogy

Imagine a big office building. The building has one street address: **100 Main Street**. Inside, there are 200 employees, each in their own office. From outside, all mail comes to "100 Main Street." Inside, a receptionist sorts the mail and delivers it to the right office.

Mail going out is the trickier direction. Sara from office 47 sends a letter to a vendor. The vendor will reply. Where does the vendor send the reply? "100 Main Street" — that's the only address the building has. So the receptionist writes "100 Main Street, Attention: Sara, office 47" on the outgoing envelope. When the reply comes back to "100 Main Street," the receptionist sees "Attention: Sara, office 47" and routes it inside.

That's NAT. Your home Wi-Fi router has one public IP address (provided by your ISP). Inside your house, every device has a private address (`192.168.x.x`). When a device sends a packet out, the router rewrites the source address from the private one to the public one, remembers which device it was, and sends the packet on. When the reply comes back to the public address, the router uses its memory to know which inside device the reply belongs to, rewrites the destination address, and forwards it inside.

### The diagram

```
[ laptop  192.168.1.42 ] ---+
                            |
[ phone   192.168.1.43 ] ---+--- [ router ] ---- public internet
                            |    public IP:
[ TV      192.168.1.44 ] ---+    203.0.113.7
```

Inside: three devices with private addresses. Outside: one router with one public address. Every outbound packet has its source rewritten to `203.0.113.7`. Every reply comes back to `203.0.113.7` and gets routed to the right inside device.

### The kinds of NAT

There are several flavors.

- **Static NAT** — a fixed one-to-one mapping. Inside `192.168.1.50` always becomes outside `203.0.113.50`. Used for hosting servers behind NAT.
- **Dynamic NAT** — the router has a small pool of public addresses and picks one at random for each new connection. As more inside devices connect, more public addresses get used. When inside devices stop using the connection, public addresses go back to the pool. Rarely used today (because you'd need many public addresses, which we don't have).
- **PAT** (Port Address Translation) — also called **NAPT** or **overloading**. Many inside devices share one public address by also rewriting the source port number. This is the kind of NAT in your home router. It's what people usually mean when they say "NAT."
- **SNAT** (Source NAT) — rewriting the source address. PAT is a specific kind of SNAT.
- **DNAT** (Destination NAT) — rewriting the destination address. Used for **port forwarding**: "any packet to my public IP on port 80 should go to my private server `192.168.1.50` on port 80."

### How PAT actually works

PAT keeps a translation table. Each entry is a 5-tuple:

```
inside addr:port    outside addr:port    protocol
192.168.1.42:54321  203.0.113.7:54321    TCP
192.168.1.43:54321  203.0.113.7:55421    TCP
```

When the laptop on `192.168.1.42` opens a TCP connection from port 54321, the router writes the first row in the table and rewrites the outgoing packet's source to `203.0.113.7:54321`. When the reply comes back to `203.0.113.7:54321`, the router looks up the table and rewrites the destination back to `192.168.1.42:54321`.

When the phone on `192.168.1.43` also picks port 54321, there'd be a collision — both inside devices using `203.0.113.7:54321`. So the router invents a different outside port (`55421`) for the phone. Now the table is unambiguous.

The router has to keep the table for as long as the connection is alive. When the connection closes, the entry can be dropped. (For UDP, where there's no clear "close," entries usually expire after a few minutes of inactivity.)

### Why NAT is annoying

NAT works, but it has costs.

- **Hard for incoming connections.** Any device behind NAT can talk out, but nobody on the outside can talk in unless someone sets up a port forward. This is bad for peer-to-peer protocols, video chat, hosting services from home, and a thousand other things. Workarounds include port forwarding, UPnP (auto-port-forwarding), STUN, TURN, hole punching, and reverse tunnels. All of them are messy.
- **State.** The router has to keep a table of every active connection. If the table fills up, new connections fail. This becomes a problem at huge NATs (CGNATs at ISPs).
- **Some protocols break.** Anything that puts the IP address inside the packet payload (FTP active mode, SIP, some games) breaks under NAT unless an "ALG" (Application Layer Gateway) on the router knows how to fix it up. ALGs are usually buggy.
- **No accountability.** From the outside, you can't tell which inside device did something. This is sometimes a feature (privacy) and sometimes a problem (one bad actor can get the whole pool blocked).

IPv6 mostly removes the need for NAT. You can give every device a global address and let them talk directly. Some networks still use NAT in IPv6 (called NPTv6) for policy reasons, but it's discouraged.

### Hairpin NAT

What if a device inside your network wants to talk to a service running on its own network, but using the public address? Example: you're inside your house, and you visit `https://yourpublicip/`, where your own home server is hosting a website with port forwarding set up. The packet goes from your laptop (inside) to the router, which sees it as destined to its own public address with a forward rule. The router has to bounce the packet back inside to the server.

This is **hairpin NAT** (because the packet does a hairpin turn at the router). Some routers don't support it, which is why "I can reach my server from outside but not from inside" is a frustrating common problem.

## Routing — The Bigger Picture

Routing is how packets get from source to destination through a network of routers. Each router makes a local decision: "given this packet's destination, which interface do I send it out on?" String enough of those local decisions together and the packet eventually arrives.

### The routing table

Every host (and every router) has a **routing table**. Each entry is a rule that says "for destinations matching this prefix, use this gateway out this interface." Here's a simplified example:

```
destination       gateway        interface
0.0.0.0/0         192.168.1.1    eth0           <- default route
192.168.1.0/24    on-link        eth0
127.0.0.0/8       on-link        lo
```

When a packet needs to be sent, the kernel walks the table and picks the **most specific** matching entry. This is called **longest-prefix match**.

If you want to send a packet to `192.168.1.99`:
- Does it match `192.168.1.0/24`? Yes (24 bits match). Specificity = 24.
- Does it match `0.0.0.0/0`? Yes (0 bits match). Specificity = 0.
- More specific wins. Send out `eth0`, no gateway needed (on-link means "the destination is directly reachable, just ARP for it").

If you want to send a packet to `8.8.8.8`:
- Does it match `192.168.1.0/24`? No.
- Does it match `0.0.0.0/0`? Yes. Specificity = 0.
- Default route wins. Send to `192.168.1.1`, which is the home router.

### Default route

The `0.0.0.0/0` route (`/0` means zero bits of prefix — matches everything) is the **default route**. It's the catch-all. If no more-specific entry matches, the default route handles it. The default route's gateway is called the **default gateway**.

In IPv6, the default route is `::/0`.

### Static vs dynamic routes

Routing table entries can be:

- **Static** — manually configured by an administrator.
- **Dynamic** — learned automatically from a routing protocol (OSPF, BGP, IS-IS, RIP, EIGRP).

Home networks are almost entirely static (the default route to your home router). Big networks are heavily dynamic, with routers exchanging information constantly to find the best paths.

### RIB and FIB

The list of routes the routing protocol thinks are best is the **Routing Information Base** (RIB). The version actually used to forward packets in hardware is the **Forwarding Information Base** (FIB). On a normal Linux box, they're basically the same thing. On big network hardware, the RIB is in software and the FIB is programmed into specialized chips for speed.

### IGP vs EGP

Routing protocols come in two flavors.

- **IGP** (Interior Gateway Protocol) — runs inside one organization. OSPF, IS-IS, RIP, EIGRP.
- **EGP** (Exterior Gateway Protocol) — runs between organizations on the internet. BGP. Just one BGP, that's how the internet glues together.

The internet's whole structure is: each organization runs its own IGP inside, and BGP connects organizations to each other. Every internet packet that crosses organizational boundaries crosses through BGP routes.

See `cs ramp-up bgp-eli5` for the BGP story.

### ECMP — multiple equal-cost paths

If two routes to the same destination have equal cost (same prefix length, same metric), a router can use both at the same time. We call this **ECMP** (Equal-Cost Multi-Path). The router hashes each packet flow (based on source IP, dest IP, ports) and picks one of the paths for that flow. Different flows go down different paths. Total bandwidth is the sum of all paths.

Most modern networks use ECMP heavily. It's how data centers spread traffic across multiple links.

## The IPv4 Header

Every IPv4 packet has a header. The header is at the front. The header is **at least 20 bytes** (usually exactly 20 — options are rare). Here's the layout, drawn as a 32-bit-wide grid (the standard way to draw protocol headers).

```
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|Version|  IHL  |     DSCP    |ECN|        Total Length         |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|        Identification        |Flags|     Fragment Offset      |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|     TTL       |   Protocol   |       Header Checksum          |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                       Source Address                          |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                    Destination Address                        |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                  Options (if IHL > 5)                         |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                          Payload                              |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

Let's walk every field at ELI5 level.

### Version (4 bits)

Always 4 for IPv4. (You'd expect it to come first, and it does.) Always 6 for IPv6. The very first 4 bits tell the receiver which version they're looking at.

### IHL (4 bits) — Internet Header Length

Length of the header itself, in 32-bit words. Minimum value is 5 (which means 5 × 4 bytes = 20 bytes). Max is 15 (60 bytes). Bigger values mean there are options. Most packets have IHL=5 — no options.

### DSCP (6 bits) and ECN (2 bits) — together, the "Type of Service" byte

The original byte was called **TOS** (Type of Service). Modern usage splits it into:

- **DSCP** (Differentiated Services Code Point) — 6 bits. A label saying "this is voice traffic" or "this is bulk traffic" so routers can prioritize.
- **ECN** (Explicit Congestion Notification) — 2 bits. Lets routers signal congestion without dropping packets.

Most home traffic just has both fields zero. Quality-of-service-aware networks use them.

### Total Length (16 bits)

Total length of the whole packet (header + payload), in bytes. Minimum 20 (just a header, no payload). Maximum 65535. In practice, packets rarely get that big — they're usually capped by the link MTU (typically 1500 bytes).

### Identification (16 bits)

A unique number assigned by the sender so that fragments of the same original packet can be reassembled. Every packet sent gets a fresh ID. If a packet is fragmented along the way, all fragments share the same ID.

### Flags (3 bits)

- **Reserved** — must be zero.
- **DF** (Don't Fragment) — if set, routers must not fragment. If the packet is too big for a link, the router drops it and sends back ICMP "Fragmentation Needed." This is how Path MTU Discovery works.
- **MF** (More Fragments) — set on every fragment except the last. When you see MF=0, you've got the final fragment.

### Fragment Offset (13 bits)

For fragmented packets, says where this fragment fits in the reassembled whole. Measured in 8-byte units. The first fragment has offset 0. The next fragment has offset (size-of-first / 8), and so on.

### TTL (8 bits) — Time To Live

We talked about this. Counts hops. Starts at 64 or 128. Each router subtracts 1. Hits zero, packet dies.

### Protocol (8 bits)

The number of the next-layer protocol — what's inside the payload.

- 1 = ICMP (ping, traceroute, etc.)
- 2 = IGMP (multicast group management)
- 6 = TCP
- 17 = UDP
- 47 = GRE (a tunnel protocol)
- 50 = ESP (IPsec encapsulation)
- 51 = AH (IPsec authentication header)
- 89 = OSPF
- 132 = SCTP

These numbers come from IANA's protocol number registry. There are over 140 of them, but TCP, UDP, and ICMP are by far the most common.

### Header Checksum (16 bits)

A small math result over just the header. If the header gets corrupted in transit (a bit flips somewhere), the checksum no longer matches and the packet gets dropped. Note: only the **header** is checksummed at the IP level. The payload's checksum is the responsibility of TCP or UDP.

Every router that decrements TTL has to recompute the checksum, because TTL is part of the header. This is a small but constant cost.

### Source Address (32 bits)

Where the packet came from. The IPv4 address of the sender.

### Destination Address (32 bits)

Where the packet is going. The IPv4 address of the destination.

### Options (variable)

Rarely used. Things like "Record Route" (every router writes its IP in the packet) and "Timestamp" (every router writes its timestamp). Almost no real traffic uses options. When IHL=5, there are no options.

### Payload

Everything after the header. Whatever protocol is named in the Protocol field.

## The IPv6 Header

The IPv6 header is **fixed at 40 bytes**. No variable length. Always exactly 40. This makes it faster to process than the IPv4 header.

```
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|Version| Traffic Class |             Flow Label                |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|       Payload Length          | Next Header   |   Hop Limit    |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                                                                |
+                       Source Address                           +
|                          (128 bits)                            |
+                                                                +
|                                                                |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                                                                |
+                    Destination Address                         +
|                          (128 bits)                            |
+                                                                +
|                                                                |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                          Payload                               |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

### Field walk

- **Version** (4 bits) — always 6.
- **Traffic Class** (8 bits) — like IPv4 DSCP+ECN. Same idea.
- **Flow Label** (20 bits) — a hint to routers that all packets with the same flow label should be treated as one flow (handy for ECMP hashing). New in IPv6.
- **Payload Length** (16 bits) — length of the payload only (not including this 40-byte header). Note: in IPv4 the equivalent counts the header, in IPv6 it doesn't. Easy to mix up.
- **Next Header** (8 bits) — same as IPv4's Protocol field, with the same numbers (6 = TCP, 17 = UDP, etc.). Or it can point to an **extension header**.
- **Hop Limit** (8 bits) — same as IPv4 TTL. Honest naming.
- **Source Address** (128 bits).
- **Destination Address** (128 bits).

That's it. No checksum (TCP/UDP and the link layer already checksum). No fragmentation fields (extension header handles those). No identification field (extension header). Eight fields total, fixed positions.

### Extension headers

When IPv6 needs the equivalent of IPv4 options, it uses **extension headers**. The Next Header field tells the receiver what comes next. If it's an extension header, that header has its own Next Header field saying what comes after, and so on. A chain of headers ending eventually with a payload.

Common extension headers:

- **Hop-by-Hop Options** (0) — must be examined by every router.
- **Routing** (43) — source routing (rare).
- **Fragment** (44) — fragmentation info.
- **Destination Options** (60) — only for the destination.
- **AH** (51) and **ESP** (50) — IPsec.

In normal traffic, you almost never see extension headers. The Next Header is just 6 (TCP) or 17 (UDP), and the payload follows immediately.

## Fragmentation

**MTU** (Maximum Transmission Unit) is the largest single packet a link can carry without breaking it up. For most Ethernet-based links, the MTU is **1500 bytes**. Some links are bigger (jumbo frames, 9000 bytes). Some are smaller (PPP over slow modems used to have 576).

If a packet is bigger than the MTU of a link, something has to break it into smaller pieces.

### IPv4 fragmentation

In IPv4, any router along the path can fragment if needed. The router takes the big packet, splits it into pieces that fit in the next MTU, copies the IP header onto each piece, fills in the Identification, MF flag, and Fragment Offset fields appropriately, and sends each piece on its way. The destination collects all the pieces, reassembles them, and hands the result up to TCP/UDP.

This sounds nice, but it's actually painful. The router has to do extra work. The destination has to collect and time out unmatched fragments. If any fragment is lost, the entire original packet is useless and has to be retransmitted. Lots of clever attacks exploit fragmentation reassembly bugs.

Most modern protocols set the **DF** (Don't Fragment) bit, which forbids router-side fragmentation. If a packet with DF set is too big for the next link, the router drops it and sends an **ICMP "Fragmentation Needed"** message back to the source. The source then knows: "the path can't carry packets bigger than X bytes." This is **Path MTU Discovery** (PMTUD). The source remembers the smaller MTU and sends smaller packets from now on.

When PMTUD works, you get the benefits of big packets where possible and small packets where necessary, with no router-side fragmentation. When PMTUD breaks (because some firewall blocks ICMP — bad firewall!), you get the dreaded "PMTUD black hole" where connections hang silently on big packets.

### IPv6 fragmentation

In IPv6, **routers cannot fragment**. Period. Only the original sender can fragment, and only by including a Fragment extension header.

If an IPv6 router receives a packet too big for the next link, it does **not** fragment. It sends an **ICMPv6 Packet Too Big** (PTB) message back to the source, including the MTU it should have used. The source remembers and either fragments using the extension header or sends smaller packets going forward.

This is much cleaner than IPv4's hopeful fragmentation. Routers do less work. Only the source has to manage fragments. The downside is that PMTUD becomes mandatory in IPv6 — networks that drop ICMPv6 PTB break IPv6.

The minimum IPv6 MTU is **1280 bytes** (RFC 8200). Every IPv6 link must support at least 1280 bytes without fragmentation. So a sender can always safely send packets up to 1280 bytes without worrying about PMTU.

### TCP and MSS

TCP avoids fragmentation by negotiating an **MSS** (Maximum Segment Size) at connection setup. MSS is essentially "MTU minus IP header minus TCP header." Each side announces its MSS. They agree on the lower of the two. From then on, neither side sends a TCP segment bigger than MSS, so fragmentation is unlikely.

UDP doesn't have an equivalent. UDP applications have to handle MTU themselves.

## Address Resolution

Once IP has decided "this packet is going out interface eth0 to local-network address `192.168.1.99`," there's still one more step. The link layer (Ethernet) needs a **MAC address** — a unique 48-bit hardware identifier baked into every network card. The packet has to be wrapped in an Ethernet frame addressed to the MAC of `192.168.1.99`.

How do we get from "I know your IP address" to "I know your MAC address"? Address resolution.

### ARP for IPv4

**ARP** (Address Resolution Protocol) is the way IPv4 finds MAC addresses for IP addresses. It's a "shouting on the local network" protocol.

The exchange:

```
1. Host A wants to send to 192.168.1.99 (Host B). A doesn't know B's MAC.
2. A broadcasts: "Who has 192.168.1.99? Tell 192.168.1.42."
   [ this is an ARP REQUEST sent to MAC ff:ff:ff:ff:ff:ff (broadcast) ]
3. Every device on the network sees the broadcast. Most ignore it.
4. Host B sees its own IP and replies: "192.168.1.99 is at MAC bb:bb:bb:bb:bb:bb"
   [ this is an ARP REPLY sent unicast back to A's MAC ]
5. Host A caches B's MAC for a while (typically 60 seconds to a few minutes)
   so it doesn't have to ARP again on every packet.
6. Now A can send actual IP packets, wrapping each one in an Ethernet frame
   addressed to bb:bb:bb:bb:bb:bb.
```

A diagram:

```
A (192.168.1.42 / aa:aa:aa:aa:aa:aa)
        |
        |   "Who has 192.168.1.99? Tell 192.168.1.42"
        |   src MAC: aa:aa:aa:aa:aa:aa, dst MAC: ff:ff:ff:ff:ff:ff
        v
[ broadcast to entire local Ethernet ]
        |
        |   ... many hosts hear it ...
        |
        v
B (192.168.1.99 / bb:bb:bb:bb:bb:bb)  "That's me!"
        |
        |   "192.168.1.99 is at bb:bb:bb:bb:bb:bb"
        |   src MAC: bb:bb:bb:bb:bb:bb, dst MAC: aa:aa:aa:aa:aa:aa
        v
A (caches the answer)
```

The cache is called the **ARP cache** or **ARP table**. You can see it with `ip neigh show` or the older `arp -an`.

**Gratuitous ARP** is when a host sends an ARP reply unsolicited. "Hi everyone, I'm 192.168.1.42 at MAC aa:aa:aa:aa:aa:aa." Used to update everyone's cache when a host moves or comes online. Also used in some failover protocols.

### NDP for IPv6

IPv6 doesn't use ARP. Instead it has **NDP** (Neighbor Discovery Protocol), which uses ICMPv6. Same general idea, but with multicast instead of broadcast.

The exchange:

```
1. Host A wants to send to 2001:db8::99. A doesn't know B's MAC.
2. A sends a Neighbor Solicitation (NS) to the solicited-node multicast address
   for B (computed from B's IPv6 address).
3. Only nodes interested in that multicast group see it. (Far less noisy than broadcast.)
4. B replies with a Neighbor Advertisement (NA), unicast to A.
5. A caches the answer.
```

The diagram:

```
A
  |  Neighbor Solicitation (NS) to ff02::1:ff00:99 (solicited-node multicast)
  |  "Who has 2001:db8::99?"
  v
[ multicast — only nodes that subscribed receive ]
  |
  v
B receives because it joined ff02::1:ff00:99
  |
  |  Neighbor Advertisement (NA), unicast back
  |  "2001:db8::99 is at MAC bb:bb:bb:bb:bb:bb"
  v
A caches.
```

NDP uses ICMPv6 messages. It also handles things ARP couldn't:

- **Router discovery** — Router Advertisement (RA) messages tell hosts about routers and prefixes.
- **DAD** (Duplicate Address Detection) — before using a new address, a host checks that nobody else has it.
- **Redirect** — a router can tell a host about a better next hop.

The cache is called the **neighbor cache**. View with `ip -6 neigh show`.

## DHCP and SLAAC

How does a brand-new device get an IP address when it joins a network? Two main answers: DHCP (for IPv4 and optionally IPv6) and SLAAC (IPv6 only).

### DHCPv4

**DHCP** (Dynamic Host Configuration Protocol) is the IPv4 way. The exchange is **DORA**:

```
1. DISCOVER  — client broadcasts "I need an IP."
              src 0.0.0.0, dst 255.255.255.255 (everyone), UDP port 67/68.
2. OFFER     — DHCP server offers an IP and configuration.
3. REQUEST   — client says "yes, I'll take that one."
4. ACK       — server confirms.
```

After ACK, the client has:
- An IP address.
- A subnet mask.
- A default gateway.
- One or more DNS servers.
- A **lease time** (how long the IP is valid before the client must renew).

The client renews the lease before it expires, usually at the halfway point. If renewal fails, the client tries again. If it can't renew, eventually the address expires and the client has to do another full DORA.

DHCP servers can give out the same IP every time to the same MAC (a "reservation") or pick from a pool randomly. Home routers usually just pick from a pool.

### SLAAC (IPv6)

**SLAAC** (Stateless Address Auto-Configuration) is the IPv6 way to auto-configure without a DHCP server. It works because IPv6 was designed for it.

The exchange:

```
1. Host comes up. Generates a link-local address (fe80::xxx) and runs DAD on it.
2. Host sends a Router Solicitation (RS) on the local link.
3. A router replies with a Router Advertisement (RA) containing one or more prefixes.
4. The host combines a prefix with an interface ID to form a global address.
   (Interface ID = either EUI-64 from the MAC, or a random number for privacy.)
5. Host runs DAD on the global address to make sure nobody else has it.
6. Host now has a working global address, no DHCP server needed.
```

Routers periodically send RAs unsolicited too, so even hosts that didn't ask will see them.

The interface ID can be derived from the MAC address (called **EUI-64**), or it can be random for privacy (per RFC 4941, "privacy extensions"). Most modern OSes use random interface IDs by default to avoid tracking concerns.

### DHCPv6

When SLAAC isn't enough — for example, you want to assign specific addresses centrally, or distribute information SLAAC can't (like NTP servers in some configs) — there's **DHCPv6**. It's similar in spirit to DHCPv4 but uses different message names (SOLICIT, ADVERTISE, REQUEST, REPLY) and runs over UDP ports 546/547 with multicast addresses instead of broadcast.

DHCPv6 has two modes:

- **Stateful** — the server hands out addresses, like DHCPv4.
- **Stateless** — the server provides extra info (DNS, etc.) but addresses come from SLAAC.

The Router Advertisement has flags telling the host which mode to use:

- **M flag** (Managed) — use stateful DHCPv6 for addresses.
- **O flag** (Other) — use stateless DHCPv6 for other config (DNS).

If both M and O are clear, pure SLAAC.

### Prefix delegation

Big networks need to give entire subnets, not just one address, to downstream networks. This is **DHCPv6 prefix delegation** (DHCPv6-PD). An ISP gives a residential customer a `/56`, say. The customer's router takes that `/56` and creates many `/64` subnets for the various Wi-Fi networks inside the house.

## Common IP Issues

Things break in IP networking all the time. Here are common error messages and what they mean.

### "No route to host"

You tried to send a packet but the kernel couldn't find any route to use. Either:

- The destination doesn't match any entry in the routing table (no default route either).
- The matching route's gateway is unreachable (default gateway is down).
- The matching route's interface is down.

Diagnosis: check `ip route show`, check `ping` to your default gateway, check whether your interface is up with `ip link show`.

### "Network unreachable"

Slightly different from "no route to host." This is what you get when there's no IP configured on the outgoing interface — the kernel can't even pick a source address.

Diagnosis: check `ip addr show`. Make sure the interface has an IP.

### "Destination host unreachable"

This is the ARP-failed (or NDP-failed) error. The kernel knows the route — the destination is on the local network — but ARP timed out and no MAC came back. The destination either isn't there or isn't responding.

Diagnosis: ping the destination. Check `ip neigh show` for failed entries. Check if the destination is actually powered on.

### "Address already in use"

You tried to bind a server to an IP+port that's already in use. Either another process has it, or the previous instance is in TIME_WAIT.

Fix: use `ss -tulpn | grep :80` to find what's using the port. Kill it, or pick a different port. To allow rebind to a TIME_WAIT port, set `SO_REUSEADDR`.

### "Cannot assign requested address"

You tried to bind to an IP that's not configured on any of your interfaces. The kernel doesn't know how to assign that IP because it doesn't have it.

Fix: check `ip addr show`. Add the address to an interface, or use a different bind IP. `INADDR_ANY` (0.0.0.0) is always safe — it means "any IP I have."

### "Connection refused"

You connected, but the destination actively refused. The destination machine sent back a TCP RST. Usually means: no service is listening on that port. (Different from "no route to host" because the destination machine is reachable — it's just not running anything on that port.)

### "Connection timed out"

You tried to connect, but no answer came back at all within the timeout. Could be:

- The destination machine is down.
- A firewall is silently dropping packets.
- A route somewhere along the way is broken.

Diagnose with `traceroute` to see where the path stops responding.

### IP conflict (DAD failure)

Someone else on the network has the same IP as you. The kernel notices via gratuitous ARP or DAD and refuses to fully use the address. Symptom: connection drops, connections to your IP go to the wrong machine, weird intermittent failures.

Fix: figure out who else has the IP. Move one machine to a different IP.

### "TTL expired in transit"

Your packet's TTL hit zero before reaching the destination. Either the destination is more than TTL hops away (rare — TTL=64 can reach almost anywhere on Earth), or there's a routing loop.

Diagnose with `traceroute`. If the trace shows the same routers repeating, you have a loop.

### MTU/PMTU black hole

Connections work for small packets (ping, DNS) but hang on big packets (file downloads). Likely cause: a router on the path drops big packets but ICMP "fragmentation needed" is being filtered. PMTUD breaks. The sender keeps trying with too-big packets and they all silently disappear.

Fix: lower the MSS at the firewall (`iptables ... -j TCPMSS --clamp-mss-to-pmtu`) or stop blocking ICMP.

## Hands-On

These commands are your friends. Most are read-only and safe. The ones that change state are clearly marked with `(root)`.

### See your own addresses

```
$ ip addr show
1: lo: <LOOPBACK,UP,LOWER_UP> mtu 65536 qdisc noqueue state UNKNOWN group default qlen 1000
    link/loopback 00:00:00:00:00:00 brd 00:00:00:00:00:00
    inet 127.0.0.1/8 scope host lo
       valid_lft forever preferred_lft forever
    inet6 ::1/128 scope host
       valid_lft forever preferred_lft forever
2: eth0: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 qdisc fq_codel state UP group default qlen 1000
    link/ether 00:11:22:33:44:55 brd ff:ff:ff:ff:ff:ff
    inet 192.168.1.42/24 brd 192.168.1.255 scope global dynamic eth0
       valid_lft 84372sec preferred_lft 84372sec
    inet6 2001:db8:abcd::1234/64 scope global dynamic mngtmpaddr
       valid_lft 86400sec preferred_lft 14400sec
    inet6 fe80::211:22ff:fe33:4455/64 scope link
       valid_lft forever preferred_lft forever
```

The `inet` lines are IPv4. The `inet6` lines are IPv6. Each interface can have many addresses. `lo` is loopback, `eth0` is a wired Ethernet, `wlan0` (or `wlpXsY` on newer naming) is Wi-Fi.

Short form: `ip a`. Same output, less typing.

### Just IPv4 or just IPv6

```
$ ip -4 addr show
1: lo: <LOOPBACK,UP,LOWER_UP> mtu 65536
    inet 127.0.0.1/8 scope host lo
2: eth0: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500
    inet 192.168.1.42/24 brd 192.168.1.255 scope global dynamic eth0

$ ip -6 addr show
1: lo: <LOOPBACK,UP,LOWER_UP> mtu 65536
    inet6 ::1/128 scope host
2: eth0: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500
    inet6 2001:db8:abcd::1234/64 scope global dynamic mngtmpaddr
    inet6 fe80::211:22ff:fe33:4455/64 scope link
```

### See your interfaces (without addresses)

```
$ ip link show
1: lo: <LOOPBACK,UP,LOWER_UP> mtu 65536 qdisc noqueue state UNKNOWN mode DEFAULT
    link/loopback 00:00:00:00:00:00 brd 00:00:00:00:00:00
2: eth0: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 qdisc fq_codel state UP mode DEFAULT
    link/ether 00:11:22:33:44:55 brd ff:ff:ff:ff:ff:ff
```

`UP` means the interface is enabled. `LOWER_UP` means a cable is plugged in (or Wi-Fi is associated). The MAC address is on the `link/ether` line.

### See your routing tables

```
$ ip route show
default via 192.168.1.1 dev eth0 proto dhcp metric 100
192.168.1.0/24 dev eth0 proto kernel scope link src 192.168.1.42 metric 100
```

Two entries. The first is the default route — anything that doesn't match more specific entries goes to `192.168.1.1` via `eth0`. The second is the LAN — anything in `192.168.1.0/24` is on `eth0` directly (no gateway).

```
$ ip -6 route show
2001:db8:abcd::/64 dev eth0 proto ra metric 100 pref medium
fe80::/64 dev eth0 proto kernel metric 256 pref medium
default via fe80::1 dev eth0 proto ra metric 100 pref medium
```

IPv6 default goes through the router's link-local address (`fe80::1`).

### See your neighbor caches

```
$ ip neigh show
192.168.1.1 dev eth0 lladdr aa:bb:cc:dd:ee:ff REACHABLE
192.168.1.99 dev eth0 lladdr 11:22:33:44:55:66 STALE
192.168.1.150 dev eth0  FAILED
```

Each entry is one cached neighbor. `REACHABLE` means we recently confirmed they're alive. `STALE` means we haven't checked recently but the entry is probably still good. `FAILED` means we tried to ARP and got nothing.

```
$ ip -6 neigh show
fe80::1 dev eth0 lladdr aa:bb:cc:dd:ee:ff REACHABLE
2001:db8:abcd::99 dev eth0 lladdr 11:22:33:44:55:66 STALE
```

Same idea, IPv6 version.

### Add an IP to an interface (root)

```
$ sudo ip addr add 192.168.99.1/24 dev eth0
```

Adds a secondary IP. Confirm with `ip addr show eth0`. To remove:

```
$ sudo ip addr del 192.168.99.1/24 dev eth0
```

### Add a static route (root)

```
$ sudo ip route add 10.10.0.0/16 via 192.168.1.1
```

Says "anything for `10.10.0.0/16` goes via `192.168.1.1`." Confirm with `ip route show`. To remove:

```
$ sudo ip route del 10.10.0.0/16
```

### See policy routing rules

```
$ ip rule show
0:      from all lookup local
32766:  from all lookup main
32767:  from all lookup default
```

Each rule says "for traffic matching X, look in routing table Y." Almost everything goes to `main`. Advanced setups use multiple tables for things like split tunneling.

### Read the kernel's raw routing table

```
$ cat /proc/net/route
Iface   Destination     Gateway         Flags   RefCnt  Use     Metric  Mask            MTU     Window  IRTT
eth0    00000000        0101A8C0        0003    0       0       100     00000000        0       0       0
eth0    0001A8C0        00000000        0001    0       0       100     00FFFFFF        0       0       0
```

That's the same table `ip route show` shows, but in the raw kernel format (little-endian hex). `0101A8C0` is `0xC0A80101` = `192.168.1.1`.

### Per-interface stats

```
$ cat /proc/net/dev
Inter-|   Receive                                                |  Transmit
 face |bytes    packets errs drop fifo frame compressed multicast|bytes    packets errs drop fifo colls carrier compressed
    lo: 12345678   45678   0    0    0     0          0         0  12345678   45678    0    0    0     0       0          0
  eth0: 987654321 1234567   0    0    0     0          0    234567  87654321  876543    0    0    0     0       0          0
```

Every interface, RX and TX byte and packet counters. Counters never reset (well, until the interface is re-created). Subtract two readings to get a rate.

### Is IP forwarding on?

```
$ cat /proc/sys/net/ipv4/ip_forward
0
```

`0` means this machine is not a router (won't forward packets between interfaces). `1` means it is. To turn it on temporarily:

```
$ echo 1 | sudo tee /proc/sys/net/ipv4/ip_forward
```

Or via sysctl:

```
$ sudo sysctl -w net.ipv4.ip_forward=1
```

For IPv6:

```
$ cat /proc/sys/net/ipv6/conf/all/forwarding
0
```

### See the local port range

```
$ cat /proc/sys/net/ipv4/ip_local_port_range
32768   60999
```

The kernel picks ephemeral ports from this range when you make outgoing connections. So your laptop can have about 28,000 outgoing TCP connections going at once (per source-IP / destination-IP pair).

### Browse all IP-related sysctls

```
$ sysctl -a 2>/dev/null | grep -E "net\.ipv[46]\." | head -20
net.ipv4.conf.all.accept_local = 0
net.ipv4.conf.all.accept_redirects = 0
net.ipv4.conf.all.accept_source_route = 0
net.ipv4.conf.all.arp_accept = 0
net.ipv4.conf.all.arp_announce = 0
net.ipv4.conf.all.arp_filter = 0
net.ipv4.conf.all.arp_ignore = 0
net.ipv4.conf.all.arp_notify = 0
net.ipv4.conf.all.bootp_relay = 0
net.ipv4.conf.all.disable_policy = 0
net.ipv4.conf.all.disable_xfrm = 0
net.ipv4.conf.all.drop_gratuitous_arp = 0
net.ipv4.conf.all.drop_unicast_in_l2_multicast = 0
net.ipv4.conf.all.force_igmp_version = 0
net.ipv4.conf.all.forwarding = 0
net.ipv4.conf.all.igmp_link_local_mcast_reports = 1
net.ipv4.conf.all.ignore_routes_with_linkdown = 0
net.ipv4.conf.all.log_martians = 0
net.ipv4.conf.all.mc_lite = 0
net.ipv4.conf.all.medium_id = 0
```

Hundreds of knobs. Most you should leave alone. Read the kernel networking docs (`/usr/share/doc/linux-doc/networking/`) before changing anything.

### Look at the legacy ARP table

```
$ arp -an
? (192.168.1.1) at aa:bb:cc:dd:ee:ff [ether] on eth0
? (192.168.1.99) at 11:22:33:44:55:66 [ether] on eth0
```

This is the old `arp` command. Same data as `ip neigh show`. The `arp` command might not be installed by default on minimal systems.

### Layer-2 ping

```
$ sudo arping -c 3 192.168.1.1
ARPING 192.168.1.1 from 192.168.1.42 eth0
Unicast reply from 192.168.1.1 [aa:bb:cc:dd:ee:ff]  0.523ms
Unicast reply from 192.168.1.1 [aa:bb:cc:dd:ee:ff]  0.471ms
Unicast reply from 192.168.1.1 [aa:bb:cc:dd:ee:ff]  0.498ms
Sent 3 probes (1 broadcast(s))
Received 3 response(s)
```

Sends ARP requests instead of ICMP echos. Useful when ICMP is blocked. Works only on the local link.

### Regular ping

```
$ ping -c 4 8.8.8.8
PING 8.8.8.8 (8.8.8.8) 56(84) bytes of data.
64 bytes from 8.8.8.8: icmp_seq=1 ttl=115 time=14.2 ms
64 bytes from 8.8.8.8: icmp_seq=2 ttl=115 time=13.9 ms
64 bytes from 8.8.8.8: icmp_seq=3 ttl=115 time=14.5 ms
64 bytes from 8.8.8.8: icmp_seq=4 ttl=115 time=14.0 ms

--- 8.8.8.8 ping statistics ---
4 packets transmitted, 4 received, 0% packet loss, time 3005ms
rtt min/avg/max/mdev = 13.892/14.150/14.512/0.232 ms
```

Sends 4 ICMP echo requests, prints replies. The TTL on each reply tells you (sort of) how many hops away the destination is. (Modern OSes start TTL at 64 or 128, so 115 = 13 hops, since 128 - 115 = 13.)

### Trace the path

```
$ traceroute 8.8.8.8
traceroute to 8.8.8.8 (8.8.8.8), 30 hops max, 60 byte packets
 1  192.168.1.1 (192.168.1.1)  0.521 ms  0.499 ms  0.476 ms
 2  10.0.0.1 (10.0.0.1)  4.123 ms  4.211 ms  4.087 ms
 3  203.0.113.1 (203.0.113.1)  8.112 ms  8.034 ms  8.198 ms
 4  * * *
 5  100.64.5.7 (100.64.5.7)  12.001 ms  11.987 ms  12.045 ms
 ...
13  8.8.8.8 (8.8.8.8)  14.012 ms  14.001 ms  13.999 ms
```

Each line is one router along the path. Numbers are round-trip times for three probes. Asterisks mean the probe didn't get an answer (router was filtering ICMP, or otherwise quiet). The final line is the destination.

### Modern combined trace + ping

```
$ mtr -rwbz 8.8.8.8
Start: 2026-04-27T14:32:11
HOST: mybox                                       Loss%   Snt   Last   Avg  Best  Wrst StDev
  1.|-- _gateway 192.168.1.1                       0.0%    10    0.5   0.5   0.4   0.7   0.1
  2.|-- 10.0.0.1                                   0.0%    10    4.1   4.2   4.0   4.5   0.2
  3.|-- 203.0.113.1                                0.0%    10    8.1   8.2   8.0   8.4   0.1
  ...
 13.|-- dns.google 8.8.8.8                         0.0%    10   14.0  14.1  14.0  14.5   0.2
```

`mtr` is `traceroute + ping` rolled into one. `-r` is report mode. `-w` is wide. `-b` shows IPs and hostnames. `-z` shows AS numbers (which network owns each hop). Best learning tool for understanding paths.

### Watch IPv4 packets fly by

```
$ sudo tcpdump -i any -n -c 10 ip
tcpdump: data link type LINUX_SLL2
tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on any, link-type LINUX_SLL2 (Linux cooked v2), snapshot length 262144 bytes
14:32:11.123456 eth0  Out IP 192.168.1.42.51234 > 8.8.8.8.443: Flags [S], seq 1234567890, win 65535, length 0
14:32:11.137821 eth0  In  IP 8.8.8.8.443 > 192.168.1.42.51234: Flags [S.], seq 9876543210, ack 1234567891, win 65535, length 0
14:32:11.137921 eth0  Out IP 192.168.1.42.51234 > 8.8.8.8.443: Flags [.], ack 1, win 1444, length 0
...
```

`-i any` listens on all interfaces. `-n` means "don't resolve names" (faster). `-c 10` means "capture 10 packets and exit." `ip` is a filter — only IPv4 packets.

### Watch IPv6 packets

```
$ sudo tcpdump -i any -n -c 10 ip6
14:32:15.001234 eth0  Out IP6 2001:db8:abcd::1234.51200 > 2001:4860:4860::8888.443: Flags [S], seq 111, win 65535, length 0
14:32:15.014567 eth0  In  IP6 2001:4860:4860::8888.443 > 2001:db8:abcd::1234.51200: Flags [S.], seq 222, ack 112, win 65535, length 0
...
```

`ip6` filter shows only IPv6.

### See firewall rules (root)

```
$ sudo nft list ruleset | head -30
table inet filter {
        chain input {
                type filter hook input priority filter; policy accept;
                ct state established,related accept
                iif "lo" accept
                ct state invalid drop
                ip protocol icmp accept
                tcp dport 22 accept
        }
        chain forward {
                type filter hook forward priority filter; policy drop;
        }
        chain output {
                type filter hook output priority filter; policy accept;
        }
}
```

The Linux firewall is in the kernel and called nftables. `nft list ruleset` shows everything. Older systems use `iptables` instead — see `cs networking iptables`.

### Resolve a name

```
$ getent hosts google.com
142.250.190.78  google.com
```

`getent` reads `/etc/nsswitch.conf` to figure out where to look (DNS, /etc/hosts, etc.) and resolves the name. This is the same path real programs use. Useful when you want to test with the same tools that production traffic uses.

### Or with `host`

```
$ host google.com
google.com has address 142.250.190.78
google.com has IPv6 address 2607:f8b0:4004:c08::64
google.com mail is handled by 10 smtp.google.com.
```

`host` is a simple DNS lookup tool. Returns A (IPv4), AAAA (IPv6), and MX (mail) records.

### IPv6 lookup specifically

```
$ dig +short google.com aaaa
2607:f8b0:4004:c08::64
2607:f8b0:4004:c1b::71
2607:f8b0:4004:c00::8a
```

`dig` (Domain Information Groper) is the deep DNS tool. `+short` cuts the output. `aaaa` asks specifically for IPv6 addresses.

### Lookup the reverse

```
$ dig +short -x 8.8.8.8
dns.google.
```

`-x` means "reverse lookup" — turn an IP into a name.

### Open a TCP connection by hand

```
$ nc -zv google.com 443
Connection to google.com (142.250.190.78) 443 port [tcp/https] succeeded!
```

`nc` (netcat) opens a TCP connection. `-z` is "just check, don't send anything." `-v` is verbose. Great for "is this port open?"

### See active TCP/UDP sockets

```
$ ss -tunap | head -10
Netid State  Recv-Q Send-Q  Local Address:Port    Peer Address:Port  Process
tcp   LISTEN 0      128            0.0.0.0:22           0.0.0.0:*       users:(("sshd",pid=1234,fd=3))
tcp   LISTEN 0      511            127.0.0.1:8080       0.0.0.0:*       users:(("nginx",pid=5678,fd=6))
tcp   ESTAB  0      0          192.168.1.42:54321  142.250.190.78:443  users:(("firefox",pid=9876,fd=44))
udp   UNCONN 0      0            0.0.0.0:68          0.0.0.0:*       users:(("dhclient",pid=2345,fd=8))
```

`ss` is the modern replacement for `netstat`. `-t` TCP, `-u` UDP, `-n` no name resolution, `-a` all states, `-p` show processes. Very powerful.

### Show socket statistics by state

```
$ ss -s
Total: 234
TCP:   45 (estab 12, closed 22, orphaned 0, timewait 11)

Transport Total     IP        IPv6
RAW       1         0         1
UDP       18        9         9
TCP       23        14        9
INET      42        23        19
FRAG      0         0         0
```

Quick summary of all sockets.

## Common Confusions

### "Is my IP my computer's ID?"

**The confusion:** Is the IP address what uniquely identifies my computer?

**The fix:** Sort of. An IP address identifies a computer **on a network at a particular time**. It can change. When you take your laptop to a coffee shop, you get a different IP. When you reboot your home router, you might get a different public IP from the ISP. The thing that doesn't change (much) is the **MAC address** of your network card — that's burned into the hardware. Even MACs can be spoofed, though, so really nothing is a perfect "ID."

### "What's the difference between a public and a private IP?"

**The confusion:** I see a `192.168.x.x` and a `203.0.113.x`. Why are they different?

**The fix:** Public IPs are globally unique and routable on the open internet. Anyone in the world can (in principle) reach a public IP. Private IPs are from RFC 1918 / fc00::/7. They live inside your network. They are not routable on the public internet — every backbone router actively drops traffic to them. Many networks can use the same private addresses without conflict.

### "Why is IPv6 still not everywhere?"

**The confusion:** It's been thirty years since IPv4 ran out, why isn't IPv6 universal?

**The fix:** Inertia, mostly. NAT makes IPv4 work "well enough" that there's no urgent need to upgrade. Training engineers in IPv6 takes time. Some old gear doesn't support it. But adoption is steadily climbing — about 45-50% of traffic to large content providers is IPv6 today. Mobile networks have been particularly aggressive. Major content (Google, Facebook, Cloudflare, Netflix) is all dual-stack.

### "Why is `10.0.0.0/8` not pingable from anywhere on the internet?"

**The confusion:** I can ping `8.8.8.8` from anywhere but not `10.5.5.5`.

**The fix:** `10.0.0.0/8` is in RFC 1918, the private address space. Every backbone router drops traffic to it. The same `10.x.x.x` is in use in millions of private networks worldwide. There's no way for a packet from outside to know which one to go to. Private addresses are confined to their own networks by design.

### "Why does my home Wi-Fi say `192.168.1.1`?"

**The confusion:** That's the router's address but my computer is `192.168.1.42`?

**The fix:** Your router has at least two IP addresses. On the LAN side (the side facing your devices), it has `192.168.1.1`. That's the one your computer talks to. On the WAN side (the side facing the ISP), it has a completely different public IP (something like `203.0.113.7`). When your packets go to the internet, they leave through the WAN side, getting NATed in the process. So your laptop sees `192.168.1.1` as the gateway, but the rest of the internet sees `203.0.113.7`.

### "Should I disable IPv6?"

**The confusion:** I've heard IPv6 causes problems.

**The fix:** Almost always **no**. Disabling IPv6 will at best speed nothing up, and at worst will silently break things (some Linux services bind to `::` expecting both stacks, modern apps prefer IPv6, etc.). If something is actually broken with IPv6, fix the broken thing rather than disabling the whole stack. There are extremely few legitimate reasons to disable IPv6 in 2026.

### "Is `127.0.0.1` the same as `localhost`?"

**The confusion:** I see them used interchangeably.

**The fix:** `localhost` is a name. `127.0.0.1` is an IPv4 address. By default, the name `localhost` resolves to `127.0.0.1` (and `::1` in IPv6). They behave identically for most purposes. The mapping comes from `/etc/hosts`. You can technically remap `localhost` to something else, but please don't.

### "Why does my computer have so many IPs?"

**The confusion:** `ip addr` shows ten different addresses. Why?

**The fix:** Each interface can have multiple addresses (like an IPv4 and several IPv6 addresses). You also have a loopback (`127.0.0.1`, `::1`). If you have a VPN running, that's another interface with another IP. Docker containers add internal interfaces with their own IPs. Adding it all up: a regular laptop on a single Wi-Fi network easily has 5+ IPs.

### "Why does ping say 'TTL=64' or 'TTL=128'?"

**The confusion:** Different OSes show different TTL values in ping replies.

**The fix:** Different OSes use different starting TTLs. Linux defaults to 64. Windows defaults to 128. Some older Cisco gear used to use 255. The TTL in the reply tells you `(starting TTL) - (number of hops)`. You can guess the OS sometimes from the TTL: if you ping a host and see TTL=58, that's probably Linux (64-6 hops). If you see TTL=124, probably Windows.

### "Why are ports 0-1023 special?"

**The confusion:** I read that I need root to bind to ports below 1024.

**The fix:** Ports 0-1023 are called **privileged ports**. The kernel requires root (or `CAP_NET_BIND_SERVICE`) to bind to them. The reasoning: classically, ports below 1024 ran trusted services (mail, web, SSH). Forcing root meant non-root users couldn't impersonate those services. The convention has stuck even though modern security models are much more sophisticated.

### "Is `0.0.0.0` an address?"

**The confusion:** I see `0.0.0.0` in some places and it's confusing.

**The fix:** `0.0.0.0` is a special address. As a **source**, it means "I don't have an IP yet" (used during DHCP DISCOVER, for example). As a **destination**, it means "this network." As a **bind address**, it means "any IP I have" — bind to all interfaces. So when a server says "listening on 0.0.0.0:8080," it means "I'm listening on port 8080 on every IP address this machine has." The IPv6 equivalent is `::`.

### "Why is my home IP different every few weeks?"

**The confusion:** I'm sure my public IP changed since last time I checked.

**The fix:** Your ISP gives you an IP via DHCP with a lease. Leases expire. When they renew, you usually get the same IP back, but not always. Some ISPs explicitly cycle addresses to discourage running servers from home. Your IPv4 address is yours but only "for now." If you need a stable public IP, you usually pay extra for a static one.

### "Why does `ipconfig /all` on Windows look so different from `ip addr` on Linux?"

**The confusion:** Different tools, different output, same underlying network.

**The fix:** They are different commands on different OSes that show similar info. Windows uses `ipconfig`. macOS and BSD use `ifconfig`. Linux uses `ip addr` (or older `ifconfig`). The protocols underneath (IP, ARP, NDP, etc.) are exactly the same on all of them — just the user-space tools differ.

## Vocabulary

| Term | Plain English |
|------|---------------|
| **IP** | Internet Protocol. The addressing system that lets devices talk across networks. |
| **IPv4** | The old version of IP, 32-bit addresses. Still dominant in 2026. |
| **IPv6** | The new version of IP, 128-bit addresses. Climbing fast. |
| **Packet** | A chunk of data with a header and payload. The basic unit of IP. |
| **Datagram** | Same idea as packet, slightly more formal. Often used for UDP. |
| **Header** | The label at the front of a packet. Contains addressing and metadata. |
| **Payload** | The actual data inside the packet. The "letter" inside the envelope. |
| **TTL** | Time To Live. Counter that prevents packets from looping forever. (IPv4 name.) |
| **Hop limit** | The IPv6 name for TTL. Counts router hops. |
| **Source** | The "from" address on a packet. |
| **Destination** | The "to" address on a packet. |
| **Octet** | An 8-bit chunk. IPv4 has four octets. |
| **Hextet** | A 16-bit chunk written in hex. IPv6 has eight hextets. |
| **Dotted-quad** | The IPv4 address format: four decimal numbers with dots between. |
| **Colon-hex** | The IPv6 address format: hex hextets with colons between. |
| **Loopback** | The special "talk to myself" address. `127.0.0.1` (IPv4) or `::1` (IPv6). |
| **Multicast** | One-to-many delivery. Anybody who joined the group gets the packet. |
| **Broadcast** | One-to-everyone-on-the-link. IPv4 only — IPv6 has no broadcast. |
| **Anycast** | Multiple machines with the same address — packet goes to the nearest. |
| **Link-local** | Address valid only on the local wire. IPv4 `169.254.0.0/16`, IPv6 `fe80::/10`. |
| **Unique-local** | IPv6 private. `fc00::/7`. Like RFC 1918 but for IPv6. |
| **Global** | An address that's globally routable on the internet. |
| **Public** | Same idea as global. A "real" internet address. |
| **Private** | An address not routable on the internet. RFC 1918 / fc00::/7. |
| **RFC 1918** | The standard that defines IPv4 private address blocks. |
| **RFC 4193** | The standard that defines IPv6 unique-local addresses. |
| **fc00** | The IPv6 unique-local prefix. (Practically `fd00::/8`.) |
| **fe80** | The IPv6 link-local prefix. |
| **169.254** | The IPv4 link-local prefix. Sign of DHCP failure when you see one. |
| **SLAAC** | Stateless Address Auto-Configuration. IPv6 way to self-configure. |
| **EUI-64** | An interface ID derived from a MAC address. Used in older SLAAC. |
| **DHCP** | Dynamic Host Configuration Protocol. Hands out IP addresses (and more). |
| **DHCPv4** | DHCP for IPv4. |
| **DHCPv6** | DHCP for IPv6. |
| **RA** | Router Advertisement. IPv6 message announcing the network's prefix. |
| **RS** | Router Solicitation. IPv6 message asking routers to identify themselves. |
| **NS** | Neighbor Solicitation. The IPv6 "who has X?" message. |
| **NA** | Neighbor Advertisement. The IPv6 "I have X" reply. |
| **NDP** | Neighbor Discovery Protocol. The IPv6 replacement for ARP. |
| **ARP** | Address Resolution Protocol. IPv4 way to find a MAC for an IP. |
| **Gratuitous ARP** | An ARP reply sent unsolicited. Useful for announcing yourself. |
| **DAD** | Duplicate Address Detection. Make sure nobody else has your address before using it. |
| **CIDR** | Classless Inter-Domain Routing. The slash-notation way of writing networks. |
| **Subnet** | A subdivision of a larger network. |
| **Mask** | The pattern that says where the network bits stop and host bits start. |
| **Prefix length** | The number after the slash in CIDR. Tells you how many bits are network. |
| **Longest-prefix match** | The routing rule: most specific matching entry wins. |
| **Default route** | The "if nothing more specific matches, use this" route. `0.0.0.0/0` or `::/0`. |
| **Default gateway** | The router that handles the default route. |
| **IGP** | Interior Gateway Protocol. Routing inside one organization. |
| **EGP** | Exterior Gateway Protocol. Routing between organizations. (Just BGP.) |
| **RIB** | Routing Information Base. The complete list of routes the kernel knows. |
| **FIB** | Forwarding Information Base. The fast-path table used to actually forward. |
| **NAT** | Network Address Translation. Mapping addresses on the way through. |
| **SNAT** | Source NAT. Rewriting the source address. |
| **DNAT** | Destination NAT. Rewriting the destination address. |
| **PAT** | Port Address Translation. SNAT with port rewriting. The home-router kind. |
| **Masquerade** | A flavor of SNAT where the source becomes the outgoing interface's IP. |
| **Hairpin NAT** | Reaching your own public service from inside the same NAT. |
| **Port forwarding** | DNAT for incoming connections. |
| **MTU** | Maximum Transmission Unit. The largest packet a link can carry. |
| **MSS** | Maximum Segment Size. TCP's per-segment cap, derived from MTU. |
| **PMTU** | Path MTU. The smallest MTU along the path. |
| **PMTUD** | Path MTU Discovery. The process of finding the PMTU. |
| **Fragment** | A piece of a too-big packet that got chopped up. |
| **MF** | More Fragments flag. Set on every fragment except the last. |
| **DF** | Don't Fragment flag. If set, routers must not fragment. |
| **Fragment offset** | Where this fragment fits in the reassembled whole. |
| **Traceroute** | A tool that maps the path of routers between you and a destination. |
| **Ping** | A tool that sends ICMP echo and times the reply. |
| **mtr** | Combined traceroute + ping. |
| **Tracepath** | A non-root traceroute alternative on Linux. |
| **hping** | A traceroute-like tool with custom packet crafting. |
| **nmap** | A port scanner. |
| **ip command** | The modern Linux network configuration tool (part of iproute2). |
| **iproute2** | The Linux package containing `ip`, `ss`, `tc`, etc. |
| **route command** | The deprecated old route-listing tool. |
| **ifconfig** | The deprecated old interface-listing tool. |
| **MAC address** | The 48-bit hardware address of a network card. |
| **OUI** | Organizationally Unique Identifier. The first 24 bits of a MAC, identifying the maker. |
| **Ethernet** | The most common link-layer protocol. Wired or wireless (Wi-Fi is "Ethernet over radio"). |
| **ARP cache** | The kernel's table of known MAC addresses for IPv4 neighbors. |
| **Neighbor cache** | The kernel's table of known MAC addresses for IPv6 neighbors. |
| **IP forwarding** | Whether a host will forward packets between its interfaces (acting as a router). |
| **rp_filter** | Reverse Path Filter. Drops packets whose source doesn't match the routing table. |
| **conntrack** | The Linux kernel's connection tracker. The brain behind stateful firewalls and NAT. |
| **iptables** | The legacy Linux firewall tool. |
| **nftables** | The modern Linux firewall, replacing iptables. |
| **nat table** | The iptables/nftables table dedicated to NAT. |
| **mangle table** | The table for packet rewriting (TTL, TOS, etc.). |
| **filter table** | The table for accept/drop firewall rules. |
| **raw table** | The table for bypass-conntrack rules. |
| **security table** | The table used by SELinux for label-based firewalling. |
| **ICMP** | Internet Control Message Protocol. Diagnostic and error messages. |
| **ICMP unreachable** | An ICMP message saying "I couldn't deliver your packet." |
| **ICMPv6 PTB** | Packet Too Big. The IPv6 equivalent of "fragmentation needed." |
| **DHCPv6-PD** | DHCPv6 Prefix Delegation. Hands out whole subnets, not just addresses. |
| **Zone** | A scope identifier. `fe80::1%eth0` means "fe80::1 on eth0." |
| **Scope** | What range an address is valid in (host, link, global). |
| **MLD** | Multicast Listener Discovery. IPv6 way to subscribe to multicast groups. |
| **IGMP** | Internet Group Management Protocol. IPv4 way to subscribe to multicast groups. |
| **IGMPv2 / v3** | Versions of IGMP. v3 supports source-specific multicast. |
| **MLDv1 / v2** | Versions of MLD. v2 supports source-specific multicast. |
| **RPKI** | Resource Public Key Infrastructure. Cryptographic origin validation for BGP. |
| **ROA** | Route Origin Authorization. A signed statement saying "this AS owns these prefixes." |
| **ASN** | Autonomous System Number. A unique ID for a network on the BGP global stage. |
| **BGP** | Border Gateway Protocol. The internet's only EGP. |
| **ECMP** | Equal-Cost Multi-Path. Load-spreading across multiple equal routes. |
| **Hash-based ECMP** | The most common ECMP — picks a path by hashing flow tuple. |
| **Traffic engineering** | Manipulating routes to achieve specific traffic patterns. |
| **MPLS** | Multi-Protocol Label Switching. A label-based forwarding tech beneath IP. |
| **VRF** | Virtual Routing and Forwarding. Multiple isolated routing tables on one device. |
| **Tunnel** | Wrapping one protocol inside another (GRE, IP-in-IP, IPsec, WireGuard). |
| **Encapsulation** | Wrapping a packet in another packet's header. |
| **Stack** | The collection of layered protocols (Application, Transport, Network, Link, Physical). |
| **Dual-stack** | Running both IPv4 and IPv6 simultaneously. |
| **6in4 / 6to4** | Tunneling IPv6 over IPv4 networks. |
| **Happy Eyeballs** | Algorithm for trying both IPv4 and IPv6 in parallel and using whichever wins. |

## Try This

Here are safe experiments to play with. None will break anything.

### Experiment 1: Add a secondary IP to loopback

This is harmless because loopback never goes anywhere.

```
$ sudo ip addr add 10.99.99.99/32 dev lo
$ ping -c 2 10.99.99.99
PING 10.99.99.99 (10.99.99.99) 56(84) bytes of data.
64 bytes from 10.99.99.99: icmp_seq=1 ttl=64 time=0.026 ms
64 bytes from 10.99.99.99: icmp_seq=2 ttl=64 time=0.029 ms
$ sudo ip addr del 10.99.99.99/32 dev lo
```

You added a fake address on loopback, pinged it (which works because it's local), then removed it. No risk.

### Experiment 2: Try a private-block ping

Ping the most common router address.

```
$ ping -c 4 192.168.1.1
PING 192.168.1.1 (192.168.1.1) 56(84) bytes of data.
64 bytes from 192.168.1.1: icmp_seq=1 ttl=64 time=0.521 ms
...
```

If your home router is `192.168.1.1`, this works. If it's something else (`192.168.0.1`, `10.0.0.1`), check `ip route show | grep default` for the right address.

### Experiment 3: Trace to your favorite IPv6 site

```
$ traceroute -6 google.com
traceroute to google.com (2607:f8b0:4004:c08::64), 30 hops max, 80 byte packets
 1  _gateway (fe80::1)  0.512 ms  ...
 2  2001:db8:abcd::1 ...
 ...
```

You'll see your packet hop through routers using IPv6 addresses. If your network doesn't have IPv6, you'll get an error.

### Experiment 4: Convert a CIDR by hand and verify

Pick `192.168.5.0/26`. By the math:
- Mask is 26 ones, 6 zeros: `11111111.11111111.11111111.11000000` = `255.255.255.192`.
- Total addresses: 2^6 = 64.
- Usable hosts: 62.
- First usable: `192.168.5.1`.
- Last usable: `192.168.5.62`.
- Broadcast: `192.168.5.63`.

Verify with `ipcalc` if installed:

```
$ ipcalc 192.168.5.0/26
Address:   192.168.5.0          11000000.10101000.00000101.00 000000
Netmask:   255.255.255.192 = 26 11111111.11111111.11111111.11 000000
Wildcard:  0.0.0.63             00000000.00000000.00000000.00 111111
=>
Network:   192.168.5.0/26       11000000.10101000.00000101.00 000000
HostMin:   192.168.5.1          11000000.10101000.00000101.00 000001
HostMax:   192.168.5.62         11000000.10101000.00000101.00 111110
Broadcast: 192.168.5.63         11000000.10101000.00000101.00 111111
Hosts/Net: 62                    Class C, Private Internet
```

The numbers match what you computed by hand.

### Experiment 5: Check the hop count to a few different sites

```
$ ping -c 1 8.8.8.8 | grep ttl
$ ping -c 1 1.1.1.1 | grep ttl
$ ping -c 1 9.9.9.9 | grep ttl
```

The TTL in the reply tells you how many hops away each is (assuming starting TTL of 64). Different DNS providers might be different distances from you.

### Experiment 6: Watch a real connection establish

```
$ sudo tcpdump -i any -n -c 6 'host 8.8.8.8'
```

Then in another terminal:

```
$ curl -s https://dns.google/resolve?name=example.com >/dev/null
```

You'll see the SYN, SYN-ACK, ACK handshake, then encrypted data, then FIN. Real packets, real protocol.

### Experiment 7: Find your public IP

```
$ curl -s https://ifconfig.me
203.0.113.7
```

Or for IPv6:

```
$ curl -s -6 https://ifconfig.me
2001:db8:abcd::1234
```

### Experiment 8: Test that PMTU works

Set DF and try to send a too-big packet on a path that probably can't handle it:

```
$ ping -M do -s 1472 -c 4 8.8.8.8
PING 8.8.8.8 (8.8.8.8) 1472(1500) bytes of data.
1480 bytes from 8.8.8.8: icmp_seq=1 ttl=115 time=14.3 ms
...
```

`-M do` means "set DF." `-s 1472` is "1472 bytes of payload" (which becomes 1500 bytes with the IP+ICMP header). If this works, your path supports 1500-byte packets. If you increase to `-s 1473`, you should see "Frag needed" errors or silent drops.

### Experiment 9: See what gets discarded by rp_filter

If you have `rp_filter` enabled, you can watch the counter increase when packets are dropped:

```
$ cat /proc/sys/net/ipv4/conf/all/rp_filter
1
$ cat /proc/net/netstat | grep -i Martian
```

Look for `IPReversePathFilter` or similar. (The exact name depends on kernel version.)

### Experiment 10: Inspect a route's metric

```
$ ip -d route show
default via 192.168.1.1 dev eth0 proto dhcp metric 100 ...
192.168.1.0/24 dev eth0 proto kernel scope link src 192.168.1.42 metric 100
```

The `metric` is a tiebreaker when multiple routes match. Lower wins. Most home routes have metric 100.

## Where to Go Next

Once this sheet feels easy, the dense engineer-grade material is one command away. Stay in the terminal.

- **`cs networking ip`** — the dense reference. Real names of every field, every flag, every protocol number.
- **`cs networking ipv4`** — version-specific, IPv4 details (header, options, classes).
- **`cs networking ipv6`** — version-specific, IPv6 details (header, addressing, NDP).
- **`cs networking ipv6-advanced`** — extension headers, prefix delegation, transition technologies.
- **`cs networking dhcp`**, **`cs networking dhcpv6`** — address assignment.
- **`cs networking arp`** — IPv4 link-layer resolution in detail.
- **`cs networking dns`** — turning names into IPs.
- **`cs networking icmp`** — error and diagnostic messages.
- **`cs networking tcp`**, **`cs networking udp`** — what runs on top of IP.
- **`cs networking tcpdump`** — packet capture in depth.
- **`cs networking iptables`** — the classic Linux firewall.
- **`cs ramp-up tcp-eli5`**, **`cs ramp-up udp-eli5`**, **`cs ramp-up icmp-eli5`** — gentle intros to the layer-4 friends.
- **`cs ramp-up bgp-eli5`** — how the internet's routes are announced.
- **`cs ramp-up binary-numbering-eli5`** — the counting/addressing math.

## See Also

- `networking/ip` — engineer-grade reference.
- `networking/ipv4` — IPv4 specifics.
- `networking/ipv6` — IPv6 specifics.
- `networking/ipv6-advanced` — extension headers and prefix delegation.
- `networking/dhcp` — IPv4 address assignment.
- `networking/dhcpv6` — IPv6 stateful config.
- `networking/dns` — names to addresses.
- `networking/arp` — IPv4 neighbor resolution.
- `networking/icmp` — diagnostic protocol.
- `networking/tcp` — reliable transport on top of IP.
- `networking/udp` — datagram transport on top of IP.
- `networking/tcpdump` — packet capture and decoding.
- `networking/iptables` — Linux firewall and NAT.
- `ramp-up/tcp-eli5` — the reliable transport, plain English.
- `ramp-up/udp-eli5` — the datagram transport, plain English.
- `ramp-up/icmp-eli5` — error messages, plain English.
- `ramp-up/bgp-eli5` — the internet's routing protocol.
- `ramp-up/linux-kernel-eli5` — the OS that runs all of this.
- `ramp-up/binary-numbering-eli5` — the counting math under address arithmetic.

## References

- **RFC 791** — Internet Protocol (IPv4, 1981). The original.
- **RFC 8200** — Internet Protocol, Version 6 (IPv6) Specification (2017). The current IPv6 standard.
- **RFC 1918** — Address Allocation for Private Internets. Defines the IPv4 private blocks.
- **RFC 4193** — Unique Local IPv6 Unicast Addresses. Defines `fc00::/7`.
- **RFC 4291** — IP Version 6 Addressing Architecture. The IPv6 addressing rules.
- **RFC 4861** — Neighbor Discovery for IPv6. NDP, the replacement for ARP.
- **RFC 4862** — IPv6 Stateless Address Autoconfiguration. SLAAC.
- **RFC 5952** — A Recommendation for IPv6 Address Text Representation. How to write IPv6 addresses.
- **RFC 826** — An Ethernet Address Resolution Protocol. ARP.
- **RFC 2131** — DHCP. The IPv4 version.
- **RFC 8415** — DHCPv6.
- **RFC 4632** — Classless Inter-Domain Routing (CIDR).
- **`man 7 ip`** — Linux man page for the IPv4 stack. Type `man 7 ip`.
- **`man 7 ipv6`** — Linux man page for the IPv6 stack. Type `man 7 ipv6`.
- **`man 8 ip`** — Linux man page for the `ip` command (iproute2). Type `man 8 ip`.
- **`man 8 ip-address`**, **`man 8 ip-route`**, **`man 8 ip-neighbour`** — sub-pages.
- **"IPv6 Essentials"** by Silvia Hagen — friendly book on IPv6 deployment.
- **"TCP/IP Illustrated, Vol 1"** by Stevens, Fall — encyclopedia of TCP/IP. Pick up after this sheet.
- **"Internetworking with TCP/IP"** by Comer — the academic alternative to Stevens.

Tip: every reference above can be read inside your terminal. Most are accessible via `man`. RFCs can be read with `rfc-tools` or just downloaded as plain text and opened in `less`. You really do not need to leave the terminal.

— End of ELI5 —

When this sheet feels boring (and it will, faster than you think), graduate to `cs networking ip`, the engineer-grade reference. After that, `cs detail networking/ip` gives you the math and academic underpinning. By the time you've read both, you'll be reading packet captures without flinching.

### One last thing before you go

Pick one command from the Hands-On section that you have not run yet. Run it right now. Read the output. Try to figure out what each part means, using the Vocabulary table as your dictionary. Don't just trust this sheet — see for yourself. IP is a real thing. It is on your computer, doing its job, right now. The commands in this sheet let you peek at it.

Reading is good. Doing is better. Type the commands. Watch the network respond.

You are now officially started on your IP journey. Welcome.

The whole point of the North Star for the `cs` tool is: never leave the terminal to learn this stuff. Everything you need is here, or one `man` page away, or one RFC away. There is no Google search you need to do to start understanding IP. You can sit at your terminal, type, watch, read, and learn forever.

Have fun. The internet is happy to be poked at. Nothing on this sheet will break anything serious. Try things. Type commands. Read what comes back. The more you do, the more it all clicks into place.

— End of ELI5 — (really this time!)
