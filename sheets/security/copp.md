# CoPP (Control Plane Policing)

Rate-limiting and filtering traffic destined to the network device CPU to protect control plane availability against floods, DoS attacks, and misbehaving protocols.

## Concepts

### Why the Control Plane Needs Protection

- The forwarding plane (ASIC/TCAM) handles millions of pps; the CPU handles thousands at best
- Any packet that requires software processing (routing protocol, management, exception) hits the CPU
- Without CoPP, a single flood of ARP, ICMP, or TTL-expired packets can starve BGP/OSPF keepalives
- A dead control plane means a dead router even when the forwarding ASIC is healthy

### CoPP vs CPPr (Control Plane Protection)

- **CoPP:** Single aggregate policy applied to all control plane traffic; one policy-map, one service-policy
- **CPPr:** Cisco IOS extension that splits control plane traffic into three sub-interfaces:
  - **Host:** Traffic destined to the router itself (BGP, SSH, SNMP)
  - **Transit:** Traffic passing through but punted to CPU (IP options, TTL=1)
  - **CEF-exception:** Packets that miss CEF entries (glean, receive, punt)
- CPPr provides finer-grained policing per sub-interface; CoPP treats the control plane as one entity
- NX-OS uses CoPP exclusively; IOS-XE supports both CoPP and CPPr

### Traffic Classes Hitting the CPU

| Category | Examples | Risk Level |
|:---|:---|:---:|
| Routing protocols | BGP, OSPF, IS-IS, EIGRP, BFD, LDP | Critical |
| Management | SSH, SNMP, NTP, RADIUS, TACACS+, Telnet | High |
| Network services | DHCP, DNS, ARP, ICMP echo | Medium |
| Exceptions | TTL expired, MTU exceeded, IP options, ICMP unreachable | Medium |
| Multicast | IGMP, PIM, MLD | Medium |
| DoS / unknown | Broadcast storms, spoofed packets, malformed frames | Highest |

### CoPP Policy Structure

```
# Three components:
# 1. class-map  — match criteria (ACL, protocol, DSCP)
# 2. policy-map — attach policer to each class
# 3. service-policy — apply to control-plane

class-map match-all COPP-BGP
 match access-group name COPP-ACL-BGP

policy-map COPP-POLICY
 class COPP-BGP
  police rate 500 pps burst 100 packets
   conform-action transmit
   exceed-action drop

control-plane
 service-policy input COPP-POLICY
```

## IOS-XE Configuration

### Basic CoPP Policy

```
! Step 1: Define ACLs for each traffic class
ip access-list extended COPP-ACL-BGP
 permit tcp any any eq bgp
 permit tcp any eq bgp any

ip access-list extended COPP-ACL-OSPF
 permit ospf any any

ip access-list extended COPP-ACL-ISIS
 permit iso any any

ip access-list extended COPP-ACL-EIGRP
 permit eigrp any any

ip access-list extended COPP-ACL-MGMT
 permit tcp any any eq 22
 permit udp any any eq 161
 permit udp any any eq 123

ip access-list extended COPP-ACL-ICMP
 permit icmp any any echo
 permit icmp any any echo-reply

ip access-list extended COPP-ACL-ARP
 permit arp any any

ip access-list extended COPP-ACL-TTL
 permit icmp any any ttl-exceeded
 permit icmp any any port-unreachable
```

### Class Maps

```
class-map match-all COPP-BGP
 match access-group name COPP-ACL-BGP

class-map match-all COPP-OSPF
 match access-group name COPP-ACL-OSPF

class-map match-all COPP-ISIS
 match access-group name COPP-ACL-ISIS

class-map match-all COPP-EIGRP
 match access-group name COPP-ACL-EIGRP

class-map match-all COPP-MGMT
 match access-group name COPP-ACL-MGMT

class-map match-all COPP-ICMP
 match access-group name COPP-ACL-ICMP

class-map match-all COPP-ARP
 match access-group name COPP-ACL-ARP

class-map match-all COPP-TTL
 match access-group name COPP-ACL-TTL
```

### Policy Map with Policers

```
policy-map COPP-POLICY
 ! Critical: routing protocols get highest rates
 class COPP-BGP
  police rate 1000 pps burst 250 packets
   conform-action transmit
   exceed-action drop

 class COPP-OSPF
  police rate 2000 pps burst 500 packets
   conform-action transmit
   exceed-action drop

 class COPP-ISIS
  police rate 2000 pps burst 500 packets
   conform-action transmit
   exceed-action drop

 class COPP-EIGRP
  police rate 1500 pps burst 400 packets
   conform-action transmit
   exceed-action drop

 ! Management: moderate rates
 class COPP-MGMT
  police rate 500 pps burst 100 packets
   conform-action transmit
   exceed-action drop

 ! ICMP: low rate, enough for troubleshooting
 class COPP-ICMP
  police rate 200 pps burst 50 packets
   conform-action transmit
   exceed-action drop

 ! ARP: moderate, depends on subnet size
 class COPP-ARP
  police rate 1000 pps burst 200 packets
   conform-action transmit
   exceed-action drop

 ! TTL-expired / exceptions: limit aggressively
 class COPP-TTL
  police rate 100 pps burst 25 packets
   conform-action transmit
   exceed-action drop

 ! Default class: catch-all for unclassified traffic
 class class-default
  police rate 250 pps burst 50 packets
   conform-action transmit
   exceed-action drop

! Step 3: Apply to control plane
control-plane
 service-policy input COPP-POLICY
```

### CPPr Sub-Interfaces (IOS-XE)

```
! Apply separate policies per sub-interface
control-plane host
 service-policy input COPP-HOST-POLICY

control-plane transit
 service-policy input COPP-TRANSIT-POLICY

control-plane cef-exception
 service-policy input COPP-CEF-POLICY
```

## NX-OS Configuration

### Default CoPP Profile

```
! NX-OS ships with a default CoPP policy
! Three built-in profiles:
!   strict  — recommended for production
!   moderate — default on most platforms
!   lenient — permissive, not recommended

! View current profile
show copp profile

! Apply a built-in profile
copp profile strict

! The profile auto-generates class-maps and policy-maps
! Classes include: copp-system-p-class-critical,
!   copp-system-p-class-important, copp-system-p-class-normal,
!   copp-system-p-class-undesirable, copp-system-p-class-default
```

### Customizing NX-OS CoPP

```
! Step 1: Copy the default profile to a custom policy
show policy-map interface control-plane

! Step 2: Modify specific class rates
policy-map type control-plane COPP-CUSTOM
 class copp-system-p-class-critical
  police cir 36000 kbps bc 1200000 bytes
   conform-action transmit
   violate-action drop

 class copp-system-p-class-important
  police cir 1500 kbps bc 48000 bytes
   conform-action transmit
   violate-action drop

 class copp-system-p-class-normal
  police cir 600 kbps bc 19200 bytes
   conform-action transmit
   violate-action drop

 class copp-system-p-class-undesirable
  police cir 200 kbps bc 6400 bytes
   conform-action transmit
   violate-action drop

! Step 3: Apply custom policy
control-plane
 service-policy input COPP-CUSTOM
```

### Adding Custom Classes on NX-OS

```
! Create an ACL for the custom traffic
ip access-list COPP-ACL-CUSTOM-APP
 permit udp any any eq 5000

! Create a class-map
class-map type control-plane match-all COPP-CUSTOM-APP
 match access-group name COPP-ACL-CUSTOM-APP

! Add to the policy-map
policy-map type control-plane COPP-CUSTOM
 class COPP-CUSTOM-APP
  police cir 500 kbps bc 16000 bytes
   conform-action transmit
   violate-action drop
```

## Protecting Routing Protocols

### BGP Protection

```
! Dedicated class for BGP with generous rate
ip access-list extended COPP-ACL-BGP
 permit tcp any any eq 179
 permit tcp any eq 179 any
 ! Include BGP multihop (TTL > 1)
 permit tcp any any eq 179 ttl gt 1
 permit tcp any eq 179 any ttl gt 1

class-map match-all COPP-BGP
 match access-group name COPP-ACL-BGP

! Rate should handle full table convergence
! 500-2000 pps depending on peer count and table size
policy-map COPP-POLICY
 class COPP-BGP
  police rate 2000 pps burst 500 packets
   conform-action transmit
   exceed-action drop
```

### OSPF / IS-IS Protection

```
! OSPF uses protocol 89
ip access-list extended COPP-ACL-OSPF
 permit ospf any 224.0.0.5/32
 permit ospf any 224.0.0.6/32
 permit ospf any host 224.0.0.5
 permit ospf any host 224.0.0.6

! IS-IS runs at L2; match on CLNS/ISO
! On IOS-XE, use DSCP matching as fallback
class-map match-all COPP-OSPF
 match access-group name COPP-ACL-OSPF

class-map match-all COPP-ISIS
 match access-group name COPP-ACL-ISIS
```

### BFD Protection

```
! BFD is time-sensitive; needs priority handling
ip access-list extended COPP-ACL-BFD
 permit udp any any range 3784 3785
 permit udp any range 3784 3785 any

class-map match-all COPP-BFD
 match access-group name COPP-ACL-BFD

! BFD should not be rate-limited aggressively
! but must be protected from spoofing
policy-map COPP-POLICY
 class COPP-BFD
  police rate 4000 pps burst 1000 packets
   conform-action transmit
   exceed-action drop
```

## Protecting Management Traffic

### SSH, SNMP, NTP, RADIUS

```
ip access-list extended COPP-ACL-SSH
 ! Only from trusted management subnets
 permit tcp 10.0.0.0/8 any eq 22
 permit tcp 172.16.0.0/12 any eq 22

ip access-list extended COPP-ACL-SNMP
 permit udp 10.0.0.0/8 any eq 161
 permit udp 10.0.0.0/8 any eq 162

ip access-list extended COPP-ACL-NTP
 permit udp any any eq 123

ip access-list extended COPP-ACL-RADIUS
 permit udp any any eq 1812
 permit udp any any eq 1813
 permit udp any any eq 1645
 permit udp any any eq 1646

class-map match-all COPP-SSH
 match access-group name COPP-ACL-SSH
class-map match-all COPP-SNMP
 match access-group name COPP-ACL-SNMP
class-map match-all COPP-NTP
 match access-group name COPP-ACL-NTP
class-map match-all COPP-RADIUS
 match access-group name COPP-ACL-RADIUS

policy-map COPP-POLICY
 class COPP-SSH
  police rate 200 pps burst 50 packets
   conform-action transmit
   exceed-action drop
 class COPP-SNMP
  police rate 500 pps burst 100 packets
   conform-action transmit
   exceed-action drop
 class COPP-NTP
  police rate 100 pps burst 25 packets
   conform-action transmit
   exceed-action drop
 class COPP-RADIUS
  police rate 300 pps burst 75 packets
   conform-action transmit
   exceed-action drop
```

## ARP, ICMP, and Exception Traffic

### ARP Rate Limiting

```
ip access-list extended COPP-ACL-ARP
 permit arp any any

class-map match-all COPP-ARP
 match access-group name COPP-ACL-ARP

! ARP rate depends on directly-connected host count
! /24 subnet: ~500 pps is generous
! /16 subnet or many L3 interfaces: increase accordingly
policy-map COPP-POLICY
 class COPP-ARP
  police rate 1000 pps burst 300 packets
   conform-action transmit
   exceed-action drop
```

### ICMP Rate Limiting

```
ip access-list extended COPP-ACL-ICMP-ECHO
 permit icmp any any echo
 permit icmp any any echo-reply

ip access-list extended COPP-ACL-ICMP-UNREACH
 permit icmp any any unreachable
 permit icmp any any time-exceeded

class-map match-all COPP-ICMP-ECHO
 match access-group name COPP-ACL-ICMP-ECHO

class-map match-all COPP-ICMP-UNREACH
 match access-group name COPP-ACL-ICMP-UNREACH

policy-map COPP-POLICY
 class COPP-ICMP-ECHO
  police rate 200 pps burst 50 packets
   conform-action transmit
   exceed-action drop
 class COPP-ICMP-UNREACH
  police rate 100 pps burst 25 packets
   conform-action transmit
   exceed-action drop
```

### TTL-Expired / IP Options

```
ip access-list extended COPP-ACL-TTL-EXPIRED
 permit icmp any any ttl-exceeded

ip access-list extended COPP-ACL-IP-OPTIONS
 permit ip any any option any-options

class-map match-all COPP-TTL-EXPIRED
 match access-group name COPP-ACL-TTL-EXPIRED

class-map match-all COPP-IP-OPTIONS
 match access-group name COPP-ACL-IP-OPTIONS

! TTL-expired is a common traceroute flood vector
! IP options should be rare in modern networks
policy-map COPP-POLICY
 class COPP-TTL-EXPIRED
  police rate 100 pps burst 25 packets
   conform-action transmit
   exceed-action drop
 class COPP-IP-OPTIONS
  police rate 50 pps burst 10 packets
   conform-action transmit
   exceed-action drop
```

## Hardware vs Software CoPP

### Platform Differences

```
! Hardware CoPP (NX-OS, IOS-XR, some IOS-XE platforms):
!   - Policing happens in the ASIC before packets reach the CPU
!   - TCAM entries enforce rate limits in hardware
!   - Drops are counted in hardware counters
!   - No CPU impact from dropped packets

! Software CoPP (older IOS, some IOS-XE platforms):
!   - Policing happens in the punt path (after ASIC, before CPU)
!   - CPU must inspect packets to classify and police
!   - Less effective under extreme flood conditions
!   - Still better than no CoPP at all

! Check platform support
show platform software punt-policer   ! IOS-XE
show system internal copp             ! NX-OS
```

### Queuing and Priority

```
! NX-OS assigns hardware queue priorities to CoPP classes:
!   Queue 0: Critical (routing protocols)
!   Queue 1: Important (management)
!   Queue 2: Normal (ICMP, ARP)
!   Queue 3: Undesirable (unclassified)

! Each queue has independent rate limits
! Critical queue drains first, preventing starvation
show hardware rate-limiter            ! NX-OS
```

## Monitoring and Verification

### Show Commands

```bash
# IOS-XE: View CoPP policy and counters
show policy-map control-plane
show policy-map control-plane input class COPP-BGP

# IOS-XE: Check for dropped packets (exceed counters)
show policy-map control-plane | include exceed|drop|class

# NX-OS: CoPP status and counters
show copp status
show policy-map interface control-plane

# NX-OS: Detailed per-class statistics
show policy-map interface control-plane class copp-system-p-class-critical

# NX-OS: CoPP profile differences
show copp diff profile strict profile moderate

# IOS-XR: CoPP equivalent (LPTS)
show lpts pifib statistics
show lpts pifib hardware statistics

# Platform-specific punt counters
show platform software punt-policer   ! IOS-XE
show system internal copp info        ! NX-OS

# CPU utilization (correlate with CoPP drops)
show processes cpu sorted             ! IOS-XE
show system resources                 ! NX-OS
```

### Logging and Alerting

```
! Enable logging for CoPP violations
policy-map COPP-POLICY
 class COPP-ICMP
  police rate 200 pps burst 50 packets
   conform-action transmit
   exceed-action drop
   ! Log exceeded packets (use cautiously; logging itself uses CPU)
   ! exceed-action drop log

! Syslog monitoring patterns
! Look for these messages:
!   %CP-6-POLICYDROP: Control-plane packet dropped
!   %COPP-4-EXCEED: CoPP class exceeded

! SNMP traps for CoPP events
snmp-server enable traps copp
snmp-server host 10.0.0.100 version 2c COMMUNITY copp

! EEM applet to alert on high CPU caused by punt traffic
event manager applet COPP-ALERT
 event syslog pattern "%COPP.*EXCEED" period 60
 action 1.0 syslog priority warnings msg "CoPP rate exceeded - investigate"
 action 2.0 cli command "show policy-map control-plane"
```

### Clearing Counters

```
! Reset CoPP counters for baseline measurement
clear control-plane counters          ! IOS-XE
clear copp statistics                 ! NX-OS
```

## Troubleshooting

### Routing Protocol Flaps Under Load

```bash
# Check if CoPP is dropping routing protocol packets
show policy-map control-plane | section BGP
show policy-map control-plane | section OSPF

# Look for non-zero exceed/drop counters on critical classes
# If routing protocol packets are being dropped:
#   1. Increase the police rate for that class
#   2. Verify ACLs are matching correctly
#   3. Ensure routing protocol traffic is not in class-default

# Verify which class is matching the traffic
show class-map COPP-BGP
debug platform control-plane classification  ! use with caution
```

### Management Access Denied

```bash
# If SSH/SNMP is being rate-limited:
show policy-map control-plane | section MGMT

# Increase management class rate
# Or restrict source to management subnet only (reduce noise)

# Verify ACL matches
show access-list COPP-ACL-SSH
show access-list COPP-ACL-SNMP
```

### Identifying Flood Sources

```bash
# Check CPU punt reasons
show platform software punt-policer   ! IOS-XE
show system internal copp info        ! NX-OS

# Identify top punt reasons
show platform software infrastructure punt detail  ! IOS-XE

# Capture punted packets for analysis
debug platform packet-trace punt      ! IOS-XE (use carefully)
ethanalyzer local interface inband display-filter "ip" ! NX-OS
```

## Tips

- Always deploy CoPP in production; a router without CoPP is an unprotected router.
- Use the strictest built-in profile as a starting point and loosen only where monitoring shows legitimate drops.
- Never put routing protocols in class-default; a flood of unclassified traffic will starve BGP and OSPF.
- Test CoPP changes in a maintenance window; a misconfigured policer can take down the entire control plane.
- Size burst values to at least 2x the expected peak rate to absorb convergence spikes.
- Restrict management ACLs to trusted source subnets; this reduces noise before the policer even engages.
- Monitor exceed counters weekly; a sudden spike in drops on a routing class often precedes a neighbor flap.
- Separate BFD into its own class with generous rates; BFD timers are aggressive and drops cause false failures.
- On NX-OS, always run `show copp status` after upgrades; defaults may change between NX-OS versions.
- Use hardware CoPP where available; software CoPP still consumes CPU cycles for classification.
- Log exceed actions sparingly; excessive logging during a flood creates a feedback loop that worsens CPU load.
- Document your CoPP policy alongside your routing policy; both are equally critical to network stability.
- Re-evaluate CoPP rates after topology changes; adding 50 new BGP peers changes the baseline significantly.

## See Also

- bgp, ospf, is-is, eigrp, bfd, iptables, nftables, firewalld, acl, snmp, radius, ids-ips

## References

- [Cisco IOS-XE CoPP Configuration Guide](https://www.cisco.com/c/en/us/td/docs/ios-xml/ios/qos_plcshp/configuration/xe-16/qos-plcshp-xe-16-book/qos-plcshp-ctrl-pln-plc.html)
- [Cisco NX-OS CoPP Configuration Guide](https://www.cisco.com/c/en/us/td/docs/switches/datacenter/nexus9000/sw/93x/security/configuration/guide/b-cisco-nexus-9000-nx-os-security-configuration-guide-93x/b-cisco-nexus-9000-nx-os-security-configuration-guide-93x_chapter_010010.html)
- [Cisco IOS-XR LPTS Configuration Guide](https://www.cisco.com/c/en/us/td/docs/routers/asr9000/software/asr9k-r7-x/lpts/configuration/guide/b-lpts-cg-asr9000-7x.html)
- [Juniper JunOS DDoS Protection (Juniper CoPP equivalent)](https://www.juniper.net/documentation/us/en/software/junos/denial-of-service/topics/concept/copp-understanding.html)
- [Arista EOS CoPP Configuration Guide](https://www.arista.com/en/um-eos/eos-control-plane-policing)
- [RFC 6192 — Protecting the Router Control Plane](https://www.rfc-editor.org/rfc/rfc6192)
- [NIST SP 800-189 — Resilient Interdomain Traffic Exchange](https://csrc.nist.gov/publications/detail/sp/800-189/final)
- [NSA Network Infrastructure Security Guide](https://media.defense.gov/2022/Jun/15/2003018261/-1/-1/0/CTR_NSA_NETWORK_INFRASTRUCTURE_SECURITY_GUIDE_20220615.PDF)
