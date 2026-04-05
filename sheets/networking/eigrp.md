# EIGRP (Enhanced Interior Gateway Routing Protocol)

Advanced distance vector (hybrid) routing protocol using DUAL algorithm for loop-free, fast-converging routing within an autonomous system.

## Concepts

### DUAL Algorithm (Diffusing Update Algorithm)

- **Feasible distance (FD):** Lowest computed metric to a destination from this router
- **Reported distance (RD):** Metric to a destination as advertised by a neighbor (also called advertised distance)
- **Successor:** Neighbor with the lowest cost path to a destination; installed in the routing table
- **Feasible successor (FS):** Backup next-hop that satisfies the feasibility condition
- **Feasibility condition:** A neighbor qualifies as FS if its reported distance < this router's current feasible distance
- **Passive state:** Route is stable; no recomputation needed
- **Active state:** Route lost its successor and no feasible successor exists; queries sent to neighbors
- **Stuck-in-active (SIA):** Active query received no reply within 3 minutes (default); neighbor is killed

### Composite Metric

```
# Classic metric formula (by default only K1 and K3 are set to 1)
Metric = 256 * [ (K1 * BW) + (K2 * BW) / (256 - Load) + (K3 * Delay) ]
         * [ K5 / (Reliability + K4) ]

# When K5 = 0 (default), the K5 term is ignored:
Metric = 256 * [ (K1 * BW) + (K3 * Delay) ]

# Where:
#   BW    = 10^7 / minimum bandwidth in kbps along the path
#   Delay = sum of delays in tens of microseconds along the path
#   K1=1, K2=0, K3=1, K4=0, K5=0  (defaults)
```

### Wide Metrics (Named Mode)

- Classic metric is 32-bit; tops out at ~4.29 billion, problematic for high-speed interfaces
- Named mode uses 64-bit wide metrics with a `rib-scale` factor
- **Throughput:** `(10^7 * 65536) / bandwidth-in-kbps` (scaled by 65536)
- **Latency:** `(delay-in-picoseconds * 65536) / 10^7`
- Wide metrics avoid metric rollover on 10G/40G/100G interfaces
- Backward compatible: scaled down for classic peers via `rib-scale`

### EIGRP Packet Types

| Type | Name   | Purpose                                      | Reliable |
|------|--------|----------------------------------------------|----------|
| 1    | Hello  | Neighbor discovery, keepalive                | No       |
| 2    | Update | Route information (topology changes)         | Yes      |
| 3    | Query  | Ask neighbors for a route (active state)     | Yes      |
| 4    | Reply  | Response to a query                          | Yes      |
| 5    | ACK    | Acknowledges reliable packets                | No       |

### Neighbor Adjacency

- Hello packets multicast to `224.0.0.10` (IPv4) or `ff02::a` (IPv6)
- Default hello/hold timers: 5/15 seconds (high-speed links), 60/180 seconds (low-speed NBMA)
- Neighbors must match: AS number, K-values, authentication, primary subnet
- RTP (Reliable Transport Protocol) ensures delivery of Update/Query/Reply packets
- Conditional receive (CR) mode for multicast flow control

### Stub Routing

- Stub routers advertise limited routes and are excluded from query scope
- Reduces SIA risk by limiting the query domain

| Stub Mode       | Advertised Routes                              |
|------------------|-------------------------------------------------|
| `receive-only`   | No routes advertised (only receives)            |
| `connected`      | Directly connected networks (default)           |
| `static`         | Statically configured routes                    |
| `summary`        | Summary routes (default)                        |
| `redistributed`  | Routes redistributed into EIGRP                 |

- Default stub configuration: `eigrp stub connected summary`

## Cisco IOS Configuration

### Classic Mode

```
router eigrp 100
 ! Explicitly set router-id
 eigrp router-id 10.0.0.1
 ! Advertise networks
 network 10.0.0.0 0.0.0.255
 network 10.1.0.0 0.0.0.255
 network 172.16.0.0 0.0.255.255
 ! Suppress hellos on LAN-facing interfaces
 passive-interface GigabitEthernet0/2
 passive-interface default
 no passive-interface GigabitEthernet0/0
 no passive-interface GigabitEthernet0/1
 ! Disable auto-summary (disabled by default since IOS 15)
 no auto-summary
 ! Set K-values (do not change unless you have a specific reason)
 metric weights 0 1 0 1 0 0
 ! Redistribute static routes
 redistribute static metric 10000 100 255 1 1500
 ! Variance for unequal-cost load balancing
 variance 2
 ! Limit maximum paths for ECMP
 maximum-paths 4
 ! Stub configuration
 eigrp stub connected summary
```

### Named Mode (Modern)

```
router eigrp MYNET
 !
 address-family ipv4 unicast autonomous-system 100
  !
  af-interface default
   passive-interface
   hello-interval 5
   hold-time 15
  exit-af-interface
  !
  af-interface GigabitEthernet0/0
   no passive-interface
   authentication mode hmac-sha-256 EIGRP_KEY
  exit-af-interface
  !
  af-interface GigabitEthernet0/1
   no passive-interface
  exit-af-interface
  !
  topology base
   variance 2
   maximum-paths 4
   redistribute static metric 10000 100 255 1 1500
   ! Route summarization at topology level
   summary-metric 10.0.0.0/8 distance 90
  exit-af-topology
  !
  network 10.0.0.0 0.0.0.255
  network 10.1.0.0 0.0.0.255
  eigrp router-id 10.0.0.1
 exit-address-family
 !
 address-family ipv6 unicast autonomous-system 100
  !
  af-interface default
   passive-interface
   shutdown
  exit-af-interface
  !
  af-interface GigabitEthernet0/0
   no passive-interface
   no shutdown
  exit-af-interface
  !
  topology base
  exit-af-topology
  !
  eigrp router-id 10.0.0.1
 exit-address-family
```

### Interface-Level Configuration

```
interface GigabitEthernet0/0
 ip address 10.0.0.1 255.255.255.0
 ! Adjust hello and hold timers
 ip hello-interval eigrp 100 5
 ip hold-time eigrp 100 15
 ! Set bandwidth for metric calculation (does not affect interface speed)
 bandwidth 1000000
 ! Set delay for metric calculation (in tens of microseconds)
 delay 10
 ! Authentication (classic mode, MD5)
 ip authentication mode eigrp 100 md5
 ip authentication key-chain eigrp 100 EIGRP_KEYS
 ! Summarize routes out this interface
 ip summary-address eigrp 100 10.0.0.0 255.0.0.0
```

### Authentication

```
! MD5 authentication (classic mode)
key chain EIGRP_KEYS
 key 1
  key-string S3cur3Pa$$w0rd
  accept-lifetime 00:00:00 Jan 1 2025 infinite
  send-lifetime 00:00:00 Jan 1 2025 infinite

interface GigabitEthernet0/0
 ip authentication mode eigrp 100 md5
 ip authentication key-chain eigrp 100 EIGRP_KEYS

! SHA-256 authentication (named mode only)
router eigrp MYNET
 address-family ipv4 unicast autonomous-system 100
  af-interface GigabitEthernet0/0
   authentication mode hmac-sha-256 MySecretPassword
  exit-af-interface
```

### Route Summarization

```
! Interface-level summary (classic mode)
interface GigabitEthernet0/0
 ip summary-address eigrp 100 10.10.0.0 255.255.0.0

! Topology-level summary (named mode)
router eigrp MYNET
 address-family ipv4 unicast autonomous-system 100
  topology base
   summary-metric 10.10.0.0/16 distance 90
  exit-af-topology

! Summary routes create a Null0 discard route to prevent loops
! Check with:
show ip route eigrp | include Null0
```

### Unequal-Cost Load Balancing

```
! Variance multiplier: include routes with metric up to (variance * best metric)
router eigrp 100
 variance 2
 ! Only feasible successors are eligible (feasibility condition must be met)
 ! Traffic share is proportional to metric by default
 traffic-share balanced
 ! Or use min to send all traffic via the best path but install backups in RIB
 traffic-share min across-interfaces

! Example:
! Successor metric     = 1000
! Feasible successor 1 = 1500  (1500 < 2*1000=2000, included)
! Feasible successor 2 = 2500  (2500 > 2*1000=2000, excluded)
```

### EIGRP for IPv6

```
! Classic mode (requires interface-level activation)
ipv6 unicast-routing
ipv6 router eigrp 100
 eigrp router-id 10.0.0.1
 no shutdown

interface GigabitEthernet0/0
 ipv6 eigrp 100

! Named mode (preferred)
! See named mode section above for address-family ipv6 configuration
```

### Redistribution

```
! Redistribute OSPF into EIGRP
router eigrp 100
 redistribute ospf 1 metric 10000 100 255 1 1500
 ! metric: bandwidth delay reliability load mtu
 ! Or use a route-map for filtering:
 redistribute ospf 1 route-map OSPF-TO-EIGRP metric 10000 100 255 1 1500

! Redistribute BGP into EIGRP
router eigrp 100
 redistribute bgp 65000 metric 10000 100 255 1 1500

! IMPORTANT: Always set a seed metric when redistributing into EIGRP
! Without a seed metric, redistributed routes have infinite metric and are unreachable

route-map OSPF-TO-EIGRP permit 10
 match ip address prefix-list ALLOWED_ROUTES
 set metric 10000 100 255 1 1500

ip prefix-list ALLOWED_ROUTES seq 10 permit 192.168.0.0/16 le 24
```

## FRRouting Configuration

```bash
vtysh -c "configure terminal"
```

```
router eigrp 100
 eigrp router-id 10.0.0.1
 network 10.0.0.0/24
 network 10.1.0.0/24
 passive-interface eth2
 redistribute connected
 redistribute static metric 10000 100 255 1 1500
 variance 2

interface eth0
 ip bandwidth eigrp 100 1000000
 ip delay eigrp 100 10
 ip hello-interval eigrp 100 5
 ip hold-time eigrp 100 15
```

## Show and Verification Commands

```bash
# View EIGRP neighbors and their state
show ip eigrp neighbors
show ip eigrp neighbors detail

# View topology table (all learned routes, successors, and feasible successors)
show ip eigrp topology
show ip eigrp topology all-links
show ip eigrp topology 10.1.0.0/24

# View only routes installed in the routing table from EIGRP
show ip route eigrp

# Check EIGRP interface status and timers
show ip eigrp interfaces
show ip eigrp interfaces detail

# Display EIGRP protocol statistics and traffic counters
show ip eigrp traffic

# View EIGRP events log (useful for troubleshooting flaps)
show ip eigrp events

# Named mode specific
show eigrp address-family ipv4 neighbors
show eigrp address-family ipv4 topology
show eigrp address-family ipv4 interfaces
show eigrp address-family ipv6 neighbors

# Check SIA status
show ip eigrp topology active

# Verify authentication
show ip eigrp neighbors detail | include Auth

# Debug (use with caution in production)
debug eigrp packets
debug eigrp packets hello
debug eigrp packets query
debug eigrp neighbors
debug eigrp fsm
```

## Troubleshooting

### Neighbor Not Forming

```bash
# 1. Verify interface is up and IP addressing is correct
show ip interface brief

# 2. Check AS number matches on both sides
show ip protocols | section eigrp

# 3. Verify K-values match (must be identical on both neighbors)
show ip protocols | include K

# 4. Check authentication configuration
show ip eigrp neighbors detail
show key chain

# 5. Verify network statements include the interface
show ip eigrp interfaces

# 6. Ensure interface is not passive
show ip protocols | include Passive

# 7. Check ACLs or firewalls blocking EIGRP (protocol 88)
show ip access-lists
# EIGRP uses IP protocol 88, multicast 224.0.0.10
```

### Stuck-in-Active (SIA)

```bash
# Identify SIA routes
show ip eigrp topology active
show ip eigrp events | include SIA

# SIA causes:
# - Unidirectional link (neighbor receives query but reply doesn't get back)
# - Neighbor CPU overload (cannot process query in time)
# - Network partition during active query
# - Query scope too broad (use stub routers or summarization to limit)

# Mitigation:
# - Configure stub routers on spoke sites
# - Summarize routes to limit query scope
# - Increase SIA timer: timers active-time 5 (minutes, default 3)
# - Check for unidirectional links (compare neighbor tables on both ends)
```

### Metric Mismatch or Suboptimal Routing

```bash
# Compare metrics on both ends
show ip eigrp topology 10.1.0.0/24
# Check bandwidth and delay values on interfaces
show interfaces GigabitEthernet0/0 | include BW|DLY

# Common issue: default bandwidth on serial/tunnel interfaces
# Serial default = 1544 kbps (T1), Tunnel default = 100 kbps
# Set correct bandwidth:
interface Tunnel0
 bandwidth 1000000
 delay 10
```

## Tips

- Always set `eigrp router-id` explicitly to avoid unexpected changes when interfaces go up or down.
- Use named mode on all new deployments; it provides SHA-256 auth, wide metrics, and cleaner configuration hierarchy.
- Never change K-values unless every router in the AS is updated simultaneously; mismatched K-values prevent adjacency.
- Configure stub routing on spoke/branch routers to reduce query scope and prevent SIA issues.
- When redistributing into EIGRP, always specify a seed metric; without one, routes are unreachable.
- Summarize routes at distribution layer to reduce topology table size and limit query propagation.
- Variance only applies to feasible successors; a route that fails the feasibility condition is never used regardless of variance.
- Use `show ip eigrp topology all-links` to see routes that failed the feasibility condition.
- On high-speed links (10G+), prefer named mode with wide metrics to avoid metric rollover.
- EIGRP uses IP protocol 88 (not TCP or UDP); ensure firewalls permit protocol 88 to multicast 224.0.0.10.
- Hello and hold timers do not need to match between neighbors (unlike OSPF), but mismatched timers can cause flapping.
- Use `passive-interface default` and selectively enable EIGRP on uplinks to reduce attack surface.

## See Also

- ospf, bgp, rip, is-is, ecmp, bfd, subnetting, ip, ipv6

## References

- [Cisco EIGRP Configuration Guide](https://www.cisco.com/c/en/us/td/docs/ios-xml/ios/iproute_eigrp/configuration/xe-16/ire-xe-16-book.html)
- [Cisco EIGRP Named Mode Configuration](https://www.cisco.com/c/en/us/td/docs/ios-xml/ios/iproute_eigrp/configuration/xe-16/ire-xe-16-book/ire-named-mode.html)
- [Cisco EIGRP Wide Metrics](https://www.cisco.com/c/en/us/td/docs/ios-xml/ios/iproute_eigrp/configuration/xe-16/ire-xe-16-book/ire-wid-met.html)
- [RFC 7868 — Cisco's Enhanced Interior Gateway Routing Protocol](https://www.rfc-editor.org/rfc/rfc7868)
- [EIGRP Informational RFC (Historic)](https://datatracker.ietf.org/doc/html/rfc7868)
- [FRRouting EIGRP Documentation](https://docs.frrouting.org/en/latest/eigrpd.html)
- [Cisco EIGRP Stub Routing](https://www.cisco.com/c/en/us/support/docs/ip/enhanced-interior-gateway-routing-protocol-eigrp/200340-Understanding-EIGRP-Stub-Routing.html)
- [Cisco EIGRP Stuck-in-Active](https://www.cisco.com/c/en/us/support/docs/ip/enhanced-interior-gateway-routing-protocol-eigrp/13676-18.html)
