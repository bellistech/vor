# gNMI and gNOI — gRPC-Based Network Management Architecture

> *gNMI (gRPC Network Management Interface) and gNOI (gRPC Network Operations Interface) represent the modern model-driven approach to network management, replacing legacy SNMP and CLI scraping with strongly-typed, streaming-capable, protobuf-encoded RPCs over HTTP/2. gNMI handles configuration and telemetry (Get/Set/Subscribe), while gNOI provides operational lifecycle services (reboot, file transfer, certificate management, OS installation). Together they form the programmatic control plane for OpenConfig-aligned network automation.*

---

## 1. gRPC Transport Layer

### HTTP/2 Foundation

gNMI and gNOI are built on gRPC, which itself runs over HTTP/2. This provides several advantages over the SSH-based transports used by NETCONF and CLI:

| Feature | HTTP/2 (gNMI) | SSH (NETCONF) | HTTP/1.1 (RESTCONF) |
|:---|:---|:---|:---|
| Multiplexing | Full stream multiplexing | Single channel per session | Request-response only |
| Header compression | HPACK | None | None |
| Flow control | Per-stream | TCP only | TCP only |
| Server push | Native (Subscribe) | Notification (RFC 5277) | SSE (limited) |
| Binary framing | Yes | No (XML text) | No (JSON/XML text) |
| Connection reuse | Multiple RPCs on one connection | One session per connection | Keep-alive possible |

### Connection Lifecycle

```
Client                                     Server (Device)
   |                                          |
   |--- TCP SYN ---→                          |
   |←-- TCP SYN-ACK ---|                      |
   |--- TLS ClientHello ---→                  |
   |←-- TLS ServerHello + Cert ---|           |
   |--- TLS Finished ---→                     |
   |←-- TLS Finished ---|                     |
   |--- HTTP/2 SETTINGS ---→                  |
   |←-- HTTP/2 SETTINGS ---|                  |
   |--- gRPC Request (Get/Set/Subscribe) ---→ |
   |←-- gRPC Response ---|                    |
   |--- gRPC Request (reuse connection) ---→  |
   |←-- gRPC Response ---|                    |
```

A single gRPC channel (TCP connection) supports multiple concurrent RPCs via HTTP/2 stream multiplexing. This is particularly valuable for Subscribe, where a long-lived stream coexists with Get/Set operations.

---

## 2. Protobuf Encoding

### gNMI Proto Definition

The gNMI service is defined in `gnmi.proto`:

```protobuf
service gNMI {
    rpc Capabilities(CapabilityRequest) returns (CapabilityResponse);
    rpc Get(GetRequest)                 returns (GetResponse);
    rpc Set(SetRequest)                 returns (SetResponse);
    rpc Subscribe(stream SubscribeRequest) returns (stream SubscribeResponse);
}
```

Key message types:

| Message | Purpose | Key Fields |
|:---|:---|:---|
| `Path` | YANG path representation | `origin`, `elem[]` (name + key map) |
| `TypedValue` | Leaf value with type | `string_val`, `int_val`, `json_ietf_val`, etc. |
| `Update` | Path + value pair | `path`, `val` |
| `Notification` | Timestamped update set | `timestamp`, `prefix`, `update[]`, `delete[]` |
| `SubscribeRequest` | Subscription parameters | `subscribe` (SubscriptionList) or `poll` |
| `SubscribeResponse` | Subscription data | `update` (Notification) or `sync_response` |

### Path Encoding

gNMI paths are encoded as a sequence of `PathElem` messages, each containing a name and optional key-value map:

```
YANG path:  /interfaces/interface[name=Ethernet1]/state/oper-status

Protobuf encoding:
  Path {
    origin: "openconfig"
    elem: [ {name: "interfaces"},
            {name: "interface", key: {"name": "Ethernet1"}},
            {name: "state"},
            {name: "oper-status"} ]
  }
```

This structured encoding avoids the ambiguity of string-based path representations and enables efficient matching in the server's path trie.

### Encoding Formats

gNMI supports multiple value encodings:

| Encoding | Proto Field | Use Case |
|:---|:---|:---|
| JSON | `json_val` | Human-readable, legacy compatibility |
| JSON_IETF | `json_ietf_val` | IETF-compliant JSON (RFC 7951) with module prefixes |
| BYTES | `bytes_val` | Opaque binary data |
| PROTO | `any_val` | Protobuf-encoded vendor models |
| ASCII | `ascii_val` | Plain text (CLI output) |
| SCALAR | Various `*_val` | Individual typed values (string, int, uint, bool, float, decimal) |

---

## 3. gNMI Specification Details

### Get RPC

Get is a unary RPC: one request, one response. The client specifies paths and an optional data type filter:

| Data Type | Content |
|:---|:---|
| `ALL` | Configuration + state |
| `CONFIG` | Configuration only (intended state) |
| `STATE` | Operational state only (derived/computed) |
| `OPERATIONAL` | Operational data not modeled as state |

The server returns a `GetResponse` containing one `Notification` per requested path, each with the current values at that path.

### Set RPC

Set is a unary RPC that supports three operations in a single transaction:

1. **Delete** — remove paths (processed first)
2. **Replace** — replace subtree at path (processed second)
3. **Update** — merge values at path (processed third)

The ordering guarantees atomicity: deletes clear the slate, replaces set the baseline, updates apply incremental changes. All operations within a single SetRequest are applied atomically — either all succeed or none do.

### Subscribe RPC

Subscribe is a bidirectional streaming RPC. The client sends `SubscribeRequest` messages, the server responds with `SubscribeResponse` messages:

**Subscription Modes:**

| Mode | Behavior | Connection Lifetime |
|:---|:---|:---|
| `ONCE` | Server sends current state, then `sync_response`, then closes | Short-lived |
| `POLL` | Server sends state on each client `poll` request | Long-lived, client-driven |
| `STREAM` | Server pushes updates continuously | Long-lived, server-driven |

**Stream Sub-modes:**

| Stream Mode | Trigger | Best For |
|:---|:---|:---|
| `ON_CHANGE` | Value changes | Boolean/enum state (oper-status, admin-status) |
| `SAMPLE` | Timer expiry | Counters, gauges (bytes, packets, CPU) |
| `TARGET_DEFINED` | Device decides per path | Mixed workloads, device optimization |

### Subscription Semantics

**Heartbeat Interval:** For `ON_CHANGE` subscriptions, the `heartbeat_interval` field forces the server to resend the current value even if unchanged. This serves as a liveness signal — if no heartbeat arrives, the collector knows the subscription or device has failed.

**Suppress Redundant:** For `SAMPLE` subscriptions, `suppress_redundant` tells the server to skip sending a sample if the value has not changed since the last transmission. This reduces bandwidth for slowly-changing values while maintaining the guaranteed maximum latency of the sample interval.

**Sync Response:** After the initial burst of current-state updates, the server sends a `sync_response = true` message. This tells the client that all initial data has been delivered and subsequent messages represent real-time changes.

---

## 4. gNMI vs NETCONF/RESTCONF Comparison

### Protocol Comparison

| Dimension | gNMI | NETCONF | RESTCONF |
|:---|:---|:---|:---|
| Transport | gRPC (HTTP/2) | SSH | HTTPS (HTTP/1.1 or 2) |
| Encoding | Protobuf + JSON/JSON_IETF | XML | JSON or XML |
| Data Model | YANG (OpenConfig focus) | YANG | YANG |
| Streaming Telemetry | Native (Subscribe) | RFC 5277 notifications | SSE (limited) |
| Transaction | Set (atomic delete/replace/update) | edit-config (candidate commit) | PATCH/PUT |
| Candidate Datastore | No (direct apply) | Yes (lock/edit/commit) | No |
| Confirmed Commit | No | Yes (RFC 6241) | No |
| Connection Overhead | Low (binary, multiplexed) | Medium (SSH + XML) | Low (HTTP) |
| Tooling Maturity | Growing (gnmic, pygnmi) | Mature (ncclient, NAPALM) | Mature (curl, requests) |
| Vendor Support | Arista, Cisco, Juniper, Nokia | Universal | Cisco, Juniper |

### When to Use Each

- **gNMI** — streaming telemetry at scale, high-frequency counter collection, environments standardized on OpenConfig
- **NETCONF** — full configuration lifecycle with candidate datastore, confirmed commit, and lock semantics
- **RESTCONF** — quick integration with existing HTTP tooling, web-based dashboards, ad-hoc queries

---

## 5. gNOI Service Architecture

### Service Catalog

gNOI defines multiple gRPC services, each handling a specific operational domain:

| Service | Proto File | Key RPCs |
|:---|:---|:---|
| `gnoi.system.System` | `system.proto` | `Reboot`, `RebootStatus`, `Time`, `Ping`, `Traceroute`, `SwitchControlProcessor` |
| `gnoi.file.File` | `file.proto` | `Get`, `Put`, `Stat`, `Remove`, `TransferToRemote` |
| `gnoi.cert.CertificateManagement` | `cert.proto` | `Install`, `Rotate`, `GetCertificates`, `RevokeCertificates` |
| `gnoi.os.OS` | `os.proto` | `Install`, `Activate`, `Verify` |
| `gnoi.healthz.Healthz` | `healthz.proto` | `Get`, `Check`, `Acknowledge` |
| `gnoi.bgp.BGP` | `bgp.proto` | `ClearBGPNeighbor` |
| `gnoi.layer2.Layer2` | `layer2.proto` | `ClearLLDPInterface`, `ClearSpanningTree` |
| `gnoi.diag.Diag` | `diag.proto` | `StartBERT`, `StopBERT`, `GetBERTResult` |
| `gnoi.mpls.MPLS` | `mpls.proto` | `ClearLSP`, `MPLSPing` |
| `gnoi.packet_link_qualification.LinkQualification` | `plq.proto` | `Create`, `Get`, `Delete`, `List` |

### OS Installation Workflow

The gNOI OS service implements a multi-stage installation process:

```
1. Install RPC (streaming)
   Client → TransferRequest (version)
   Client → TransferContent (chunked image data, ~64KB per message)
   Client → TransferEnd
   Server → InstallResponse (status: IN_PROGRESS, COMPLETE, or ERROR)

2. Activate RPC
   Client → ActivateRequest (version, no_reboot optional)
   Server → ActivateResponse (activation status)

3. Verify RPC (post-reboot)
   Client → VerifyRequest
   Server → VerifyResponse (running version, activation status)
```

This design supports zero-touch OS upgrades with verification gates at each stage.

### Certificate Rotation

The cert service implements hitless certificate rotation:

```
1. Rotate RPC (streaming)
   Client → GenerateCSRRequest (key type, key size)
   Server → GenerateCSRResponse (CSR in PEM)
   [Client signs CSR with CA, obtains certificate]
   Client → LoadCertificateRequest (certificate chain)
   Server → LoadCertificateResponse (OK or error)
   Client → FinalizeRequest
   Server → connection switches to new certificate
```

If the new certificate causes connection failure, the server automatically rolls back to the previous certificate after a configurable timeout.

---

## 6. Telemetry Pipeline Design

### Collection Architecture

A production gNMI telemetry pipeline typically follows this pattern:

```
Network Devices                  Collection Tier           Storage + Analytics
┌─────────────┐
│  Router 1   │─── gNMI ──→ ┌──────────────┐
│  (gNMI srv) │              │   gnmic /     │          ┌─────────────┐
└─────────────┘              │   telegraf    │──write──→│  InfluxDB / │
┌─────────────┐              │              │          │  Prometheus │
│  Router 2   │─── gNMI ──→ │  (collector)  │          └─────────────┘
│  (gNMI srv) │              └──────────────┘                │
└─────────────┘                     │                        │
┌─────────────┐                     │                   ┌────▼────┐
│  Switch 1   │─── gNMI ──→        │                   │ Grafana │
│  (gNMI srv) │              ┌──────▼──────┐            └─────────┘
└─────────────┘              │   Kafka /    │
                             │   NATS       │──→ Long-term analytics
                             └─────────────┘
```

### Scaling Considerations

| Factor | Guidance |
|:---|:---|
| Subscription count | ~100-500 per collector instance (depends on path depth) |
| Sample interval | 10-30s for counters, on-change for state (avoid <5s intervals) |
| Path granularity | Subscribe to specific leaves, not entire subtrees |
| Connection count | One gRPC channel per device (multiplexed streams) |
| Collector HA | Run multiple collectors with device sharding |
| Back-pressure | Use Kafka/NATS buffer between collector and storage |

### Data Volume Estimation

$$\text{Messages/sec} = \sum_{d \in \text{devices}} \sum_{s \in \text{subs}_d} \frac{\text{paths}_s}{\text{interval}_s}$$

Example: 100 devices, each with 48 interfaces, counters sampled at 30s:

$$\text{Messages/sec} = 100 \times \frac{48}{30} = 160 \text{ msg/s}$$

At ~500 bytes per Notification message, this is approximately 80 KB/s — manageable for a single collector.

---

## 7. gNMI Performance Characteristics

### Latency Profile

| Operation | Typical Latency | Bottleneck |
|:---|:---|:---|
| Capabilities | 5-50 ms | Server model enumeration |
| Get (single leaf) | 10-100 ms | Data retrieval from device |
| Get (large subtree) | 100 ms - 5s | Serialization + transport |
| Set (single update) | 50-500 ms | Commit to running config |
| Set (bulk replace) | 100 ms - 10s | Config compilation |
| Subscribe initial sync | 1-30s | Full state enumeration |
| Subscribe update | 1-50 ms | Event propagation |

### Comparison with SNMP Polling

| Metric | SNMP Polling | gNMI Subscribe |
|:---|:---|:---|
| Minimum interval | ~60s practical | ~1s (sample), instant (on-change) |
| CPU per poll | Walk entire MIB | Incremental updates only |
| Bandwidth | Full table every poll | Delta on change |
| Data freshness | poll_interval / 2 average | Near real-time |
| Encoding overhead | ASN.1/BER | Protobuf (2-10x smaller) |
| Connection state | Stateless (UDP) | Stateful (gRPC stream) |

---

## 8. Dial-In vs Dial-Out Telemetry

### Dial-In (Collector-Initiated)

The collector (gnmic, telegraf) initiates a gRPC connection to the device's gNMI server and issues Subscribe RPCs. This is the standard gNMI model.

**Advantages:**
- Collector controls subscription lifecycle
- Easy to add/remove subscriptions dynamically
- Standard gNMI API — portable across vendors

**Challenges:**
- Requires inbound connectivity to devices (firewall rules)
- Collector must know device addresses
- Connection failures require collector-side retry logic

### Dial-Out (Device-Initiated)

The device initiates a gRPC connection to a pre-configured collector address and pushes telemetry data. This is vendor-specific (Cisco MDT, Juniper JTI).

**Advantages:**
- No inbound connectivity required to devices
- Devices push through NAT/firewalls
- Subscription config lives on device (infrastructure-as-code friendly)

**Challenges:**
- Vendor-specific configuration syntax
- Less dynamic — subscription changes require device config change
- Non-standard — different encoding options per vendor

### Hybrid Architecture

Production deployments often combine both:
- **Dial-in** for interactive queries (Get/Set) and ad-hoc troubleshooting
- **Dial-out** for bulk telemetry collection across thousands of devices

---

## See Also

- netconf
- restconf
- yang-models
- pyats
- opentelemetry
- prometheus

## References

- gNMI specification: https://github.com/openconfig/reference/blob/master/rpc/gnmi/gnmi-specification.md
- gNOI repository: https://github.com/openconfig/gnoi
- gnmic documentation: https://gnmic.openconfig.net/
- gRPC documentation: https://grpc.io/docs/
- OpenConfig reference: https://github.com/openconfig/reference
- RFC 7950 — YANG 1.1: https://datatracker.ietf.org/doc/html/rfc7950
