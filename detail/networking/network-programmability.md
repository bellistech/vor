# Network Programmability — Model-Driven Network Management

> *Network programmability replaces CLI screen-scraping with structured, machine-readable APIs built on YANG data models. NETCONF provides transactional configuration over SSH, RESTCONF maps YANG to RESTful HTTP, and gNMI delivers high-performance streaming telemetry over gRPC. Together they form the foundation of intent-based networking, where operators declare desired state and devices converge automatically.*

---

## 1. YANG Data Modeling Language

### The Problem YANG Solves

Before YANG, every network vendor defined its own CLI syntax, SNMP MIBs were flat and limited, and automation meant parsing unstructured text output with fragile regular expressions. YANG provides a single, vendor-neutral schema language that formally specifies what data a device exposes, what constraints apply, and what operations are available.

YANG was designed by the NETMOD working group (RFC 6020, updated in RFC 7950 for YANG 1.1) with explicit goals: human-readable, machine-parseable, extensible, and protocol-independent.

### YANG Module Anatomy

A YANG module is a self-contained unit defining a namespace, imports, typedefs, groupings, and the data tree:

```
module <name> {
    namespace "<URI>";       // globally unique identifier
    prefix <short-name>;     // used in XPath and XML

    import <other-module> { prefix <p>; }   // use types/groupings from other modules
    include <submodule>;                     // split large modules into submodules

    revision <date> { description "..."; }  // version tracking

    // Body: containers, lists, leafs, rpcs, notifications
}
```

The `namespace` is a URI (not a URL) that uniquely identifies the module. The `prefix` is a short alias used when referencing nodes from this module in XPath expressions or XML instances.

### Core Node Types

**Container** -- a grouping node that holds other nodes but has no value itself. In XML, it becomes an element with child elements. A container can be a "presence container" (`presence "..."`) which means its existence alone carries meaning (e.g., `container logging { presence "enables logging"; }`).

**List** -- a sequence of entries identified by one or more keys. Each entry is a complete set of the list's child nodes. Keys must be leafs and must be unique within the list:

```yang
list route {
    key "prefix";
    leaf prefix { type inet:ipv4-prefix; }
    leaf next-hop { type inet:ipv4-address; }
    leaf metric { type uint32; }
}
```

This defines a routing table where each route is uniquely identified by its prefix. In NETCONF XML, list entries appear as repeated elements. In RESTCONF URLs, the key value appears in the path: `/routes/route=10.0.0.0%2F8`.

**Leaf** -- a single scalar value. Every leaf has a type (built-in or typedef). Built-in types include `string`, `int8..int64`, `uint8..uint64`, `boolean`, `enumeration`, `bits`, `binary`, `empty`, `union`, `leafref`, `identityref`, `instance-identifier`.

**Leaf-list** -- an ordered or unordered sequence of scalar values (like an array of leafs):

```yang
leaf-list dns-server {
    type inet:ipv4-address;
    ordered-by user;
}
```

### Typedefs and Groupings

Typedefs create reusable named types with constraints:

```yang
typedef percentage {
    type uint8 {
        range "0..100";
    }
    description "A value between 0 and 100 inclusive";
}

leaf cpu-utilization {
    type percentage;
}
```

Groupings define reusable sets of nodes, instantiated with `uses`:

```yang
grouping address-family {
    leaf ipv4-unicast { type boolean; default true; }
    leaf ipv6-unicast { type boolean; default false; }
}

container bgp {
    list neighbor {
        key "address";
        leaf address { type inet:ip-address; }
        uses address-family;    // inlines all the grouping's nodes here
    }
}
```

### Augment and Deviation

**Augment** allows one module to add nodes to another module's data tree without modifying the original. This is how vendors extend standard models:

```yang
// In a vendor-specific module
augment "/ietf-if:interfaces/ietf-if:interface" {
    when "ietf-if:type = 'ianaift:ethernetCsmacd'";
    container cisco-qos {
        leaf ingress-policy { type string; }
        leaf egress-policy { type string; }
    }
}
```

The `when` clause restricts the augmentation to Ethernet interfaces only.

**Deviation** documents how an implementation differs from the standard model. This is not a way to change the model; it is a formal declaration of non-compliance:

```yang
deviation "/ietf-if:interfaces/ietf-if:interface/ietf-if:mtu" {
    deviate replace {
        type uint16 {
            range "68..9216";   // hardware limits
        }
    }
}

deviation "/ietf-if:interfaces/ietf-if:interface/ietf-if:link-up-down-trap-enable" {
    deviate not-supported;  // this device does not implement this leaf
}
```

### YANG XPath Constraints

YANG uses XPath 1.0 for `when`, `must`, and `leafref` path expressions:

```yang
leaf vlan-id {
    type uint16 { range "1..4094"; }
}

leaf vlan-name {
    type string;
    must "../vlan-id >= 1 and ../vlan-id <= 4094" {
        error-message "VLAN ID must be in range 1-4094";
    }
}

// leafref — pointer to another node's value
leaf interface-ref {
    type leafref {
        path "/interfaces/interface/name";
    }
    description "Must reference an existing interface name";
}
```

### OpenConfig vs IETF vs Native Models

Three model families coexist:

| Aspect | IETF Models | OpenConfig Models | Native Models |
|:---|:---|:---|:---|
| Maintained by | IETF working groups | Network operators (Google, MSFT, ATT) | Individual vendors |
| Naming | ietf-interfaces, ietf-routing | openconfig-interfaces, openconfig-bgp | Cisco-IOS-XE-native, junos-conf |
| Config/state split | Separate trees (RFC 8040) | Unified config+state containers | Varies |
| Versioning | RFC revision dates | Semantic versioning (openconfig-version) | Platform release tied |
| Coverage | Core protocols only | Growing but not exhaustive | Full platform features |
| Portability | High (standardized) | High (multi-vendor) | None (vendor-specific) |

OpenConfig models use a consistent pattern: every list entry has a `/config` subtree for writable parameters and a `/state` subtree mirroring those values plus operational counters. This makes it unambiguous which leafs are configurable.

---

## 2. NETCONF Protocol

### Protocol Layers

NETCONF (RFC 6241) defines four layers:

1. **Transport** -- SSH (RFC 6242, mandatory), TLS (RFC 7589, optional). SSH uses the `netconf` subsystem on port 830. Messages are framed with `]]>]]>` (NETCONF 1.0) or chunked framing with `\n#<length>\n` (NETCONF 1.1).

2. **Messages** -- XML-encoded RPC request/reply pairs. Every `<rpc>` carries a `message-id` attribute; the server echoes it in `<rpc-reply>`.

3. **Operations** -- the verbs (get, get-config, edit-config, copy-config, delete-config, lock, unlock, close-session, kill-session, commit, validate, discard-changes).

4. **Content** -- the actual configuration and state data, structured per YANG models.

### Capability Exchange

When a NETCONF session opens, both sides exchange `<hello>` messages listing their capabilities:

```xml
<hello xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <capabilities>
    <capability>urn:ietf:params:netconf:base:1.1</capability>
    <capability>urn:ietf:params:netconf:capability:candidate:1.0</capability>
    <capability>urn:ietf:params:netconf:capability:confirmed-commit:1.1</capability>
    <capability>urn:ietf:params:netconf:capability:validate:1.1</capability>
    <capability>urn:ietf:params:netconf:capability:xpath:1.0</capability>
    <capability>urn:ietf:params:netconf:capability:rollback-on-error:1.0</capability>
  </capabilities>
  <session-id>42</session-id>
</hello>
```

Key capabilities and what they unlock:

| Capability | Effect |
|:---|:---|
| `:candidate` | Enables candidate datastore for staged commits |
| `:confirmed-commit` | Commit with automatic rollback if not confirmed within timeout |
| `:validate` | Enables `<validate>` RPC to check config before committing |
| `:xpath` | Allows XPath expressions in filter elements |
| `:rollback-on-error` | Rolls back entire edit-config if any part fails |
| `:with-defaults` | Controls how default values appear in replies |
| `:yang-library` | Device publishes its YANG module inventory |

### Datastore Model

The **running** datastore is always present and represents the active configuration. The **candidate** datastore (optional) acts as a scratch area where changes can be staged, validated, and committed atomically. The **startup** datastore (optional) persists across reboots.

Workflow with candidate:

```
1. lock(candidate)          -- prevent other sessions from editing
2. edit-config(candidate)   -- stage changes
3. validate(candidate)      -- syntax/semantic check
4. commit                   -- apply candidate -> running atomically
5. unlock(candidate)        -- release lock

On failure: discard-changes  -- revert candidate to match running
```

Confirmed commit adds a safety net: `<commit><confirmed/><confirm-timeout>120</confirm-timeout></commit>` applies the change but auto-rolls back in 120 seconds unless a second `<commit/>` confirms it. This prevents locking yourself out of a remote device.

### Subtree vs XPath Filtering

**Subtree filtering** (always supported) uses XML structure matching:

```xml
<filter type="subtree">
  <interfaces xmlns="urn:ietf:params:xml:ns:yang:ietf-interfaces">
    <interface>
      <name>eth0</name>
      <!-- only return nodes under interface named eth0 -->
    </interface>
  </interfaces>
</filter>
```

The filter acts as a template: the server returns only data matching the structure. Empty elements are "select" nodes (return everything beneath). Elements with values are "content match" nodes (filter by value).

**XPath filtering** (requires `:xpath` capability) is more expressive:

```xml
<filter type="xpath"
  select="/interfaces/interface[enabled='true' and type='ianaift:ethernetCsmacd']"/>
```

XPath can express conditions that subtree filtering cannot, such as selecting interfaces where `in-octets > 1000000` or joining across subtrees.

### edit-config Operations

The `operation` attribute on elements within `edit-config` controls behavior:

| Operation | Behavior |
|:---|:---|
| `merge` (default) | Merge supplied data with existing config |
| `replace` | Replace the entire target subtree |
| `create` | Create only if it does not exist; error otherwise |
| `delete` | Delete; error if it does not exist |
| `remove` | Delete if it exists; no error if absent |

```xml
<interface operation="replace">
  <name>eth0</name>
  <description>Replaced entire interface config</description>
  <enabled>true</enabled>
</interface>
```

The `default-operation` attribute on `<edit-config>` sets the default for all elements: `merge` (default), `replace`, or `none`. Using `none` forces you to specify the operation on every element, preventing accidental merges.

---

## 3. RESTCONF Protocol

### Mapping YANG to HTTP

RESTCONF (RFC 8040) provides a RESTful interface to the same YANG-modeled data that NETCONF serves. The key insight is that YANG's tree structure maps naturally to URI paths:

```
YANG tree:                          RESTCONF URI:
/interfaces                         /restconf/data/ietf-interfaces:interfaces
  /interface[name="eth0"]           /restconf/data/ietf-interfaces:interfaces/interface=eth0
    /description                    /restconf/data/ietf-interfaces:interfaces/interface=eth0/description
    /enabled                        /restconf/data/ietf-interfaces:interfaces/interface=eth0/enabled
```

List keys are encoded in the URL as `=<key>` after the list name. For composite keys: `=<key1>,<key2>`. Special characters are percent-encoded (`/` becomes `%2F`).

### Resource Types

RESTCONF defines two resource types:

- **Data resource** (`/restconf/data/...`) -- configuration and state data. Supports GET, POST, PUT, PATCH, DELETE.
- **Operations resource** (`/restconf/operations/...`) -- YANG RPCs and actions. Supports POST only.

Additional well-known resources:

- `/restconf/yang-library-version` -- which YANG library version the server implements
- `/.well-known/host-meta` -- discovery of the RESTCONF API root

### HTTP Method Semantics

| Method | YANG Equivalent | Behavior | Success Code |
|:---|:---|:---|:---|
| GET | get / get-config | Read data | 200 |
| POST (to parent) | edit-config (create) | Create new list entry or child | 201 |
| PUT | edit-config (replace) | Create or replace entire resource | 201 or 204 |
| PATCH | edit-config (merge) | Merge changes into existing | 204 |
| DELETE | edit-config (delete) | Remove resource | 204 |

PATCH is generally preferred over PUT because it only modifies the fields you supply, leaving the rest untouched. PUT replaces the entire resource, which means you must supply all mandatory fields even if you only want to change one.

### Content Negotiation

RESTCONF supports two encodings:

```
Accept: application/yang-data+json     -- JSON (more common for automation)
Accept: application/yang-data+xml      -- XML

Content-Type: application/yang-data+json   -- for request bodies
```

The JSON encoding follows RFC 7951 rules: module names prefix the first occurrence of a node from a different module (`"ietf-interfaces:interfaces"`), and YANG identities become strings (`"iana-if-type:ethernetCsmacd"`).

### Query Parameters

RESTCONF supports several query parameters for GET requests:

| Parameter | Example | Purpose |
|:---|:---|:---|
| `depth` | `?depth=2` | Limit response depth (1 = only the requested node) |
| `fields` | `?fields=name;enabled` | Return only specific leafs |
| `content` | `?content=config` | `config` / `nonconfig` / `all` |
| `with-defaults` | `?with-defaults=report-all` | Control default value reporting |
| `filter` | `?filter=/interface[enabled='true']` | XPath-style filtering (vendor extension) |

### RESTCONF vs NETCONF Comparison

| Feature | NETCONF | RESTCONF |
|:---|:---|:---|
| Transport | SSH (port 830) | HTTPS (port 443) |
| Encoding | XML only | JSON or XML |
| Session state | Yes (lock, session-id) | No (stateless REST) |
| Transactions | candidate + commit | Per-request (or PATCH) |
| Streaming events | NETCONF notifications | SSE (Server-Sent Events, RFC 8040 sec. 6) |
| Firewall friendly | Needs port 830 open | Standard HTTPS |
| Tooling | ncclient, specialized | curl, requests, any HTTP library |
| Bulk operations | Single edit-config with full tree | Multiple HTTP requests |

NETCONF is better for atomic multi-device transactions (lock, edit, validate, commit). RESTCONF is better for lightweight queries, web integrations, and environments where only HTTPS is allowed.

---

## 4. gNMI and Streaming Telemetry

### Why gNMI Exists

NETCONF and RESTCONF were designed primarily for configuration management. While they can retrieve operational state, they use a request-response pattern that does not scale for high-frequency telemetry. gNMI (gRPC Network Management Interface) was created by the OpenConfig consortium specifically to address:

1. **High-volume telemetry** -- push thousands of counters per second without polling overhead
2. **Low latency** -- sub-second event detection via ON_CHANGE subscriptions
3. **Efficient encoding** -- Protobuf binary serialization (much smaller than XML/JSON)
4. **Bidirectional streaming** -- gRPC natively supports server-push and client-streaming

### gNMI RPCs

gNMI defines four RPCs in its Protobuf service definition:

**Capabilities** -- returns the set of YANG models the device supports, supported encodings (JSON_IETF, PROTO, ASCII), and gNMI version:

```protobuf
rpc Capabilities(CapabilityRequest) returns (CapabilityResponse);
```

**Get** -- retrieves a snapshot of data at one or more paths. Equivalent to NETCONF get/get-config but supports requesting specific data types (CONFIG, STATE, OPERATIONAL, ALL):

```protobuf
rpc Get(GetRequest) returns (GetResponse);
// GetRequest contains: path[], type, encoding
```

**Set** -- modifies configuration. Supports three operations in a single atomic request:

```protobuf
rpc Set(SetRequest) returns (SetResponse);
// SetRequest contains: delete[], replace[], update[]
```

- `delete` -- remove paths entirely
- `replace` -- replace subtree at path (like PUT)
- `update` -- merge data at path (like PATCH)

Operations within a single SetRequest are applied atomically.

**Subscribe** -- the core telemetry RPC. Opens a bidirectional stream:

```protobuf
rpc Subscribe(stream SubscribeRequest) returns (stream SubscribeResponse);
```

### Subscription Modes In Depth

**STREAM** mode opens a persistent connection. Within STREAM, two sub-modes control when updates are pushed:

- **SAMPLE** -- the device pushes the current value at a fixed interval (e.g., every 10 seconds). Best for counters (interface bytes, CPU utilization) where you want regular data points regardless of change.

- **ON_CHANGE** -- the device pushes only when the value changes. Best for state (interface up/down, BGP neighbor state, route additions). Dramatically reduces bandwidth for stable values but can burst during convergence events.

**ONCE** mode sends a single snapshot of all requested paths and then closes the stream. Equivalent to Get but using the Subscribe RPC (useful when you want the same path encoding).

**POLL** mode keeps the stream open but only sends data when the client sends a `Poll` message on the stream. The client controls the timing. Less common than STREAM but useful for on-demand dashboards.

### Dial-In vs Dial-Out

**Dial-in** (standard gNMI) -- the collector (client) initiates a gRPC connection to the device (server) and subscribes. This is the native gNMI model. Requires the collector to know device addresses and have network reachability.

**Dial-out** -- the device initiates the connection to the collector. Configured on the device with collector addresses and subscription paths. Useful when devices are behind NAT or firewalls. Not part of the gNMI spec; implemented as vendor extensions (Cisco MDT, Junos JTI).

### Path Encoding

gNMI paths use a structured representation rather than XPath strings:

```
Path {
    elem: [
        PathElem { name: "interfaces" },
        PathElem { name: "interface", key: {"name": "eth0"} },
        PathElem { name: "state" },
        PathElem { name: "counters" }
    ]
}
```

In gnmic CLI and most tools, this is written as a simplified string: `/interfaces/interface[name=eth0]/state/counters`.

Wildcards are supported: `/interfaces/interface[name=*]/state/oper-status` subscribes to the operational status of all interfaces.

### Telemetry Data Pipeline

A production gNMI telemetry stack typically involves:

```
Network Devices (gNMI servers)
    |
    v  gRPC streams (Protobuf)
Collectors (gnmic, Telegraf, pipeline)
    |
    v  Write to TSDB
Time-Series Database (Prometheus, InfluxDB, Kafka)
    |
    v  Query
Visualization (Grafana, Kibana)
    |
    v  Alerts
Alert Manager (PagerDuty, Slack)
```

gnmic supports direct output to Prometheus (pull or push gateway), Kafka, NATS, InfluxDB, and file. Telegraf's `gnmi` input plugin provides similar functionality.

---

## 5. Model-Driven Telemetry vs SNMP Polling

### Fundamental Architecture Difference

SNMP is a **pull** model: the NMS asks each device "what are your counters right now?" at fixed intervals. The device must process the request, walk its internal data structures, encode the response, and send it back. During the poll, CPU spikes on the device.

Model-driven telemetry is a **push** model: the device continuously monitors subscribed paths and pushes updates to the collector. The device's forwarding ASIC often provides counters natively; the telemetry subsystem simply reads them and streams to gRPC.

### Quantitative Comparison

| Metric | SNMP Polling | gNMI Streaming |
|:---|:---|:---|
| Typical resolution | 5 minutes | 1-10 seconds (SAMPLE) |
| Fastest detection | Equal to poll interval | Sub-second (ON_CHANGE) |
| CPU impact on device | Spikes each poll cycle | Steady low overhead |
| Bandwidth (idle) | Zero between polls | Near-zero (ON_CHANGE) |
| Bandwidth (active) | Proportional to OID count | Proportional to change rate |
| Encoding overhead | ASN.1 BER (~30% overhead) | Protobuf (~5% overhead) |
| Transport | UDP (unreliable) | gRPC/TCP/TLS (reliable) |
| Schema language | MIB (SMIv2) | YANG |
| Data richness | Flat OID namespace | Hierarchical tree with constraints |
| Multi-value atomicity | Each OID polled separately | Entire subtree in one update |

### When to Use Each

SNMP remains appropriate for:
- Legacy devices with no YANG/gNMI support
- Simple up/down monitoring where 5-minute resolution suffices
- Trap-based event notification (syslog is the real competitor here)
- Environments with existing MIB-based tooling

gNMI/model-driven telemetry is superior for:
- High-frequency counter monitoring (interface utilization, queue depth)
- Real-time event detection (link flaps, BGP state changes)
- Large-scale networks (10,000+ devices) where polling does not scale
- Automation pipelines that need structured, typed data

---

## 6. pyang, YANG Explorer, and Tooling

### pyang

pyang is the reference YANG validator and code generator. Essential operations:

```bash
# Validate syntax and semantics
pyang --lint module.yang

# Tree output (the standard visualization)
pyang -f tree module.yang
# +--rw indicates read-write (config)
# +--ro indicates read-only (state)
# ? after type means optional
# * after name means list

# Generate DSDL schemas (for XML validation)
pyang -f dsdl module.yang

# Generate YIN (XML representation of YANG)
pyang -f yin module.yang

# Generate sample XML instance
pyang -f sample-xml-skeleton --sample-xml-skeleton-doctype=config module.yang

# Check backward compatibility
pyang --check-update-from old-module.yang new-module.yang
```

### YANG Catalog and Model Discovery

The YANG Catalog (yangcatalog.org) indexes YANG modules from all major vendors and standards bodies. It provides:
- Searchable index of 40,000+ modules
- Module impact analysis (who imports/augments what)
- YANG validation service
- API for programmatic module discovery

On a device, NETCONF advertises supported models via the `<hello>` capabilities list or the YANG Library (RFC 8525): `GET /restconf/data/ietf-yang-library:yang-library`.

### ncclient Advanced Patterns

```python
from ncclient import manager
from lxml import etree

# Async dispatch for multiple devices
import concurrent.futures

def get_interfaces(host):
    with manager.connect(host=host, port=830,
                         username="admin", password="pass",
                         hostkey_verify=False) as m:
        return m.get_config("running", filter=("subtree", """
            <interfaces xmlns="urn:ietf:params:xml:ns:yang:ietf-interfaces"/>
        """))

hosts = ["10.0.0.1", "10.0.0.2", "10.0.0.3"]
with concurrent.futures.ThreadPoolExecutor(max_workers=10) as pool:
    results = pool.map(get_interfaces, hosts)
    for host, result in zip(hosts, results):
        root = etree.fromstring(result.xml.encode())
        ifaces = root.findall(".//{urn:ietf:params:xml:ns:yang:ietf-interfaces}interface")
        print(f"{host}: {len(ifaces)} interfaces")
```

### pygnmi — Python gNMI Client

```python
from pygnmi.client import gNMIclient

with gNMIclient(
    target=("192.168.1.1", 9339),
    username="admin", password="pass",
    insecure=True
) as gc:
    # Capabilities
    caps = gc.capabilities()

    # Get
    result = gc.get(path=["/interfaces/interface[name=eth0]/state"])

    # Set (update)
    gc.set(update=[
        ("/interfaces/interface[name=eth0]/config/description",
         {"description": "Managed by pygnmi"})
    ])

    # Subscribe (blocking generator)
    subscribe_request = {
        "subscription": [
            {"path": "/interfaces/interface/state/counters",
             "mode": "sample", "sample_interval": 10_000_000_000}  # 10s in ns
        ],
        "mode": "stream",
        "encoding": "json_ietf"
    }
    for response in gc.subscribe2(subscribe=subscribe_request):
        print(response)
```

---

## 7. Practical Workflows

### Configuration Backup and Diff

```python
# NETCONF-based config backup with diff
from ncclient import manager
from datetime import datetime
import difflib

def backup_config(host, label):
    with manager.connect(host=host, port=830,
                         username="admin", password="pass",
                         hostkey_verify=False, device_params={"name": "csr"}) as m:
        config = m.get_config("running").xml
        filename = f"backup_{host}_{label}_{datetime.now():%Y%m%d_%H%M%S}.xml"
        with open(filename, "w") as f:
            f.write(config)
        return filename

# Compare two backups
def diff_configs(file1, file2):
    with open(file1) as f1, open(file2) as f2:
        diff = difflib.unified_diff(
            f1.readlines(), f2.readlines(),
            fromfile=file1, tofile=file2
        )
        return "".join(diff)
```

### Atomic Multi-Step Configuration

```python
# Candidate datastore workflow: configure BGP + route-policy atomically
with manager.connect(host="router1", port=830,
                     username="admin", password="pass",
                     hostkey_verify=False) as m:

    with m.locked("candidate"):
        # Step 1: add prefix-list
        m.edit_config(target="candidate", config=prefix_list_xml)

        # Step 2: add route-map referencing prefix-list
        m.edit_config(target="candidate", config=route_map_xml)

        # Step 3: add BGP neighbor referencing route-map
        m.edit_config(target="candidate", config=bgp_neighbor_xml)

        # Validate entire candidate
        m.validate("candidate")

        # Commit atomically (all or nothing)
        # Use confirmed-commit for safety on remote devices
        m.commit(confirmed=True, timeout="120")

        # ... verify connectivity ...

        # Confirm (prevent rollback)
        m.commit()
```

### gNMI Telemetry to Prometheus

```yaml
# gnmic configuration file (gnmic.yaml)
targets:
  spine1:
    address: 10.0.0.1:9339
    username: admin
    password: pass
    insecure: true
  spine2:
    address: 10.0.0.2:9339
    username: admin
    password: pass
    insecure: true

subscriptions:
  interface-counters:
    paths:
      - /interfaces/interface/state/counters
    stream-mode: sample
    sample-interval: 10s
    encoding: json_ietf

  bgp-state:
    paths:
      - /network-instances/network-instance/protocols/protocol/bgp/neighbors/neighbor/state
    stream-mode: on-change
    encoding: json_ietf

outputs:
  prometheus-output:
    type: prometheus
    listen: ":9804"
    path: /metrics
    metric-prefix: gnmi
    append-subscription-name: true
```

```bash
# Run gnmic as a Prometheus exporter
gnmic --config gnmic.yaml subscribe

# Prometheus scrape config (prometheus.yml)
# scrape_configs:
#   - job_name: 'gnmi'
#     static_configs:
#       - targets: ['localhost:9804']
```

---

## 8. Security Considerations

### Transport Security

All three protocols mandate or strongly recommend TLS:

- **NETCONF over SSH** -- inherits SSH key exchange and encryption. Host key verification is critical; disabling it (`hostkey_verify=False`) is acceptable only in labs.
- **RESTCONF over HTTPS** -- standard TLS certificate validation. Use proper CA-signed certificates in production.
- **gNMI over gRPC/TLS** -- mutual TLS (mTLS) is recommended. The client presents a certificate; the server validates it. This provides both encryption and client authentication.

### Authentication and Authorization

NETCONF and RESTCONF typically use local credentials or TACACS+/RADIUS for AAA. gNMI authentication is handled at the gRPC metadata level (username/password in the call metadata) or via mTLS client certificates.

NACM (NETCONF Access Control Model, RFC 8341) provides fine-grained authorization:

```xml
<!-- Allow monitoring group to read but not write -->
<rule>
    <name>deny-write-monitoring</name>
    <module-name>*</module-name>
    <access-operations>create update delete</access-operations>
    <action>deny</action>
</rule>
```

### Rate Limiting and Session Limits

Production deployments should limit:
- Maximum concurrent NETCONF sessions (prevent resource exhaustion)
- RESTCONF request rate (standard HTTP rate limiting)
- gNMI subscription count per client (prevent telemetry overload)
- Subscription sample intervals (minimum 1 second to avoid flooding)

---

## 9. Summary of Protocols and Trade-offs

| Dimension | NETCONF | RESTCONF | gNMI |
|:---|:---|:---|:---|
| Primary use case | Transactional config | Lightweight config queries | Streaming telemetry + config |
| Transport | SSH | HTTPS | gRPC (HTTP/2 + TLS) |
| Encoding | XML | JSON or XML | Protobuf, JSON_IETF |
| Statefulness | Stateful (sessions, locks) | Stateless | Stateful (subscriptions) |
| Streaming | Notifications (limited) | SSE (limited) | Native (Subscribe RPC) |
| Atomicity | Candidate + commit | Per-request | Set (delete+replace+update) |
| Maturity | RFC 6241 (2011) | RFC 8040 (2017) | OpenConfig spec (2017+) |
| Best for | Network-wide atomic changes | Web integrations, simple queries | Real-time monitoring at scale |

## Prerequisites

- data modeling, XML/JSON encoding, SSH, TLS/HTTPS, gRPC, protocol buffers, REST APIs

---

*Network programmability transforms infrastructure from a collection of individually configured boxes into a programmable fabric governed by data models. YANG defines the contract between operator intent and device behavior, NETCONF provides the transactional guarantees, RESTCONF lowers the barrier to entry, and gNMI delivers the real-time visibility that modern networks demand. The shift from CLI to model-driven management is not optional -- it is the foundation upon which intent-based networking, closed-loop automation, and self-healing infrastructure are built.*
