# Network Programmability (YANG, NETCONF, RESTCONF, gNMI)

> Manage network devices programmatically using YANG data models, NETCONF/RESTCONF/gNMI protocols, streaming telemetry, and model-driven automation instead of CLI screen-scraping.

## YANG Data Modeling

### YANG Fundamentals

```
# YANG (Yet Another Next Generation) — RFC 7950
# Defines the structure, constraints, and types of configuration/state data
# Used by NETCONF, RESTCONF, and gNMI as the data schema language

# Key constructs:
#   module      — top-level unit of YANG, defines a namespace
#   container   — grouping node (no value itself, holds children)
#   list        — ordered/unordered collection keyed by one or more leafs
#   leaf        — single scalar value (string, uint32, boolean, etc.)
#   leaf-list   — array of scalar values
#   choice/case — mutually exclusive branches
#   augment     — extend another module's tree
#   deviation   — document differences from a standard model
#   typedef     — reusable custom type
#   grouping    — reusable set of nodes (used with "uses")
#   rpc         — remote procedure call definition
#   notification — event notification definition
```

### YANG Module Structure

```yang
module example-interfaces {
    namespace "urn:example:interfaces";
    prefix ex-if;

    import ietf-inet-types {
        prefix inet;
    }

    revision 2024-01-01 {
        description "Initial revision";
    }

    typedef interface-speed {
        type enumeration {
            enum 1G;
            enum 10G;
            enum 25G;
            enum 100G;
        }
    }

    container interfaces {
        list interface {
            key "name";
            leaf name {
                type string;
            }
            leaf description {
                type string;
            }
            leaf enabled {
                type boolean;
                default true;
            }
            leaf speed {
                type interface-speed;
            }
            container ipv4 {
                leaf address {
                    type inet:ipv4-address;
                }
                leaf prefix-length {
                    type uint8 {
                        range "0..32";
                    }
                }
            }
        }
    }
}
```

### YANG Augment and Deviation

```yang
# Augment — add nodes to another module's tree
augment "/ex-if:interfaces/ex-if:interface" {
    leaf qos-policy {
        type string;
        description "QoS policy applied to this interface";
    }
}

# Deviation — document how an implementation differs from the model
deviation "/ex-if:interfaces/ex-if:interface/ex-if:speed" {
    deviate replace {
        type enumeration {
            enum 1G;
            enum 10G;
            # This device does not support 25G or 100G
        }
    }
}
```

### pyang — YANG Validation and Tree Output

```bash
# Install pyang
pip install pyang

# Validate a YANG module
pyang --lint example-interfaces.yang

# Generate tree view (canonical format for documentation)
pyang -f tree example-interfaces.yang
# module: example-interfaces
#   +--rw interfaces
#      +--rw interface* [name]
#         +--rw name           string
#         +--rw description?   string
#         +--rw enabled?       boolean
#         +--rw speed?         interface-speed
#         +--rw ipv4
#            +--rw address?        inet:ipv4-address
#            +--rw prefix-length?  uint8

# Generate UML diagram
pyang -f uml example-interfaces.yang -o interfaces.uml

# Generate sample XML skeleton
pyang -f sample-xml-skeleton example-interfaces.yang

# Validate against multiple models (with search path)
pyang --path /usr/share/yang/modules/ietf -f tree ietf-interfaces.yang
```

## NETCONF

### NETCONF Architecture

```
# NETCONF — RFC 6241
# Transport: SSH (port 830), TLS optional
# Encoding: XML (RPC request/reply wrapped in <rpc>/<rpc-reply>)
# Protocol stack:
#   Application
#   ├── Content    — YANG-modeled configuration/state data
#   ├── Operations — get, get-config, edit-config, lock, commit, etc.
#   ├── Messages   — RPC, RPC-reply, notification
#   └── Transport  — SSH subsystem "netconf"

# Datastores:
#   running    — currently active configuration
#   candidate  — staging area (requires :candidate capability)
#   startup    — loaded at boot (requires :startup capability)
```

### NETCONF Operations

```xml
<!-- get-config: retrieve configuration data -->
<rpc xmlns="urn:ietf:params:xml:ns:netconf:base:1.0" message-id="1">
  <get-config>
    <source><running/></source>
    <filter type="subtree">
      <interfaces xmlns="urn:ietf:params:xml:ns:yang:ietf-interfaces"/>
    </filter>
  </get-config>
</rpc>

<!-- get: retrieve config + operational state -->
<rpc xmlns="urn:ietf:params:xml:ns:netconf:base:1.0" message-id="2">
  <get>
    <filter type="xpath"
      select="/interfaces/interface[name='GigabitEthernet1']"/>
  </get>
</rpc>

<!-- edit-config: modify configuration -->
<rpc xmlns="urn:ietf:params:xml:ns:netconf:base:1.0" message-id="3">
  <edit-config>
    <target><running/></target>
    <config>
      <interfaces xmlns="urn:ietf:params:xml:ns:yang:ietf-interfaces">
        <interface>
          <name>GigabitEthernet2</name>
          <description>Uplink to core</description>
          <enabled>true</enabled>
        </interface>
      </interfaces>
    </config>
  </edit-config>
</rpc>

<!-- lock/unlock: prevent concurrent edits -->
<rpc xmlns="urn:ietf:params:xml:ns:netconf:base:1.0" message-id="4">
  <lock><target><candidate/></target></lock>
</rpc>

<!-- commit: apply candidate to running -->
<rpc xmlns="urn:ietf:params:xml:ns:netconf:base:1.0" message-id="5">
  <commit/>
</rpc>
```

### ncclient — Python NETCONF Client

```python
from ncclient import manager

# Connect to IOS-XE device
with manager.connect(
    host="192.168.1.1",
    port=830,
    username="admin",
    password="cisco123",
    hostkey_verify=False,
    device_params={"name": "csr"}
) as m:

    # List server capabilities
    for cap in m.server_capabilities:
        print(cap)

    # Get running config (subtree filter)
    filter_xml = """
    <filter>
      <native xmlns="http://cisco.com/ns/yang/Cisco-IOS-XE-native">
        <interface/>
      </native>
    </filter>
    """
    result = m.get_config("running", filter_xml)
    print(result)

    # Edit config — set interface description
    config_xml = """
    <config>
      <native xmlns="http://cisco.com/ns/yang/Cisco-IOS-XE-native">
        <interface>
          <GigabitEthernet>
            <name>2</name>
            <description>Configured via NETCONF</description>
          </GigabitEthernet>
        </interface>
      </native>
    </config>
    """
    m.edit_config(target="running", config=config_xml)
```

```python
# ncclient with candidate datastore and commit
with manager.connect(host="junos-router", port=830,
                     username="admin", password="juniper123",
                     hostkey_verify=False) as m:
    with m.locked("candidate"):
        m.edit_config(target="candidate", config=config_xml)
        m.validate("candidate")
        m.commit()

# XPath filter (requires :xpath capability)
result = m.get(filter=("xpath", "/interfaces/interface[enabled='true']"))
```

## RESTCONF

### RESTCONF Basics

```
# RESTCONF — RFC 8040
# REST API for YANG-modeled data over HTTPS
# Port: 443 (HTTPS required)
# Media types:
#   application/yang-data+json   — JSON encoding
#   application/yang-data+xml    — XML encoding
# URL structure:
#   https://<host>/restconf/data/<yang-module:container>/<list=key>
#   https://<host>/restconf/operations/<yang-module:rpc>

# HTTP method mapping:
#   GET     — read config or state data
#   POST    — create a new resource / invoke RPC
#   PUT     — create or replace a resource
#   PATCH   — merge changes into a resource
#   DELETE  — remove a resource
```

### RESTCONF on IOS-XE

```bash
# Discover RESTCONF root
curl -s -k -u admin:cisco123 \
    https://192.168.1.1/.well-known/host-meta \
    -H "Accept: application/xrd+xml"

# Get all interfaces (JSON)
curl -s -k -u admin:cisco123 \
    https://192.168.1.1/restconf/data/ietf-interfaces:interfaces \
    -H "Accept: application/yang-data+json"

# Get a specific interface
curl -s -k -u admin:cisco123 \
    "https://192.168.1.1/restconf/data/ietf-interfaces:interfaces/interface=GigabitEthernet1" \
    -H "Accept: application/yang-data+json"

# Create a loopback interface (POST)
curl -s -k -u admin:cisco123 \
    https://192.168.1.1/restconf/data/ietf-interfaces:interfaces \
    -H "Content-Type: application/yang-data+json" \
    -X POST -d '{
      "ietf-interfaces:interface": {
        "name": "Loopback100",
        "description": "Created via RESTCONF",
        "type": "iana-if-type:softwareLoopback",
        "enabled": true,
        "ietf-ip:ipv4": {
          "address": [{
            "ip": "10.100.0.1",
            "netmask": "255.255.255.255"
          }]
        }
      }
    }'

# Update interface description (PATCH — merge)
curl -s -k -u admin:cisco123 \
    "https://192.168.1.1/restconf/data/ietf-interfaces:interfaces/interface=GigabitEthernet2" \
    -H "Content-Type: application/yang-data+json" \
    -X PATCH -d '{
      "ietf-interfaces:interface": {
        "description": "Updated via RESTCONF"
      }
    }'

# Delete an interface
curl -s -k -u admin:cisco123 \
    "https://192.168.1.1/restconf/data/ietf-interfaces:interfaces/interface=Loopback100" \
    -X DELETE

# Invoke RPC (save running config)
curl -s -k -u admin:cisco123 \
    https://192.168.1.1/restconf/operations/cisco-ia:save-config \
    -H "Content-Type: application/yang-data+json" \
    -X POST
```

### RESTCONF with Python requests

```python
import requests
import json

BASE_URL = "https://192.168.1.1/restconf/data"
HEADERS = {
    "Accept": "application/yang-data+json",
    "Content-Type": "application/yang-data+json"
}
AUTH = ("admin", "cisco123")

# GET all interfaces
resp = requests.get(
    f"{BASE_URL}/ietf-interfaces:interfaces",
    headers=HEADERS, auth=AUTH, verify=False
)
interfaces = resp.json()
for iface in interfaces["ietf-interfaces:interfaces"]["interface"]:
    print(f"{iface['name']}: {iface.get('description', 'N/A')}")

# PATCH — update BGP neighbor
bgp_patch = {
    "Cisco-IOS-XE-bgp:neighbor": {
        "id": "10.0.0.2",
        "remote-as": 65002,
        "description": "RESTCONF-managed peer"
    }
}
resp = requests.patch(
    f"{BASE_URL}/Cisco-IOS-XE-native:native/router/bgp=65001/neighbor",
    headers=HEADERS, auth=AUTH, verify=False,
    json=bgp_patch
)
print(resp.status_code)  # 204 = success
```

## gNMI (gRPC Network Management Interface)

### gNMI Architecture

```
# gNMI — gRPC-based protocol for streaming and configuration
# Transport: HTTP/2 + TLS (gRPC)
# Encoding: Protobuf (binary), JSON_IETF, or ASCII
# Port: typically 9339 or 57400

# RPCs:
#   Capabilities  — discover supported models and encodings
#   Get           — retrieve config/state at a path
#   Set           — update/replace/delete configuration
#   Subscribe     — streaming telemetry

# Subscription modes:
#   STREAM        — continuous push
#   ONCE          — one-time snapshot
#   POLL          — client-triggered refresh

# STREAM sub-modes:
#   ON_CHANGE     — push only when value changes (event-driven)
#   SAMPLE        — push at fixed intervals (e.g., every 10s)

# Dial-in  — device initiates gRPC connection to collector
# Dial-out — collector connects to device (more common with gNMI)
```

### gnmic — gNMI CLI Client

```bash
# Install gnmic
bash -c "$(curl -sL https://get.gnmic.io)"

# Capabilities
gnmic -a 192.168.1.1:9339 -u admin -p cisco123 --insecure capabilities

# Get interface counters
gnmic -a 192.168.1.1:9339 -u admin -p cisco123 --insecure \
    get --path "/interfaces/interface[name=Ethernet1]/state/counters"

# Get full interface config
gnmic -a 192.168.1.1:9339 -u admin -p cisco123 --insecure \
    get --path "/interfaces/interface[name=Ethernet1]" \
    --type config

# Set interface description
gnmic -a 192.168.1.1:9339 -u admin -p cisco123 --insecure \
    set --update-path "/interfaces/interface[name=Ethernet1]/config/description" \
    --update-value "Configured via gNMI"

# Replace entire interface config
gnmic -a 192.168.1.1:9339 -u admin -p cisco123 --insecure \
    set --replace-path "/interfaces/interface[name=Ethernet1]/config" \
    --replace-file interface-config.json

# Delete a configuration path
gnmic -a 192.168.1.1:9339 -u admin -p cisco123 --insecure \
    set --delete "/interfaces/interface[name=Loopback100]"

# Subscribe — STREAM SAMPLE every 10 seconds
gnmic -a 192.168.1.1:9339 -u admin -p cisco123 --insecure \
    subscribe --path "/interfaces/interface/state/counters" \
    --stream-mode sample --sample-interval 10s

# Subscribe — ON_CHANGE for interface oper-status
gnmic -a 192.168.1.1:9339 -u admin -p cisco123 --insecure \
    subscribe --path "/interfaces/interface/state/oper-status" \
    --stream-mode on-change

# Subscribe to multiple paths, output to Prometheus
gnmic -a 192.168.1.1:9339 -u admin -p cisco123 --insecure \
    subscribe \
    --path "/interfaces/interface/state/counters" \
    --path "/network-instances/network-instance/protocols/protocol/bgp/neighbors/neighbor/state" \
    --stream-mode sample --sample-interval 30s \
    --format prototext
```

## OpenConfig vs Native Models

```
# OpenConfig — vendor-neutral YANG models
#   Maintained by network operators (Google, Microsoft, AT&T, etc.)
#   Path style: /openconfig-interfaces:interfaces/interface[name=...]/config
#   Pros: portable across vendors, consistent structure
#   Cons: may lack vendor-specific features, lagging coverage

# Native models — vendor-specific YANG modules
#   Cisco IOS-XE: Cisco-IOS-XE-native, Cisco-IOS-XE-bgp, etc.
#   Cisco NX-OS:  Cisco-NX-OS-device
#   Junos:        junos-conf-*, junos-qfx-conf-*
#   Arista EOS:   arista-*, openconfig with extensions
#   Pros: full feature coverage, matches CLI 1:1
#   Cons: vendor lock-in, different per platform

# Strategy: use OpenConfig where possible, augment with native for gaps
```

## Model-Driven Telemetry vs SNMP

```
# SNMP polling (pull model):
#   Manager polls device every N seconds
#   CPU spike on device during each poll cycle
#   Resolution limited by poll interval (typically 5min)
#   UDP transport — no delivery guarantee
#   MIB/OID-based — flat namespace, limited expressiveness

# Model-driven telemetry (push model):
#   Device pushes data to collector (streaming)
#   ON_CHANGE — event-driven, sub-second detection
#   SAMPLE — periodic push, offloads polling from collector
#   gRPC/TCP — reliable delivery, TLS encryption
#   YANG-based — rich hierarchical data models
#   1000x more efficient for high-volume counters
```

## Platform-Specific Enablement

### IOS-XE

```
! Enable NETCONF
netconf-yang
netconf-yang ssh port 830

! Enable RESTCONF
restconf
ip http secure-server

! Enable gNMI (17.x+)
gnxi
gnxi server
gnxi secure-port 9339
```

### NX-OS

```
! Enable NETCONF
feature netconf

! Enable RESTCONF
feature restconf

! Enable gRPC telemetry
feature grpc
grpc port 57400
grpc certificate /bootflash/gnmi-cert.pem
```

### Junos

```
# NETCONF is built in (SSH subsystem)
set system services netconf ssh

# gRPC / Junos Telemetry Interface
set system services extension-service request-response grpc ssl port 32767
set system services extension-service notification allow-clients address 0.0.0.0/0
```

## Tips

- Use `pyang -f tree` output in documentation; it is the standard way to communicate YANG structure.
- RESTCONF PATCH (merge) is safer than PUT (replace) when you only want to change one field.
- NETCONF `edit-config` with `operation="merge"` is the default; use `operation="replace"` explicitly when you need to wipe and rewrite a subtree.
- gNMI ON_CHANGE subscriptions detect link flaps in under a second; SAMPLE is better for counters.
- Always lock the candidate datastore before editing on Junos; concurrent commits cause conflicts.
- ncclient auto-detects device type via `device_params`; use `"csr"` for IOS-XE, `"nexus"` for NX-OS, `"junos"` for Juniper.
- OpenConfig models version slowly; check `openconfig-version` in the module revision before relying on a feature.
- YANG Explorer (Cisco) provides a GUI to browse device models and build RPC payloads interactively.

## See Also

- snmp, grpc, http, curl

## References

- [RFC 7950 — The YANG 1.1 Data Modeling Language](https://www.rfc-editor.org/rfc/rfc7950)
- [RFC 6241 — Network Configuration Protocol (NETCONF)](https://www.rfc-editor.org/rfc/rfc6241)
- [RFC 6242 — Using the NETCONF Protocol over SSH](https://www.rfc-editor.org/rfc/rfc6242)
- [RFC 8040 — RESTCONF Protocol](https://www.rfc-editor.org/rfc/rfc8040)
- [RFC 8340 — YANG Tree Diagrams](https://www.rfc-editor.org/rfc/rfc8340)
- [RFC 8641 — Subscription to YANG Notifications for Datastore Updates](https://www.rfc-editor.org/rfc/rfc8641)
- [gNMI Specification — openconfig/gnmi (GitHub)](https://github.com/openconfig/gnmi)
- [OpenConfig YANG Models (GitHub)](https://github.com/openconfig/public)
- [pyang Documentation](https://github.com/mbj4668/pyang)
- [ncclient Documentation](https://ncclient.readthedocs.io/)
- [gnmic Documentation](https://gnmic.openconfig.net/)
- [Cisco IOS-XE Programmability Guide](https://www.cisco.com/c/en/us/td/docs/ios-xml/ios/prog/configuration/1611/b_1611_programmability_cg.html)
- [Junos Automation and DevOps Guide](https://www.juniper.net/documentation/us/en/software/junos/automation-scripting/index.html)
