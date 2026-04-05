# IP SLA — Active Network Measurement and Path Verification

> *IP Service Level Agreement is Cisco's active monitoring framework that injects synthetic probe traffic into the network to measure reachability, latency, jitter, packet loss, and application response times in real time. Unlike passive monitoring that observes existing traffic, IP SLA generates controlled test packets on schedule, enabling deterministic measurement of path quality and automated failover when service levels degrade below defined thresholds. Combined with enhanced object tracking, IP SLA transforms a router from a simple packet forwarder into a network-aware decision engine that adapts routing, FHRP priorities, and policy-based forwarding based on measured reality rather than protocol state alone.*

---

## 1. Active vs. Passive Measurement

### The Measurement Paradigm

Network performance measurement falls into two categories:

**Passive monitoring** observes real user traffic via SNMP counters, NetFlow, sFlow, or packet captures. It measures what is already happening but cannot detect problems before users are affected, and it provides no data when traffic volume is low.

**Active monitoring** generates synthetic probe packets that traverse the same paths as production traffic. It provides continuous measurement regardless of traffic volume, detects failures before users notice, and produces consistent, comparable metrics across time periods.

IP SLA is an active monitoring system embedded in Cisco IOS, IOS-XE, and NX-OS. The source router generates probe packets, sends them to a target (which may be a configured IP SLA responder or a real service endpoint), and measures the response characteristics.

### Why Active Probing Matters

Consider a dual-ISP site where the primary link's BGP session is up but the ISP's upstream transit provider has failed. The BGP session remains established because it peers with the directly connected ISP router. Static routes via that ISP remain in the routing table. Passive monitoring sees the interface as up and the BGP session as established. Only an active probe to a target beyond the ISP (such as a public DNS server) reveals that end-to-end connectivity has failed.

This is the core value proposition of IP SLA: it measures what matters to applications — end-to-end path quality — rather than what is visible to routing protocols — adjacency state with the next hop.

---

## 2. Probe Architecture

### The IP SLA Engine

The IP SLA engine runs as a process on the router's control plane. It consists of:

- **Scheduler:** Manages probe timing, ensuring probes fire at configured intervals and distributing probe execution to avoid CPU spikes
- **Probe generator:** Constructs and sends probe packets with appropriate timestamps and sequence numbers
- **Collector:** Receives responses, calculates metrics (RTT, jitter, packet loss), and stores results in the statistics database
- **Reactor:** Monitors metrics against configured thresholds and triggers actions (SNMP traps, track state changes) when thresholds are breached
- **History engine:** Maintains rolling buffers of probe results for trend analysis

### Probe Packet Flow

For an ICMP echo probe, the flow is straightforward:

1. The scheduler triggers the probe at the configured frequency interval
2. The probe generator creates an ICMP echo request with a timestamp in the payload
3. The target responds with a standard ICMP echo reply
4. The collector calculates RTT from the embedded timestamp
5. The result is stored in the statistics database and checked against thresholds

For a UDP jitter probe, the flow is more complex:

1. The scheduler triggers the probe
2. The source sends a control message to the IP SLA responder on the target
3. The responder opens a temporary port and confirms readiness
4. The source sends a configurable number of UDP packets (default 10) at configurable intervals (default 20ms)
5. Each packet carries a send timestamp; the responder adds a receive timestamp and a send-back timestamp before returning it
6. The source records the return timestamp, giving four timestamps per packet
7. From these four timestamps, the collector calculates one-way delay (source-to-destination and destination-to-source), round-trip time, and jitter in both directions

### The Four Timestamps

The UDP jitter probe uses four timestamps per packet to decompose round-trip time:

| Timestamp | Label | Captured By |
|:---|:---|:---|
| T1 | Send time at source | Source router |
| T2 | Receive time at target | Responder |
| T3 | Send-back time at target | Responder |
| T4 | Receive time at source | Source router |

From these:

- **Source-to-destination delay:** T2 - T1 (requires clock synchronization for absolute value; jitter calculation does not require sync)
- **Destination-to-source delay:** T4 - T3
- **Round-trip time:** (T4 - T1) - (T3 - T2), which eliminates responder processing time
- **Source-to-destination jitter:** |(T2n - T1n) - (T2(n-1) - T1(n-1))|
- **Destination-to-source jitter:** |(T4n - T3n) - (T4(n-1) - T3(n-1))|

The jitter calculation uses the difference of differences, which cancels out any constant clock offset between source and target. This means accurate jitter measurement does not require NTP synchronization between devices, though NTP is still recommended for meaningful one-way delay values.

---

## 3. Probe Types in Depth

### ICMP Echo

The simplest probe type. Sends standard ICMP echo requests and measures RTT from the ICMP echo reply. Works against any IP host without requiring an IP SLA responder. Limitations: cannot measure one-way delay or jitter (only RTT); ICMP may be rate-limited or deprioritized by intermediate devices, giving inaccurate latency readings; some firewalls block ICMP entirely.

Best used for: basic reachability monitoring, static route failover triggers, and gateway health checks.

### UDP Jitter

The most versatile probe type. Sends a burst of timestamped UDP packets to an IP SLA responder and calculates RTT, one-way delay (both directions), jitter (both directions), and packet loss (both directions). The responder adds its own timestamps, enabling decomposition of the path into source-to-target and target-to-source segments.

The number of packets per test and the inter-packet interval are configurable. Default is 10 packets at 20ms intervals, completing in 200ms. For statistical significance, increase to 50-100 packets. The inter-packet interval should match the traffic pattern being simulated; for VoIP, use 20ms to match the G.729 codec's packet interval.

Best used for: WAN quality monitoring, VoIP readiness assessment, SLA compliance verification.

### UDP Jitter with Codec Simulation (VoIP Probes)

An extension of UDP jitter that simulates a specific voice codec's traffic pattern and calculates voice quality metrics directly. Supported codecs include G.711 a-law, G.711 mu-law, G.729A, and G.723.1.

The probe automatically sets the correct packet size and interval for the selected codec. For G.711 a-law with a 20ms packetization period, each probe packet is 172 bytes (160 bytes payload + 12 bytes RTP header) sent every 20ms.

From the measured jitter, latency, and packet loss, the probe calculates:

- **ICPIF (Calculated Planning Impairment Factor):** A numeric value where lower is better. Derived from the E-model (ITU-T G.107). Values below 20 indicate acceptable voice quality.
- **MOS (Mean Opinion Score):** Converted from ICPIF on a 1-5 scale. Values above 4.0 are considered good; below 3.6 is poor.

The advantage-factor compensates for user expectations. Users of mobile phones tolerate more impairment than users of landlines. Setting advantage-factor to 10 for a mobile VoIP deployment adjusts the MOS calculation upward.

### TCP Connect

Measures TCP three-way handshake time to a target host and port. The probe sends a SYN, measures the time to receive SYN-ACK, completes the handshake with ACK, then immediately tears down with RST. No application data is exchanged.

This probe does not require an IP SLA responder because it connects to a real TCP service. However, the probe generates real TCP connections that may appear in the target's connection logs and may trigger intrusion detection alerts if the port is unexpected.

Best used for: monitoring application server availability (HTTP/443, database ports), SYN flood detection (abnormally high TCP connect times suggest a saturated server).

### HTTP

Fetches a complete HTTP transaction and breaks down the response time into DNS lookup, TCP connect, HTTP transaction, and total time. Supports GET and raw request modes. The probe can follow redirects and validate response content.

The HTTP probe is heavier weight than other probe types because it involves full HTTP protocol processing on the router's control plane. Set frequency conservatively (60-120 seconds) to avoid excessive CPU load on the measuring router.

Best used for: web application monitoring, CDN availability verification, URL-based health checking.

### DNS

Sends a DNS query (A record lookup by default) to a specified DNS server and measures response time. Validates that the DNS server is responsive and returning results.

Best used for: DNS infrastructure monitoring, validating recursive resolver availability, measuring DNS resolution latency from branch sites.

### DHCP

Sends a DHCP DISCOVER message and measures the time to receive a DHCP OFFER. The probe does not complete the DHCP handshake (no REQUEST/ACK) and does not consume an IP address from the pool.

Best used for: DHCP server availability monitoring, detecting DHCP pool exhaustion (slow or no response).

### Path Echo and Path Jitter

Path echo performs a traceroute to the target and then sends ICMP echo probes to each discovered hop, measuring per-hop RTT. Path jitter extends this with jitter measurement per hop. These probes are useful for identifying which segment of a multi-hop path is introducing latency or jitter.

These probes are resource-intensive because they test every hop. Use them for troubleshooting rather than continuous monitoring.

---

## 4. Scheduling and Lifecycle

### Probe Lifecycle States

An IP SLA probe transitions through these states:

1. **Configured:** Probe parameters are set but no schedule is defined; the probe does not run
2. **Pending:** A schedule with start-time pending is configured; the probe is ready but waiting for a manual or time-based trigger
3. **Active:** The probe is running and collecting data
4. **Expired:** The probe's life timer has expired; it has stopped running but statistics remain in memory
5. **Aged out:** The ageout timer has expired after the probe stopped; all statistics are purged from memory

### Frequency vs. Timeout

The frequency parameter sets how often the probe runs (in seconds). The timeout parameter sets how long the probe waits for a response before declaring failure (in milliseconds).

Critical relationship: the timeout must be less than the frequency multiplied by 1000. If frequency is 10 seconds and timeout is 15000 milliseconds, the timeout exceeds the frequency and the next probe attempt will be delayed. Most implementations enforce timeout < frequency, but misconfiguration can lead to probe overlap and inaccurate measurements.

Recommended practice: set timeout to no more than half the frequency. For a 10-second frequency, use a 5000ms timeout. This provides ample headroom and ensures clean separation between probe cycles.

### Group Scheduling

When running many probes, simultaneous execution creates CPU spikes. Group scheduling distributes probe execution evenly across the schedule period.

For 10 probes with a 60-second frequency and 60-second schedule period: the scheduler spaces probes 6 seconds apart (60/10). Probe 1 fires at T+0, probe 2 at T+6, probe 3 at T+12, and so on. This smooths CPU utilization from a spike of 10 probes every 60 seconds to 1 probe every 6 seconds.

The schedule-period should match the frequency. If schedule-period is less than frequency, not all probes will run in each cycle. If schedule-period exceeds frequency, probes from consecutive cycles may overlap.

---

## 5. Thresholds and Reaction Framework

### Threshold Types

IP SLA supports three threshold evaluation methods:

**Immediate:** The reaction triggers on a single probe result exceeding the threshold. Fast response but susceptible to transient spikes causing false positives.

**Consecutive:** The reaction triggers only after N consecutive probe results exceed the threshold. More robust against transient conditions. A consecutive threshold of 3 with a 10-second frequency means a minimum 30-second detection time.

**Average:** The reaction triggers when the average over the last N probe results exceeds the threshold. Smooths out jitter in measurements but introduces latency in detection proportional to the averaging window.

### Reaction Actions

When a threshold is breached, IP SLA can take three types of action:

- **SNMP trap (trapOnly):** Sends an SNMP notification to the configured trap receiver. Requires SNMP infrastructure to be actionable. The trap includes the probe ID, the metric that triggered, and the measured value.
- **Trigger operation (triggerOnly):** Starts a secondary IP SLA probe (the "triggered" probe). Used for escalation — a failed ICMP echo probe can trigger a more detailed path-echo or UDP-jitter probe to gather diagnostic data.
- **Both (trapAndTrigger):** Sends a trap and starts a triggered probe simultaneously.

The most common operational pattern uses IP SLA reactions to change the state of a tracking object, which in turn modifies routing. This path does not use the reaction-configuration command directly but instead relies on the track object polling the IP SLA operation's return code.

---

## 6. Enhanced Object Tracking

### Track Object Model

Enhanced object tracking decouples the "what to monitor" decision from the "what to do about it" action. A track object monitors a condition and presents a simple up/down state. Consumers of the track object (static routes, HSRP, PBR, EEM) react to state changes without knowing what is being tracked.

Track objects can monitor:

- **IP SLA state:** Up when the probe's return code is OK; down when it is timeout or over-threshold
- **IP SLA reachability:** Up when the target is reachable; down on timeout (does not consider threshold violations)
- **Interface line-protocol:** Up when the interface is operationally up
- **Interface IP routing:** Up when the interface has a valid IP address and is participating in routing
- **IP route reachability:** Up when the specified prefix exists in the routing table
- **IP route metric threshold:** Up when the route's metric is within specified bounds

### Delay Timers

Track objects support delay timers that debounce state transitions:

- **delay down N:** The tracked object must be down for N seconds before the track transitions to down
- **delay up N:** The tracked object must be up for N seconds before the track transitions to up

These timers prevent routing churn caused by transient failures. Best practice is to set the up delay longer than the down delay. A down delay of 30 seconds catches genuine failures quickly while filtering brief glitches. An up delay of 60 seconds ensures the path has genuinely recovered before restoring it to service, preventing repeated failover/failback cycles during an intermittent fault.

### Boolean and Weighted Track Lists

Track lists combine multiple track objects using boolean logic or weighted thresholds:

**Boolean AND:** All member objects must be up for the list to be up. Useful when multiple conditions must simultaneously hold — for example, both the ISP link must be up AND the IP SLA probe to a target beyond the ISP must succeed.

**Boolean OR:** At least one member object must be up. Useful for redundant monitoring — if any one of several probes succeeds, the path is considered viable.

**Weighted threshold:** Each member object is assigned a numeric weight. The list is up when the sum of weights of up members meets the up-threshold. This allows sophisticated policies — a primary link with weight 60 and two backup links with weight 20 each. The list remains up as long as either the primary or both backups are working (threshold 40).

**Percentage threshold:** Similar to weighted but uses the percentage of up members. If 4 objects are tracked and the up threshold is 75%, at least 3 must be up.

---

## 7. Integration Patterns

### Static Route Failover

The most common IP SLA use case. A tracked static route is installed in the routing table only when its track object is in the up state. When the track transitions to down, the route is withdrawn and a backup route (with higher administrative distance) takes over.

Design considerations:

- **Probe target selection:** Probe a target beyond the ISP, not the ISP gateway. The ISP gateway may respond while upstream connectivity is broken. Public DNS servers (8.8.8.8, 1.1.1.1) are popular targets but may be rate-limited. Dedicated monitoring endpoints are preferable when available.
- **Probe source interface:** The probe must be sourced from the interface connected to the ISP being tested. If the probe's source is not pinned, a default route change during failover can cause the probe to exit via the backup ISP, creating a feedback loop where the primary probe succeeds via the backup path and restores the primary route, which causes the probe to fail again.
- **Pinned probe routes:** Add a host route for the probe target pointing at the specific ISP gateway. This ensures the probe always exits via the intended path regardless of default route changes.

### HSRP/VRRP Priority Adjustment

IP SLA tracking integrates with Hot Standby Router Protocol and Virtual Router Redundancy Protocol to trigger priority decrements when monitored paths fail. The active router decrements its priority by a configured amount when a track object goes down. If the decremented priority falls below the standby router's priority, preemption causes a failover.

The decrement value must be calculated carefully. If the active router has priority 110, the standby has priority 100, and a single link failure should trigger failover, the decrement must be at least 11 (dropping active to 99, below standby's 100). Multiple track objects with individual decrements allow granular control — losing one link reduces priority but does not trigger failover; losing two links drops priority below the threshold.

### Policy-Based Routing with Verification

PBR's set ip next-hop verify-availability command integrates with track objects. Traffic matching the route-map is forwarded to the specified next-hop only if its track object is up. Multiple next-hops can be configured with sequence numbers; the router uses the first one whose track is up.

This pattern is more flexible than static route failover for specific traffic flows. For example, VoIP traffic can be policy-routed via a low-latency MPLS path (tracked by a UDP jitter probe), while bulk data uses a cheaper internet path (tracked by an ICMP echo probe). If the MPLS path degrades, VoIP traffic fails over to the internet path while bulk traffic is unaffected.

### EEM Automation

Embedded Event Manager applets can be triggered by track state changes, enabling complex automated responses beyond routing changes:

- Shut down interfaces to force protocol reconvergence
- Send email alerts to NOC teams
- Execute diagnostic commands and log the output
- Modify QoS policies to prioritize traffic during degraded conditions
- Generate syslog messages with custom severity levels
- Execute Tcl scripts for complex decision logic

The track event trigger provides the track number and new state (up or down) to the EEM applet, allowing different actions for failure and recovery.

---

## 8. VoIP Quality Measurement

### The E-Model and MOS

The E-model (ITU-T G.107) is a computational model for predicting voice quality. It produces an R-factor (0-100) that accounts for:

- Codec impairment (fixed per codec)
- Delay impairment (increases with one-way delay)
- Loss impairment (increases with packet loss)
- Jitter impairment (derived from jitter buffer size and overflow rate)
- Advantage factor (user tolerance based on access type)

The R-factor is converted to MOS using a defined formula:

| R-Factor | MOS | User Satisfaction |
|:---:|:---:|:---|
| 90-100 | 4.3-4.5 | Very satisfied |
| 80-90 | 4.0-4.3 | Satisfied |
| 70-80 | 3.6-4.0 | Some users dissatisfied |
| 60-70 | 3.1-3.6 | Many users dissatisfied |
| < 60 | < 3.1 | Nearly all users dissatisfied |

### ICPIF Calculation

IP SLA calculates the ICPIF (Calculated Planning Impairment Factor) from measured network characteristics:

ICPIF = Icodec + Idelay + Iloss - A

Where:
- Icodec = impairment from codec compression (fixed per codec; G.711=0, G.729=11)
- Idelay = impairment from one-way delay (increases non-linearly above 150ms)
- Iloss = impairment from packet loss (increases rapidly; 1% loss is noticeable)
- A = advantage factor (compensation for user expectations)

| ICPIF Value | Quality |
|:---:|:---|
| 0-10 | Excellent |
| 11-20 | Good |
| 21-30 | Adequate |
| 31-40 | Poor |
| > 40 | Unacceptable |

### Codec Parameter Summary

| Codec | Bandwidth | Packetization | Payload Size | MOS (optimal) |
|:---|:---:|:---:|:---:|:---:|
| G.711 a-law | 64 kbps | 20 ms | 160 bytes | 4.4 |
| G.711 mu-law | 64 kbps | 20 ms | 160 bytes | 4.4 |
| G.729A | 8 kbps | 20 ms | 20 bytes | 3.9 |
| G.723.1 (6.3k) | 6.3 kbps | 30 ms | 24 bytes | 3.7 |

---

## 9. Dual-ISP Failover Design

### Architecture

A dual-ISP site connects to two Internet service providers. The goal is automatic failover: when the primary ISP fails, traffic shifts to the backup ISP without manual intervention. IP SLA provides the failure detection mechanism; enhanced object tracking provides the decision logic; tracked static routes provide the routing change.

### Design Components

**Probe placement:** Each ISP link gets its own IP SLA probe. The probe target should be beyond the ISP's edge router — a publicly reachable IP that routes independently of either ISP. Using a target within the ISP's own network only detects failures within that ISP; using a target that routes through both ISPs means a failure of one ISP could still allow the probe to succeed via the other.

**Source pinning:** Each probe must be sourced from the interface connected to its respective ISP. Without source pinning, a probe intended for ISP1 could route out ISP2 after a default route change, producing misleading results.

**Route pinning:** A static host route for each probe target, pointing at the respective ISP gateway, ensures probe traffic always uses the intended path regardless of default route state.

**Track delay timers:** Down delay prevents momentary packet loss from triggering failover. Up delay prevents premature failback during an intermittent fault. Typical values: down 30 seconds, up 60 seconds.

**Administrative distance:** The primary default route has AD 1 (default for static routes). The backup default route has AD 10 (or any value higher than 1). When the primary route is withdrawn (track goes down), the backup route is automatically installed.

### Failure Scenarios

| Scenario | ISP1 Probe | ISP2 Probe | Track 1 | Track 2 | Active Route |
|:---|:---:|:---:|:---:|:---:|:---|
| Both ISPs up | OK | OK | Up | Up | ISP1 (lower AD) |
| ISP1 down | Timeout | OK | Down | Up | ISP2 |
| ISP2 down | OK | Timeout | Up | Down | ISP1 |
| Both down | Timeout | Timeout | Down | Down | No default route |

### Convergence Timeline

The total failover time from ISP failure to traffic flowing via backup:

1. Probe detection: frequency x consecutive threshold (e.g., 10s x 3 = 30s)
2. Track delay down timer (e.g., 30s)
3. Route table update (near-instant once track transitions)
4. ARP resolution for new next-hop if not cached (1-2s)

Total: approximately 60-62 seconds with conservative settings. Can be reduced to 15-20 seconds with aggressive probe frequency (5s) and shorter delay timers (10s down), at the cost of increased susceptibility to false positives.

---

## 10. NX-OS Platform Differences

### Feature Enablement

Unlike IOS where IP SLA is available by default, NX-OS requires explicit feature enablement:

- `feature sla sender` — enables the ability to configure and run IP SLA probes
- `feature sla responder` — enables the IP SLA responder function

Without these features enabled, IP SLA commands are not available in the configuration mode.

### Supported Probe Types

NX-OS supports a subset of IOS IP SLA probe types. ICMP echo, UDP jitter, TCP connect, and DNS probes are generally available. HTTP and DHCP probes may have limited or no support depending on the NX-OS version and platform.

### Configuration Syntax Differences

NX-OS uses the same general structure as IOS but with some syntax variations:

- Routes use CIDR notation: `ip route 0.0.0.0/0` instead of `ip route 0.0.0.0 0.0.0.0`
- Track objects and IP SLA commands follow the same syntax as IOS in most cases
- Verification commands are identical: `show ip sla configuration`, `show ip sla statistics`, `show track`

### VDC and VRF Considerations

On Nexus platforms with Virtual Device Contexts, IP SLA probes operate within the VDC where they are configured. Probes cannot cross VDC boundaries. For VRF-aware probes, specify the VRF within the probe configuration to ensure packets are generated in the correct routing context.

---

## 11. Scaling and Performance Considerations

### Control Plane Impact

IP SLA probes are processed by the router's control plane (Route Processor). Each probe consumes CPU cycles for packet generation, timestamp processing, metric calculation, and statistics storage. The aggregate impact scales with:

- Number of active probes
- Probe frequency (shorter intervals = more CPU per second)
- Probe type complexity (UDP jitter with 100 packets > ICMP echo with 1 packet)
- Statistics and history retention depth

### Scaling Guidelines

| Platform Class | Maximum Probes | Recommended Probes | Minimum Frequency |
|:---|:---:|:---:|:---:|
| Low-end (ISR 1000) | 50 | 20-30 | 30 sec |
| Mid-range (ISR 4000) | 500 | 100-200 | 10 sec |
| High-end (ASR 1000) | 2000 | 500-1000 | 5 sec |
| Nexus 9000 | 500 | 100-200 | 10 sec |

These are approximate guidelines. Actual limits depend on the router's overall CPU load from routing protocols, management sessions, ACLs, and other features. Monitor CPU utilization when deploying IP SLA at scale.

### Memory Consumption

Each IP SLA operation consumes memory for:

- Configuration data (~1-2 KB per probe)
- Statistics buffer (~4-8 KB per probe, scales with history depth)
- Enhanced history (~16-32 KB per probe if enabled)

Use the ageout parameter to automatically purge statistics from probes that have expired, preventing memory accumulation from one-time diagnostic probes.

---

## 12. Operational Best Practices

### Probe Design

1. **Choose the right probe type.** Use ICMP echo for simple reachability. Use UDP jitter for WAN quality measurement. Use TCP connect for application port monitoring. Do not use a complex probe when a simple one suffices.

2. **Select meaningful targets.** The probe target should represent what you are trying to protect. For internet failover, probe a target beyond both ISPs. For branch-to-datacenter monitoring, probe the actual application server or its load balancer VIP.

3. **Pin probe sources and routes.** Always specify source-ip or source-interface. Always create host routes for probe targets pointing at the intended gateway. This prevents feedback loops during failover.

4. **Use appropriate thresholds.** Set thresholds based on measured baselines, not guesses. Run probes for a week without reactions to establish normal RTT, jitter, and loss patterns, then set thresholds at 2-3x the normal values.

### Track Object Design

5. **Always use delay timers.** Bare track objects with no delay timers react to every transient condition. Use delay down 30 up 60 as a starting point and adjust based on operational experience.

6. **Use track lists for complex decisions.** Combine IP SLA tracking with interface tracking using boolean logic. A track that requires both the interface to be up AND the probe to succeed prevents the probe from falsely reporting success when the interface is administratively down.

7. **Document track object assignments.** Track objects are identified by number. Maintain a tracking plan that maps track numbers to their purpose, associated probes, and consuming features (routes, HSRP, PBR).

### Monitoring and Maintenance

8. **Monitor IP SLA health.** Regularly check `show ip sla statistics` for unexpected timeout rates. A probe that times out 5% of the time may indicate an underlying issue even if it has not triggered failover.

9. **Audit probe targets.** Verify that probe targets remain valid. A decommissioned server used as a probe target will cause permanent track-down state and route withdrawal.

10. **Correlate with NTP.** Ensure all devices running IP SLA are synchronized to the same NTP source. While jitter measurement does not require synchronized clocks, one-way delay values and cross-device metric comparison are meaningless without accurate time.

---

## Prerequisites

- Familiarity with static routing and administrative distance concepts
- Understanding of HSRP/VRRP operation for FHRP integration scenarios
- Basic SNMP knowledge for trap-based alerting
- Understanding of TCP/IP fundamentals (ICMP, UDP, TCP handshake)
- Access to Cisco IOS 12.4+ or IOS-XE 3.x+ or NX-OS 7.x+ for lab exercises
- NTP configuration for accurate timestamp-based measurements

## References

- [ITU-T G.107 — The E-model: A Computational Model for Use in Transmission Planning](https://www.itu.int/rec/T-REC-G.107)
- [ITU-T G.114 — One-Way Transmission Time](https://www.itu.int/rec/T-REC-G.114)
- [RFC 2925 — Definitions of Managed Objects for Remote Ping, Traceroute, and Lookup Operations](https://www.rfc-editor.org/rfc/rfc2925)
- [RFC 4710 — Real-Time Application Quality-of-Service Monitoring (RAQMON)](https://www.rfc-editor.org/rfc/rfc4710)
- [RFC 5357 — A Two-Way Active Measurement Protocol (TWAMP)](https://www.rfc-editor.org/rfc/rfc5357)
- [Cisco IOS IP SLA Configuration Guide](https://www.cisco.com/c/en/us/td/docs/ios-xml/ios/ipsla/configuration/xe-16/sla-xe-16-book.html)
- [Cisco IOS IP SLA Command Reference](https://www.cisco.com/c/en/us/td/docs/ios-xml/ios/ipsla/command/sla-cr-book.html)
- [Cisco Enhanced Object Tracking Configuration Guide](https://www.cisco.com/c/en/us/td/docs/ios-xml/ios/ipapp_eot/configuration/xe-16/ipapp-eot-xe-16-book.html)
- [Cisco NX-OS IP SLA Configuration Guide](https://www.cisco.com/c/en/us/td/docs/switches/datacenter/nexus9000/sw/93x/ip-sla/configuration/guide/b-cisco-nexus-9000-series-nx-os-ip-sla-configuration-guide-93x.html)
- [Cisco EEM Configuration Guide](https://www.cisco.com/c/en/us/td/docs/ios-xml/ios/eem/configuration/xe-16/eem-xe-16-book.html)
