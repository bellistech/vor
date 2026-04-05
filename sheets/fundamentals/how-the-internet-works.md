# How the Internet Works (From Your WiFi to the World)

A tiered guide to the global network of networks.

## ELI5

### A Giant Road System

The internet is like a giant system of roads connecting cities all over the world.

Your **computer** is your house. The **websites** you visit (Google, YouTube,
Wikipedia) live in big warehouses called **data centers** -- buildings full of
thousands of computers that never turn off.

To get from your house to those warehouses, you need a road. Your **Internet
Service Provider** (ISP) is the on-ramp. They connect your house to the highway
system.

```
Your           ISP            Big           Data
House  ------> On-Ramp -----> Highway ----> Center
(Computer)    (Comcast,       (The          (Where
               AT&T,          Internet      websites
               Verizon)       Backbone)     live)
```

### How a Web Page Gets to You

When you type "google.com" in your browser:

1. Your computer asks a **phone book** (called DNS): "What is Google's address?"
2. The phone book answers: "Google lives at 142.250.80.4"
3. Your computer sends a request down the road to that address
4. Google's warehouse computer builds the page and sends it back
5. Your browser shows you the page

The whole trip takes less than a second -- even though the data might travel
thousands of miles.

### The Wires Under the Ocean

How does the internet work between continents? There are huge cables lying on
the ocean floor, connecting North America to Europe, Asia, Africa, and everywhere
else. These **undersea cables** carry almost all international internet traffic.
Satellites handle very little of it.

```
North                    Atlantic                    Europe
America  ---- [Cable on Ocean Floor] ---- [London]
[New York]               3,500 miles
```

### Copies Closer to Home

If everyone in your city watched the same YouTube video by connecting all the way
to YouTube's data center in California, the roads would be jammed. Instead,
popular content is copied to smaller warehouses closer to you. These are called
**CDNs** (Content Delivery Networks) -- like local libraries that keep copies of
popular books so you do not have to drive to the central warehouse.

---

## Middle School

### Your Path to the Internet

When you connect to the internet from home, the path looks like this:

```
Your Device --> WiFi --> Home Router --> ISP --> Internet Backbone --> Destination
```

Your **home router** is the border between your private network and the public
internet. It has two addresses: a private one (like 192.168.1.1) for your home
network and a public one assigned by your ISP.

### Internet Service Providers (Tiers)

Not all ISPs are equal. They are organized into tiers:

| Tier | What They Do | Examples |
|:---:|:---|:---|
| Tier 1 | Own the backbone. Reach the entire internet without paying anyone else. | Lumen (CenturyLink), NTT, Cogent, Telia |
| Tier 2 | Regional networks. Pay Tier 1 for some routes, peer with others for free. | Comcast, Vodafone, Cox |
| Tier 3 | Local ISPs. Buy access from Tier 1 or 2 and sell it to homes and businesses. | Small local providers |

Your home ISP (Tier 3 or 2) connects you to a Tier 1 backbone, which connects
to every other network on earth.

### Data Centers

A data center is a building designed to house thousands of servers. They have:

- Redundant power (generators, UPS batteries)
- Massive cooling systems (servers generate enormous heat)
- Multiple internet connections from different providers
- Physical security (biometric locks, cameras, guards)

Big companies (Google, Amazon, Microsoft) build their own data centers. Smaller
companies rent space in shared data centers (colocation).

### Undersea Cables

Over 400 submarine cables crisscross the ocean floor, carrying 99% of
intercontinental data. A single modern cable can carry over 200 terabits per
second. They are about the diameter of a garden hose and are laid by specialized
ships.

### How a Google Search Travels

```
1. You type "cats" and press Enter
2. Your computer --> WiFi --> home router
3. Router --> ISP local exchange (a few miles away)
4. ISP --> regional backbone (maybe 100 miles)
5. Backbone --> Google's nearest data center (could be in your state)
6. Google's servers search billions of pages in < 0.5 seconds
7. Results travel back the same path in reverse
8. Total round trip: typically 20-80 milliseconds
```

### CDNs (Content Delivery Networks)

CDNs are networks of servers spread across the world that cache (copy) content
closer to users:

```
Without CDN:                          With CDN:
User in Tokyo --> California          User in Tokyo --> Tokyo CDN server
(200ms round trip)                    (5ms round trip)
```

Major CDN providers: Cloudflare, Akamai, Fastly, AWS CloudFront. When you watch
a Netflix show, the video streams from a CDN server near you, not from Netflix's
main data center.

### WiFi to Router to ISP to Backbone

```
Your Phone            Home Router          ISP                  Backbone
(192.168.1.5) ------> (192.168.1.1) -----> (10.0.0.1) -------> (Core Routers)
   WiFi (2.4/5 GHz)     Ethernet/Fiber      Fiber/Coax          Fiber (100+ Gbps)
   ~100 Mbps             1 Gbps              50-1000 Mbps        Terabits/sec
```

Each step is a faster, higher-capacity link. Your home connection is the
narrowest part of the path (the "last mile").

---

## High School

### BGP and Autonomous Systems

The internet is not one network. It is tens of thousands of independently
operated networks called **Autonomous Systems** (AS), each identified by a
unique **ASN** (Autonomous System Number):

| ASN | Organization |
|:---|:---|
| AS15169 | Google |
| AS13335 | Cloudflare |
| AS16509 | Amazon (AWS) |
| AS7018 | AT&T |
| AS3356 | Lumen (CenturyLink) |

**BGP** (Border Gateway Protocol) is the routing protocol that connects all these
ASes. BGP routers exchange **path vectors** -- lists of ASNs a packet must
traverse to reach a destination. Each network applies policies to decide which
paths to prefer.

```
AS 7018 (AT&T) --> AS 3356 (Lumen) --> AS 15169 (Google)
     or
AS 7018 (AT&T) --> AS 13335 (Cloudflare) --> AS 15169 (Google)
```

BGP chooses based on path length, business relationships, and local policy.

### DNS Hierarchy

DNS is not a single phone book. It is a hierarchy of name servers:

```
Root Servers (13 clusters, letters A-M)
    |
    v
TLD Servers (.com, .org, .net, .io, etc.)
    |
    v
Authoritative Servers (google.com's own DNS servers)
    |
    v
Your Answer: google.com = 142.250.80.4
```

**Resolution process:**

1. Your computer asks its configured **recursive resolver** (often your ISP or 8.8.8.8)
2. Resolver asks a **root server**: "Where is .com?"
3. Root says: "Ask the .com TLD server at 192.5.6.30"
4. Resolver asks the TLD server: "Where is google.com?"
5. TLD says: "Ask Google's authoritative server at 216.239.32.10"
6. Resolver asks the authoritative server: "What is google.com's IP?"
7. Authoritative answers: "142.250.80.4"
8. Resolver caches the answer and returns it to your computer

### HTTP/HTTPS and the TLS Handshake

**HTTP** (HyperText Transfer Protocol) is how browsers request web pages.
**HTTPS** adds encryption via **TLS** (Transport Layer Security).

The TLS 1.3 handshake establishes an encrypted connection:

```
Client                              Server
  |--- ClientHello ----------------->|  Supported ciphers, random value
  |<-- ServerHello, Certificate -----|  Chosen cipher, server's certificate
  |    + Key Share                   |  Server's public key for key exchange
  |--- Key Share, Finished --------->|  Client's public key, verify
  |<-- Finished ---------------------|  Server confirms
  |=== Encrypted Data ===============|  All traffic now encrypted
```

TLS 1.3 completes in 1 round trip (1-RTT). TLS 1.2 required 2 round trips.

### Load Balancers

When millions of users access the same website, one server cannot handle the
load. A **load balancer** distributes requests across many servers:

```
                    +-- Server 1
Users --> [LB] -----+-- Server 2
                    +-- Server 3
                    +-- Server 4
```

Load balancing strategies:
- **Round robin**: each request goes to the next server in rotation
- **Least connections**: send to the server with the fewest active connections
- **Consistent hashing**: same client always goes to the same server

### Anycast

**Anycast** assigns the same IP address to servers in multiple locations. When
you connect, BGP routing sends you to the nearest one:

```
Same IP: 1.1.1.1 (Cloudflare DNS)

User in London --> London server (1.1.1.1)
User in Tokyo  --> Tokyo server  (1.1.1.1)
User in NYC    --> NYC server    (1.1.1.1)
```

Anycast is used for DNS, CDNs, and DDoS mitigation. If one site goes down,
traffic automatically routes to the next closest.

### Peering vs. Transit

Networks connect in two ways:

- **Peering**: two networks exchange traffic for free (mutual benefit).
  Usually at an **IXP** (Internet Exchange Point) -- a physical building where
  networks plug into the same switch.
- **Transit**: one network pays another to carry its traffic to the rest of the
  internet. Tier 3 ISPs pay Tier 2, which may pay Tier 1.

Major IXPs: DE-CIX (Frankfurt), AMS-IX (Amsterdam), LINX (London), Equinix
(multiple cities). DE-CIX handles over 14 Tbps peak traffic.

### Traceroute -- Seeing the Path

`traceroute` shows every router hop between you and a destination:

```bash
traceroute google.com
# 1  192.168.1.1       1.2 ms   (your router)
# 2  10.0.0.1          5.4 ms   (ISP local)
# 3  72.14.233.105    12.1 ms   (ISP backbone)
# 4  108.170.246.1    15.6 ms   (Google edge)
# 5  142.250.80.4     18.3 ms   (Google server)
```

It works by sending packets with increasing TTL (Time To Live) values. Each
router decrements TTL by 1. When TTL hits 0, the router sends back an error
message, revealing its identity.

---

## College

### BGP Path Selection Algorithm

BGP routers receive multiple paths to the same prefix and select the best using
an ordered decision process:

1. **Highest LOCAL_PREF** (local policy preference -- customer > peer > transit)
2. **Shortest AS_PATH** (fewest ASes to traverse)
3. **Lowest ORIGIN** (IGP < EGP < Incomplete)
4. **Lowest MED** (Multi-Exit Discriminator -- neighbor's preference)
5. **eBGP over iBGP** (prefer external routes)
6. **Lowest IGP metric** to the BGP next hop
7. **Oldest route** (stability)
8. **Lowest router ID** (tiebreaker)

LOCAL_PREF typically encodes business relationships:

```
Customer routes:  LOCAL_PREF = 100  (preferred -- they pay you)
Peer routes:      LOCAL_PREF = 80   (free exchange)
Transit routes:   LOCAL_PREF = 60   (you pay them)
```

### RPKI and Route Origin Validation

**RPKI** (Resource Public Key Infrastructure) prevents BGP hijacking by
cryptographically binding IP prefixes to authorized ASNs.

Components:
- **ROA** (Route Origin Authorization): signed statement that AS X may
  originate prefix Y with max length Z
- **RIR trust anchors**: ARIN, RIPE, APNIC, LACNIC, AFRINIC issue certificates
- **Validators**: software that downloads ROAs and feeds them to BGP routers

Validation states:
- **Valid**: prefix matches a ROA
- **Invalid**: prefix conflicts with a ROA (potential hijack)
- **Not Found**: no ROA exists for this prefix

As of 2025, roughly 50% of IPv4 routes have ROAs. Route-origin validation is
deployed by most major networks but enforcement (dropping invalid routes) varies.

### Submarine Cable Topology

The global submarine cable network forms an irregular mesh:

- **~550 cables** in service worldwide
- **Highest capacity routes**: transatlantic (~500+ Tbps total), transpacific
  (~400+ Tbps total), Europe-Asia
- **Chokepoints**: Strait of Malacca, Suez Canal, English Channel, Luzon Strait
- **Landing stations**: hardened buildings where cables come ashore and connect
  to terrestrial fiber

A single cable break rarely causes outages because routes have redundancy, but
regional chokepoints (e.g., multiple cables through the Red Sea) create
correlated failure risk.

### CDN Cache Hierarchies

Large CDNs use multi-tier caching:

```
Origin Server (data center)
    |
    v
Shield/Mid-Tier Cache (one per region)
    |
    v
Edge Cache (one per city/PoP)
    |
    v
User
```

- **Edge hit**: fastest, served from the nearest PoP
- **Mid-tier hit**: edge missed, shield has it, avoids origin round-trip
- **Origin fetch**: cache miss everywhere, full round-trip to origin

Cache key: typically URL + relevant headers (Accept-Encoding, Vary). Cache
invalidation is the hard part -- purge APIs, TTL expiry, and stale-while-
revalidate strategies balance freshness vs. efficiency.

### Anycast Routing and Catchment

Anycast works because BGP naturally routes to the topologically closest
announcement of a prefix. The **catchment area** of an anycast node is the set
of source IPs that BGP routes to it.

Challenges:
- **Affinity**: TCP connections must stay pinned to one node. If BGP
  reconverges mid-connection, the session breaks (mitigated by ECMP hashing
  and connection tracking).
- **Uneven load**: catchment areas depend on BGP topology, not geography.
  One node might attract disproportionate traffic.
- **Failover**: removing a BGP announcement withdraws the node. Convergence
  time (seconds to minutes) determines failover speed.

### DDoS Mitigation

Distributed Denial-of-Service attacks flood a target with traffic. Defense
operates at multiple layers:

- **Blackholing** (RTBH -- Remotely Triggered Black Hole): advertise the
  victim's prefix with a community that tells upstream routers to drop all
  traffic to it. Stops the attack but also blocks legitimate users.
- **Scrubbing**: route traffic through a scrubbing center that filters malicious
  packets and forwards clean traffic. Providers: Cloudflare, Akamai Prolexic,
  AWS Shield.
- **Rate limiting**: drop packets exceeding a threshold per source IP or per
  flow.
- **Anycast absorption**: distribute attack traffic across many PoPs so no
  single location is overwhelmed.
- **SYN cookies**: respond to SYN floods without allocating state until the
  handshake completes.

### Internet Governance

The internet has no single authority. Governance is distributed:

| Organization | Role |
|:---|:---|
| **IANA** | Manages DNS root zone, IP address pools, protocol parameters |
| **ICANN** | Coordinates DNS, domain name policy, accredits registrars |
| **RIRs** (ARIN, RIPE, APNIC, LACNIC, AFRINIC) | Allocate IP addresses and ASNs by region |
| **IETF** | Develops internet standards (RFCs) via open process |
| **IEEE** | Ethernet, WiFi (802.x) standards |
| **W3C** | Web standards (HTML, CSS, WebAssembly) |
| **ITU** | International telecommunications treaties |

### Network Neutrality -- Technical Mechanisms

Network neutrality means ISPs treat all traffic equally. Without it, ISPs can:

- **Throttle**: reduce bandwidth for specific services (e.g., slow Netflix)
- **Prioritize**: use DiffServ/DSCP markings to give preferred traffic lower
  latency
- **Block**: drop packets to certain destinations entirely
- **Zero-rate**: exempt certain services from data caps

Technical enforcement relies on **Deep Packet Inspection** (DPI) to classify
traffic by application, and **traffic shaping** (token bucket, leaky bucket) to
control rates per class.

### IPv4 Exhaustion and CGNAT

IANA exhausted its free IPv4 pool in 2011. RIRs have since run out or are
rationing. Solutions:

- **CGNAT** (Carrier-Grade NAT): ISP places thousands of customers behind a
  single public IP. Customers get private addresses (100.64.0.0/10, RFC 6598).
  Breaks end-to-end connectivity, complicates P2P and hosting.
- **IPv6 transition**: dual-stack (run both), 6to4 tunnels, NAT64/DNS64
  (translate IPv6 clients to IPv4 servers).
- **IP address markets**: organizations buy and sell IPv4 blocks. Prices:
  approximately $40-55 per address (2025).

---

## Tips

- Use `dig +trace example.com` to see the full DNS resolution path from root to answer
- `mtr` combines ping and traceroute for continuous path analysis
- `curl -w "%{time_connect} %{time_starttls} %{time_total}\n" -o /dev/null -s https://example.com` measures connection timing
- Check your public IP and ASN: `curl ipinfo.io`
- Submarine cable maps: submarinecablemap.com shows every cable and landing station
- BGP looking glasses (e.g., lg.he.net) let you see how routes look from different networks
- `whois <IP>` reveals the ASN and organization that owns an address

## See Also

- how-networking-works
- dns
- bgp
- http
- tls
- traceroute
- nat
- ipv4
- ipv6
- load-balancing
- cdn
- ddos

## References

- RFC 4271 (BGP-4)
- RFC 8205 (BGPsec Protocol Specification)
- RFC 6480-6488 (RPKI framework)
- RFC 6598 (CGNAT shared address space)
- RFC 8446 (TLS 1.3)
- RFC 1034/1035 (DNS)
- RFC 3168 (ECN), RFC 5575 (Flow Specification)
- Clark, D.D. "The Design Philosophy of the DARPA Internet Protocols" (1988)
- TeleGeography, "Submarine Cable Map" (submarinecablemap.com)
- Geoff Huston, "BGP in 2024" (APNIC blog series)
- Kurose & Ross, "Computer Networking: A Top-Down Approach"
- CAIDA, "AS Relationships and Internet Topology" datasets
