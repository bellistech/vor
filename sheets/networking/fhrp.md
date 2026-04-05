# FHRP (First Hop Redundancy — HSRP, VRRP, GLBP)

Provides gateway redundancy so hosts never lose their default gateway when a router fails.

## Quick Comparison

| Feature              | HSRP                     | VRRP                     | GLBP                     |
|----------------------|--------------------------|--------------------------|--------------------------|
| Standard             | Cisco proprietary        | RFC 5798 (v3), 3768 (v2) | Cisco proprietary        |
| Terminology          | Active / Standby         | Master / Backup          | AVG / AVF                |
| Default Priority     | 100                      | 100                      | 100                      |
| Preemption           | Disabled by default      | Enabled by default       | Disabled by default      |
| Virtual MAC (v2)     | 0000.0c9f.fXXX           | 0000.5e00.01XX           | 0007.b400.XXYY           |
| Multicast (IPv4)     | 224.0.0.2 (v1/v2)       | 224.0.0.18               | 224.0.0.102              |
| Multicast (IPv6)     | ff02::66 (v2)           | ff02::12 (v3)            | ff02::66                 |
| Hello / Hold         | 3s / 10s                 | 1s / 3x hello            | 3s / 10s                 |
| Load Balancing       | Manual (multi-group)     | Manual (multi-group)     | Native (AVF)             |
| Group Range v1       | 0-255                    | 1-255 (v2)               | 0-1023                   |
| Group Range v2       | 0-4095                   | 1-255 (v3)               | 0-1023                   |
| IPv6 Support         | v2 only                  | v3 only                  | Yes                      |
| Authentication       | Plaintext / MD5          | None (v3), text (v2)     | Plaintext / MD5          |
| Transport            | UDP 1985                 | IP protocol 112          | UDP 3222                 |

## HSRP Configuration

### Basic HSRP v2

```
! Enable HSRP version 2 globally on the interface
interface GigabitEthernet0/0
 ip address 10.0.1.2 255.255.255.0
 standby version 2
 standby 1 ip 10.0.1.1
 standby 1 priority 110
 standby 1 preempt
 standby 1 timers 1 3
 standby 1 authentication md5 key-string S3cur3Key!
 standby 1 track 1 decrement 20
!
! Object tracking
track 1 interface GigabitEthernet0/1 line-protocol
```

### HSRP v2 IPv6

```
interface GigabitEthernet0/0
 ipv6 address 2001:db8:1::2/64
 standby version 2
 standby 2 ipv6 autoconfig
 standby 2 priority 110
 standby 2 preempt
```

### HSRP on NX-OS (vPC)

```
! NX-OS requires feature hsrp
feature hsrp

interface Vlan100
 ip address 10.0.100.2/24
 hsrp version 2
 hsrp 100
  ip 10.0.100.1
  priority 110
  preempt
  timers 1 3
  authentication md5 key-string V3ryS3cure
```

### HSRP with vPC Peer-Gateway

```
! Both vPC peers — allows forwarding traffic destined to peer's HSRP MAC
vpc domain 1
 peer-gateway

! Fabric peering (NX-OS 7.0.3+) — HSRP hellos over peer-link fabric
interface Vlan100
 hsrp version 2
 hsrp 100
  ip 10.0.100.1
  priority 110
  preempt
  fabricpath-peering         ! NX-OS fabric peering for HSRP
```

### Multi-Group (Manual Load Balancing)

```
! Router A — active for group 1, standby for group 2
interface GigabitEthernet0/0
 standby version 2
 standby 1 ip 10.0.1.1
 standby 1 priority 110
 standby 1 preempt
 standby 2 ip 10.0.1.2
 standby 2 priority 90

! Router B — standby for group 1, active for group 2
interface GigabitEthernet0/0
 standby version 2
 standby 1 ip 10.0.1.1
 standby 1 priority 90
 standby 2 ip 10.0.1.2
 standby 2 priority 110
 standby 2 preempt

! Half the hosts point to 10.0.1.1, half to 10.0.1.2
```

## VRRP Configuration

### Basic VRRP v3

```
! VRRP v3 uses "vrrp address-family" syntax on IOS-XE
interface GigabitEthernet0/0
 ip address 10.0.1.3 255.255.255.0
 vrrp 1 address-family ipv4
  address 10.0.1.1 primary
  priority 110
  preempt delay minimum 30
  track 1 decrement 20
  timers advertise 100          ! in centiseconds (1 second)
  exit-vrrp
```

### VRRP v3 IPv6

```
interface GigabitEthernet0/0
 ipv6 address 2001:db8:1::3/64
 vrrp 1 address-family ipv6
  address 2001:db8:1::1 primary
  address fe80::1
  priority 110
  exit-vrrp
```

### VRRP with Real IP as Virtual

```
! Master can use its own real IP as the virtual IP
! Priority becomes 255 automatically (owner mode)
interface GigabitEthernet0/0
 ip address 10.0.1.1 255.255.255.0
 vrrp 1 address-family ipv4
  address 10.0.1.1 primary        ! same as real IP
  ! priority auto-set to 255
  exit-vrrp
```

### Legacy VRRP v2 (older IOS)

```
interface GigabitEthernet0/0
 ip address 10.0.1.3 255.255.255.0
 vrrp 1 ip 10.0.1.1
 vrrp 1 priority 110
 vrrp 1 preempt
 vrrp 1 timers advertise 1
 vrrp 1 authentication text MyPass
```

## GLBP Configuration

### Basic GLBP with Load Balancing

```
interface GigabitEthernet0/0
 ip address 10.0.1.2 255.255.255.0
 glbp 1 ip 10.0.1.1
 glbp 1 priority 110
 glbp 1 preempt
 glbp 1 load-balancing round-robin
 glbp 1 timers 3 10
 glbp 1 authentication md5 key-string GlbpK3y!
```

### GLBP Load Balancing Methods

```
! Round-robin — default, cycles through AVFs
glbp 1 load-balancing round-robin

! Weighted — proportional to weight value
glbp 1 load-balancing weighted
glbp 1 weighting 200 lower 50 upper 100
glbp 1 weighting track 1 decrement 50

! Host-dependent — consistent hash of client MAC, same client always hits same AVF
glbp 1 load-balancing host-dependent
```

### GLBP Redirect and Timeout Timers

```
! Redirect timer: how long AVG continues to redirect clients to old AVF (default 600s)
! Timeout timer: how long before AVG removes the AVF from the table (default 14400s)
glbp 1 timers redirect 600 14400
```

### GLBP with Interface Tracking

```
glbp 1 weighting 100 lower 50 upper 80
glbp 1 weighting track 1 decrement 30
glbp 1 weighting track 2 decrement 20

track 1 interface GigabitEthernet0/1 line-protocol
track 2 ip route 0.0.0.0/0 reachability
```

## Interface Tracking

### Object Tracking (All Protocols)

```
! Track interface line-protocol state
track 1 interface GigabitEthernet0/1 line-protocol

! Track IP reachability via static route
track 2 ip route 10.0.0.0/8 reachability

! Track IP SLA (ping probe)
ip sla 1
 icmp-echo 8.8.8.8
 frequency 5
ip sla schedule 1 start-time now life forever
track 3 ip sla 1 reachability

! Track list — boolean logic
track 10 list boolean and
 object 1
 object 2

! Apply to HSRP
standby 1 track 1 decrement 20
standby 1 track 10 decrement 50

! Apply to VRRP
vrrp 1 address-family ipv4
 track 1 decrement 20

! Apply to GLBP
glbp 1 weighting track 1 decrement 30
```

## HSRP State Machine (Quick Reference)

```
+--------+     +-------+     +--------+     +-------+     +---------+     +--------+
| Init   |---->| Learn |---->| Listen |---->| Speak |---->| Standby |---->| Active |
+--------+     +-------+     +--------+     +-------+     +---------+     +--------+
  No VIP       Learning       Listening      Sending       Ready to        Forwarding
  known        VIP from       for hellos     hellos,       take over       traffic
               Active                        contending
```

## VRRP State Machine (Quick Reference)

```
+------------+     +--------+     +--------+
| Initialize |---->| Backup |<--->| Master |
+------------+     +--------+     +--------+
  Interface up      Listening      Forwarding,
                    for adverts    sending adverts
```

## Troubleshooting Commands

### HSRP

```
show standby                        ! Full HSRP state
show standby brief                  ! Summary: Grp/Pri/State/Active/Standby/VIP
show standby vlan 100               ! Specific VLAN/interface
show standby | include Active|Standby|Priority
debug standby events                ! State transitions
debug standby packets               ! Hello/coup/resign packets
debug standby errors                ! Error conditions
show track                          ! Object tracking state
show track brief                    ! Track summary
```

### VRRP

```
show vrrp                           ! Full VRRP state
show vrrp brief                     ! Summary table
show vrrp interface Gi0/0           ! Specific interface
show vrrp all                       ! All groups including inactive
debug vrrp events                   ! State changes
debug vrrp packets                  ! Advertisement packets
```

### GLBP

```
show glbp                           ! Full GLBP state (AVG + AVF info)
show glbp brief                     ! Summary table
show glbp | include AVG|AVF|State
debug glbp events                   ! State transitions
debug glbp packets                  ! Hello and redirect packets
debug glbp errors                   ! Errors
```

### Common Verification Steps

```
! 1. Check FHRP state on both routers
show standby brief          (or show vrrp brief / show glbp brief)

! 2. Verify virtual IP is reachable from hosts
ping 10.0.1.1

! 3. Check virtual MAC in ARP table on hosts
show ip arp | include 10.0.1.1

! 4. Verify tracking objects
show track
show track brief

! 5. Test failover — shut primary uplink and verify transition
interface GigabitEthernet0/1
 shutdown
show standby brief          ! Verify standby took over

! 6. Check multicast group membership
show ip mroute 224.0.0.2    ! HSRP
show ip mroute 224.0.0.18   ! VRRP
show ip mroute 224.0.0.102  ! GLBP
```

## Design Decision Matrix

| Scenario                        | Recommended   | Reason                              |
|---------------------------------|---------------|-------------------------------------|
| Multi-vendor environment        | VRRP          | Open standard, interoperable        |
| Cisco-only, simple HA           | HSRP v2       | Mature, well-understood, wide NX-OS |
| Cisco-only, need load balancing | GLBP          | Native per-host load balancing      |
| vPC / MLAG environment          | HSRP v2       | Best vPC integration, peer-gateway  |
| IPv6 required                   | VRRP v3       | Best IPv6 support, dual-stack       |
| Large group count needed        | HSRP v2       | 4096 groups                         |
| Minimal config, fast failover   | VRRP          | Preempt on by default, 1s hello     |

## Timer Tuning Cheatsheet

```
! Aggressive HSRP (sub-second)
standby 1 timers msec 200 msec 750

! Aggressive VRRP
vrrp 1 address-family ipv4
 timers advertise 15           ! 150ms (centiseconds in v3)

! Conservative (stable WAN)
standby 1 timers 5 15
```

## Tips

- Always use HSRP v2 over v1 — larger group range (4096 vs 256), millisecond timers, IPv6 support.
- VRRP preempts by default — add `preempt delay minimum 30` to prevent flapping during boot.
- GLBP AVG election works like HSRP active election — highest priority, then highest IP wins.
- With vPC, always enable `peer-gateway` — prevents packets from being blackholed when destined to the peer's router MAC.
- Track upstream interfaces or IP SLA probes, not just line-protocol — a link can be up but routing broken.
- Use MD5 authentication on HSRP/GLBP in production — prevents rogue routers from joining the group.
- Set preempt delay on the primary to avoid flapping during router reload or OSPF convergence.
- In GLBP, if the AVG fails, a new AVG is elected but existing AVF assignments persist until redirect timer expires.
- HSRP and GLBP use UDP; VRRP uses IP protocol 112 directly — firewall rules differ.
- Multi-group HSRP/VRRP achieves load balancing but requires hosts to be split across gateways (use DHCP pools or DNS round-robin).

## See Also

- stp (Spanning Tree Protocol — L2 redundancy)
- etherchannel (LAG/LACP/PAgP)
- vpc (Virtual Port-Channel / MLAG)
- routing-fundamentals (default gateway, static routes)
- ospf (dynamic routing for upstream tracking)

## References

- [RFC 5798 — VRRPv3](https://datatracker.ietf.org/doc/html/rfc5798)
- [RFC 3768 — VRRPv2](https://datatracker.ietf.org/doc/html/rfc3768)
- [Cisco HSRP Configuration Guide](https://www.cisco.com/c/en/us/td/docs/ios-xml/ios/ipapp_fhrp/configuration/xe-16/fhp-xe-16-book/fhp-hsrp.html)
- [Cisco GLBP Configuration Guide](https://www.cisco.com/c/en/us/td/docs/ios-xml/ios/ipapp_fhrp/configuration/xe-16/fhp-xe-16-book/fhp-glbp.html)
- [Cisco NX-OS HSRP Configuration](https://www.cisco.com/c/en/us/td/docs/switches/datacenter/nexus9000/sw/93x/unicast/configuration/guide/b-cisco-nexus-9000-series-nx-os-unicast-routing-configuration-guide-93x/b-cisco-nexus-9000-series-nx-os-unicast-routing-configuration-guide-93x_chapter_01110.html)
- [Cisco vPC Design Guide — FHRP with vPC](https://www.cisco.com/c/en/us/products/collateral/switches/nexus-5000-series-switches/design_guide_c07-625857.html)
