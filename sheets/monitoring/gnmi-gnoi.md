# gNMI / gNOI (gRPC Network Management and Operations)

gRPC-based network management (Get/Set/Subscribe) and operations (OS install, cert rotation, file transfer) — model-driven telemetry and device lifecycle over HTTP/2 with protobuf encoding.

## gNMI Operations

### gNMI Get

```bash
# gnmic — the standard gNMI CLI client
# Install: go install github.com/openconfig/gnmic@latest
#   or: brew install gnmic

# Get entire config
gnmic -a 10.0.0.1:57400 \
  -u admin -p cisco123 \
  --insecure \
  get --path /

# Get specific path
gnmic -a 10.0.0.1:57400 \
  -u admin -p cisco123 \
  --insecure \
  get --path /interfaces/interface[name=Ethernet1]/state/counters

# Get with specific encoding
gnmic -a 10.0.0.1:57400 \
  -u admin -p cisco123 \
  --insecure \
  get --path /system/config/hostname \
  --encoding json_ietf

# Get from multiple paths
gnmic -a 10.0.0.1:57400 \
  -u admin -p cisco123 \
  --insecure \
  get \
  --path /interfaces/interface[name=Ethernet1]/state/oper-status \
  --path /interfaces/interface[name=Ethernet1]/state/admin-status

# Get with data type filter
gnmic -a 10.0.0.1:57400 \
  -u admin -p cisco123 \
  --insecure \
  get --path /interfaces --type config       # config only
  # types: config, state, operational, all
```

### gNMI Set

```bash
# Set (update) a single leaf
gnmic -a 10.0.0.1:57400 \
  -u admin -p cisco123 \
  --insecure \
  set --update-path /system/config/hostname \
      --update-value "router1"

# Set with JSON value
gnmic -a 10.0.0.1:57400 \
  -u admin -p cisco123 \
  --insecure \
  set --update-path /interfaces/interface[name=Ethernet1]/config \
      --update-value '{"enabled": true, "description": "uplink"}'

# Replace entire subtree
gnmic -a 10.0.0.1:57400 \
  -u admin -p cisco123 \
  --insecure \
  set --replace-path /interfaces/interface[name=Loopback0]/config \
      --replace-value '{"name":"Loopback0","enabled":true}'

# Delete a path
gnmic -a 10.0.0.1:57400 \
  -u admin -p cisco123 \
  --insecure \
  set --delete /interfaces/interface[name=Loopback99]

# Set from file
gnmic -a 10.0.0.1:57400 \
  -u admin -p cisco123 \
  --insecure \
  set --update-path /network-instances \
      --update-file config.json

# Multiple operations in one RPC
gnmic -a 10.0.0.1:57400 \
  -u admin -p cisco123 \
  --insecure \
  set \
  --update-path /system/config/hostname --update-value "router1" \
  --update-path /system/config/domain-name --update-value "lab.local" \
  --delete /interfaces/interface[name=Loopback99]
```

### gNMI Capabilities

```bash
# Discover supported models, encodings, gNMI version
gnmic -a 10.0.0.1:57400 \
  -u admin -p cisco123 \
  --insecure \
  capabilities
# Returns: supported models, encodings, gNMI version
```

## gNMI Subscribe

### Subscription modes

```bash
# ONCE — single snapshot, then close
gnmic -a 10.0.0.1:57400 \
  -u admin -p cisco123 \
  --insecure \
  subscribe --path /interfaces/interface/state/counters \
  --mode once

# POLL — client triggers each update
gnmic -a 10.0.0.1:57400 \
  -u admin -p cisco123 \
  --insecure \
  subscribe --path /interfaces/interface/state/oper-status \
  --mode poll

# STREAM with ON_CHANGE — push on value change
gnmic -a 10.0.0.1:57400 \
  -u admin -p cisco123 \
  --insecure \
  subscribe --path /interfaces/interface/state/oper-status \
  --mode stream \
  --stream-mode on-change

# STREAM with SAMPLE — periodic push
gnmic -a 10.0.0.1:57400 \
  -u admin -p cisco123 \
  --insecure \
  subscribe --path /interfaces/interface/state/counters \
  --mode stream \
  --stream-mode sample \
  --sample-interval 10s

# STREAM with TARGET_DEFINED — device chooses mode per path
gnmic -a 10.0.0.1:57400 \
  -u admin -p cisco123 \
  --insecure \
  subscribe --path /interfaces/interface/state \
  --mode stream \
  --stream-mode target-defined
```

### Subscribe with suppress_redundant and heartbeat

```bash
# ON_CHANGE with heartbeat (sends value even if unchanged)
gnmic -a 10.0.0.1:57400 \
  -u admin -p cisco123 \
  --insecure \
  subscribe --path /interfaces/interface/state/oper-status \
  --mode stream \
  --stream-mode on-change \
  --heartbeat-interval 60s

# SAMPLE with suppress_redundant (skip if value unchanged)
gnmic -a 10.0.0.1:57400 \
  -u admin -p cisco123 \
  --insecure \
  subscribe --path /interfaces/interface/state/counters \
  --mode stream \
  --stream-mode sample \
  --sample-interval 10s \
  --suppress-redundant
```

### Multiple subscriptions

```bash
# Subscribe to multiple paths
gnmic -a 10.0.0.1:57400 \
  -u admin -p cisco123 \
  --insecure \
  subscribe \
  --path /interfaces/interface/state/counters \
  --path /network-instances/network-instance/protocols/protocol/bgp/neighbors/neighbor/state \
  --mode stream \
  --stream-mode sample \
  --sample-interval 30s
```

## gNMI Path Encoding

### Path format

```bash
# OpenConfig-style paths
/interfaces/interface[name=Ethernet1]/state/oper-status
/network-instances/network-instance[name=default]/protocols/protocol[identifier=BGP][name=bgp]/bgp/neighbors/neighbor[neighbor-address=10.0.0.1]/state

# YANG module-prefixed paths
/openconfig-interfaces:interfaces/interface[name=Ethernet1]/state
/openconfig-network-instance:network-instances/network-instance[name=default]

# Wildcard (all list entries)
/interfaces/interface[name=*]/state/oper-status

# Escaped characters in keys
/interfaces/interface[name=GigabitEthernet0/0/0]/state
```

### Path origin

```bash
# Specify origin (openconfig, native, etc.)
gnmic -a 10.0.0.1:57400 \
  -u admin -p cisco123 \
  --insecure \
  get --path "origin=openconfig:/interfaces/interface[name=Ethernet1]/state"

# Native model path (vendor-specific)
gnmic -a 10.0.0.1:57400 \
  -u admin -p cisco123 \
  --insecure \
  get --path "origin=native:/Cisco-IOS-XR-ifmgr-oper:interface-properties"
```

## gNMI with TLS

### TLS configuration

```bash
# With CA certificate
gnmic -a 10.0.0.1:57400 \
  -u admin -p cisco123 \
  --tls-ca ca.pem \
  get --path /system/config/hostname

# With mutual TLS (mTLS)
gnmic -a 10.0.0.1:57400 \
  --tls-ca ca.pem \
  --tls-cert client.pem \
  --tls-key client.key \
  get --path /system/config/hostname

# Skip TLS verification (lab only)
gnmic -a 10.0.0.1:57400 \
  -u admin -p cisco123 \
  --skip-verify \
  get --path /system/config/hostname
```

## gNOI Services

### gNOI System operations

```bash
# System reboot
gnmic -a 10.0.0.1:57400 \
  -u admin -p cisco123 \
  --insecure \
  gnoi system reboot \
  --method COLD \
  --delay 60s \
  --message "Scheduled maintenance"
# Methods: UNKNOWN, COLD, POWERDOWN, HALT, WARM, NSF, POWERUP

# System time
gnmic -a 10.0.0.1:57400 \
  -u admin -p cisco123 \
  --insecure \
  gnoi system time

# System ping
gnmic -a 10.0.0.1:57400 \
  -u admin -p cisco123 \
  --insecure \
  gnoi system ping \
  --destination 8.8.8.8 \
  --count 5

# System traceroute
gnmic -a 10.0.0.1:57400 \
  -u admin -p cisco123 \
  --insecure \
  gnoi system traceroute \
  --destination 8.8.8.8
```

### gNOI File operations

```bash
# Get file from device
gnmic -a 10.0.0.1:57400 \
  -u admin -p cisco123 \
  --insecure \
  gnoi file get \
  --remote /var/log/messages \
  --local ./device_log.txt

# Put file to device
gnmic -a 10.0.0.1:57400 \
  -u admin -p cisco123 \
  --insecure \
  gnoi file put \
  --local ./config_snippet.txt \
  --remote /tmp/config_snippet.txt

# Stat (file info)
gnmic -a 10.0.0.1:57400 \
  -u admin -p cisco123 \
  --insecure \
  gnoi file stat \
  --path /var/log/messages

# Remove file
gnmic -a 10.0.0.1:57400 \
  -u admin -p cisco123 \
  --insecure \
  gnoi file remove \
  --remote /tmp/old_config.txt
```

### gNOI Certificate management

```bash
# Install certificate
gnmic -a 10.0.0.1:57400 \
  -u admin -p cisco123 \
  --insecure \
  gnoi cert install \
  --id "web-cert" \
  --cert-file server.pem \
  --key-file server.key \
  --ca-cert-file ca.pem

# Rotate certificate
gnmic -a 10.0.0.1:57400 \
  -u admin -p cisco123 \
  --insecure \
  gnoi cert rotate \
  --id "web-cert" \
  --cert-file new_server.pem \
  --key-file new_server.key \
  --ca-cert-file ca.pem

# Get certificate info
gnmic -a 10.0.0.1:57400 \
  -u admin -p cisco123 \
  --insecure \
  gnoi cert get-certs
```

### gNOI OS operations

```bash
# Install OS image
gnmic -a 10.0.0.1:57400 \
  -u admin -p cisco123 \
  --insecure \
  gnoi os install \
  --version "17.6.1" \
  --package ./ios_xe_17.6.1.bin

# Activate installed OS
gnmic -a 10.0.0.1:57400 \
  -u admin -p cisco123 \
  --insecure \
  gnoi os activate \
  --version "17.6.1" \
  --no-reboot

# Verify OS
gnmic -a 10.0.0.1:57400 \
  -u admin -p cisco123 \
  --insecure \
  gnoi os verify
```

### gNOI Healthz

```bash
# Check device health
gnmic -a 10.0.0.1:57400 \
  -u admin -p cisco123 \
  --insecure \
  gnoi healthz get \
  --path /components/component[name=CPU0]

# Check with acknowledgement
gnmic -a 10.0.0.1:57400 \
  -u admin -p cisco123 \
  --insecure \
  gnoi healthz check
```

## Device Configuration for gNMI

### IOS-XR gNMI server

```bash
# Enable gRPC server
grpc
 port 57400
 no-tls                                      # lab only — disable TLS
 address-family dual                          # IPv4 + IPv6
 max-request-per-user 32
 max-request-total 128
!

# With TLS
grpc
 port 57400
 tls-mutual
 certificate-id grpc-cert
!
```

### IOS-XE gNMI (NETCONF-YANG must be enabled first)

```bash
# Enable NETCONF and RESTCONF first
netconf-yang
restconf

# Enable gNMI
gnxi
 state
 server
  transport grpc port 57400
  no secure-server                            # lab only
!

# With TLS
gnxi
 server
  transport grpc port 57400
  secure-server
  secure-trustpoint gnmi-tp
!
```

### NX-OS gRPC

```bash
feature grpc                                  # enable gRPC agent
grpc certificate                              # configure certificate
grpc port 50051                               # set gRPC port
```

### JunOS gNMI (gRPC)

```bash
set system services extension-service request-response grpc clear-text port 57400
set system services extension-service request-response grpc skip-authentication
# Production:
set system services extension-service request-response grpc ssl port 57400
set system services extension-service request-response grpc ssl local-certificate gnmi-cert
```

## gnmic as Prometheus Exporter

### gnmic configuration file

```yaml
# gnmic.yaml — run gnmic as a telemetry collector
targets:
  10.0.0.1:57400:
    username: admin
    password: cisco123
    insecure: true
  10.0.0.2:57400:
    username: admin
    password: cisco123
    insecure: true

subscriptions:
  interface_counters:
    paths:
      - /interfaces/interface/state/counters
    mode: stream
    stream-mode: sample
    sample-interval: 30s

  bgp_neighbors:
    paths:
      - /network-instances/network-instance/protocols/protocol/bgp/neighbors/neighbor/state
    mode: stream
    stream-mode: on-change

outputs:
  prometheus_output:
    type: prometheus
    listen: :9804
    path: /metrics
    metric-prefix: gnmic
    append-subscription-name: true
```

### Run gnmic as collector

```bash
# Start gnmic with config file
gnmic --config gnmic.yaml subscribe

# Prometheus scrape config
# scrape_configs:
#   - job_name: 'gnmic'
#     static_configs:
#       - targets: ['gnmic-host:9804']

# gnmic also supports these outputs:
#   - InfluxDB, Kafka, NATS, Stan, file, UDP, TCP
```

## gNMI Dial-In vs Dial-Out

### Dial-in (collector initiates connection)

```bash
# Standard model: collector (gnmic) connects to device
# Device runs gNMI server, collector is the client
gnmic -a 10.0.0.1:57400 subscribe --path /interfaces
#       ^^^ collector connects TO device
```

### Dial-out (device initiates connection)

```bash
# IOS-XR dial-out telemetry (MDT)
telemetry model-driven
 destination-group COLLECTOR
  address-family ipv4 10.0.1.100 port 57500
   encoding self-describing-gpb
   protocol grpc no-tls
  !
 !
 sensor-group INTERFACES
  sensor-path openconfig-interfaces:interfaces/interface/state/counters
 !
 subscription SUB1
  sensor-group-id INTERFACES sample-interval 30000
  destination-id COLLECTOR
 !
!
# Device pushes telemetry TO collector — no inbound connections needed
```

## Python gNMI Client

### Using pygnmi

```python
from pygnmi.client import gNMIclient

with gNMIclient(
    target=('10.0.0.1', 57400),
    username='admin',
    password='cisco123',
    insecure=True
) as gc:
    # Get
    result = gc.get(path=['/interfaces/interface[name=Ethernet1]/state'])
    print(result)

    # Set
    gc.set(update=[
        ('/system/config/hostname', {'hostname': 'router1'})
    ])

    # Subscribe (ONCE)
    subscribe = {
        'subscription': [
            {'path': '/interfaces/interface/state/counters',
             'mode': 'sample', 'sample_interval': 10000000000}
        ],
        'mode': 'once',
        'encoding': 'json_ietf'
    }
    for response in gc.subscribe2(subscribe=subscribe):
        print(response)
```

## See Also

- netconf
- restconf
- yang-models
- pyats
- opentelemetry

## References

- gNMI specification: https://github.com/openconfig/reference/blob/master/rpc/gnmi/gnmi-specification.md
- gNOI repository: https://github.com/openconfig/gnoi
- gnmic documentation: https://gnmic.openconfig.net/
- OpenConfig: https://www.openconfig.net/
- pygnmi: https://github.com/akarneliuk/pygnmi
