# Peering and Transit

Interconnection strategies for exchanging traffic between autonomous systems — peering (direct, typically settlement-free) vs transit (paid upstream providing full routing table access).

## Concepts

### Peering vs Transit vs Paid Peering

- **Settlement-Free Peering (SFP):** Two networks exchange traffic at no cost; each party only carries traffic destined to its own customers
- **Paid Peering:** One party pays the other for direct interconnection; common when traffic ratios are heavily skewed
- **Transit:** Upstream provider carries traffic to/from the entire internet; customer pays per-Mbps or commit rate; provider advertises customer prefixes to all peers and upstreams
- **Partial Transit:** Provider only advertises routes to a subset of destinations (e.g., regional or a specific set of peers)

### IXP (Internet Exchange Point) Architecture

- **Layer 2 IXP:** Shared Ethernet fabric (VLAN or dedicated switch); members connect and peer directly; most common model (DE-CIX, AMS-IX, LINX)
- **Layer 3 IXP:** IXP operates as an AS and participates in routing; less common
- **Route Server:** IXP-operated BGP speaker that collects and redistributes routes between members; eliminates need for bilateral peering sessions
- **Peering LAN:** Shared subnet (typically /23 or /24 IPv4 + /64 IPv6) where all IXP members have port addresses
- **Resiliency:** Dual-switch fabrics, multiple PoPs, redundant route servers

### Peering Policy Types

- **Open:** Will peer with anyone who meets basic technical requirements (has ASN, prefix, 24x7 NOC)
- **Selective:** Evaluates peering requests based on traffic volume, geographic overlap, mutual benefit
- **Restrictive:** Only peers with networks meeting strict criteria (minimum traffic, geographic presence, traffic ratios)
- **Peering coordinators** publish policy on PeeringDB and network websites

### PeeringDB

- Public database of networks, IXPs, and facilities
- Lists ASN, peering policy, traffic levels, IXP memberships, facility presence
- Essential for identifying potential peers and verifying peering requirements

```bash
# Query PeeringDB API for a network
curl -s "https://www.peeringdb.com/api/net?asn=13335" | jq '.data[0] | {name, policy_general, info_traffic, info_ratio}'

# List all networks at a specific IXP
curl -s "https://www.peeringdb.com/api/netixlan?ixlan_id=42" | jq '.data[] | {asn, name, ipaddr4, ipaddr6, speed}'

# Find IXPs in a city
curl -s "https://www.peeringdb.com/api/ix?city=Amsterdam" | jq '.data[] | {name, id, country}'

# Search facilities (data centers)
curl -s "https://www.peeringdb.com/api/fac?name__contains=Equinix" | jq '.data[] | {name, city, country}'
```

## BGP Peering Configuration

### Basic eBGP Peering at IXP

```
router bgp 65001
 bgp router-id 10.0.0.1
 ! Direct peer at IXP
 neighbor 198.51.100.2 remote-as 65002
 neighbor 198.51.100.2 description peer-ExampleNet
 address-family ipv4 unicast
  ! Only accept their prefixes
  neighbor 198.51.100.2 prefix-list PEER-65002-IN in
  neighbor 198.51.100.2 prefix-list MY-PREFIXES out
  ! Hard limit on received prefixes — protect RIB
  neighbor 198.51.100.2 maximum-prefix 1000 90 restart 15
 exit-address-family
 address-family ipv6 unicast
  neighbor 2001:db8:ixp::2 activate
  neighbor 2001:db8:ixp::2 prefix-list PEER-65002-V6-IN in
  neighbor 2001:db8:ixp::2 maximum-prefix 200 90 restart 15
 exit-address-family
```

### eBGP Multihop for PNI (Private Network Interconnect)

```
router bgp 65001
 ! PNI — direct cross-connect, loopback peering
 neighbor 10.255.0.2 remote-as 65002
 neighbor 10.255.0.2 ebgp-multihop 2
 neighbor 10.255.0.2 update-source Loopback0
 neighbor 10.255.0.2 description PNI-ExampleNet-DFW1
 address-family ipv4 unicast
  neighbor 10.255.0.2 maximum-prefix 5000 80
 exit-address-family
```

### TTL Security (GTSM — RFC 5082)

```
! Generalized TTL Security Mechanism — drop packets with TTL < 254
! Protects against spoofed BGP packets from remote attackers
router bgp 65001
 neighbor 198.51.100.2 ttl-security hops 1
 ! Cannot combine with ebgp-multihop on same neighbor
```

### IXP Route Server Peering

```
router bgp 65001
 ! Route server at IXP (typically uses IXP's AS or private AS)
 neighbor 198.51.100.253 remote-as 65534
 neighbor 198.51.100.253 description IXP-RS1
 address-family ipv4 unicast
  ! Route server preserves original next-hop — must accept third-party next-hops
  neighbor 198.51.100.253 next-hop-self  ! Do NOT set this for RS peering
  ! Accept routes with any AS in path (RS passes through many ASes)
  neighbor 198.51.100.253 prefix-list BOGON-FILTER in
  neighbor 198.51.100.253 maximum-prefix 50000 90
 exit-address-family
```

### Prefix Limits and Filtering

```
! Maximum prefix — tear down session if exceeded
neighbor 198.51.100.2 maximum-prefix 1000 90 restart 15
! 1000 = max prefixes, 90 = warning at 90%, restart 15 = retry in 15 minutes

! Prefix-list for inbound filtering
ip prefix-list PEER-65002-IN seq 10 permit 203.0.113.0/24
ip prefix-list PEER-65002-IN seq 20 permit 198.18.0.0/15 le 24
ip prefix-list PEER-65002-IN seq 999 deny 0.0.0.0/0 le 32

! Always filter bogons inbound
ip prefix-list BOGON-FILTER seq 10 deny 0.0.0.0/8 le 32
ip prefix-list BOGON-FILTER seq 20 deny 10.0.0.0/8 le 32
ip prefix-list BOGON-FILTER seq 30 deny 127.0.0.0/8 le 32
ip prefix-list BOGON-FILTER seq 40 deny 169.254.0.0/16 le 32
ip prefix-list BOGON-FILTER seq 50 deny 172.16.0.0/12 le 32
ip prefix-list BOGON-FILTER seq 60 deny 192.168.0.0/16 le 32
ip prefix-list BOGON-FILTER seq 70 deny 224.0.0.0/4 le 32
ip prefix-list BOGON-FILTER seq 80 deny 240.0.0.0/4 le 32
ip prefix-list BOGON-FILTER seq 999 permit 0.0.0.0/0 le 24
```

## BGP Communities for Traffic Engineering

### Standard Communities (RFC 1997)

```
! Community format: ASN:VALUE (16-bit each)
! Common conventions:
!   PEER_AS:0     — do not advertise to PEER_AS
!   PEER_AS:ASN   — prepend once toward PEER_AS
!   0:PEER_AS     — blackhole at PEER_AS

route-map SET-COMMUNITY permit 10
 set community 65001:100 additive
 ! 65001:100 = learned from peer (local tagging)

route-map SET-COMMUNITY permit 20
 set community 65001:200 additive
 ! 65001:200 = learned from transit
```

### Large Communities (RFC 8092)

```
! Format: ASN:Function:Parameter (32-bit each)
! Needed when ASN > 65535 (4-byte ASes)

route-map TAG-ORIGIN permit 10
 set large-community 394500:1:1 additive
 ! 394500:1:1 = learned at IXP location 1

route-map TAG-ORIGIN permit 20
 set large-community 394500:2:100 additive
 ! 394500:2:100 = learned from transit provider in region 100
```

### Blackhole Communities (RFC 7999)

```
! Well-known blackhole community: 65535:666
! Signal upstream to drop traffic to a prefix (DDoS mitigation)

ip prefix-list BLACKHOLE-OK permit 203.0.113.0/24 ge 32 le 32
! Only allow /32 blackholes from your own space

route-map BLACKHOLE permit 10
 match ip address prefix-list BLACKHOLE-OK
 set community 65535:666
 set origin igp

! Announce a /32 to trigger blackhole upstream
network 203.0.113.100/32 route-map BLACKHOLE

! Upstream must accept /32s and match community 65535:666 to null-route
```

### Traffic Engineering via Communities

```
! Common transit provider community actions:
!   TRANSIT_AS:0        — do not advertise to any peer
!   TRANSIT_AS:PEER_AS  — do not advertise to PEER_AS
!   TRANSIT_AS:X00      — prepend X times to all peers
!   TRANSIT_AS:X0Y      — prepend X times to peer group Y

! Example: prepend 3x toward AS 174 (Cogent) via transit AS 3356 (Lumen)
route-map TO-LUMEN permit 10
 set community 3356:3174 additive
 ! Meaning: prepend 3 times toward AS 174

! Example: do not announce to AS 6939 (Hurricane Electric)
route-map TO-TRANSIT permit 10
 set community 3356:06939 additive
```

## RPKI and IRR

### RPKI/ROA (Resource Public Key Infrastructure)

```bash
# RPKI validates that an AS is authorized to originate a prefix
# ROA (Route Origin Authorization) = signed object binding prefix to AS

# Check ROA status using routinator
routinator vrps --format json | jq '.[] | select(.prefix == "203.0.113.0/24")'

# States:
#   Valid    — ROA exists and matches (prefix + origin AS)
#   Invalid  — ROA exists but does not match (REJECT these)
#   NotFound — no ROA exists (accept with lower preference)

# Configure RPKI validation in FRRouting
rpki
 rpki cache 10.0.0.100 3323 preference 1
 rpki cache 10.0.0.101 3323 preference 2
 exit

router bgp 65001
 address-family ipv4 unicast
  ! Drop RPKI-invalid routes
  neighbor 198.51.100.2 route-map RPKI-FILTER in

route-map RPKI-FILTER deny 10
 match rpki invalid

route-map RPKI-FILTER permit 20
 match rpki valid
 set local-preference 200

route-map RPKI-FILTER permit 30
 match rpki notfound
 set local-preference 100
```

### IRR (Internet Routing Registry)

```bash
# IRR databases: RADB, RIPE, ARIN, APNIC, AFRINIC, etc.
# Used to generate prefix-lists from registered route objects

# Query a route object
whois -h whois.radb.net 203.0.113.0/24

# Query an AS-SET (aggregated set of ASNs)
whois -h whois.radb.net AS-EXAMPLE

# Resolve AS-SET members recursively
whois -h whois.radb.net -i origin -T route AS65001
```

### bgpq4 — Automated Prefix-List Generation from IRR

```bash
# Install bgpq4
# macOS: brew install bgpq4
# Debian: apt install bgpq4

# Generate Cisco-style prefix-list from AS-SET
bgpq4 -4 -l PEER-65002-IN AS-EXAMPLE
# Output:
# ip prefix-list PEER-65002-IN permit 203.0.113.0/24
# ip prefix-list PEER-65002-IN permit 198.51.100.0/24

# Generate for Juniper (JunOS)
bgpq4 -4 -J -l PEER-65002-IN AS-EXAMPLE

# Generate for BIRD
bgpq4 -4 -b -l PEER-65002-IN AS-EXAMPLE

# IPv6 prefix-list
bgpq4 -6 -l PEER-65002-V6-IN AS-EXAMPLE

# Include more specific prefixes up to /24
bgpq4 -4 -R 24 -l PEER-65002-IN AS-EXAMPLE

# Generate AS-path filter (only allow these ASNs)
bgpq4 -4 -f 65002 -l AS-PATH-65002 AS-EXAMPLE

# Use specific IRR source
bgpq4 -4 -S RIPE,RADB -l PEER-IN AS-EXAMPLE

# Automate with cron — regenerate prefix-lists daily
# 0 4 * * * /usr/bin/bgpq4 -4 -l PEER-65002-IN AS-EXAMPLE > /etc/frr/prefix-65002.conf && vtysh -c "configure terminal" -c "$(cat /etc/frr/prefix-65002.conf)"
```

## Transit Provider Selection

### Key Evaluation Criteria

- **Path diversity:** How many upstream providers does the transit have? Single-homed transit is a risk
- **Geographic reach:** Points of presence near your users; latency to key destinations
- **Peering richness:** Number of settlement-free peers; check PeeringDB
- **Traffic engineering support:** What BGP communities do they offer?
- **SLA terms:** Uptime guarantee, packet loss, latency, jitter, MTTR
- **DDoS mitigation:** Blackhole community support, scrubbing services
- **Pricing model:** 95th percentile, committed rate, burstable, per-GB
- **RPKI support:** Do they validate and drop RPKI-invalid routes?

### BGP Looking Glass

```bash
# Looking glass — view routing table from a remote perspective
# Useful for verifying your announcements are visible

# Common looking glass tools
# Telnet-based (legacy):
telnet route-views.routeviews.org
# Then: show ip bgp 203.0.113.0/24

# Web-based:
# https://lg.he.net/ (Hurricane Electric)
# https://www.bgp4.as/looking-glasses

# RIPE RIS / RIPEstat
curl -s "https://stat.ripe.net/data/looking-glass/data.json?resource=203.0.113.0/24" | jq '.data.rrcs'

# Check if your prefix is visible globally
curl -s "https://stat.ripe.net/data/visibility/data.json?resource=203.0.113.0/24" | jq '.data.visibilities'

# BGPStream for real-time monitoring
# pip install pybgpstream
bgpstream -p routeviews -f "prefix exact 203.0.113.0/24"
```

### Multi-Homing with Transit

```
! Dual-transit setup with preference control
router bgp 65001
 ! Primary transit
 neighbor 10.1.0.1 remote-as 3356
 neighbor 10.1.0.1 description transit-lumen-primary
 ! Secondary transit
 neighbor 10.2.0.1 remote-as 174
 neighbor 10.2.0.1 description transit-cogent-secondary

 address-family ipv4 unicast
  network 203.0.113.0/24

  ! Prefer primary inbound — prepend on secondary
  neighbor 10.2.0.1 route-map PREPEND-OUT out

  ! Prefer primary outbound — set local-pref higher
  neighbor 10.1.0.1 route-map PRIMARY-IN in
  neighbor 10.2.0.1 route-map SECONDARY-IN in
 exit-address-family

route-map PRIMARY-IN permit 10
 set local-preference 200

route-map SECONDARY-IN permit 10
 set local-preference 100

route-map PREPEND-OUT permit 10
 set as-path prepend 65001 65001
```

## Operational Commands

```bash
# FRRouting / Quagga / Cisco-style
# Show all BGP peers and their state
vtysh -c "show bgp summary"

# Show received routes from a peer (requires soft-reconfiguration or add-path)
vtysh -c "show bgp ipv4 unicast neighbors 198.51.100.2 received-routes"

# Show advertised routes to a peer
vtysh -c "show bgp ipv4 unicast neighbors 198.51.100.2 advertised-routes"

# Show communities attached to a prefix
vtysh -c "show bgp ipv4 unicast 203.0.113.0/24"

# Soft reset (apply policy changes without tearing session)
vtysh -c "clear bgp ipv4 unicast 198.51.100.2 soft in"
vtysh -c "clear bgp ipv4 unicast 198.51.100.2 soft out"

# RPKI status
vtysh -c "show rpki prefix-table"
vtysh -c "show rpki cache-connection"
```

```bash
# BIRD
birdc show protocols all
birdc show route for 203.0.113.0/24 all
birdc show route export <peer_protocol_name>
birdc show route import <peer_protocol_name>
```

## Best Practices

- Always filter inbound from peers using IRR-generated prefix-lists; regenerate daily with bgpq4.
- Set `maximum-prefix` on every eBGP session — both peers and transit; a route leak can push 900k+ prefixes.
- Use RPKI validation and drop invalid routes; prefer valid over not-found with local-preference.
- Deploy TTL security (GTSM) on all directly connected eBGP sessions to prevent spoofed TCP resets.
- Tag routes with communities at ingress to indicate origin (peer vs transit, location, IXP); makes traffic engineering systematic.
- Use RFC 7999 blackhole communities for DDoS mitigation; coordinate with upstreams in advance and test during maintenance.
- Peer at multiple IXPs for redundancy; a single IXP failure should not partition your peering.
- Maintain accurate PeeringDB records; other networks use this to evaluate peering requests.
- For PNI (private interconnect), monitor utilization and upgrade proactively; congested PNIs degrade worse than transit.
- Document community meanings in a public community guide; transit customers and peers need to know your scheme.
- Automate prefix-list generation from IRR data; manual prefix-lists go stale and cause reachability issues.
- Monitor RPKI ROA coverage for your own prefixes; create ROAs in your RIR portal for every announcement.

## See Also

- bgp, bgp-advanced, mpls, ipv4, ipv6

## References

- [RFC 7999 — Blackhole Community (RTBH)](https://www.rfc-editor.org/rfc/rfc7999)
- [RFC 1997 — BGP Communities Attribute](https://www.rfc-editor.org/rfc/rfc1997)
- [RFC 8092 — BGP Large Communities](https://www.rfc-editor.org/rfc/rfc8092)
- [RFC 5082 — Generalized TTL Security Mechanism (GTSM)](https://www.rfc-editor.org/rfc/rfc5082)
- [RFC 6811 — BGP Prefix Origin Validation (RPKI)](https://www.rfc-editor.org/rfc/rfc6811)
- [RFC 7454 — BGP Operations and Security (BCP 194)](https://www.rfc-editor.org/rfc/rfc7454)
- [PeeringDB](https://www.peeringdb.com/)
- [Euro-IX — European Internet Exchange Association](https://www.euro-ix.net/)
- [bgpq4 — BGP Filter Generator](https://github.com/bgp/bgpq4)
- [RIPE RIPEstat — BGP Routing Data](https://stat.ripe.net/)
- [Packet Clearing House — IXP Directory](https://www.pch.net/ixp/dir)
