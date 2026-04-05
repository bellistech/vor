# Model-Driven Telemetry (Streaming Network Telemetry)

Push-based telemetry that streams structured data from network devices using YANG models, replacing periodic SNMP polling.

## Architecture Overview

### MDT components

```bash
# Telemetry flow:
# Device (Publisher) → Transport (gRPC/TCP/UDP) → Receiver (Telegraf/gnmic)
#                                                       → TSDB (Prometheus/InfluxDB)
#                                                       → Visualization (Grafana)
#
# Key concepts:
# - Sensor path: YANG model path to data (e.g., openconfig-interfaces:interfaces)
# - Subscription: Sensor path + cadence + transport
# - Encoding: How data is serialized (GPB, GPB-KV, JSON)
# - Transport: How data is delivered (gRPC, TCP, UDP)
```

## Transport Protocols

### gRPC (recommended)

```bash
# Supports both dial-in and dial-out
# - TLS encryption
# - Bidirectional streaming
# - HTTP/2 multiplexing
# - Default port: 57400 (IOS-XR), 57500 (NX-OS)
```

### TCP

```bash
# Dial-out only
# - Simple, no framing overhead
# - No encryption (unless tunneled)
# - Useful for legacy collectors
# - Default port: configurable
```

### UDP

```bash
# Dial-out only
# - Lowest overhead
# - No delivery guarantee
# - Best for high-frequency, loss-tolerant data
# - MTU considerations for large payloads
```

## Encoding Formats

### GPB (Google Protocol Buffers) — compact

```bash
# Binary encoding, smallest payload
# Requires .proto file for decoding
# Best for high-volume telemetry
# ~10x smaller than JSON
```

### GPB-KV (Key-Value variant)

```bash
# Self-describing GPB format
# No .proto file needed for decoding
# Slightly larger than compact GPB
# Easier to parse generically
# Most commonly used in production
```

### JSON

```bash
# Human-readable
# Largest payload (~10x GPB)
# Easiest to debug
# Good for development/testing
# Not recommended for production at scale
```

## Dial-In vs Dial-Out

### Dial-in (device as server)

```bash
# Collector connects TO the device
# Device listens on gRPC port
# Collector controls subscription lifecycle
# Better for: on-demand queries, troubleshooting
# Requires: device reachable from collector, firewall rules

# Example: gnmic dial-in subscription
gnmic subscribe \
  --address 10.0.0.1:57400 \
  --username admin --password secret \
  --path "/interfaces/interface/state/counters" \
  --stream-mode sample \
  --sample-interval 10s \
  --encoding json_ietf
```

### Dial-out (device as client)

```bash
# Device connects TO the collector
# Subscription configured on device
# Device initiates connection
# Better for: production monitoring, NAT traversal
# Requires: collector reachable from device
```

## IOS-XE Configuration

### gRPC dial-in

```bash
# Enable gRPC server
conf t
netconf-yang
telemetry ietf subscription 100
 encoding encode-kvgpb
 filter xpath /interfaces-ios-xe-oper:interfaces/interface/statistics
 source-address 10.0.0.1
 stream yang-push
 update-policy periodic 1000
 receiver ip address 10.0.0.200 57000 protocol grpc-tcp
end
```

### Periodic subscription

```bash
conf t
telemetry ietf subscription 101
 encoding encode-kvgpb
 filter xpath /process-cpu-ios-xe-oper:cpu-usage/cpu-utilization/five-seconds
 source-address 10.0.0.1
 stream yang-push
 update-policy periodic 500
 receiver ip address 10.0.0.200 57000 protocol grpc-tcp
end
```

### On-change subscription

```bash
conf t
telemetry ietf subscription 102
 encoding encode-kvgpb
 filter xpath /bgp-state-data/neighbors/neighbor/connection/state
 source-address 10.0.0.1
 stream yang-push
 update-policy on-change
 receiver ip address 10.0.0.200 57000 protocol grpc-tcp
end
```

### Verify subscriptions

```bash
show telemetry ietf subscription all
show telemetry ietf subscription 100 detail
show telemetry ietf subscription 100 receiver
show telemetry internal connection
show platform software yang-management process
```

## IOS-XR Configuration

### gRPC dial-out

```bash
conf t
telemetry model-driven
 destination-group COLLECTOR
  address-family ipv4 10.0.0.200 port 57000
   encoding self-describing-gpb
   protocol grpc no-tls
  !
 !
 sensor-group INTERFACES
  sensor-path Cisco-IOS-XR-infra-statsd-oper:infra-statistics/interfaces/interface/latest/generic-counters
 !
 sensor-group CPU
  sensor-path Cisco-IOS-XR-wdsysmon-fd-oper:system-monitoring/cpu-utilization
 !
 sensor-group BGP
  sensor-path Cisco-IOS-XR-ipv4-bgp-oper:bgp/instances/instance/instance-active/default-vrf/neighbors/neighbor
 !
 subscription INFRA
  sensor-group-id INTERFACES sample-interval 10000
  sensor-group-id CPU sample-interval 5000
  destination-id COLLECTOR
 !
 subscription ROUTING
  sensor-group-id BGP sample-interval 30000
  destination-id COLLECTOR
 !
commit
end
```

### Verify on IOS-XR

```bash
show telemetry model-driven subscription
show telemetry model-driven subscription INFRA internal
show telemetry model-driven sensor-group
show telemetry model-driven destination
show telemetry model-driven summary
```

## NX-OS Configuration

### Telemetry subscription

```bash
conf t
feature telemetry

telemetry
  destination-group 100
    ip address 10.0.0.200 port 57000 protocol gRPC encoding GPB
  sensor-group 100
    path sys/intf depth unbounded
    path sys/bgp depth unbounded
  sensor-group 200
    path sys/procsys/sysmem depth 0
    path sys/proccpu depth 0
  subscription 100
    dst-grp 100
    snsr-grp 100 sample-interval 10000
    snsr-grp 200 sample-interval 5000
end
```

### Verify on NX-OS

```bash
show telemetry transport
show telemetry data collector details
show telemetry control database subscriptions
show telemetry control database sensor-paths
```

## JunOS Configuration

### OpenConfig telemetry (JTI)

```bash
set services analytics streaming-server COLLECTOR remote-address 10.0.0.200
set services analytics streaming-server COLLECTOR remote-port 57000
set services analytics export-profile INTERFACES reporting-rate 10
set services analytics export-profile INTERFACES format gpb
set services analytics sensor IFACE server-name COLLECTOR
set services analytics sensor IFACE export-name INTERFACES
set services analytics sensor IFACE resource /interfaces/interface/state/counters/

# gRPC dial-in (gNMI)
set system services extension-service request-response grpc clear-text port 57400
set system services extension-service request-response grpc skip-authentication
```

### Verify on JunOS

```bash
show agent sensors
show analytics streaming-server
show analytics sensor-data IFACE
```

## gnmic (gNMI CLI Client)

### Install gnmic

```bash
# Linux
bash -c "$(curl -sL https://get-gnmic.openconfig.net)"

# macOS
brew install openconfig/gnmic/gnmic
```

### Subscribe to telemetry

```bash
# Sample mode (periodic)
gnmic subscribe \
  --address 10.0.0.1:57400 \
  -u admin -p secret \
  --path "/interfaces/interface/state/counters" \
  --stream-mode sample \
  --sample-interval 10s \
  --encoding json_ietf

# On-change mode
gnmic subscribe \
  --address 10.0.0.1:57400 \
  -u admin -p secret \
  --path "/network-instances/network-instance/protocols/protocol/bgp/neighbors/neighbor/state" \
  --stream-mode on-change \
  --encoding json_ietf

# Multiple paths
gnmic subscribe \
  --address 10.0.0.1:57400 \
  -u admin -p secret \
  --path "/interfaces/interface/state/counters" \
  --path "/system/cpus/cpu/state" \
  --stream-mode sample \
  --sample-interval 10s
```

### Get (single read)

```bash
gnmic get \
  --address 10.0.0.1:57400 \
  -u admin -p secret \
  --path "/interfaces/interface[name=Ethernet1]/state" \
  --encoding json_ietf
```

### Set (config push via gNMI)

```bash
gnmic set \
  --address 10.0.0.1:57400 \
  -u admin -p secret \
  --update-path "/interfaces/interface[name=Ethernet1]/config/description" \
  --update-value "Uplink to spine1"
```

### gnmic as collector (dial-out receiver)

```bash
# gnmic.yaml
username: admin
password: secret
encoding: json_ietf

subscriptions:
  interfaces:
    paths:
      - /interfaces/interface/state/counters
    stream-mode: sample
    sample-interval: 10s
  bgp:
    paths:
      - /network-instances/network-instance/protocols/protocol/bgp/
    stream-mode: on-change

targets:
  spine1:
    address: 10.0.0.1:57400
  spine2:
    address: 10.0.0.2:57400
  leaf1:
    address: 10.0.0.11:57400

outputs:
  prometheus:
    type: prometheus
    listen: :9804
    path: /metrics
    metric-prefix: gnmic
    append-subscription-name: true
```

```bash
gnmic --config gnmic.yaml subscribe
```

## Telegraf as Receiver

### telegraf.conf for telemetry

```bash
# /etc/telegraf/telegraf.conf

# gRPC dial-out receiver
[[inputs.cisco_telemetry_mdt]]
  transport = "grpc"
  service_address = ":57000"

# gnmi dial-in (Telegraf connects to device)
[[inputs.gnmi]]
  addresses = ["10.0.0.1:57400"]
  username = "admin"
  password = "secret"
  encoding = "json_ietf"

  [[inputs.gnmi.subscription]]
    name = "interface_counters"
    origin = "openconfig"
    path = "/interfaces/interface/state/counters"
    subscription_mode = "sample"
    sample_interval = "10s"

  [[inputs.gnmi.subscription]]
    name = "bgp_neighbors"
    origin = "openconfig"
    path = "/network-instances/network-instance/protocols/protocol/bgp/neighbors/neighbor/state"
    subscription_mode = "on-change"

# Output to Prometheus
[[outputs.prometheus_client]]
  listen = ":9273"
  metric_version = 2

# Output to InfluxDB
[[outputs.influxdb_v2]]
  urls = ["http://localhost:8086"]
  token = "$INFLUX_TOKEN"
  organization = "netops"
  bucket = "telemetry"
```

### Run Telegraf

```bash
telegraf --config /etc/telegraf/telegraf.conf --test
telegraf --config /etc/telegraf/telegraf.conf
```

## Prometheus Integration

### prometheus.yml scrape config

```yaml
scrape_configs:
  - job_name: 'gnmic'
    static_configs:
      - targets: ['localhost:9804']
    scrape_interval: 15s

  - job_name: 'telegraf'
    static_configs:
      - targets: ['localhost:9273']
    scrape_interval: 15s

  - job_name: 'snmp_exporter'
    static_configs:
      - targets: ['10.0.0.1', '10.0.0.2']
    metrics_path: /snmp
    params:
      module: [if_mib]
    relabel_configs:
      - source_labels: [__address__]
        target_label: __param_target
      - target_label: __address__
        replacement: localhost:9116
```

### Useful PromQL for network telemetry

```bash
# Interface utilization (bits/sec)
rate(interface_counters_in_octets[5m]) * 8

# Interface utilization percentage (1G link)
rate(interface_counters_in_octets[5m]) * 8 / 1000000000 * 100

# BGP neighbor state changes
changes(bgp_neighbor_session_state[1h])

# CPU utilization
system_cpu_utilization_percent

# Top 10 interfaces by traffic
topk(10, rate(interface_counters_in_octets[5m]) * 8)

# Error rate per interface
rate(interface_counters_in_errors[5m])

# Packet discard rate
rate(interface_counters_in_discards[5m]) / rate(interface_counters_in_pkts[5m]) * 100
```

## Grafana Dashboards

### Dashboard JSON model (interface panel)

```bash
# Key panel types for network telemetry:
# - Time series: interface counters, CPU/memory over time
# - Stat: current BGP neighbor count, uptime
# - Table: interface status summary
# - Gauge: link utilization percentage
# - Alert list: triggered alerts
#
# Variables for dashboard:
# - $device: label_values(device)
# - $interface: label_values(interface_name{device="$device"})
#
# Common queries:
# Interface throughput:
#   rate(interface_counters_out_octets{device="$device",interface_name="$interface"}[5m]) * 8
# BGP state:
#   bgp_neighbor_session_state{device="$device"}
```

## Key Sensor Paths

### OpenConfig paths (cross-vendor)

```bash
# Interfaces
/interfaces/interface/state/counters
/interfaces/interface/state/oper-status
/interfaces/interface/state/admin-status

# BGP
/network-instances/network-instance/protocols/protocol/bgp/neighbors/neighbor/state
/network-instances/network-instance/protocols/protocol/bgp/global/state

# System
/system/cpus/cpu/state
/system/memory/state
/system/processes/process/state

# LLDP
/lldp/interfaces/interface/neighbors/neighbor/state

# Routing
/network-instances/network-instance/afts/ipv4-unicast/ipv4-entry
```

### Cisco IOS-XR native paths

```bash
# Interface stats
Cisco-IOS-XR-infra-statsd-oper:infra-statistics/interfaces/interface/latest/generic-counters

# CPU
Cisco-IOS-XR-wdsysmon-fd-oper:system-monitoring/cpu-utilization

# Memory
Cisco-IOS-XR-nto-misc-oper:memory-summary/nodes/node/summary

# BGP neighbors
Cisco-IOS-XR-ipv4-bgp-oper:bgp/instances/instance/instance-active/default-vrf/neighbors/neighbor

# OSPF neighbors
Cisco-IOS-XR-ipv4-ospf-oper:ospf/processes/process/default-vrf/adjacency-information
```

### Cisco IOS-XE native paths

```bash
# CPU
/process-cpu-ios-xe-oper:cpu-usage/cpu-utilization

# Memory
/memory-ios-xe-oper:memory-statistics/memory-statistic

# Interfaces
/interfaces-ios-xe-oper:interfaces/interface/statistics

# BGP
/bgp-state-data/neighbors/neighbor

# Environment
/environment-ios-xe-oper:environment-sensors
```

## Cadence Selection Guide

### Recommended sample intervals

```bash
# High frequency (5-10 seconds):
# - Interface counters (for real-time dashboards)
# - CPU/memory (capacity monitoring)
# - QoS queue stats

# Medium frequency (30-60 seconds):
# - BGP neighbor state (periodic check)
# - OSPF neighbor state
# - Routing table size

# On-change (event-driven):
# - BGP neighbor state transitions
# - Interface up/down
# - Configuration changes
# - LLDP neighbor changes

# Low frequency (5-10 minutes):
# - Hardware inventory
# - Software version
# - License status
```

## Telemetry vs SNMP

### Comparison

```bash
# SNMP (pull-based):
# + Universal support
# + Simple to set up
# - Polling overhead on device CPU
# - Fixed OID tree (rigid schema)
# - 10-30 second practical minimum poll interval
# - No on-change notification (traps are unreliable)
# - Text encoding (inefficient)

# MDT (push-based):
# + Device pushes data (lower CPU impact at scale)
# + YANG models (structured, versioned)
# + Sub-second cadence possible
# + On-change subscriptions
# + Binary encoding (GPB) — 10x more efficient
# + gRPC transport (TLS, streaming, multiplexed)
# - Not universally supported
# - More complex initial setup
# - Vendor-specific sensor paths alongside OpenConfig
```

## See Also

- SNMP
- gNMI/gNOI
- Prometheus
- Grafana
- OpenTelemetry
- NetFlow/IPFIX

## References

- OpenConfig: https://www.openconfig.net/
- gnmic: https://gnmic.openconfig.net/
- Cisco MDT: https://www.cisco.com/c/en/us/td/docs/iosxr/ncs5500/telemetry/
- Telegraf gNMI: https://github.com/influxdata/telegraf/tree/master/plugins/inputs/gnmi
- YANG Catalog: https://yangcatalog.org/
- gRPC: https://grpc.io/
