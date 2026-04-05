# IP SLA (Service Level Agreement Probes and Tracking)

Active measurement framework for monitoring network performance metrics (latency, jitter, packet loss, reachability) by generating synthetic traffic between devices, enabling automated failover when thresholds are breached.

## Concepts

### Probe Types

- **ICMP echo:** Basic reachability and round-trip time measurement; simplest probe type, no responder needed
- **UDP jitter:** Measures one-way delay, jitter, and packet loss using timestamped UDP packets; requires IP SLA responder on the target
- **TCP connect:** Tests TCP handshake to a specific port; measures connection setup time; no responder needed if the port is open
- **HTTP:** Fetches a URL and measures DNS lookup time, TCP connect time, HTTP transaction time, and total RTT
- **DNS:** Sends a DNS query to a server and measures response time; validates name resolution availability
- **DHCP:** Sends a DHCP DISCOVER and measures the time to receive an OFFER; tests DHCP service health
- **FTP:** Tests FTP server reachability and file transfer performance
- **VoIP (UDP jitter with codec):** Simulates voice traffic with specific codec parameters (G.711, G.729) to calculate MOS scores and ICPIF values
- **Path echo:** Discovers the path via traceroute and measures hop-by-hop latency
- **Path jitter:** Measures jitter on each hop along a path

### Key Metrics

- **RTT (Round-Trip Time):** Total time for probe packet to reach target and return
- **Jitter:** Variation in one-way delay between consecutive packets
- **MOS (Mean Opinion Score):** Voice quality rating derived from jitter, latency, and packet loss (1.0-5.0 scale)
- **ICPIF (Calculated Planning Impairment Factor):** Numeric voice quality impairment value; lower is better
- **Packet loss:** Percentage of probe packets that fail to return

### IP SLA Responder

- Dedicated listener on the target device that timestamps packets for accurate one-way measurements
- Required for UDP jitter, UDP echo, and TCP connect probes to non-application ports
- Not required for ICMP echo, HTTP, DNS, or DHCP probes (they use real services)
- Responder listens on a configurable port and handles IP SLA control messages

## IOS Configuration

### ICMP Echo Probe

```
! Basic ICMP echo probe to monitor reachability
ip sla 1
 icmp-echo 10.0.0.1 source-ip 192.168.1.1
 ! Probe frequency in seconds
 frequency 10
 ! Timeout before declaring failure (milliseconds)
 timeout 5000
 ! Type of Service byte for QoS marking
 tos 160
 ! Size of the probe packet in bytes
 request-data-size 64
 ! Tag for identification in show commands
 tag INTERNET-GW-MONITOR
 ! Threshold for rising-threshold reaction (milliseconds)
 threshold 1000
 ! Owner string for administrative tracking
 owner NETWORK-OPS

! Schedule the probe
ip sla schedule 1 life forever start-time now
```

### UDP Jitter Probe

```
! Enable responder on target device first
ip sla responder

! On the source device: UDP jitter probe
ip sla 10
 udp-jitter 10.0.0.2 16384 source-ip 192.168.1.1
 ! Number of packets per test
 num-packets 100
 ! Interval between packets (milliseconds)
 interval 20
 frequency 60
 timeout 5000
 ! Verify data integrity
 verify-data
 ! Set probe packet size
 request-data-size 172
 ! Precision for jitter values
 precision microseconds
 tag WAN-JITTER-MONITOR

ip sla schedule 10 life forever start-time now
```

### VoIP Probe with Codec Simulation

```
! UDP jitter probe simulating G.711 codec for MOS calculation
ip sla 20
 udp-jitter 10.0.0.3 16386 codec g711alaw
 ! Codec advantage factor for MOS calculation
 advantage-factor 0
 frequency 60
 tag VOIP-QUALITY

ip sla schedule 20 life forever start-time now
```

### TCP Connect Probe

```
! Monitor web server availability
ip sla 30
 tcp-connect 10.0.0.4 443
 frequency 30
 timeout 5000
 tag HTTPS-SERVER-CHECK

ip sla schedule 30 life forever start-time now
```

### HTTP Probe

```
! Monitor web application response
ip sla 40
 http get http://10.0.0.5/health
 ! HTTP version
 http-raw-request "GET /health HTTP/1.1\r\nHost: app.example.com\r\n\r\n"
 frequency 60
 timeout 10000
 tag APP-HEALTH

ip sla schedule 40 life forever start-time now
```

### DNS Probe

```
! Monitor DNS resolution
ip sla 50
 dns www.example.com name-server 8.8.8.8
 frequency 30
 timeout 5000
 tag DNS-CHECK

ip sla schedule 50 life forever start-time now
```

### DHCP Probe

```
! Monitor DHCP service availability
ip sla 60
 dhcp 10.0.0.6
 frequency 120
 timeout 10000
 tag DHCP-CHECK

ip sla schedule 60 life forever start-time now
```

## Scheduling

### Start-Time Options

```
! Start immediately
ip sla schedule 1 life forever start-time now

! Start at a specific time
ip sla schedule 1 life forever start-time 08:00:00 Jan 1 2026

! Start after a delay (seconds)
ip sla schedule 1 life forever start-time after 00:05:00

! Pending (configured but not running)
ip sla schedule 1 life forever start-time pending
```

### Life and Ageout

```
! Run forever
ip sla schedule 1 life forever start-time now

! Run for a specific duration (seconds)
ip sla schedule 1 life 86400 start-time now

! Ageout: remove probe from memory after being inactive (seconds)
ip sla schedule 1 life 3600 ageout 7200 start-time now

! Recurring: restart the probe when life expires
ip sla schedule 1 life 3600 recurring start-time now
```

### Scheduling Multiple Probes

```
! Group scheduling for multiple probes
ip sla group schedule 100 1-10 schedule-period 60 frequency 60 life forever start-time now
! Distributes 10 probes (IDs 1-10) evenly across a 60-second period
! Prevents all probes from firing simultaneously
```

## Thresholds and Reactions

### Reaction Configuration (IOS 15+)

```
! React when RTT exceeds threshold
ip sla reaction-configuration 1 react rtt threshold-type immediate threshold-value 500 action-type trapAndTrigger

! React on timeout (unreachable)
ip sla reaction-configuration 1 react timeout threshold-type immediate action-type trapAndTrigger

! React on jitter exceeding threshold
ip sla reaction-configuration 10 react jitterSDAvg threshold-type immediate threshold-value 30 action-type trapAndTrigger

! React on packet loss
ip sla reaction-configuration 10 react PacketLossSD threshold-type immediate threshold-value 5 action-type trapAndTrigger

! React on MOS falling below threshold
ip sla reaction-configuration 20 react MOS threshold-type immediate threshold-value 350 action-type trapAndTrigger

! React after consecutive failures
ip sla reaction-configuration 1 react timeout threshold-type consecutive 3 action-type trapAndTrigger

! React on average over X operations
ip sla reaction-configuration 1 react rtt threshold-type average 5 action-type trapAndTrigger
```

### Reaction Actions

- **trapOnly:** Send an SNMP trap
- **triggerOnly:** Activate a triggered probe
- **trapAndTrigger:** Both SNMP trap and trigger
- **none:** No action (just record)

## Tracking Objects

### Basic IP SLA Tracking

```
! Track the state of IP SLA probe 1 (up/down based on return code)
track 1 ip sla 1 state

! Track reachability of IP SLA probe 1
track 2 ip sla 1 reachability

! Track with delay (avoid flapping)
track 1 ip sla 1 state
 delay down 30 up 60
```

### Threshold-Based Tracking

```
! Track if RTT exceeds threshold (probe is "down" when over threshold)
track 3 ip sla 10 state
 delay down 10 up 30
```

### Boolean Track Lists

```
! Track list using AND logic (all must be up)
track 10 list boolean and
 object 1
 object 2
 object 3

! Track list using OR logic (any one must be up)
track 20 list boolean or
 object 1
 object 2

! Track list with weighted threshold
track 30 list threshold weight
 object 1 weight 50
 object 2 weight 30
 object 3 weight 20
 threshold weight up 60 down 40
! Object weight must sum to >= up-threshold for track to be up

! Track list with percentage threshold
track 40 list threshold percentage
 object 1
 object 2
 object 3
 object 4
 threshold percentage up 75 down 25
```

## Integration with Static Routes

### Tracked Static Routes

```
! Primary route via ISP1 (tracked by IP SLA probe)
ip route 0.0.0.0 0.0.0.0 203.0.113.1 track 1

! Backup route via ISP2 (higher AD, installed when track 1 goes down)
ip route 0.0.0.0 0.0.0.0 198.51.100.1 10

! Or both tracked for mutual failover
ip route 0.0.0.0 0.0.0.0 203.0.113.1 track 1
ip route 0.0.0.0 0.0.0.0 198.51.100.1 track 2
```

### Dual-ISP Failover (Complete Example)

```
! --- ISP1 probe ---
ip sla 1
 icmp-echo 8.8.8.8 source-interface GigabitEthernet0/0
 frequency 10
 timeout 2000
 threshold 1000
 tag ISP1-PROBE
ip sla schedule 1 life forever start-time now

! --- ISP2 probe ---
ip sla 2
 icmp-echo 8.8.4.4 source-interface GigabitEthernet0/1
 frequency 10
 timeout 2000
 threshold 1000
 tag ISP2-PROBE
ip sla schedule 2 life forever start-time now

! --- Track objects with delay to prevent flapping ---
track 1 ip sla 1 reachability
 delay down 30 up 60
track 2 ip sla 2 reachability
 delay down 30 up 60

! --- Tracked default routes ---
ip route 0.0.0.0 0.0.0.0 203.0.113.1 track 1
ip route 0.0.0.0 0.0.0.0 198.51.100.1 10 track 2

! --- Probe target routes (force probe traffic out correct interface) ---
ip route 8.8.8.8 255.255.255.255 203.0.113.1
ip route 8.8.4.4 255.255.255.255 198.51.100.1
```

## Integration with HSRP

```
! HSRP priority decrement when tracked object goes down
interface GigabitEthernet0/0
 standby 1 ip 192.168.1.1
 standby 1 priority 110
 standby 1 preempt delay reload 60
 standby 1 track 1 decrement 20
 ! If track 1 goes down: priority drops from 110 to 90
 ! Standby router (priority 100) takes over

! Track multiple objects
interface GigabitEthernet0/0
 standby 1 ip 192.168.1.1
 standby 1 priority 120
 standby 1 preempt
 standby 1 track 1 decrement 15
 standby 1 track 2 decrement 15
 ! Both links down: 120 - 15 - 15 = 90 (below standby's 100)
```

## Integration with PBR (Policy-Based Routing)

```
! Policy route with tracked next-hop
route-map PBR-TRAFFIC permit 10
 match ip address INTERESTING-TRAFFIC
 set ip next-hop verify-availability 203.0.113.1 1 track 1
 set ip next-hop verify-availability 198.51.100.1 2 track 2

! Apply to interface
interface GigabitEthernet0/2
 ip policy route-map PBR-TRAFFIC
```

## EEM Integration with IP SLA

```
! EEM applet triggered by IP SLA track state change
track 1 ip sla 1 reachability
 delay down 30 up 60

event manager applet ISP1-DOWN
 event track 1 state down
 action 1.0 syslog msg "ISP1 link DOWN - failover to ISP2"
 action 2.0 cli command "enable"
 action 3.0 cli command "configure terminal"
 action 4.0 cli command "interface GigabitEthernet0/0"
 action 5.0 cli command "shutdown"
 action 6.0 cli command "end"
 action 7.0 mail server "10.0.0.10" to "noc@example.com" from "router@example.com" subject "ISP1 FAILOVER" body "ISP1 link failure detected. Failover to ISP2 activated."

event manager applet ISP1-UP
 event track 1 state up
 action 1.0 syslog msg "ISP1 link UP - restoring primary path"
 action 2.0 cli command "enable"
 action 3.0 cli command "configure terminal"
 action 4.0 cli command "interface GigabitEthernet0/0"
 action 5.0 cli command "no shutdown"
 action 6.0 cli command "end"
```

## NX-OS Configuration

### ICMP Echo Probe

```
! NX-OS IP SLA configuration
feature sla sender

ip sla 1
 icmp-echo 10.0.0.1 source-ip 192.168.1.1
 frequency 10
 timeout 5000

ip sla schedule 1 life forever start-time now

! NX-OS tracking
feature sla responder

track 1 ip sla 1 reachability
 delay down 30 up 60

ip route 0.0.0.0/0 203.0.113.1 track 1
ip route 0.0.0.0/0 198.51.100.1 5
```

### NX-OS UDP Jitter

```
feature sla sender

ip sla 10
 udp-jitter 10.0.0.2 16384 source-ip 192.168.1.1
 num-packets 100
 interval 20
 frequency 60
 timeout 5000

ip sla schedule 10 life forever start-time now
```

## Show Commands

### Probe Status and Results

```bash
# Show all configured IP SLA operations
show ip sla configuration

# Show specific probe configuration
show ip sla configuration 1

# Show current operational state
show ip sla operational-state

# Show probe statistics summary
show ip sla statistics

# Show detailed statistics for a specific probe
show ip sla statistics 1

# Show aggregated statistics over time
show ip sla statistics aggregated 1

# Show enhanced history
show ip sla enhanced-history 1

# Show reaction configuration
show ip sla reaction-configuration

# Show reaction trigger information
show ip sla reaction-trigger

# Show IP SLA responder status
show ip sla responder

# Show all tracking objects
show track

# Show specific track object
show track 1

# Show brief track summary
show track brief

# Show track interface brief (useful for HSRP integration)
show track interface brief
```

### NX-OS Show Commands

```bash
# NX-OS equivalents
show ip sla configuration
show ip sla statistics
show ip sla operational-state
show track
show track brief
```

### Debugging

```bash
# Debug IP SLA operations (use with caution in production)
debug ip sla trace
debug ip sla error

# Debug tracking
debug track

# Verify probe is sending packets
show ip sla statistics 1 | include Number of successes|Number of failures

# Check for timeout issues
show ip sla statistics 1 | include Timeout|RTT

# Verify track state changes
show track 1 | include State|Latest
```

## Enhanced Object Tracking

### Interface Tracking

```
! Track interface line-protocol state
track 50 interface GigabitEthernet0/0 line-protocol

! Track interface IP routing state
track 51 interface GigabitEthernet0/0 ip routing
```

### IP Route Tracking

```
! Track if a route exists in the routing table
track 60 ip route 10.0.0.0/24 reachability

! Track route with metric threshold
track 61 ip route 10.0.0.0/24 metric threshold
 threshold metric up 100 down 200
```

### Combining Track Types

```
! Combine IP SLA tracking with interface tracking
track 1 ip sla 1 reachability
track 50 interface GigabitEthernet0/0 line-protocol

! Boolean AND: both probe succeeds AND interface is up
track 100 list boolean and
 object 1
 object 50
 delay down 10 up 30

! Use combined track for routing decision
ip route 0.0.0.0 0.0.0.0 203.0.113.1 track 100
```

## Tips

- Always pin probe target routes with static routes pointing to the correct ISP gateway; without them, a default route change can cause the probe itself to reroute through the wrong path and give false results.
- Use delay timers on track objects (delay down 30 up 60) to prevent flapping during transient failures; the up delay should be longer than the down delay to confirm genuine recovery.
- The IP SLA responder must be enabled on the target for UDP jitter probes; ICMP echo probes work without a responder because they use the standard ICMP stack.
- Set the probe frequency higher than the timeout value; if timeout is 5000ms and frequency is 10s, the probe has 5 seconds to complete before the next one starts.
- For dual-ISP failover, probe a target beyond the ISP (like 8.8.8.8) rather than the ISP gateway itself; the gateway might respond while upstream connectivity is broken.
- Group scheduling distributes probe execution across the schedule period to avoid CPU spikes; use it when running more than 10 simultaneous probes.
- VoIP probes with codec simulation provide MOS scores directly on the router; use advantage-factor to account for user expectation in specific environments.
- Track lists with weighted thresholds allow sophisticated failover logic; assign higher weights to more important paths to control when failover triggers.
- EEM applets triggered by track state changes can perform automated remediation beyond simple route withdrawal, including interface shutdown, email alerts, and configuration changes.
- On NX-OS, enable the sla sender and sla responder features before configuring probes; they are not enabled by default unlike IOS.

## See Also

- bfd, hsrp, vrrp, pbr, eigrp, ospf, bgp, eem, snmp

## References

- [Cisco IOS IP SLA Configuration Guide](https://www.cisco.com/c/en/us/td/docs/ios-xml/ios/ipsla/configuration/xe-16/sla-xe-16-book.html)
- [Cisco NX-OS IP SLA Configuration Guide](https://www.cisco.com/c/en/us/td/docs/switches/datacenter/nexus9000/sw/93x/ip-sla/configuration/guide/b-cisco-nexus-9000-series-nx-os-ip-sla-configuration-guide-93x.html)
- [Cisco Enhanced Object Tracking Configuration Guide](https://www.cisco.com/c/en/us/td/docs/ios-xml/ios/ipapp_eot/configuration/xe-16/ipapp-eot-xe-16-book.html)
- [RFC 2925 — Definitions of Managed Objects for Remote Ping, Traceroute, and Lookup Operations](https://www.rfc-editor.org/rfc/rfc2925)
- [RFC 4710 — Real-Time Application Quality-of-Service Monitoring (RAQMON)](https://www.rfc-editor.org/rfc/rfc4710)
- [Cisco IP SLA Command Reference](https://www.cisco.com/c/en/us/td/docs/ios-xml/ios/ipsla/command/sla-cr-book.html)
- [Cisco EEM Configuration Guide](https://www.cisco.com/c/en/us/td/docs/ios-xml/ios/eem/configuration/xe-16/eem-xe-16-book.html)
