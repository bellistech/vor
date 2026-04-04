# BFD (Bidirectional Forwarding Detection)

Lightweight protocol for sub-second failure detection on forwarding paths, designed to work alongside routing protocols like BGP, OSPF, and IS-IS rather than replace their native keepalive mechanisms.

## Concepts

### Operating Modes

- **Asynchronous mode:** Both peers send periodic BFD control packets; failure detected when packets stop arriving (the standard mode)
- **Echo mode:** A peer sends echo packets that are looped back by the remote end without protocol processing; allows very fast detection with reduced CPU load on the remote side
- **Demand mode:** BFD packets only sent when explicitly requested (rarely used in practice)

### Timers

- **Transmit interval:** How often BFD control packets are sent
- **Receive interval:** Minimum interval at which a peer expects to receive packets
- **Detect multiplier:** Number of missed packets before declaring session down (e.g., 3 missed = session down)
- **Detection time = receive-interval x detect-multiplier** (e.g., 300ms x 3 = 900ms)

### Session Types

- **Single-hop:** Directly connected peers (TTL=255 for security)
- **Multi-hop:** Peers separated by multiple hops (for eBGP multihop, loopback peerings)
- **Micro-BFD:** Per-member-link BFD on LAG/bond interfaces; if enough members fail, the aggregate is brought down

## FRRouting Configuration

### Basic BFD Peer

```
bfd
 # Define a BFD peer
 peer 10.1.1.2
  # Detect multiplier: session down after 3 missed packets
  detect-multiplier 3
  # Minimum receive interval (milliseconds)
  receive-interval 300
  # Minimum transmit interval (milliseconds)
  transmit-interval 300
  # Echo mode (optional, reduces remote CPU usage)
  echo-mode
  # Echo interval (milliseconds)
  echo-interval 50
 exit

 # Multi-hop BFD peer (for loopback-to-loopback peering)
 peer 10.0.0.2 multihop local-address 10.0.0.1
  detect-multiplier 3
  receive-interval 300
  transmit-interval 300
 exit
```

### BFD Profiles

```
bfd
 # Define a reusable profile
 profile FAST-DETECT
  detect-multiplier 3
  receive-interval 100
  transmit-interval 100
 exit

 profile RELAXED
  detect-multiplier 5
  receive-interval 1000
  transmit-interval 1000
 exit

 # Apply profile to a peer
 peer 10.1.1.2
  profile FAST-DETECT
 exit
```

## Integration with Routing Protocols

### BGP + BFD

```
router bgp 65001
 # Enable BFD for a specific neighbor
 neighbor 10.1.1.2 bfd
 # With a specific BFD profile
 neighbor 10.1.1.2 bfd profile FAST-DETECT
 # Check-control-plane-failure: bring down BGP if BFD detects control plane issue
 neighbor 10.1.1.2 bfd check-control-plane-failure
```

### OSPF + BFD

```
interface eth0
 # Enable BFD on OSPF interface
 ip ospf bfd
 # With BFD profile
 ip ospf bfd profile FAST-DETECT
```

### IS-IS + BFD

```
interface eth0
 # Enable BFD on IS-IS interface
 isis bfd
 # With BFD profile
 isis bfd profile FAST-DETECT
```

### Static Routes + BFD

```
# Monitor next-hop reachability for static routes
ip route 10.10.0.0/16 10.1.1.2 bfd

# BFD peer must also be configured for the next-hop
bfd
 peer 10.1.1.2
  detect-multiplier 3
  receive-interval 300
  transmit-interval 300
 exit
```

## Cisco IOS Equivalents

```
! Global BFD template
bfd-template single-hop FAST-DETECT
 interval min-tx 100 min-rx 100 multiplier 3

! BFD on interface (for OSPF, IS-IS)
interface GigabitEthernet0/0
 bfd interval 300 min_rx 300 multiplier 3

! BGP with BFD
router bgp 65001
 neighbor 10.1.1.2 fall-over bfd

! OSPF with BFD
router ospf 1
 bfd all-interfaces

! IS-IS with BFD
router isis
 bfd all-interfaces

! Static route with BFD
ip route static bfd GigabitEthernet0/0 10.1.1.2
ip route 10.10.0.0 255.255.0.0 10.1.1.2
```

## Show Commands

```bash
# All BFD peers and their state
vtysh -c "show bfd peers"

# Detailed peer info (timers, counters, uptime)
vtysh -c "show bfd peers detail"

# BFD peer counters (control packet stats)
vtysh -c "show bfd peer counters"

# BFD peers in brief format
vtysh -c "show bfd peers brief"

# Specific peer
vtysh -c "show bfd peers peer 10.1.1.2"

# Check which protocols are registered with BFD
vtysh -c "show bfd peers detail" | grep -i "client"
```

## Linux Kernel BFD

```bash
# Linux kernel does not natively implement BFD
# Options for BFD on Linux:
# 1. FRRouting (recommended) — full BFD with protocol integration
# 2. OpenBFDD — standalone BFD daemon
# 3. Some NIC offloads support hardware BFD (vendor-specific)

# Verify FRRouting bfdd is running
systemctl status frr
# bfdd should be listed in the FRR daemons
grep "bfdd" /etc/frr/daemons
```

```
# /etc/frr/daemons — enable BFD daemon
bfdd=yes
```

## Troubleshooting

### Session Flapping

```bash
# Check for timer mismatch (negotiate should handle this, but verify)
vtysh -c "show bfd peers detail" | grep -i "interval\|multiplier"

# Look for packet loss on the link causing sporadic detection
vtysh -c "show bfd peer counters" | grep -i "drop\|error"

# Increase detect-multiplier to tolerate transient packet loss
# multiplier 5 with 300ms interval = 1.5s detection (more stable)

# Check CPU load — BFD packets can be delayed under heavy CPU
# Consider hardware-offloaded BFD if available
```

### Session Not Coming Up

```bash
# Verify peer IP is correct and reachable
ping 10.1.1.2

# Check that BFD is configured on BOTH sides
# BFD requires bilateral configuration

# Verify bfdd daemon is running
systemctl status frr | grep bfdd

# For multihop BFD: check that local-address is correct
# and that TTL security is not blocking (TTL must be >= 254 for single-hop)
```

### TTL Security

```bash
# BFD single-hop uses TTL=255 by default (GTSM: Generalized TTL Security)
# Packets with TTL < 255 are dropped to prevent remote spoofing
# Multi-hop BFD has configurable minimum TTL

# If BFD packets are being filtered, check:
iptables -L -n | grep -i "ttl\|bfd"

# BFD uses UDP port 3784 (single-hop) and 4784 (multi-hop)
# Verify firewall allows these ports
iptables -L -n | grep -E "3784|4784"
```

## Tips

- BFD provides fast failure detection; routing protocol convergence is a separate (and usually slower) step after BFD triggers.
- Start with conservative timers (300ms x 3 = 900ms) and tune down only after confirming stability; aggressive timers on lossy links cause flapping.
- Echo mode reduces CPU on the remote peer because echo packets are looped in the forwarding plane, not processed by the control plane.
- Always enable BFD on iBGP sessions over loopback using multi-hop BFD; single-hop BFD only works for directly connected peers.
- Micro-BFD on LAGs detects per-member failures; without it, a partially failed LAG may black-hole traffic on the dead member.
- BFD detection time should be faster than the routing protocol's dead timer to be useful (e.g., BFD 900ms vs OSPF dead 40s).
- On hardware platforms, offload BFD to the ASIC when possible; software-based BFD competes with other control-plane processes for CPU.
- BFD sessions consume memory and CPU proportional to the number of peers and timer aggressiveness; do not enable on hundreds of peers with 50ms timers without testing.
- UDP ports 3784 (single-hop) and 4784 (multi-hop) must be permitted through any intermediate firewalls or ACLs.
- When BFD brings down a BGP session, the BGP hold timer still applies for the actual session teardown; BFD just accelerates the detection, not the BGP state machine itself.

## See Also

- bgp, ospf, is-is, ecmp, ip

## References

- [RFC 5880 — Bidirectional Forwarding Detection (BFD)](https://www.rfc-editor.org/rfc/rfc5880)
- [RFC 5881 — BFD for IPv4 and IPv6 (Single Hop)](https://www.rfc-editor.org/rfc/rfc5881)
- [RFC 5882 — Generic Application of BFD](https://www.rfc-editor.org/rfc/rfc5882)
- [RFC 5883 — BFD for Multihop Paths](https://www.rfc-editor.org/rfc/rfc5883)
- [RFC 7130 — BFD on Link Aggregation Group (LAG) Interfaces](https://www.rfc-editor.org/rfc/rfc7130)
- [FRRouting BFD Documentation](https://docs.frrouting.org/en/latest/bfd.html)
- [Cisco IOS BFD Configuration Guide](https://www.cisco.com/c/en/us/td/docs/ios-xml/ios/iproute_bfd/configuration/xe-16/irb-xe-16-book.html)
- [Juniper BFD Configuration Guide](https://www.juniper.net/documentation/us/en/software/junos/high-availability/topics/topic-map/bfd.html)
- [Arista EOS BFD Configuration Guide](https://www.arista.com/en/um-eos/eos-bidirectional-forwarding-detection)
- [BIRD Internet Routing Daemon — BFD](https://bird.network.cz/?get_doc&v=20&f=bird-6.html)
