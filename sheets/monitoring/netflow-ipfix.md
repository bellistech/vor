# NetFlow / IPFIX (Network Traffic Flow Analysis)

Export and collect IP traffic flow records for visibility into who is talking to whom, how much, and when.

## Flow Concepts

### What is a flow

```bash
# A flow = unidirectional sequence of packets sharing:
#   Source IP, Destination IP, Source Port, Destination Port,
#   IP Protocol, Type of Service (ToS), Input Interface
# This 7-tuple is the "flow key" in NetFlow v5

# Flow lifecycle:
#   1. First packet arrives → new flow created in cache
#   2. Subsequent matching packets → counters updated
#   3. Flow expires → exported to collector
```

### Flow expiration triggers

```bash
# Active timeout   — flow has been active for N seconds (default 1800s / 30 min)
# Inactive timeout — no packets for N seconds (default 15s)
# Cache full       — oldest flow evicted
# TCP FIN/RST      — connection teardown detected
# Forced export    — manual cache clear or config change
```

## NetFlow v5 (Fixed Format)

### Enable NetFlow v5 on IOS

```bash
# Enter interface
interface GigabitEthernet0/0
 ip flow ingress                              # capture incoming traffic
 ip flow egress                               # capture outgoing traffic (optional)
 exit

# Global configuration
ip flow-export version 5                      # set export version
ip flow-export destination 10.0.1.100 2055    # collector IP and UDP port
ip flow-export source Loopback0               # source IP for export packets
ip flow-cache timeout active 1                # active timeout in minutes
ip flow-cache timeout inactive 15             # inactive timeout in seconds
```

### Show NetFlow v5 status

```bash
show ip flow export                           # export config and stats
show ip flow interface                        # interfaces with flow enabled
show ip cache flow                            # flow cache contents (top talkers)
show ip cache verbose flow                    # detailed cache with per-flow info
```

### v5 packet format (48-byte header + 48-byte records)

```bash
# Header fields (24 bytes):
#   version(2) count(2) sysUptime(4) unix_secs(4) unix_nsecs(4)
#   flow_sequence(4) engine_type(1) engine_id(1) sampling(2)

# Record fields (48 bytes each, up to 30 per packet):
#   srcaddr(4) dstaddr(4) nexthop(4) input(2) output(2)
#   dPkts(4) dOctets(4) first(4) last(4)
#   srcport(2) dstport(2) pad1(1) tcp_flags(1) prot(1) tos(1)
#   src_as(2) dst_as(2) src_mask(1) dst_mask(1) pad2(2)
```

## NetFlow v9 (Template-Based)

### Enable NetFlow v9 on IOS

```bash
ip flow-export version 9                      # template-based format
ip flow-export destination 10.0.1.100 9996    # collector IP and port
ip flow-export source Loopback0
ip flow-export template refresh-rate 20       # resend template every 20 packets
ip flow-export template timeout-rate 30       # resend template every 30 minutes

interface GigabitEthernet0/0
 ip flow ingress
 ip flow egress
```

### v9 template structure

```bash
# Template FlowSet — defines field layout:
#   Template ID (256-65535), Field Count, then Field Type + Field Length pairs
#
# Data FlowSet — actual flow records referencing a Template ID
#
# Options Template — router metadata (sampling info, interface names)

# Common field types:
#   1  = IN_BYTES          2  = IN_PKTS         4  = PROTOCOL
#   7  = L4_SRC_PORT       8  = IPV4_SRC_ADDR   11 = L4_DST_PORT
#   12 = IPV4_DST_ADDR     14 = OUTPUT_SNMP      15 = IPV4_NEXT_HOP
#   21 = LAST_SWITCHED     22 = FIRST_SWITCHED   34 = SAMPLING_INTERVAL
```

## IPFIX (RFC 7011)

### Key differences from NetFlow v9

```bash
# IPFIX = "IP Flow Information Export" (IETF standard)
# Based on NetFlow v9 but with important improvements:
#   - SCTP, TCP, and UDP transport (v9 only UDP)
#   - Variable-length fields (v9 fixed only)
#   - Enterprise-specific Information Elements (vendor extensions)
#   - Structured data types (basicList, subTemplateList)
#   - Standardized Information Element registry (IANA)
#   - Version field = 10 (v9 = 9)
#   - Template withdrawal messages
#   - Options Template for metering process metadata

# IPFIX default port: TCP/UDP 4739, SCTP 4740
# NetFlow v9 default port: UDP 9996 (or 2055)
```

### IPFIX message format

```bash
# Message header (16 bytes):
#   Version(2)=10  Length(2)  ExportTime(4)  SequenceNumber(4)  ObservationDomainID(4)

# Set types:
#   Set ID 2 = Template Set
#   Set ID 3 = Options Template Set
#   Set ID >= 256 = Data Set (references a Template ID)
```

## Flexible NetFlow (IOS-XE / IOS 15+)

### Define a flow record

```bash
flow record CUSTOM-RECORD
 description Custom traffic analysis record
 match ipv4 source address                    # flow key field
 match ipv4 destination address
 match ipv4 protocol
 match transport source-port
 match transport destination-port
 match interface input
 match ipv4 tos
 collect counter bytes long                   # non-key collected field
 collect counter packets long
 collect timestamp sys-uptime first
 collect timestamp sys-uptime last
 collect ipv4 dscp
 collect ipv4 ttl minimum
 collect ipv4 ttl maximum
 collect transport tcp flags
 collect interface output
 collect routing next-hop address ipv4
```

### Define a flow exporter

```bash
flow exporter EXPORT-TO-COLLECTOR
 description Send flows to central collector
 destination 10.0.1.100
 source Loopback0
 transport udp 9996
 export-protocol netflow-v9                   # or ipfix
 template data timeout 60                     # template refresh interval
 option interface-table                       # export interface names
 option sampler-table                         # export sampling config
 option application-table                     # export NBAR app info
```

### Define a flow monitor

```bash
flow monitor TRAFFIC-MONITOR
 description Monitor all traffic flows
 record CUSTOM-RECORD                         # reference the record
 exporter EXPORT-TO-COLLECTOR                 # reference the exporter
 cache timeout active 60                      # active timeout seconds
 cache timeout inactive 15                    # inactive timeout seconds
 cache entries 16384                          # max flow cache entries
 statistics packet protocol                   # enable protocol stats
```

### Apply flow monitor to interface

```bash
interface GigabitEthernet0/0/0
 ip flow monitor TRAFFIC-MONITOR input        # ingress monitoring
 ip flow monitor TRAFFIC-MONITOR output       # egress monitoring
```

### Show Flexible NetFlow status

```bash
show flow record                              # configured records
show flow exporter                            # configured exporters
show flow monitor                             # configured monitors
show flow monitor TRAFFIC-MONITOR cache       # cached flows
show flow monitor TRAFFIC-MONITOR statistics  # monitor stats
show flow exporter EXPORT-TO-COLLECTOR statistics  # export stats
show flow interface                           # interfaces with monitors
```

## NX-OS NetFlow Configuration

### NX-OS flow record and exporter

```bash
feature netflow                               # enable feature first

flow record NX-RECORD
 match ipv4 source address
 match ipv4 destination address
 match ipv4 protocol
 match transport source-port
 match transport destination-port
 collect counter bytes long
 collect counter packets long
 collect timestamp sys-uptime first
 collect timestamp sys-uptime last

flow exporter NX-EXPORTER
 destination 10.0.1.100 use-vrf management
 transport udp 2055
 source mgmt0
 version 9
  template data timeout 120

flow monitor NX-MONITOR
 record NX-RECORD
 exporter NX-EXPORTER
 cache timeout active 60
 cache timeout inactive 15

interface Ethernet1/1
 ip flow monitor NX-MONITOR input
 ip flow monitor NX-MONITOR output
```

### NX-OS verification

```bash
show feature | include netflow                # verify feature enabled
show flow record                              # show records
show flow exporter                            # show exporters
show flow monitor                             # show monitors
show flow cache                               # show cached flows
show flow timeout                             # show timeout values
```

## Sampled NetFlow

### Configure sampling (IOS-XE)

```bash
# Deterministic sampling — every Nth packet
sampler DETERMINISTIC-SAMPLER
 mode deterministic 1 out-of 100              # sample 1 in 100 packets

# Random sampling — probabilistic
sampler RANDOM-SAMPLER
 mode random 1 out-of 1000                    # sample 1 in 1000 packets

# Apply sampler to flow monitor on interface
interface GigabitEthernet0/0/0
 ip flow monitor TRAFFIC-MONITOR sampler DETERMINISTIC-SAMPLER input
```

### Sampling impact on accuracy

```bash
# Sampled flow byte/packet counts must be multiplied by the sampling rate
# Actual traffic = reported_value * sampling_rate
#
# 1:100 sampling with 500 reported packets = ~50,000 actual packets
#
# Trade-offs:
#   1:1 (unsampled)    — full accuracy, highest CPU/memory
#   1:100              — good for capacity planning, some flow loss
#   1:1000             — DDoS detection, top talkers, low overhead
#   1:10000            — very high-speed links (100G+), coarse view
```

### Show sampler status

```bash
show sampler                                  # all configured samplers
show sampler DETERMINISTIC-SAMPLER            # specific sampler stats
show flow monitor TRAFFIC-MONITOR statistics  # verify sampling applied
```

## sFlow Comparison

### sFlow vs NetFlow / IPFIX

```bash
# sFlow (RFC 3176):
#   - Packet sampling + counter polling (two mechanisms)
#   - Samples raw packet headers (first 128 bytes typical)
#   - Stateless — no flow cache on the device
#   - Lower device resource usage (no cache to maintain)
#   - Less accurate per-flow (statistical sampling)
#   - Multi-vendor standard (widely supported on switches)
#   - UDP-only export
#   - Good for: real-time visibility, high-port-density switches
#
# NetFlow/IPFIX:
#   - Full flow tracking with cache on device
#   - Aggregated flow records (not raw packets)
#   - Stateful — maintains flow state until expiry
#   - Higher device resource usage (CPU + memory for cache)
#   - More accurate per-flow (especially unsampled)
#   - Good for: billing, forensics, compliance, detailed analysis
```

## Collectors

### nfdump / nfcapd (CLI collector and analysis)

```bash
# Start collector daemon
nfcapd -w -D -p 2055 -l /var/nfdump/data     # listen on UDP 2055, write to dir
nfcapd -w -D -p 9996 -l /var/nfdump/data -T all  # all extensions

# Query collected flows
nfdump -r /var/nfdump/data/nfcapd.202604050000  # read specific file
nfdump -R /var/nfdump/data -o long            # read directory, long output
nfdump -R /var/nfdump/data -s srcip/bytes     # top source IPs by bytes
nfdump -R /var/nfdump/data -s dstport/flows   # top destination ports by flows
nfdump -R /var/nfdump/data -s record/bytes    # top flows by bytes

# Filters
nfdump -R /var/nfdump/data 'src ip 10.0.1.0/24'          # source subnet
nfdump -R /var/nfdump/data 'dst port 443'                  # HTTPS traffic
nfdump -R /var/nfdump/data 'proto tcp and bytes > 1000000' # large TCP flows
nfdump -R /var/nfdump/data 'src ip 10.0.1.5 and dst ip 192.168.1.1'

# Time window
nfdump -R /var/nfdump/data -t 2026/04/05.08:00-2026/04/05.17:00

# Aggregate and sort
nfdump -R /var/nfdump/data -A srcip,dstport -s record/bytes -n 20
nfdump -R /var/nfdump/data -o 'fmt:%sa %da %sp %dp %pr %byt %pkt %fl'
```

### ntopng (web-based collector)

```bash
# Start ntopng with NetFlow/IPFIX collection
ntopng -i tcp://127.0.0.1:5556               # ZMQ input from nprobe
ntopng --zmq-collector-port 5556              # direct ZMQ collector

# nprobe as NetFlow-to-ntopng bridge
nprobe --zmq tcp://127.0.0.1:5556 -i none -3 2055  # collect on 2055, forward ZMQ

# ntopng web UI: http://localhost:3000
# Default credentials: admin/admin
# Dashboards: Flows, Hosts, ASNs, VLANs, Protocols, Alerts
```

### Elastiflow (Elasticsearch-based)

```bash
# Elastiflow receives NetFlow/IPFIX/sFlow via Logstash or custom collector
# Configuration in /etc/logstash/conf.d/

# Logstash input for NetFlow v5/v9
# input {
#   udp {
#     port => 2055
#     codec => netflow { versions => [5, 9] }
#     type => "netflow"
#   }
# }

# Logstash input for IPFIX
# input {
#   udp {
#     port => 4739
#     codec => netflow { versions => [10] }
#     type => "ipfix"
#   }
# }

# Kibana dashboards for traffic analysis, top talkers, geo maps
```

## Linux Flow Tools

### softflowd (software NetFlow probe)

```bash
# Install
apt install softflowd                         # Debian/Ubuntu
yum install softflowd                         # RHEL/CentOS

# Run as NetFlow exporter on a Linux host or tap
softflowd -i eth0 -n 10.0.1.100:2055 -v 9    # export v9 to collector
softflowd -i eth0 -n 10.0.1.100:4739 -v 10   # export IPFIX
softflowd -i eth0 -n 10.0.1.100:2055 -v 5 -t maxlife=300  # v5, 5 min active

# Options
softflowd -i eth0 -n 10.0.1.100:2055 -v 9 \
  -t tcp.rst=30 \                             # RST timeout
  -t tcp.fin=10 \                             # FIN timeout
  -t maxlife=1800 \                           # max flow lifetime
  -t expint=60                                # export interval

# Query running softflowd
softflowctl /var/run/softflowd.ctl statistics # show stats
softflowctl /var/run/softflowd.ctl dump-flows # dump current flows
softflowctl /var/run/softflowd.ctl expire-all # force expire all flows
```

### pmacctd (traffic accounting daemon)

```bash
# pmacctd is part of pmacct — full-featured traffic accounting
apt install pmacct

# Basic config (/etc/pmacct/pmacctd.conf):
# daemonize: true
# interface: eth0
# plugins: nfprobe
# nfprobe_receiver: 10.0.1.100:2055
# nfprobe_version: 9
# nfprobe_timeouts: tcp.rst=30:maxlife=1800:expint=60
# aggregate: src_host, dst_host, src_port, dst_port, proto, tos

# Start pmacctd
pmacctd -f /etc/pmacct/pmacctd.conf

# Query accounting data
pmacct -s -T bytes                            # top talkers by bytes
pmacct -s -e src_host,dst_host,proto          # show src/dst/proto
```

### Linux kernel flow tools

```bash
# tc (traffic control) with flow classifier
tc filter add dev eth0 parent 1:0 protocol ip handle 1 \
  flow hash keys src,dst divisor 1024

# conntrack for stateful flow tracking
conntrack -L                                  # list all tracked connections
conntrack -L -p tcp --dport 443               # HTTPS connections
conntrack -C                                  # connection count
conntrack -E                                  # real-time event stream
```

## Analysis Use Cases

### Capacity planning

```bash
# Top bandwidth consumers over 24 hours
nfdump -R /var/nfdump/data -t 2026/04/04.00:00-2026/04/05.00:00 \
  -s srcip/bytes -n 20

# Protocol distribution
nfdump -R /var/nfdump/data -s proto/bytes

# Traffic by hour (time series)
nfdump -R /var/nfdump/data -A srcip -t 2026/04/05.08:00-2026/04/05.09:00 \
  -o 'fmt:%ts %sa %byt' | sort -t, -k3 -rn

# Interface utilization trends
nfdump -R /var/nfdump/data -A inif -s record/bytes
```

### DDoS detection

```bash
# Sudden spike in flows to a single destination
nfdump -R /var/nfdump/data -s dstip/flows -n 10   # top targets by flow count

# SYN flood — high flow count, low packet count per flow
nfdump -R /var/nfdump/data 'flags S and not flags ARFPU' -s dstip/flows

# Amplification attack — large responses from known amplifiers
nfdump -R /var/nfdump/data 'src port in [53, 123, 161, 1900] and bytes > 100000' \
  -s dstip/bytes

# Volumetric — top destinations by bytes in short window
nfdump -R /var/nfdump/data -t 2026/04/05.14:00-2026/04/05.14:05 \
  -s dstip/bytes -n 10
```

### Billing and accounting

```bash
# Per-customer traffic (by source subnet)
nfdump -R /var/nfdump/data -A srcip4/24 -s record/bytes -n 50

# 95th percentile calculation (from 5-minute samples)
# Export per-interval bytes, sort, take 95th percentile value
nfdump -R /var/nfdump/data -M /var/nfdump/data -A srcip \
  -o 'fmt:%byt' | sort -n | awk 'END{print NR*0.95" "NR}'
```

### Forensics and incident response

```bash
# All flows involving a compromised host
nfdump -R /var/nfdump/data 'host 10.0.5.99' -o long

# Lateral movement — internal-to-internal on unusual ports
nfdump -R /var/nfdump/data \
  'src ip 10.0.0.0/8 and dst ip 10.0.0.0/8 and not dst port in [22,80,443,53,3389]' \
  -s record/bytes

# Data exfiltration — large outbound flows
nfdump -R /var/nfdump/data \
  'src ip 10.0.0.0/8 and not dst ip 10.0.0.0/8 and bytes > 50000000' \
  -s record/bytes -n 20

# DNS tunneling — high volume to single external DNS
nfdump -R /var/nfdump/data 'dst port 53 and bytes > 10000' -s dstip/bytes
```

## Performance Impact

### CPU and memory considerations

```bash
# Flow cache sizing (Cisco IOS):
#   Default cache: 4096 entries (IOS), 64K entries (IOS-XE)
#   Each entry: ~64 bytes
#   Memory = cache_entries * 64 bytes
#
# CPU impact:
#   Unsampled:  3-5% CPU overhead on moderate traffic
#   1:100:      < 1% CPU overhead
#   1:1000:     negligible
#
# High-traffic recommendations:
#   - Enable sampling on links > 1 Gbps
#   - Use hardware-based NetFlow (ASICs) where available
#   - Increase active timeout to reduce export volume
#   - Use IPFIX over TCP for reliable delivery on congested management planes

# Monitor router impact
show processes cpu | include flow              # CPU from flow processes
show memory summary                            # memory usage
```

### Collector sizing

```bash
# Rough collector storage estimate:
#   Each flow record: ~50-100 bytes (compressed)
#   1000 flows/sec = ~4-8 GB/day
#   10000 flows/sec = ~40-80 GB/day
#
# Recommended collector resources:
#   < 5K flows/sec:   2 CPU, 4 GB RAM, 100 GB disk
#   5-50K flows/sec:  4 CPU, 16 GB RAM, 1 TB disk
#   > 50K flows/sec:  8+ CPU, 32+ GB RAM, SSD storage, distributed arch
```

## Tips

- Always set `ip flow-export source` to a loopback for stable source IP across link failures.
- Use NetFlow v9 or IPFIX over v5 — template-based formats support IPv6 and custom fields.
- Set active timeout to 60 seconds (not the 30-minute default) for near-real-time visibility.
- Sampling is essential on high-speed links (10G+). Start with 1:1000 and tune down if accuracy is needed.
- Export to a dedicated management VRF to keep flow data off the production data plane.
- IPFIX over TCP or SCTP prevents flow record loss on congested collector links.
- Monitor `show flow exporter statistics` for dropped/failed exports — a sign of collector overload.
- Collector time synchronization (NTP) is critical — flow timestamps rely on exporter system clock.
- Pair NetFlow with SNMP interface counters to validate flow data completeness.
- For forensics, keep at least 30 days of raw flow data — compressed storage is cheap.

## See Also

- prometheus
- grafana
- opentelemetry
- wireshark
- tcpdump
- snmp

## References

- [Cisco NetFlow Overview](https://www.cisco.com/c/en/us/products/ios-nx-os-software/ios-netflow/index.html)
- [RFC 7011 — IPFIX Protocol Specification](https://www.rfc-editor.org/rfc/rfc7011)
- [RFC 7012 — IPFIX Information Model](https://www.rfc-editor.org/rfc/rfc7012)
- [RFC 7013 — IPFIX Guidelines for Information Element Definitions](https://www.rfc-editor.org/rfc/rfc7013)
- [RFC 7014 — IPFIX Per-SCTP-Stream and Template-Set Scoping](https://www.rfc-editor.org/rfc/rfc7014)
- [RFC 7015 — IPFIX File Format](https://www.rfc-editor.org/rfc/rfc7015)
- [RFC 3176 — sFlow Specification](https://www.rfc-editor.org/rfc/rfc3176)
- [IANA IPFIX Information Elements Registry](https://www.iana.org/assignments/ipfix/ipfix.xhtml)
- [Cisco Flexible NetFlow Configuration Guide (IOS-XE)](https://www.cisco.com/c/en/us/td/docs/ios-xml/ios/fnetflow/configuration/xe-17/fnf-xe-17-book.html)
- [nfdump Documentation](https://github.com/phaag/nfdump)
- [ntopng Documentation](https://www.ntop.org/products/traffic-analysis/ntop/)
- [softflowd Manual](https://github.com/djmdjm/softflowd)
- [pmacct Documentation](http://www.pmacct.net/)
- [ElastiFlow Community Edition](https://github.com/robcowart/elastiflow)
