# CoPP — Control Plane Policing Architecture and Implementation

> *Control Plane Policing protects the most vulnerable component of any network device: the general-purpose CPU. While forwarding ASICs handle millions of packets per second through hardware lookup tables, the control plane processor handles routing protocol adjacencies, management sessions, and exception packets at rates orders of magnitude lower. CoPP applies QoS policy to the internal punt path between the forwarding engine and the CPU, ensuring that critical traffic (BGP keepalives, OSPF hellos) is never starved by floods of low-priority or malicious packets. This document covers the architectural foundations, traffic classification taxonomy, platform-specific implementations, rate engineering, and operational monitoring required to deploy CoPP effectively across IOS-XE, NX-OS, IOS-XR, and JunOS environments.*

---

## 1. The Control Plane as Attack Surface

### 1.1 Forwarding Plane vs Control Plane Asymmetry

Every modern network device has two fundamentally different processing domains. The forwarding plane (data plane) uses purpose-built ASICs or NPUs to perform lookup, rewrite, and switching at line rate. A Memory or a Memory for a Memory is not an issue because switching tables were fetched from the forwarding engines of the ASIC already. The control plane runs on a general-purpose CPU (typically ARM or x86) and handles everything the ASIC cannot: routing protocol state machines, CLI and API sessions, SNMP polling, exception packets, and system management.

The asymmetry is stark:

| Component | Typical Throughput | Processing Type |
|:---|:---|:---|
| Forwarding ASIC | 1-12.8 Tbps | Hardware lookup (TCAM, LPM) |
| Control plane CPU | 10,000-100,000 pps | Software processing (OS kernel) |

This means a single 1 Gbps flood of 64-byte packets (~1.48 Mpps) can overwhelm the CPU by a factor of 15-150x, even though the ASIC handles it without breaking a sweat.

### 1.2 Packet Punt Mechanisms

Not all packets traverse the forwarding plane exclusively. Certain categories are "punted" from the ASIC to the CPU via an internal path:

**Directly addressed packets:** Any packet with a destination IP matching a router interface address. This includes routing protocol packets (BGP to port 179, OSPF to 224.0.0.5), management traffic (SSH to port 22, SNMP to port 161), and services (NTP, DHCP relay).

**Exception packets:** Packets that the ASIC cannot handle autonomously. These include TTL=0 or TTL=1 packets requiring ICMP time-exceeded generation, packets with IP options set, packets requiring fragmentation when DF bit is set (ICMP need-frag), and packets matching a receive adjacency in the FIB (glean for incomplete ARP entries).

**Protocol snooping:** Certain protocols are snooped by the CPU even when not directly addressed. ARP requests/replies on connected subnets, IGMP membership reports for multicast group management, LACP and STP BPDUs for link aggregation and spanning tree.

**ACL logging:** When an ACL entry includes the `log` keyword, matching packets are copied to the CPU for syslog generation.

### 1.3 Real-World Attack Scenarios

Without CoPP, several attack patterns can destabilize the control plane:

**Routing protocol starvation:** An attacker floods the router with millions of ICMP echo requests. The CPU queue fills with ICMP processing. BGP keepalive timers expire because the BGP process cannot read from its socket. BGP sessions tear down. Routes withdraw. Traffic black-holes. The forwarding ASIC is perfectly healthy; the network is down because the CPU is drowning.

**ARP exhaustion:** An attacker sends ARP requests for every IP in a /16 subnet (65,534 requests). Each triggers a glean adjacency punt to the CPU. The CPU attempts to ARP for each address, flooding the ARP table and consuming memory and processing cycles.

**Traceroute amplification:** Each hop in a traceroute generates an ICMP time-exceeded message on the CPU. An attacker sending thousands of traceroutes per second to addresses behind the router forces the CPU to generate thousands of ICMP responses.

**Management plane probing:** Continuous SSH brute-force attempts, SNMP community string scanning, or NTP amplification reflection all consume CPU cycles for connection setup, authentication failure processing, and response generation.

---

## 2. CoPP Architecture

### 2.1 Policy Components

CoPP uses the Modular QoS CLI (MQC) framework, the same three-component model used for interface QoS:

**Class-maps** define the match criteria. Each class-map identifies a category of control plane traffic using ACLs, protocol matches, DSCP values, or combinations thereof. The `match-all` keyword requires all criteria to match (logical AND); `match-any` requires at least one (logical OR).

```
class-map match-all COPP-ROUTING-BGP
 description "BGP control traffic"
 match access-group name COPP-ACL-BGP
```

**Policy-maps** attach policing actions to each class. The policer defines a rate (in pps, bps, or kbps) and a burst size, along with actions for conforming, exceeding, and optionally violating traffic.

```
policy-map COPP-POLICY
 class COPP-ROUTING-BGP
  police rate 2000 pps burst 500 packets
   conform-action transmit
   exceed-action drop
```

**Service-policy** binds the policy-map to the control plane. Unlike interface QoS, CoPP is applied to the logical `control-plane` target rather than a physical interface.

```
control-plane
 service-policy input COPP-POLICY
```

### 2.2 The Punt Path

Understanding where CoPP sits in the packet processing pipeline is critical:

```
                  Ingress         Forwarding         Punt Path
 Packet ------> Interface -----> ASIC Lookup -----> Punt Queue -----> CoPP -----> CPU
                                      |                                           |
                                      |                                     Process/Drop
                                      v
                                 Forward out
                                 egress port
```

On platforms with hardware CoPP (NX-OS, modern IOS-XE on Catalyst 9000, IOS-XR), the policing happens in the ASIC itself. The ASIC classifies the punt-bound packet against TCAM entries and applies the rate limit before the packet enters the punt queue. Packets dropped by hardware CoPP never reach the CPU at all.

On platforms with software CoPP (older IOS, some IOS-XE on ISR), the policing happens in the CPU's input path. The packet has already been punted and consumed queue resources, but is classified and potentially dropped before the protocol stack processes it. This is less effective under extreme floods because the punt queue itself can overflow.

### 2.3 CoPP vs CPPr

Cisco IOS introduced Control Plane Protection (CPPr) as an extension to CoPP that provides finer granularity:

**CoPP (aggregate model):** A single policy-map applied to all control plane traffic. All punt-bound packets enter one policy and are classified by class-maps within it.

**CPPr (sub-interface model):** The control plane is split into three logical sub-interfaces:

| Sub-interface | Traffic Type | Examples |
|:---|:---|:---|
| Host | Destined to the router itself | BGP, SSH, SNMP, NTP |
| Transit | Passing through, punted to CPU | IP options, TTL=1 |
| CEF-exception | CEF/FIB misses | Glean, receive, punt |

Each sub-interface gets its own service-policy:

```
control-plane host
 service-policy input COPP-HOST-POLICY

control-plane transit
 service-policy input COPP-TRANSIT-POLICY

control-plane cef-exception
 service-policy input COPP-CEF-EXCEPTION-POLICY
```

CPPr allows independent rate engineering for each traffic category. For example, transit traffic (traceroute-generated TTL-expired) can be policed very aggressively without affecting host traffic (BGP keepalives). CoPP treats them as one pool, requiring careful class ordering to achieve the same effect.

NX-OS does not support CPPr; it uses CoPP exclusively with hardware-based classification. IOS-XR uses LPTS (Local Packet Transport Services), a different architecture that achieves similar goals through per-protocol hardware policers.

---

## 3. Traffic Classification Taxonomy

### 3.1 Critical Traffic — Routing Protocols

Routing protocol traffic is the highest-priority class because dropping these packets directly causes adjacency failures, route withdrawals, and forwarding black holes.

**BGP (TCP port 179):** BGP uses TCP, so both the SYN/ACK handshake and ongoing keepalive/update traffic must be classified. Match on destination port 179 AND source port 179 to capture both directions. BGP is relatively low-rate during steady state (keepalives every 60 seconds per peer) but bursts during convergence events (full table resync is millions of UPDATE messages).

**OSPF (IP protocol 89):** OSPF uses raw IP protocol 89 with multicast destinations 224.0.0.5 (AllSPFRouters) and 224.0.0.6 (AllDRouters). Hello packets are small and periodic (every 10 seconds on broadcast, 30 on NBMA). LSA floods during convergence can be bursty; size the policer to accommodate SPF events.

**IS-IS (ISO/CLNS):** IS-IS runs directly on Layer 2, not IP. On IOS-XE, classify using protocol matching or DSCP. IS-IS hello intervals are typically 10 seconds (L1/L2), with LSP floods during topology changes. IS-IS is inherently harder to spoof because it requires L2 adjacency.

**EIGRP (IP protocol 88):** EIGRP uses IP protocol 88 with multicast 224.0.0.10. Hello intervals are 5 seconds on most interfaces. EIGRP can be bursty during DUAL computation but typically at moderate rates.

**BFD (UDP ports 3784-3785):** BFD is uniquely time-sensitive because it operates on sub-second timers (commonly 50-300ms intervals with 3x multiplier). BFD packets are small (24-54 bytes) but must not be delayed or dropped. Some platforms process BFD in the ASIC rather than the CPU (hardware-offloaded BFD), making CoPP classification unnecessary for those sessions.

**LDP (TCP port 646, UDP port 646):** LDP discovery uses UDP multicast (224.0.0.2) and sessions use TCP. LDP is typically low-rate but critical for MPLS label distribution.

### 3.2 Important Traffic — Management

Management traffic is essential for operating the device but can typically tolerate small amounts of loss without catastrophic failure.

**SSH/Telnet (TCP 22/23):** Interactive sessions. Rate limit to prevent brute-force login attempts. Always restrict source addresses to management subnets in the ACL.

**SNMP (UDP 161/162):** Polling and traps. NMS polling cycles can generate bursts, especially with large walk operations. Traps are infrequent but important; consider a separate class for trap traffic if traps must not be dropped.

**NTP (UDP 123):** Time synchronization. NTP is low-rate (typically one query every 64-1024 seconds per server) but critical for log correlation and certificate validation. NTP amplification attacks are a common DDoS vector; rate-limiting incoming NTP protects against reflection.

**RADIUS/TACACS+ (UDP 1812-1813, TCP 49):** Authentication traffic. Bursty during login storms (e.g., after a network-wide reboot) but otherwise infrequent. Dropping authentication traffic causes login failures for administrators.

**DNS (UDP/TCP 53):** If the router runs a DNS resolver or proxy. Typically low-rate.

**Syslog (UDP 514):** Outbound syslog is generated by the CPU; inbound syslog is unusual but possible in certain architectures.

### 3.3 Normal Traffic — Network Services

**ARP (Ethertype 0x0806):** ARP requests and replies on connected subnets. Rate depends on subnet size and host churn. A /24 with active hosts generates modest ARP traffic; a /16 with scanning activity can generate thousands of ARP requests per second. ARP is often the most common punt reason on access-layer switches.

**ICMP echo/reply:** Ping for reachability testing. Low priority; rate-limit aggressively. An Internet-facing router may receive thousands of unsolicited pings per second.

**IGMP (IP protocol 2):** Multicast group membership. Rate depends on the number of multicast groups and receivers. Typically moderate on distribution/access layers, minimal on core routers.

**DHCP (UDP 67/68):** DHCP relay or server traffic. Bursty during boot storms (power restoration events) but otherwise infrequent.

### 3.4 Exception Traffic

**TTL-expired:** Generated when the router receives a packet with TTL=1 and must send ICMP time-exceeded. Traceroute is the primary source; rate-limit aggressively because traceroute floods are a common DoS vector.

**MTU-exceeded:** ICMP need-fragmentation (type 3, code 4) when a packet exceeds the outgoing MTU with DF bit set. Important for PMTUD to function; do not drop entirely but rate-limit.

**IP options:** Packets with IP option headers (record route, timestamp, etc.) are punted to the CPU because the ASIC cannot process options. Extremely rare in legitimate traffic; any significant volume indicates scanning or attack. Rate-limit to near-zero.

**CEF/FIB exceptions:** Packets matching a receive or glean adjacency. Glean means the router knows the connected subnet but has no ARP entry for the specific host; it must ARP and punt the first packet. Rate-limit to prevent ARP scanning floods.

### 3.5 Undesirable Traffic

The catch-all class-default must always be policed. Any traffic not matched by an explicit class falls here. This is where DoS floods of unclassified traffic land. Set the rate low enough to protect the CPU but high enough to allow occasional legitimate unclassified packets through for debugging.

---

## 4. Platform-Specific Implementations

### 4.1 Cisco NX-OS

NX-OS ships with CoPP enabled by default. Three built-in profiles provide pre-configured policies:

| Profile | Use Case | Default Class Rate |
|:---|:---|:---|
| strict | Production | Lowest rates, tightest policing |
| moderate | Default on most platforms | Balanced rates |
| lenient | Lab/testing | Permissive, minimal policing |

NX-OS CoPP is hardware-based. The supervisor module programs TCAM entries in the forwarding ASIC that classify and police punt-bound traffic before it reaches the CPU. This means:

- Packets dropped by CoPP never consume CPU cycles
- CoPP counters are maintained in hardware
- Classification uses TCAM space (shared with ACLs and QoS)
- Rate limits are enforced per-ASIC, not globally

Default NX-OS CoPP classes:

| Class | Traffic | Default Rate (strict) |
|:---|:---|:---|
| copp-system-p-class-critical | BGP, OSPF, BFD, LACP | 36000 kbps |
| copp-system-p-class-important | SSH, SNMP, NTP | 1500 kbps |
| copp-system-p-class-normal | ICMP, ARP, IGMP | 600 kbps |
| copp-system-p-class-undesirable | All other | 200 kbps |
| copp-system-p-class-l2-default | L2 control (STP, LLDP) | 600 kbps |

Customization requires creating a custom policy-map of type `control-plane`:

```
policy-map type control-plane COPP-CUSTOM
 class copp-system-p-class-critical
  police cir 48000 kbps bc 1500000 bytes
   conform-action transmit
   violate-action drop
```

After modifying the policy, it must be applied to the control plane:

```
control-plane
 service-policy input COPP-CUSTOM
```

NX-OS does not support CPPr sub-interfaces. All classification is done within the single CoPP policy.

### 4.2 Cisco IOS-XE

IOS-XE CoPP varies by platform. Catalyst 9000 series implements hardware CoPP using the UADP ASIC. ISR 4000/1000 series typically uses software CoPP.

IOS-XE does not ship with a default CoPP policy on most platforms. The administrator must create and apply the policy manually. This is a critical gap; many production IOS-XE routers run without any CoPP protection.

IOS-XE uses the standard MQC framework with pps-based policing:

```
policy-map COPP-POLICY
 class COPP-BGP
  police rate 2000 pps burst 500 packets
   conform-action transmit
   exceed-action drop
```

IOS-XE supports both CoPP and CPPr. CPPr provides the host/transit/cef-exception sub-interface model for finer-grained control.

Catalyst 9000 hardware CoPP integrates with the platform's punt-path architecture. The UADP ASIC classifies punt-bound packets and applies per-class rate limits. Punt policer statistics are available via:

```
show platform software punt-policer
show platform software infrastructure punt
```

### 4.3 Cisco IOS-XR

IOS-XR uses LPTS (Local Packet Transport Services) instead of the MQC-based CoPP model. LPTS is a pre-IFIB (Internal Forwarding Information Base) that classifies locally-destined traffic and applies per-protocol hardware policers.

LPTS operates in two tiers:

**Pre-IFIB (hardware):** The first tier runs in the line card ASIC. It classifies packets by protocol, port, and source/destination and applies coarse rate limits. Packets exceeding the hardware policer are dropped before reaching the route processor.

**IFIB (software):** The second tier runs on the route processor. It provides finer-grained classification and policing for packets that passed the hardware tier.

LPTS policer rates are configured per-protocol:

```
lpts pifib hardware police
 flow bgp known rate 4000
 flow bgp default rate 300
 flow ospf unicast known rate 3000
 flow ospf multicast known rate 3000
 flow ssh known rate 300
 flow snmp rate 300
 flow ntp default rate 200
 flow icmp default rate 200
 flow arp rate 2000
```

The "known" qualifier matches traffic from established protocol sessions (e.g., configured BGP neighbors), while "default" matches unknown sources. This allows generous rates for legitimate peers and restrictive rates for unknown sources.

### 4.4 Juniper JunOS

JunOS uses DDoS Protection (formerly host-inbound traffic filter) for control plane policing. The architecture is conceptually similar to CoPP but uses JunOS-specific constructs.

JunOS DDoS protection operates at three levels:

**Subscriber level:** Per-interface rate limits for each protocol.

**Protocol group level:** Aggregate rate limits across all interfaces for a protocol group.

**System level:** Global aggregate rate limit as a last-resort backstop.

```
set system ddos-protection protocols bgp aggregate bandwidth 20000
set system ddos-protection protocols bgp aggregate burst 5000
set system ddos-protection protocols ospf aggregate bandwidth 10000
set system ddos-protection protocols icmp aggregate bandwidth 2000
```

JunOS also supports firewall filters applied to the loopback interface (lo0) as a complementary mechanism:

```
set firewall filter PROTECT-RE term ALLOW-BGP from protocol tcp port bgp
set firewall filter PROTECT-RE term ALLOW-BGP then policer BGP-POLICER
set firewall filter PROTECT-RE term ALLOW-BGP then accept
set interfaces lo0 unit 0 family inet filter input PROTECT-RE
```

---

## 5. Rate Engineering

### 5.1 Determining Baseline Rates

Effective CoPP requires understanding the legitimate traffic rates for each class. Under-policing drops valid traffic; over-policing provides insufficient protection.

**Methodology:**

1. Deploy CoPP in monitor mode (conform-action transmit, exceed-action transmit) with counters enabled
2. Collect baseline statistics over 7-14 days covering normal operations and at least one maintenance event
3. Identify the peak rate for each class during normal operations
4. Identify the peak rate during convergence events (link failures, BGP full table resync)
5. Set the police rate to 2-3x the convergence peak rate
6. Set the burst to accommodate the longest sustained burst observed

**Example baseline collection:**

| Class | Steady State | Convergence Peak | Recommended Rate | Burst |
|:---|:---:|:---:|:---:|:---:|
| BGP | 50 pps | 800 pps | 2000 pps | 500 pkts |
| OSPF | 20 pps | 400 pps | 1000 pps | 250 pkts |
| SSH | 10 pps | 30 pps | 200 pps | 50 pkts |
| SNMP | 100 pps | 300 pps | 800 pps | 200 pkts |
| ICMP | 50 pps | 200 pps | 500 pps | 100 pkts |
| ARP | 200 pps | 1000 pps | 3000 pps | 500 pkts |
| TTL-expired | 10 pps | 50 pps | 150 pps | 30 pkts |

### 5.2 Burst Size Calculation

The burst parameter (bc — committed burst) determines how many packets can be sent in a single burst before the policer starts dropping. For token-bucket policers:

- Tokens are replenished at the configured rate (e.g., 2000 pps adds 2000 tokens per second)
- The bucket can hold at most `burst` tokens
- Each packet consumes one token
- When the bucket is empty, packets are classified as exceeding

For convergence scenarios where a routing protocol needs to send a burst of LSAs or BGP UPDATEs, the burst size must accommodate the entire burst without dropping:

A practical rule: set burst to the maximum number of packets expected in a single convergence event, or 25-50% of the rate, whichever is larger. For BGP with 800K routes from a single peer, a full table resync sends approximately 8000-16000 UPDATE messages. The burst should be at least 4000-8000 to accommodate resync without drops.

### 5.3 Rate Units: pps vs bps

IOS-XE CoPP supports both pps (packets per second) and bps (bits per second) policing. NX-OS typically uses kbps.

**pps policing** is more intuitive for protocol traffic because protocol packet rates are well-understood (OSPF sends one hello every 10 seconds, BGP keepalive every 60 seconds). It also provides consistent protection regardless of packet size.

**bps policing** is more common in NX-OS and can be useful when the concern is bandwidth consumption rather than packet rate. However, bps policing allows more small packets through than large ones, which may not match the protection intent.

When converting between units, consider average packet sizes for each protocol:

| Protocol | Typical Packet Size | 1000 pps equivalent |
|:---|:---:|:---:|
| BGP keepalive | 19 bytes + headers = ~63 bytes | ~504 kbps |
| BGP UPDATE | 200-4000 bytes | ~500 kbps - 32 Mbps |
| OSPF hello | ~60 bytes | ~480 kbps |
| ICMP echo | 64-1500 bytes | ~512 kbps - 12 Mbps |
| ARP | 28 bytes + headers = ~42 bytes | ~336 kbps |

### 5.4 Conform, Exceed, and Violate Actions

CoPP supports three-action policing with single-rate or dual-rate token buckets:

**Single-rate (1r2c — one rate, two colors):** Packets are either conforming (within rate+burst) or exceeding (above rate+burst). Actions: transmit or drop.

**Single-rate (1r3c — one rate, three colors):** Adds a violate action when traffic exceeds both the committed burst (bc) and excess burst (be). Useful for graduated responses: transmit conforming, remark exceeding (lower DSCP), drop violating.

**Dual-rate (2r3c — two rates, three colors):** Separate committed information rate (CIR) and peak information rate (PIR) with independent burst sizes. Conforming traffic is within CIR, exceeding is between CIR and PIR, violating exceeds PIR.

For CoPP, the most common configuration is 1r2c: transmit conforming, drop exceeding. More sophisticated deployments use 1r3c to remark rather than drop, allowing traffic to be further classified downstream.

---

## 6. Monitoring and Operations

### 6.1 Verifying CoPP Policy

The primary monitoring command shows the active policy with real-time counters:

```
! IOS-XE
show policy-map control-plane

! Sample output:
!  Control Plane
!   Service-policy input: COPP-POLICY
!     Class-map: COPP-BGP (match-all)
!       0 packets, 0 bytes
!       5 minute offered rate 0000 bps
!       Match: access-group name COPP-ACL-BGP
!       police:
!         rate 2000 pps, burst 500 packets
!         conformed 15234 packets; actions: transmit
!         exceeded 0 packets; actions: drop
!         conformed 0000 bps, exceeded 0000 bps
```

Key fields to monitor:

- **exceeded packets:** Any non-zero value means legitimate traffic may be affected. Investigate immediately for critical classes (routing protocols).
- **offered rate:** The total rate of traffic matching this class. Compare against the policer rate to assess headroom.
- **conformed packets:** Validates that the class is matching the intended traffic.

### 6.2 NX-OS CoPP Monitoring

```
! Comprehensive CoPP status
show copp status

! Per-class statistics
show policy-map interface control-plane

! Sample output:
! Control Plane
!   service-policy input copp-system-p-policy-strict
!     class-map copp-system-p-class-critical (match-any)
!       set cos 7
!       police cir 36000 kbps , bc 1200000 bytes
!       module 1 :
!         transmitted 1847234 packets;
!         dropped 0 packets;

! Compare profiles
show copp diff profile strict profile moderate
```

### 6.3 Alerting on CoPP Drops

Proactive alerting prevents CoPP drops from silently causing routing instability. Implement monitoring at multiple layers:

**SNMP polling:** Poll CoPP counters via SNMP and alert when exceed counters increment for critical classes. The relevant MIB objects are in CISCO-CONTROL-PLANE-POLICYMAP-MIB.

**EEM (Embedded Event Manager):** Configure EEM applets to trigger on CoPP-related syslog messages:

```
event manager applet COPP-CRITICAL-ALERT
 event syslog pattern "%CP-.*POLICYDROP" period 60 maxrun 30
 action 1.0 syslog priority critical msg "CoPP dropping control plane traffic"
 action 2.0 cli command "enable"
 action 3.0 cli command "show policy-map control-plane | redirect flash:copp-alert.txt"
```

**Streaming telemetry:** Modern platforms support gRPC-based model-driven telemetry for CoPP counters. Subscribe to the relevant YANG model for real-time counter streaming to your monitoring platform:

```
! IOS-XE telemetry subscription for CoPP
telemetry ietf subscription 100
 encoding encode-kvgpb
 filter xpath /control-plane-policing/copp-policy
 stream yang-push
 update-policy periodic 3000
 receiver ip address 10.0.0.100 57000 protocol grpc-tcp
```

### 6.4 Baseline Deviation Detection

Establish baseline counter rates and alert on deviation:

1. Record CoPP conform/exceed counters every 5 minutes via SNMP or telemetry
2. Calculate rolling 24-hour averages for each class
3. Alert when current rate exceeds 2x the 24-hour average (potential flood)
4. Alert when conform rate drops below 50% of average (possible misclassification or connectivity issue)
5. Alert on any exceed counter increment for critical classes (routing protocols)

### 6.5 Capacity Planning

As the network grows, CoPP rates must be revisited:

- Adding 10 new BGP peers increases steady-state BGP pps by ~10 keepalives/minute plus convergence traffic
- Adding a /16 connected subnet increases ARP baseline proportionally
- Deploying BFD with 50ms timers on 100 interfaces generates 2000 pps baseline
- Enabling SNMP polling from additional NMS stations increases SNMP pps linearly

Review CoPP counters quarterly. If the conform rate for any class approaches 70% of the policer rate during normal operations, increase the rate proactively before the next convergence event causes drops.

---

## 7. Best Practices and Common Pitfalls

### 7.1 Design Principles

**Explicit classification over default:** Every known protocol should have its own class-map. Traffic in class-default is, by definition, unclassified, and the default class should have the most restrictive policer. If routing protocol traffic accidentally falls into class-default because of a misconfigured ACL, it will be policed at the default rate and likely dropped.

**Source restriction in ACLs:** Management traffic ACLs (SSH, SNMP, NTP, RADIUS) should restrict source addresses to known management subnets. This provides defense-in-depth: even if the policer is generous, only traffic from trusted sources is classified into the management class.

**Separate BFD from other routing protocols:** BFD has uniquely strict timing requirements. A 50ms BFD timer with 3x multiplier means a session fails after 150ms of packet loss. BFD should be in its own class with generous rates, or better yet, offloaded to hardware (available on Nexus 9000, ASR 9000, Catalyst 9000).

**Test before deployment:** Apply CoPP in monitor mode first (all actions set to transmit). Collect counters for at least one convergence event (planned failover or maintenance). Only then switch exceed-action to drop.

### 7.2 Common Pitfalls

**Forgetting both directions of TCP protocols:** BGP uses TCP port 179. A class-map matching only `destination port 179` misses return traffic where the source port is 179. Always match both `eq bgp` in source and destination.

**ACL ordering in class-maps:** When using `match-all` with multiple ACL matches, all must match. When using `match-any`, the first match wins. Incorrect ACL structure can cause traffic to match the wrong class or no class at all.

**Not accounting for convergence bursts:** Steady-state BGP is ~1 keepalive per peer per minute. But a full table withdrawal and re-advertisement from a single peer can send 10,000+ UPDATE messages in seconds. If the burst is too small, BGP UPDATEs are dropped during the most critical moment: convergence.

**Ignoring class-default:** Leaving class-default without a policer means all unclassified traffic has unlimited access to the CPU. This is the most common CoPP deployment error.

**NX-OS upgrade profile reset:** Some NX-OS upgrades reset the CoPP profile to the default. Always verify CoPP policy after any NX-OS upgrade using `show copp status` and `show policy-map interface control-plane`.

**Over-aggressive ICMP policing:** Setting ICMP to 10 pps sounds safe but breaks PMTUD. ICMP type 3 code 4 (fragmentation needed) is essential for TCP MSS negotiation. Either separate ICMP types into distinct classes or ensure the rate accommodates PMTUD.

### 7.3 Operational Procedures

**Change management:** Treat CoPP policy changes with the same rigor as routing policy changes. A misconfigured CoPP can take down more devices than a misconfigured route-map because it affects the control plane's ability to maintain all protocol adjacencies.

**Pre-maintenance verification:** Before any maintenance window, record CoPP counters as a baseline. After the maintenance activity (link deactivation, peer migration), verify that no critical class is showing exceed drops.

**Post-incident review:** After any routing instability incident, include CoPP counter review in the root-cause analysis. Many "unexplained" BGP flaps trace back to CoPP dropping keepalives during a concurrent traffic flood.

**Documentation:** Maintain a CoPP policy document that maps each class-map to its intended traffic, the rationale for the configured rate, and the last date the rate was validated against actual traffic patterns.

---

## 8. Integration with Other Security Controls

### 8.1 Infrastructure ACLs (iACLs)

Infrastructure ACLs filter traffic destined to router infrastructure addresses at the network edge. iACLs complement CoPP by reducing the volume of traffic that reaches the CoPP policer in the first place. Deploy iACLs on all external-facing interfaces to block traffic to router loopback and interface addresses from untrusted sources.

### 8.2 Receive ACLs (rACLs)

Receive ACLs are applied specifically to traffic destined to the router itself (similar to CoPP's scope). On platforms that support rACLs in hardware, they can drop unwanted traffic before it enters the punt path. rACLs are a binary permit/deny mechanism; CoPP adds rate-limiting on top of access control.

### 8.3 GTSM (Generalized TTL Security Mechanism)

GTSM (RFC 5082) protects routing protocols by requiring that packets have a TTL of 255, meaning they must originate from a directly connected neighbor (TTL cannot be 255 after traversing any router). GTSM is implemented in the ASIC and drops non-conforming packets before they reach CoPP, reducing load on the policer.

```
! BGP GTSM
router bgp 65000
 neighbor 10.0.0.2 ttl-security hops 1
```

### 8.4 MD5/TCP-AO Authentication

Routing protocol authentication (BGP MD5, TCP-AO, OSPF authentication, IS-IS authentication) causes the router to drop unauthenticated packets before they are processed. This reduces CPU load from spoofed routing packets but does not eliminate it entirely because the authentication check itself consumes CPU cycles. CoPP should still police authenticated protocol traffic to handle authentication processing floods.

---

## Prerequisites

- Understanding of QoS concepts: class-maps, policy-maps, service-policy, policing
- Familiarity with ACL construction (extended IP ACLs, protocol matching)
- Knowledge of routing protocol operation (BGP state machine, OSPF hello/dead timers, BFD intervals)
- Understanding of the difference between forwarding plane (ASIC/TCAM) and control plane (CPU)
- Access to `show policy-map control-plane` output on a production device for baseline analysis

---

## References

- [RFC 6192 — Protecting the Router Control Plane](https://www.rfc-editor.org/rfc/rfc6192)
- [RFC 5082 — Generalized TTL Security Mechanism (GTSM)](https://www.rfc-editor.org/rfc/rfc5082)
- [RFC 5925 — The TCP Authentication Option (TCP-AO)](https://www.rfc-editor.org/rfc/rfc5925)
- [Cisco IOS-XE Control Plane Policing Configuration Guide](https://www.cisco.com/c/en/us/td/docs/ios-xml/ios/qos_plcshp/configuration/xe-16/qos-plcshp-xe-16-book/qos-plcshp-ctrl-pln-plc.html)
- [Cisco NX-OS CoPP Configuration Guide (Nexus 9000)](https://www.cisco.com/c/en/us/td/docs/switches/datacenter/nexus9000/sw/93x/security/configuration/guide/b-cisco-nexus-9000-nx-os-security-configuration-guide-93x/b-cisco-nexus-9000-nx-os-security-configuration-guide-93x_chapter_010010.html)
- [Cisco IOS-XR LPTS Configuration Guide (ASR 9000)](https://www.cisco.com/c/en/us/td/docs/routers/asr9000/software/asr9k-r7-x/lpts/configuration/guide/b-lpts-cg-asr9000-7x.html)
- [Juniper JunOS DDoS Protection](https://www.juniper.net/documentation/us/en/software/junos/denial-of-service/topics/concept/copp-understanding.html)
- [Arista EOS Control Plane Policing](https://www.arista.com/en/um-eos/eos-control-plane-policing)
- [NIST SP 800-189 — Resilient Interdomain Traffic Exchange: BGP Security and DDoS Mitigation](https://csrc.nist.gov/publications/detail/sp/800-189/final)
- [NSA Network Infrastructure Security Guide](https://media.defense.gov/2022/Jun/15/2003018261/-1/-1/0/CTR_NSA_NETWORK_INFRASTRUCTURE_SECURITY_GUIDE_20220615.PDF)
- [FIRST CVSS — Scoring Network Infrastructure Vulnerabilities](https://www.first.org/cvss/)
