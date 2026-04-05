# MPLS TE (Traffic Engineering with RSVP-TE)

Constraint-based routing and signaling framework that builds explicit label-switched paths with bandwidth guarantees, enabling deterministic traffic placement across MPLS networks.

## Concepts

### Core Architecture

- **TE Tunnel:** Unidirectional LSP from headend to tailend with explicit path and bandwidth constraints
- **CSPF (Constrained Shortest Path First):** Modified Dijkstra algorithm that prunes links not meeting constraints before computing SPF
- **Headend:** Ingress router that initiates and maintains the TE tunnel
- **Tailend:** Egress router; the tunnel destination
- **Midpoint:** Transit LSR along the tunnel path that maintains RSVP state
- **TE Database (TED):** Repository of network topology and link attributes (bandwidth, affinity, TE metric) flooded by IGP-TE extensions
- **Autoroute:** Mechanism to inject TE tunnel as next-hop into the IGP routing table

### RSVP-TE Signaling

- **PATH Message:** Sent headend-to-tailend; carries ERO, sender template, traffic parameters
- **RESV Message:** Sent tailend-to-headend; confirms reservation, carries label and RRO
- **ERO (Explicit Route Object):** Ordered list of hops the LSP must traverse (strict or loose)
- **RRO (Record Route Object):** Records the actual path taken; returned in RESV for verification
- **Session Attribute Object:** Carries tunnel name, setup/hold priority, flags (local protection, SE style)
- **RSVP Refresh:** Periodic PATH/RESV messages maintain soft state (default 30 seconds)
- **Hello Messages:** Detect neighbor failure faster than refresh timeout (default 3 seconds, 3.5x multiplier)

### TE Metrics and Constraints

| Attribute              | Source         | Purpose                                                |
|------------------------|----------------|--------------------------------------------------------|
| Maximum Bandwidth      | IGP-TE flood   | Total link capacity                                    |
| Maximum Reservable BW  | IGP-TE flood   | Bandwidth available for TE reservation                 |
| Unreserved BW (8 prios)| IGP-TE flood   | Available BW per setup priority (0-7)                  |
| TE Metric              | Admin config    | Separate metric for CSPF (independent of IGP metric)   |
| Admin Group (Affinity) | Admin config    | 32-bit bitmask for link coloring / exclusion           |
| SRLG                   | Admin config    | Shared Risk Link Group for diverse path computation    |

### Bandwidth Reservation

- Tunnels reserve bandwidth along the path during RSVP signaling
- Eight priority levels (0 = highest, 7 = lowest); split into setup priority and hold priority
- Setup priority must be numerically lower than or equal to hold priority
- A new tunnel can preempt an existing tunnel if its setup priority is higher (lower number) than the existing tunnel's hold priority
- Bandwidth is accounted per-priority in the TED; CSPF checks unreserved BW at the tunnel's setup priority

### CSPF Algorithm

1. Start with the full network topology from the TED
2. Prune links that do not meet bandwidth requirement at the tunnel's setup priority
3. Prune links excluded by affinity/admin-group constraints
4. Prune links in excluded SRLG groups (if configured)
5. Run Dijkstra on the pruned topology using TE metric (or IGP metric if TE metric not set)
6. If multiple equal-cost paths exist, apply tiebreakers (random, least-fill, most-fill)
7. Result: explicit path encoded as ERO and signaled via RSVP PATH

## IGP TE Extensions

### OSPF TE Extensions (RFC 3630)

```
! IOS: Enable OSPF TE extensions
router ospf 1
 mpls traffic-eng router-id Loopback0
 mpls traffic-eng area 0

! Interface TE attributes
interface GigabitEthernet0/0
 mpls traffic-eng tunnels
 ip rsvp bandwidth 1000000 1000000
 ! Format: ip rsvp bandwidth <interface-bw-kbps> <single-flow-max-kbps>
```

### IS-IS TE Extensions (RFC 5305)

```
! IOS: Enable IS-IS TE extensions
router isis 1
 mpls traffic-eng router-id Loopback0
 mpls traffic-eng level-2
 metric-style wide
 ! Wide metrics required for TE

interface GigabitEthernet0/0
 ip router isis 1
 mpls traffic-eng tunnels
 ip rsvp bandwidth 1000000 1000000
```

### IOS-XR IGP TE

```
! IOS-XR: OSPF TE
router ospf 1
 mpls traffic-eng router-id Loopback0
 area 0
  mpls traffic-eng
  interface GigabitEthernet0/0/0/0
   !
  !
 !

! IOS-XR: IS-IS TE
router isis 1
 address-family ipv4 unicast
  mpls traffic-eng router-id Loopback0
  mpls traffic-eng level-2
  metric-style wide
 !
```

## TE Tunnel Configuration

### Cisco IOS — Basic Tunnel

```
interface Tunnel0
 ip unnumbered Loopback0
 tunnel mode mpls traffic-eng
 tunnel destination 10.0.0.5
 tunnel mpls traffic-eng bandwidth 100000
 ! Bandwidth in kbps
 tunnel mpls traffic-eng priority 3 3
 ! Setup-priority hold-priority (0-7, lower = higher priority)
 tunnel mpls traffic-eng path-option 1 explicit name PATH-PRIMARY
 tunnel mpls traffic-eng path-option 2 dynamic
 ! Fallback to dynamic CSPF if explicit fails
```

### Cisco IOS — Explicit Path

```
ip explicit-path name PATH-PRIMARY enable
 next-address strict 10.1.1.2
 next-address strict 10.2.2.2
 next-address strict 10.3.3.2

ip explicit-path name PATH-LOOSE enable
 next-address loose 10.2.2.2
 ! Loose: CSPF can insert additional hops between here and next entry
 next-address strict 10.3.3.2
```

### Cisco IOS-XR — TE Tunnel

```
interface tunnel-te0
 ipv4 unnumbered Loopback0
 destination 10.0.0.5
 signalled-bandwidth 100000
 priority 3 3
 path-option 1 explicit name PATH-PRIMARY
 path-option 10 dynamic
 autoroute announce
 !

explicit-path name PATH-PRIMARY
 index 10 next-address strict ipv4 unicast 10.1.1.2
 index 20 next-address strict ipv4 unicast 10.2.2.2
 index 30 next-address strict ipv4 unicast 10.3.3.2
```

## Affinity / Admin Groups

### Configuration

```
! IOS: Define colors (admin-group names)
mpls traffic-eng attribute-flags name RED 0x1
mpls traffic-eng attribute-flags name BLUE 0x2
mpls traffic-eng attribute-flags name GREEN 0x4

! Apply affinity to a link
interface GigabitEthernet0/0
 mpls traffic-eng attribute-flags 0x3
 ! Binary 011 = belongs to RED (bit 0) and BLUE (bit 1)

! Tunnel affinity constraint
interface Tunnel0
 tunnel mpls traffic-eng affinity 0x1 mask 0x1
 ! Include links with bit 0 set (RED), ignore other bits
```

### Affinity Matching Logic

```
Link is included if: (link-attribute-flags AND mask) == (affinity AND mask)

Example:
  Link flags  = 0x3  (binary: 0011)
  Affinity    = 0x1  (binary: 0001)
  Mask        = 0x1  (binary: 0001)

  (0x3 AND 0x1) = 0x1
  (0x1 AND 0x1) = 0x1
  0x1 == 0x1 -> MATCH (link included in CSPF)
```

### IOS-XR Affinity (Named Mode)

```
mpls traffic-eng
 affinity-map RED 0
 affinity-map BLUE 1
 affinity-map GREEN 2

interface tunnel-te0
 affinity include RED
 affinity exclude BLUE
```

## Auto-Bandwidth

### Concept

- Dynamically adjusts tunnel bandwidth based on measured traffic rate
- Samples traffic at configurable intervals
- Adjusts bandwidth at adjustment-interval boundaries
- Prevents oscillation with min/max bounds and adjustment threshold

### IOS Configuration

```
interface Tunnel0
 tunnel mpls traffic-eng auto-bw
  frequency 300
  ! Sample interval in seconds (how often traffic rate is measured)
  max-bw 500000
  ! Maximum bandwidth allowed (kbps)
  min-bw 10000
  ! Minimum bandwidth (kbps)
  adjustment-threshold 10
  ! Only adjust if change exceeds 10% of current bandwidth
  overflow-threshold 20 3
  ! Trigger immediate resize if traffic exceeds BW by 20% for 3 consecutive samples
  collect-bw
  ! Enable bandwidth collection even before auto-bw adjusts
```

### IOS-XR Configuration

```
interface tunnel-te0
 auto-bw
  bw-limit min 10000 max 500000
  adjustment-threshold 10 min 1000
  application 300
  overflow threshold 20 limit 3
  underflow threshold 20 limit 3
 !
```

## Fast Reroute (FRR)

### Facility Backup (Link/Node Protection)

```
! IOS: Enable FRR on the tunnel
interface Tunnel0
 tunnel mpls traffic-eng fast-reroute

! IOS: Create backup tunnel on the PLR (Point of Local Repair)
interface Tunnel100
 ip unnumbered Loopback0
 tunnel mode mpls traffic-eng
 tunnel destination 10.0.0.3
 ! Destination = MP (Merge Point), next-next-hop for node protection
 tunnel mpls traffic-eng bandwidth 0
 ! Backup tunnel typically uses 0 (best-effort) bandwidth
 tunnel mpls traffic-eng priority 7 7

! IOS: Assign backup tunnel to protected interface
interface GigabitEthernet0/0
 mpls traffic-eng backup-path Tunnel100

! Node protection: backup tunnel destination is the next-next-hop
! Link protection: backup tunnel destination is the next-hop (via alternate path)
```

### One-to-One Backup (1:1)

```
! IOS: One-to-one (detour) backup
interface Tunnel0
 tunnel mpls traffic-eng fast-reroute
 ! When no backup tunnel is configured, RSVP attempts per-LSP detour

! Each midpoint creates a detour LSP around the protected resource
! Detour merges back to the original LSP downstream of the failure
```

### IOS-XR FRR

```
interface tunnel-te0
 fast-reroute

mpls traffic-eng
 interface GigabitEthernet0/0/0/0
  backup-path tunnel-te 100
 !

! Node protection with SRLG awareness
interface tunnel-te100
 ipv4 unnumbered Loopback0
 destination 10.0.0.3
 path-option 1 dynamic
  exclude srlg
 !
```

### FRR Comparison

| Feature             | Facility Backup                  | One-to-One (Detour)             |
|---------------------|----------------------------------|---------------------------------|
| Scalability         | High (one backup per interface)  | Low (one detour per protected LSP) |
| Bandwidth guarantee | Shared across all protected LSPs | Per-LSP bandwidth               |
| Config complexity   | Requires pre-built backup tunnels| Automatic (no backup tunnels)   |
| Protection          | Link or Node                     | Link or Node                    |
| Merge point         | Explicit (backup tunnel tailend) | Automatic (downstream of failure)|

## TE Metric vs IGP Metric

### Configuration

```
! IOS: Set TE metric on a link (independent of IGP metric)
interface GigabitEthernet0/0
 mpls traffic-eng administrative-weight 100
 ! Default: TE metric = IGP metric if not configured

! IOS: Tell tunnel to use TE metric for CSPF
interface Tunnel0
 tunnel mpls traffic-eng path-selection metric te
 ! Options: igp, te
```

### Use Cases

- TE metric lets you steer traffic differently for TE tunnels vs IGP routing
- Example: high-latency satellite link has low IGP metric (cheap) but high TE metric (avoid for TE)
- When TE metric is not configured, CSPF uses the IGP metric by default

## Autoroute and Traffic Steering

### Autoroute Announce

```
! IOS: Make IGP use the tunnel as next-hop for destinations beyond the tailend
interface Tunnel0
 tunnel mpls traffic-eng autoroute announce
 ! All prefixes reachable via the tailend (and beyond) use this tunnel
 tunnel mpls traffic-eng autoroute metric relative -5
 ! Adjust metric: relative (-5 from IGP metric) or absolute (fixed value)
```

### Autoroute Destination

```
! IOS: Only use tunnel for traffic destined to the tailend address itself
interface Tunnel0
 tunnel mpls traffic-eng autoroute destination
```

### Static Route over Tunnel

```
! Force specific traffic over the TE tunnel
ip route 10.10.0.0 255.255.0.0 Tunnel0
```

### Policy-Based Routing

```
! IOS: PBR to steer traffic based on access-list match
route-map TE-STEERING permit 10
 match ip address ACL-VOICE
 set interface Tunnel0

interface GigabitEthernet0/1
 ip policy route-map TE-STEERING
```

## Class-Based Tunnel Selection (CBTS)

```
! IOS: Map DSCP values to specific TE tunnels
interface Tunnel0
 tunnel mpls traffic-eng exp 5 6
 ! EF/CS6 traffic uses this tunnel (low-latency path)

interface Tunnel1
 tunnel mpls traffic-eng exp 0 1 2 3 4
 ! Best-effort traffic uses this tunnel (high-bandwidth path)

interface Tunnel2
 tunnel mpls traffic-eng exp-default
 ! Default for any EXP not explicitly mapped
```

## TE for Load Balancing

### Multiple Parallel Tunnels

```
! IOS: Two tunnels to same destination for load sharing
interface Tunnel0
 tunnel destination 10.0.0.5
 tunnel mpls traffic-eng bandwidth 50000
 tunnel mpls traffic-eng path-option 1 explicit name PATH-VIA-NORTH
 tunnel mpls traffic-eng autoroute announce
 tunnel mpls traffic-eng load-share 10

interface Tunnel1
 tunnel destination 10.0.0.5
 tunnel mpls traffic-eng bandwidth 50000
 tunnel mpls traffic-eng path-option 1 explicit name PATH-VIA-SOUTH
 tunnel mpls traffic-eng autoroute announce
 tunnel mpls traffic-eng load-share 10
 ! load-share values determine relative traffic distribution
```

### Unequal Cost Load Sharing

```
! Different load-share values for weighted distribution
interface Tunnel0
 tunnel mpls traffic-eng load-share 30
 ! 30/(30+10) = 75% of traffic

interface Tunnel1
 tunnel mpls traffic-eng load-share 10
 ! 10/(30+10) = 25% of traffic
```

## Show and Verify Commands

### IOS Show Commands

```
! TE tunnel status and path
show mpls traffic-eng tunnels
show mpls traffic-eng tunnels name Tunnel0

! RSVP reservations
show ip rsvp reservation
show ip rsvp sender

! TE database (topology with TE attributes)
show mpls traffic-eng topology
show mpls traffic-eng topology brief

! Link attributes (bandwidth, affinity)
show mpls traffic-eng link-management bandwidth-allocation

! CSPF path computation
show mpls traffic-eng topology path destination 10.0.0.5 bandwidth 100000

! Auto-bandwidth
show mpls traffic-eng tunnels tunnel0 auto-bw

! FRR status
show mpls traffic-eng fast-reroute database
show ip rsvp fast-reroute

! RSVP interface
show ip rsvp interface
show ip rsvp neighbor
```

### IOS-XR Show Commands

```
show mpls traffic-eng tunnels
show mpls traffic-eng tunnels detail
show rsvp reservation
show rsvp sender
show mpls traffic-eng topology
show mpls traffic-eng link-management bandwidth-allocation
show mpls traffic-eng fast-reroute database
show mpls traffic-eng auto-bw
```

## Troubleshooting

### Common Issues

```
! Tunnel is down: CSPF failed to find a path
show mpls traffic-eng tunnels name Tunnel0
! Check: "Last PCALC Error:" field for failure reason

! Insufficient bandwidth
show mpls traffic-eng topology
! Verify unreserved bandwidth at the tunnel's setup priority level

! Affinity mismatch
show mpls traffic-eng link-management advertisements
! Verify link attribute-flags match tunnel affinity/mask

! RSVP signaling failure
debug ip rsvp
show ip rsvp request
! Check for PathErr or ResvErr messages

! Autoroute not injecting routes
show ip route
show ip cef 10.10.0.0
! Verify: tunnel must be UP and autoroute announce configured

! FRR not protecting
show mpls traffic-eng fast-reroute database
! Verify backup tunnel is up and assigned to the protected interface
```

### Debug Commands

```
! IOS: Targeted debugging
debug mpls traffic-eng tunnels signaling
debug ip rsvp path
debug ip rsvp resv
debug mpls traffic-eng path computation
debug mpls traffic-eng auto-bw

! IOS-XR
debug mpls traffic-eng signalling
debug mpls traffic-eng path
```

## Tips

- Always configure RSVP bandwidth on every TE-enabled interface; without it, RSVP cannot reserve bandwidth and tunnels may fail to signal.
- Use `autoroute announce` to make the tunnel transparent to IGP; use static routes or PBR for fine-grained control.
- Auto-bandwidth prevents bandwidth waste but introduces re-signaling; set appropriate thresholds to avoid flapping.
- Facility backup (FRR) scales better than one-to-one detour; prefer it in production networks with many tunnels.
- TE metric defaults to IGP metric when not explicitly configured; set it independently when you want different TE and IGP path preferences.
- FRR provides sub-50ms protection but does not guarantee bandwidth on the backup path unless explicitly configured.
- Always verify the TED (`show mpls traffic-eng topology`) to confirm TE attributes are being flooded correctly by the IGP.
- RSVP soft state means tunnels require periodic refresh; losing refresh messages for too long tears down the tunnel.
- Setup priority should never be numerically lower (higher priority) than hold priority on the same tunnel.
- SRLG-aware FRR requires consistent SRLG configuration across all routers in the TE domain.

## See Also

- mpls, mpls-vpn, ospf, is-is, bgp, segment-routing, bfd, ecmp, cos-qos

## References

- [RFC 3209 — RSVP-TE: Extensions to RSVP for LSP Tunnels](https://www.rfc-editor.org/rfc/rfc3209)
- [RFC 3630 — Traffic Engineering (TE) Extensions to OSPF Version 2](https://www.rfc-editor.org/rfc/rfc3630)
- [RFC 5305 — IS-IS Extensions for Traffic Engineering](https://www.rfc-editor.org/rfc/rfc5305)
- [RFC 4090 — Fast Reroute Extensions to RSVP-TE](https://www.rfc-editor.org/rfc/rfc4090)
- [RFC 3785 — Use of Interior Gateway Protocol (IGP) Metric as a Second MPLS TE Metric](https://www.rfc-editor.org/rfc/rfc3785)
- [RFC 2205 — Resource ReSerVation Protocol (RSVP)](https://www.rfc-editor.org/rfc/rfc2205)
- [RFC 4875 — Extensions to RSVP-TE for Point-to-Multipoint TE LSPs](https://www.rfc-editor.org/rfc/rfc4875)
- [Cisco MPLS TE Configuration Guide (IOS XE)](https://www.cisco.com/c/en/us/td/docs/ios-xml/ios/mp_te_path/configuration/xe-16/mp-te-path-xe-16-book.html)
- [Cisco IOS-XR MPLS TE Configuration Guide](https://www.cisco.com/c/en/us/td/docs/routers/asr9000/software/mpls/configuration/guide/b-mpls-cg-asr9000.html)
