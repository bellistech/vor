# NDP (Neighbor Discovery Protocol for IPv6)

Neighbor Discovery Protocol replaces ARP, ICMP Router Discovery, and ICMP Redirect in IPv6. Defined in RFC 4861, NDP uses ICMPv6 message types 133-137 to perform router discovery, prefix discovery, parameter discovery, address resolution, next-hop determination, neighbor unreachability detection (NUD), and duplicate address detection (DAD). All NDP messages are sent with a hop limit of 255 and link-local source addresses, providing basic on-link verification.

---

## Message Types

### Router Solicitation (RS) — ICMPv6 Type 133

```bash
# Sent by hosts at boot to discover routers immediately (instead of waiting for periodic RA)
# Source: host link-local (or :: during DAD)
# Destination: ff02::2 (all-routers multicast)

# Send RS manually with rdisc6 (from ndisc6 package)
rdisc6 eth0

# Send RS and show full decoded output
rdisc6 -1 -v eth0

# Capture RS traffic with tcpdump
sudo tcpdump -i eth0 -n 'icmp6 and ip6[40] == 133'
```

### Router Advertisement (RA) — ICMPv6 Type 134

```bash
# Sent periodically by routers (or in response to RS)
# Contains: prefixes, MTU, hop limit, M/O flags, router lifetime, reachable time
# Source: router link-local
# Destination: ff02::1 (all-nodes multicast)

# Listen for RA on an interface
rdisc6 eth0

# Decode RA fields in detail
rdisc6 -1 -v eth0
# Shows: hop limit, M/O flags, router lifetime, reachable time, retrans timer
# Shows: prefix info (prefix, length, L/A flags, valid/preferred lifetime)
# Shows: MTU option, source link-layer address option

# Capture RA traffic
sudo tcpdump -i eth0 -n -vv 'icmp6 and ip6[40] == 134'

# RA flags that control address configuration:
# M (Managed) flag = 1 → use DHCPv6 for addresses
# O (Other)   flag = 1 → use DHCPv6 for DNS/options only
# A (Autonomous) flag in prefix option → use SLAAC for this prefix
```

### Neighbor Solicitation (NS) — ICMPv6 Type 135

```bash
# IPv6 equivalent of ARP request
# Sent to solicited-node multicast address (ff02::1:ffXX:XXXX)
# Used for: address resolution, DAD, NUD probes

# Resolve a neighbor address manually (ndisc6 package)
ndisc6 2001:db8::1 eth0

# Resolve link-local address
ndisc6 fe80::1 eth0

# Verbose output showing NS/NA exchange
ndisc6 -v 2001:db8::1 eth0

# Capture NS traffic
sudo tcpdump -i eth0 -n 'icmp6 and ip6[40] == 135'
```

### Neighbor Advertisement (NA) — ICMPv6 Type 136

```bash
# Response to NS (or unsolicited to announce changes)
# Contains: target link-layer address, flags (R/S/O)
# R = Router flag — sender is a router
# S = Solicited flag — response to NS (vs unsolicited)
# O = Override flag — update existing cache entry

# Capture NA traffic
sudo tcpdump -i eth0 -n 'icmp6 and ip6[40] == 136'

# Watch for unsolicited NA (gratuitous, like gratuitous ARP)
sudo tcpdump -i eth0 -n 'icmp6 and ip6[40] == 136' -vv
```

### Redirect — ICMPv6 Type 137

```bash
# Sent by routers to inform hosts of a better first-hop for a destination
# Similar to ICMP Redirect in IPv4 but more tightly specified

# Capture Redirect messages
sudo tcpdump -i eth0 -n 'icmp6 and ip6[40] == 137'

# Check if host accepts redirects
sysctl net.ipv6.conf.eth0.accept_redirects
# 1 = accept (default for hosts), 0 = reject (recommended for routers)

# Disable redirect acceptance (security hardening)
sudo sysctl -w net.ipv6.conf.all.accept_redirects=0

# Disable sending redirects (on routers, if not desired)
sudo sysctl -w net.ipv6.conf.all.send_redirects=0
```

## Neighbor Cache Management

### Viewing the Neighbor Cache

```bash
# Show the full IPv6 neighbor cache (equivalent to ARP table)
ip -6 neigh show

# Output format: <address> dev <iface> lladdr <mac> <state>
# States: REACHABLE, STALE, DELAY, PROBE, FAILED, INCOMPLETE, NOARP, PERMANENT

# Show neighbors on a specific interface
ip -6 neigh show dev eth0

# Show only reachable neighbors
ip -6 neigh show nud reachable

# Show failed resolutions
ip -6 neigh show nud failed

# Monitor neighbor cache changes in real time
ip monitor neigh
```

### Modifying the Neighbor Cache

```bash
# Add a static neighbor entry (permanent)
sudo ip -6 neigh add 2001:db8::1 lladdr 00:1a:2b:3c:4d:5e dev eth0

# Replace an existing entry (add or update)
sudo ip -6 neigh replace 2001:db8::1 lladdr 00:1a:2b:3c:4d:5e dev eth0

# Change state of existing entry
sudo ip -6 neigh change 2001:db8::1 lladdr 00:1a:2b:3c:4d:5e dev eth0 nud permanent

# Delete a specific neighbor entry
sudo ip -6 neigh del 2001:db8::1 dev eth0

# Flush all neighbor entries on an interface
sudo ip -6 neigh flush dev eth0

# Flush only stale entries
sudo ip -6 neigh flush nud stale dev eth0
```

## Solicited-Node Multicast

### Address Derivation

```bash
# Each unicast/anycast address maps to a solicited-node multicast group
# Formed from: ff02::1:ff00:0/104 + last 24 bits of the address
# This limits multicast to hosts sharing the same last 24 bits

# Example: 2001:db8::1a2b:3cff:fe4d:5678
# Last 24 bits: 4d:56:78
# Solicited-node: ff02::1:ff4d:5678

# Each interface joins solicited-node groups for all its addresses
# View multicast group memberships
ip -6 maddr show dev eth0

# Efficiency: NS is sent to solicited-node multicast, not broadcast
# Only hosts with matching last-24-bits process the packet
# On a /64 with 1000 hosts, typically only 1 host processes each NS
```

## Duplicate Address Detection (DAD)

### How DAD Works

```bash
# Before using any unicast address, host sends NS for its own address
# Source: :: (unspecified)
# Destination: solicited-node multicast of the tentative address
# If NA received → address is duplicate → marked "dadfailed"

# Check for DAD failures
ip -6 addr show dadfailed

# View tentative addresses (DAD in progress)
ip -6 addr show tentative

# Number of DAD probes sent before declaring success (default: 1)
sysctl net.ipv6.conf.eth0.dad_transmits

# Increase DAD probes for higher confidence
sudo sysctl -w net.ipv6.conf.eth0.dad_transmits=3

# Disable DAD entirely (not recommended in production)
sudo sysctl -w net.ipv6.conf.eth0.dad_transmits=0

# Enhanced DAD (RFC 7527) — optimistic DAD for faster address use
sysctl net.ipv6.conf.eth0.optimistic_dad
sudo sysctl -w net.ipv6.conf.eth0.optimistic_dad=1
```

## Neighbor Unreachability Detection (NUD)

### NUD States and Transitions

```bash
# NUD ensures neighbors are still reachable after initial resolution
# State machine: INCOMPLETE → REACHABLE → STALE → DELAY → PROBE → FAILED

# REACHABLE: confirmed reachable (upper-layer hint or NUD probe reply)
# STALE: reachability not recently confirmed (still usable, starts NUD on traffic)
# DELAY: traffic sent, waiting before probing (5 seconds default)
# PROBE: sending unicast NS probes (up to 3, then FAILED)

# Tune reachable time (base, in milliseconds — actual is randomized 0.5x-1.5x)
sysctl net.ipv6.conf.eth0.base_reachable_time_ms
# Default: 30000 (30 seconds)

# Change reachable time
sudo sysctl -w net.ipv6.conf.eth0.base_reachable_time_ms=20000

# Retransmission timer for NS probes (milliseconds)
sysctl net.ipv6.conf.eth0.retrans_time_ms
# Default: 1000 (1 second)

# Number of unicast NS probes before declaring FAILED
# Controlled by: net.ipv6.neigh.eth0.ucast_solicit (default: 3)
sysctl net.ipv6.neigh.eth0.ucast_solicit
```

## Router Configuration (radvd)

### Basic radvd Setup

```bash
# Install radvd — the Router Advertisement Daemon
sudo apt install radvd     # Debian/Ubuntu
sudo dnf install radvd     # RHEL/Fedora

# /etc/radvd.conf — minimal configuration
# interface eth0
# {
#     AdvSendAdvert on;
#     MinRtrAdvInterval 3;
#     MaxRtrAdvInterval 10;
#     AdvManagedFlag off;         # M flag — off means use SLAAC
#     AdvOtherConfigFlag on;      # O flag — on means use DHCPv6 for DNS
#
#     prefix 2001:db8:1:1::/64
#     {
#         AdvOnLink on;           # L flag — prefix is on-link
#         AdvAutonomous on;       # A flag — use for SLAAC
#         AdvPreferredLifetime 600;
#         AdvValidLifetime 1200;
#     };
#
#     RDNSS 2001:db8:1:1::53
#     {
#         AdvRDNSSLifetime 1200;
#     };
# };

# Check config syntax
radvd -c

# Start radvd
sudo systemctl start radvd
sudo systemctl enable radvd

# Debug mode (foreground, verbose)
sudo radvd -n -d 5 -m stderr
```

### RA Guard (Layer 2 Protection)

```bash
# RA Guard prevents rogue Router Advertisements on the network
# Implemented on managed switches (Cisco, Juniper, etc.)

# Linux bridge — block RAs from non-router ports using ebtables
sudo ebtables -A FORWARD -i eth1 -p IPv6 --ip6-proto ipv6-icmp \
  --ip6-icmp-type 134 -j DROP

# nftables — drop RAs from untrusted interfaces
sudo nft add rule bridge filter forward \
  iifname "eth1" icmpv6 type nd-router-advert drop

# Verify RA Guard is working — should see no RAs from blocked ports
sudo tcpdump -i br0 -n 'icmp6 and ip6[40] == 134'
```

## Secure Neighbor Discovery (SEND)

### SEND Overview (RFC 3971)

```bash
# SEND adds cryptographic protection to NDP messages
# Uses CGA (Cryptographically Generated Addresses) and RSA signatures
# Protects against: spoofed NS/NA, rogue RA, NDP redirect attacks

# SEND components:
# 1. CGA — address derived from public key hash (proves address ownership)
# 2. RSA Signature option — signs NDP messages with sender's private key
# 3. Timestamp + Nonce — replay protection
# 4. Authorization Delegation Discovery — validates router authority

# Generate a CGA keypair (using cga-gen from the send-tools package)
# Note: SEND has limited real-world deployment
# Most networks rely on RA Guard + MLD snooping instead

# Check kernel SEND support
grep -r SEND /boot/config-$(uname -r) 2>/dev/null
```

## Monitoring and Troubleshooting

### Comprehensive NDP Monitoring

```bash
# Monitor all NDP traffic on an interface
sudo tcpdump -i eth0 -n icmp6 -vv

# Filter by specific NDP message type
sudo tcpdump -i eth0 -n 'icmp6 and ip6[40] == 133'   # RS
sudo tcpdump -i eth0 -n 'icmp6 and ip6[40] == 134'   # RA
sudo tcpdump -i eth0 -n 'icmp6 and ip6[40] == 135'   # NS
sudo tcpdump -i eth0 -n 'icmp6 and ip6[40] == 136'   # NA
sudo tcpdump -i eth0 -n 'icmp6 and ip6[40] == 137'   # Redirect

# Watch neighbor cache changes live
ip -s -6 neigh show

# Monitor NDP events with ip monitor
ip monitor neigh

# Count NDP packets per type over 60 seconds
timeout 60 tcpdump -i eth0 -n icmp6 2>/dev/null | \
  awk '/ICMP6, router solicitation/{rs++}
       /ICMP6, router advertisement/{ra++}
       /ICMP6, neighbor solicitation/{ns++}
       /ICMP6, neighbor advertisement/{na++}
       END{printf "RS:%d RA:%d NS:%d NA:%d\n",rs,ra,ns,na}'

# Check for NDP table overflow
ip -6 -s neigh show | grep -c "FAILED"

# View NDP-related kernel statistics
cat /proc/net/snmp6 | grep -i "Icmp6InNeighbor\|Icmp6OutNeighbor\|Icmp6InRouter\|Icmp6OutRouter"
```

### NDP Cache Tuning

```bash
# Maximum neighbor cache entries (prevent table overflow on busy networks)
sysctl net.ipv6.neigh.eth0.gc_thresh1    # min entries before GC starts (default: 128)
sysctl net.ipv6.neigh.eth0.gc_thresh2    # soft max (default: 512)
sysctl net.ipv6.neigh.eth0.gc_thresh3    # hard max (default: 1024)

# Increase for large subnets (e.g., data center /64 with thousands of hosts)
sudo sysctl -w net.ipv6.neigh.default.gc_thresh1=1024
sudo sysctl -w net.ipv6.neigh.default.gc_thresh2=4096
sudo sysctl -w net.ipv6.neigh.default.gc_thresh3=8192

# GC interval (how often stale entries are swept, in seconds)
sysctl net.ipv6.neigh.eth0.gc_interval

# Stale entry timeout (seconds before STALE entry is eligible for GC)
sysctl net.ipv6.neigh.eth0.gc_stale_time
```

## NDP and Firewalls

### Essential NDP Firewall Rules

```bash
# CRITICAL: Never block NDP in IPv6 firewalls — connectivity depends on it
# Minimum ICMPv6 types to allow for NDP:

# ip6tables — allow all NDP message types
sudo ip6tables -A INPUT -p icmpv6 --icmpv6-type router-solicitation -j ACCEPT
sudo ip6tables -A INPUT -p icmpv6 --icmpv6-type router-advertisement -j ACCEPT
sudo ip6tables -A INPUT -p icmpv6 --icmpv6-type neighbour-solicitation -j ACCEPT
sudo ip6tables -A INPUT -p icmpv6 --icmpv6-type neighbour-advertisement -j ACCEPT
sudo ip6tables -A INPUT -p icmpv6 --icmpv6-type redirect -j ACCEPT

# nftables — allow NDP (cleaner syntax)
sudo nft add rule inet filter input icmpv6 type {
  nd-router-solicit, nd-router-advert,
  nd-neighbor-solicit, nd-neighbor-advert,
  nd-redirect
} accept

# Also allow MLD (Multicast Listener Discovery) for proper multicast operation
sudo ip6tables -A INPUT -p icmpv6 --icmpv6-type 130 -j ACCEPT  # MLDv1 Query
sudo ip6tables -A INPUT -p icmpv6 --icmpv6-type 131 -j ACCEPT  # MLDv1 Report
sudo ip6tables -A INPUT -p icmpv6 --icmpv6-type 132 -j ACCEPT  # MLDv1 Done
sudo ip6tables -A INPUT -p icmpv6 --icmpv6-type 143 -j ACCEPT  # MLDv2 Report
```

---

## Tips

- All NDP messages must have hop limit 255. If a received NDP packet has any other hop limit, it was routed (not on-link) and must be discarded. This is the fundamental NDP security property.
- Never block ICMPv6 types 133-137 in your firewall. Unlike IPv4 where blocking ICMP is sometimes acceptable, IPv6 connectivity completely breaks without NDP.
- The solicited-node multicast optimization means NDP scales far better than ARP broadcast on large subnets. Only hosts with matching last-24-bit addresses process each NS.
- If `ip -6 neigh show` shows many FAILED entries, your neighbor cache thresholds (gc_thresh) may be too low for the subnet size. Increase them.
- Run `ip monitor neigh` to watch cache transitions in real time during troubleshooting. Rapid REACHABLE-to-STALE-to-PROBE cycling indicates an intermittent link.
- radvd is the standard Linux RA daemon. Always run `radvd -c` to validate config before restarting — a syntax error kills the daemon silently.
- SEND (RFC 3971) provides cryptographic NDP protection but has minimal real-world deployment. Use RA Guard on managed switches and MLD snooping as practical alternatives.
- For data centers with thousands of hosts per /64, tune gc_thresh3 to at least 8192 to prevent neighbor cache overflow and connectivity drops.
- DAD failures (shown by `ip -6 addr show dadfailed`) usually indicate a duplicate static address or a misconfigured VM/container with a cloned MAC.
- The NUD state machine ensures stale entries are re-verified before causing packet loss. If you see unexplained drops, check that NUD probes are not being blocked by a middlebox.

---

## See Also

- ipv6, slaac, dhcpv6, icmp, arp

## References

- [RFC 4861 — Neighbor Discovery for IP version 6](https://www.rfc-editor.org/rfc/rfc4861)
- [RFC 4862 — IPv6 Stateless Address Autoconfiguration](https://www.rfc-editor.org/rfc/rfc4862)
- [RFC 3971 — SEcure Neighbor Discovery (SEND)](https://www.rfc-editor.org/rfc/rfc3971)
- [RFC 7527 — Enhanced Duplicate Address Detection](https://www.rfc-editor.org/rfc/rfc7527)
- [RFC 6775 — Neighbor Discovery Optimization for Low-Power and Lossy Networks](https://www.rfc-editor.org/rfc/rfc6775)
- [RFC 5765 — Security Implications of IPv6 on IPv4 Networks](https://www.rfc-editor.org/rfc/rfc5765)
- [Linux Kernel — IPv6 Neighbor Discovery Sysctl](https://www.kernel.org/doc/html/latest/networking/ip-sysctl.html)
- [ndisc6 — IPv6 Diagnostic Tools](https://www.remlab.net/ndisc6/)
- [radvd — Router Advertisement Daemon](https://radvd.litech.org/)
