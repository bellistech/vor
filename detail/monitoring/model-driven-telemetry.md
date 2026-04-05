# The Architecture of Model-Driven Telemetry — From Poll to Push

> *Model-driven telemetry replaces the SNMP poll cycle with device-initiated streaming, using YANG-modeled data paths, binary encodings, and gRPC transport. Its mathematical foundations are in sampling theory, queuing, and time-series compression.*

---

## 1. Push vs Pull Telemetry

### The Problem

Traditional network monitoring polls devices at fixed intervals. At scale, polling becomes a bottleneck — both for the monitoring system (which must query thousands of devices) and for the devices (which must respond to thousands of queries).

### Pull Model (SNMP)

```
Collector → GET request → Device → Response → Collector
           (every T seconds)
```

**Cost per cycle**:

$$C_{pull} = N \times M \times T_{request}$$

Where $N$ = devices, $M$ = metrics per device, $T_{request}$ = time per SNMP GET.

For 1,000 devices, 100 metrics each, 50ms per request:

$$C_{pull} = 1000 \times 100 \times 0.05 = 5000 \text{ seconds per poll cycle}$$

At a 30-second poll interval, the system cannot complete a cycle before the next one starts. This is the **polling wall**.

### Push Model (MDT)

```
Device → Stream data → Collector
        (at configured cadence)
```

**Cost per cycle**:

$$C_{push} = N \times T_{receive}$$

Where $T_{receive}$ is the time to process an incoming message (typically < 1ms with GPB encoding).

$$C_{push} = 1000 \times 0.001 = 1 \text{ second}$$

### Scaling Comparison

| Devices | Metrics/Device | SNMP (30s poll) | MDT (10s stream) |
|:---:|:---:|:---:|:---:|
| 100 | 50 | 250s (feasible) | 0.1s |
| 1,000 | 100 | 5,000s (broken) | 1s |
| 10,000 | 200 | 100,000s (impossible) | 10s |

### The Fundamental Advantage

SNMP scales as $O(N \times M)$ — every metric on every device requires a request-response.

MDT scales as $O(N)$ — each device streams all metrics in a single connection. The collector's work is receiving and parsing, not requesting.

---

## 2. Streaming Telemetry Architecture

### The Problem

A production telemetry system must handle hundreds of thousands of data points per second, store them efficiently, and make them queryable for dashboards and alerts.

### End-to-End Architecture

```
┌──────────────────┐
│  Network Devices │
│  (Publishers)    │
│                  │
│ ┌──────────────┐ │
│ │ Sensor Paths │ │    gRPC/TCP
│ │ Subscriptions│ │────────────────┐
│ │ Encoding     │ │                │
│ └──────────────┘ │                ▼
└──────────────────┘        ┌──────────────┐
                            │  Collector   │
                            │  (gnmic,     │
                            │   Telegraf,  │
                            │   Pipeline)  │
                            └──────┬───────┘
                                   │
                          ┌────────┼────────┐
                          ▼        ▼        ▼
                    ┌────────┐ ┌──────┐ ┌──────┐
                    │ TSDB   │ │Kafka │ │Alert │
                    │(Prom,  │ │      │ │Mgr   │
                    │InfluxDB│ │      │ │      │
                    └───┬────┘ └──┬───┘ └──────┘
                        │        │
                        ▼        ▼
                    ┌────────┐ ┌──────┐
                    │Grafana │ │Stream│
                    │        │ │Proc. │
                    └────────┘ └──────┘
```

### Data Flow Rates

For a network with $N$ devices, $S$ sensor paths per device, $F$ fields per path, at cadence $C$ seconds:

$$\text{Messages/sec} = \frac{N \times S}{C}$$

$$\text{Data points/sec} = \frac{N \times S \times F}{C}$$

### Worked Example

| Parameter | Value |
|:---|:---:|
| Devices ($N$) | 500 |
| Sensor paths ($S$) | 10 |
| Fields per path ($F$) | 20 |
| Cadence ($C$) | 10s |

$$\text{Messages/sec} = \frac{500 \times 10}{10} = 500$$

$$\text{Data points/sec} = \frac{500 \times 10 \times 20}{10} = 10{,}000$$

At GPB-KV encoding (~100 bytes/field):

$$\text{Bandwidth} = 10{,}000 \times 100 = 1{,}000{,}000 \text{ bytes/sec} \approx 8 \text{ Mbps}$$

This is trivial bandwidth. The bottleneck is TSDB write throughput, not network capacity.

---

## 3. GPB Encoding Efficiency

### The Problem

Telemetry data must be encoded for transport. The encoding choice directly impacts bandwidth consumption, CPU usage on both ends, and system scalability.

### Encoding Comparison

| Encoding | Size (relative) | Encode Speed | Decode Speed | Self-Describing |
|:---|:---:|:---:|:---:|:---:|
| JSON | 10x | Slow | Slow | Yes |
| GPB-KV | 3x | Fast | Fast | Yes |
| GPB (compact) | 1x | Fastest | Fastest | No |

### Why GPB Is Smaller

JSON encodes field names as strings in every message:

```json
{"in_octets": 1234567890, "out_octets": 987654321, "in_errors": 0}
```

Total: ~70 bytes for 3 fields.

GPB encodes field numbers (not names) as varints:

```
field 1 (varint): 08 d2 85 d8 cc 04  (6 bytes)
field 2 (varint): 10 b1 ea 8e d6 03  (6 bytes)
field 3 (varint): 18 00              (2 bytes)
```

Total: ~14 bytes for 3 fields. **5x compression** with zero information loss.

### Varint Encoding

GPB uses variable-length integer encoding where small numbers use fewer bytes:

| Value Range | Bytes Used |
|:---|:---:|
| 0 - 127 | 1 |
| 128 - 16,383 | 2 |
| 16,384 - 2,097,151 | 3 |
| 2,097,152 - 268,435,455 | 4 |

Interface counters (large values) use more bytes, but field tags (small numbers) are compact. The net result is significant space savings.

### Bandwidth Impact at Scale

For 10,000 data points/sec:

| Encoding | Bandwidth | Daily Storage |
|:---|:---:|:---:|
| JSON | ~8 Mbps | ~86 GB |
| GPB-KV | ~2.4 Mbps | ~26 GB |
| GPB (compact) | ~0.8 Mbps | ~8.6 GB |

Over 30 days, the difference between JSON and compact GPB is **2.3 TB vs 258 GB**.

---

## 4. Dial-In vs Dial-Out Trade-Offs

### The Problem

The telemetry subscription can be initiated by either the collector (dial-in) or the device (dial-out). Each model has architectural implications.

### Dial-In (Collector-Initiated)

```
Collector ──connect──> Device (gRPC server)
Collector ──subscribe──> Device
Device ──stream──> Collector
```

**Characteristics**:
- Collector manages subscription lifecycle
- Device must run gRPC server (port open)
- Collector must be able to reach device management plane
- Subscription state lives on collector

### Dial-Out (Device-Initiated)

```
Device ──connect──> Collector (gRPC server)
Device ──stream──> Collector
```

**Characteristics**:
- Subscription configured on device (persistent)
- Collector is passive receiver
- Works through NAT (device initiates connection)
- Subscription state lives on device

### Comparison Matrix

| Factor | Dial-In | Dial-Out |
|:---|:---|:---|
| Connection direction | Collector → Device | Device → Collector |
| Firewall friendliness | Requires inbound to device | Outbound from device (easier) |
| NAT traversal | Difficult | Natural |
| Subscription management | Centralized (collector) | Distributed (each device) |
| Dynamic subscriptions | Easy (change at collector) | Hard (change on each device) |
| Device CPU | gRPC server overhead | gRPC client (lighter) |
| Scalability | Collector manages N connections | Collector accepts N connections |
| Troubleshooting | Easy (query device on demand) | Config review on device |
| gNMI support | Native | Not standard (vendor extensions) |

### Production Recommendation

**Use dial-out** for:
- Steady-state production monitoring
- Large-scale deployments (1000+ devices)
- Environments with NAT or strict firewalls
- Stable, well-defined telemetry requirements

**Use dial-in** for:
- Dynamic/ad-hoc monitoring
- Troubleshooting sessions
- Environments where gNMI is the standard
- When subscription changes are frequent

### Hybrid Architecture

Many production deployments use both:

```
Dial-out: Production monitoring (always-on, stable subscriptions)
Dial-in: Troubleshooting (on-demand, temporary subscriptions)
```

---

## 5. Telemetry Pipeline Design

### The Problem

Raw telemetry data from devices must be transformed, enriched, and routed to appropriate backends. A well-designed pipeline handles this at scale.

### Pipeline Stages

```
Receive → Decode → Transform → Enrich → Route → Store
```

| Stage | Function | Example |
|:---|:---|:---|
| Receive | Accept gRPC/TCP streams | Telegraf input plugin |
| Decode | Parse GPB/JSON payload | Proto deserialization |
| Transform | Normalize field names, units | bits → Mbps |
| Enrich | Add metadata (site, role) | Lookup from NetBox |
| Route | Fan-out to multiple backends | Prometheus + Kafka |
| Store | Write to TSDB | InfluxDB write API |

### Collector Comparison

| Collector | Protocol Support | Output Plugins | Scalability | Complexity |
|:---|:---|:---:|:---:|:---:|
| Telegraf | gNMI, MDT (gRPC/TCP) | 40+ | Single-node | Low |
| gnmic | gNMI, gRPC | Prometheus, Kafka, NATS, file | Clustered | Medium |
| Pipeline (Cisco) | MDT (gRPC/TCP/UDP) | Kafka, InfluxDB, dump | Single-node | Low |
| OpenTelemetry | OTLP, gRPC | Many | Distributed | High |

### gnmic Clustering

gnmic supports clustering for high availability:

```
                    ┌──────────┐
                    │ Consul / │
                    │ NATS     │
                    └────┬─────┘
                ┌────────┼────────┐
                ▼        ▼        ▼
          ┌──────────┐ ┌──────────┐ ┌──────────┐
          │ gnmic-1  │ │ gnmic-2  │ │ gnmic-3  │
          │(targets  │ │(targets  │ │(targets  │
          │ 1-100)   │ │ 101-200) │ │ 201-300) │
          └──────────┘ └──────────┘ └──────────┘
```

Targets are distributed across cluster members using a locker (Consul or NATS). If a member fails, its targets are redistributed.

---

## 6. YANG Path Resolution

### The Problem

Sensor paths reference YANG model nodes. Understanding YANG tree structure is essential for selecting the right paths and interpreting the data.

### YANG Tree Structure

```
module: openconfig-interfaces
  +--rw interfaces
     +--rw interface* [name]
        +--rw name          -> ../config/name
        +--rw config
        |  +--rw name          string
        |  +--rw type          identityref
        |  +--rw mtu           uint16
        |  +--rw enabled       boolean
        |  +--rw description   string
        +--ro state
        |  +--ro name          string
        |  +--ro type          identityref
        |  +--ro oper-status   enumeration
        |  +--ro counters
        |     +--ro in-octets       counter64
        |     +--ro out-octets      counter64
        |     +--ro in-errors       counter64
        |     +--ro out-errors      counter64
        +--rw subinterfaces
```

### Path Syntax

gNMI paths use `/` separators with `[key=value]` for list entries:

| Path | What It Returns |
|:---|:---|
| `/interfaces` | All interfaces (entire tree) |
| `/interfaces/interface` | All interface entries |
| `/interfaces/interface[name=Ethernet1]` | Single interface |
| `/interfaces/interface/state/counters` | Counters for all interfaces |
| `/interfaces/interface[name=Ethernet1]/state/counters/in-octets` | Single counter |

### OpenConfig vs Native Models

| Aspect | OpenConfig | Native (Vendor) |
|:---|:---|:---|
| Scope | Cross-vendor subset | Full feature coverage |
| Path format | `/openconfig-*:` | `/Cisco-IOS-XR-*:` |
| Stability | Versioned, backward-compatible | May change between releases |
| Coverage | 60-80% of common features | 100% of platform features |
| Adoption | Arista, Cisco, Juniper, Nokia | Vendor-specific |

### Path Discovery

To find available paths on a device:

```
gnmic capabilities → supported models → YANG tree → sensor paths
```

$$\text{Available paths} = \bigcup_{m \in \text{models}} \text{tree}(m)$$

---

## 7. TSDB Selection for Telemetry

### The Problem

Telemetry generates time-series data at high velocity. The TSDB must handle high write throughput, efficient compression, and fast queries for dashboards.

### TSDB Comparison for Network Telemetry

| TSDB | Write Speed | Compression | Query Language | Retention | HA |
|:---|:---:|:---:|:---|:---:|:---:|
| Prometheus | 1M samples/s | Good | PromQL | Days-weeks | Federation/Thanos |
| InfluxDB | 500K points/s | Good | InfluxQL/Flux | Configurable | Enterprise |
| VictoriaMetrics | 2M+ samples/s | Excellent | MetricsQL | Long-term | Cluster |
| TimescaleDB | 500K+ rows/s | Good (pg) | SQL | Unlimited | pg HA |
| Mimir | 1M+ samples/s | Good | PromQL | Long-term | Native |

### Storage Estimation

$$\text{Storage/day} = \frac{\text{data points/sec} \times 86{,}400 \times \text{bytes/point}}{\text{compression ratio}}$$

For 10,000 data points/sec:

| TSDB | Bytes/point (compressed) | Storage/day | Storage/year |
|:---|:---:|:---:|:---:|
| Prometheus | 1.3 bytes | 1.1 GB | 401 GB |
| VictoriaMetrics | 0.7 bytes | 0.6 GB | 219 GB |
| InfluxDB | 2.0 bytes | 1.7 GB | 620 GB |

### Cardinality Considerations

$$\text{Cardinality} = |\text{devices}| \times |\text{interfaces}| \times |\text{metrics}|$$

For 500 devices, 48 interfaces each, 20 metrics per interface:

$$\text{Cardinality} = 500 \times 48 \times 20 = 480{,}000 \text{ unique series}$$

Prometheus handles up to ~10M active series. VictoriaMetrics handles 100M+. High cardinality is the most common scaling bottleneck.

---

## 8. Telemetry-Driven Automation

### The Problem

Telemetry data can drive automated responses to network events, closing the loop between monitoring and remediation.

### Closed-Loop Architecture

```
┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐
│ Telemetry│───>│ Analytics│───>│ Decision │───>│ Action   │
│ Stream   │    │ Engine   │    │ Engine   │    │ Engine   │
└──────────┘    └──────────┘    └──────────┘    └──────────┘
     ▲                                               │
     │                                               │
     └───────────────────────────────────────────────┘
                    Feedback loop
```

### Event-Driven Automation Examples

| Telemetry Event | Detection | Automated Response |
|:---|:---|:---|
| Interface down | On-change subscription | Reroute traffic, open ticket |
| BGP neighbor down | State change to Idle | Verify config, attempt reset |
| CPU > 90% | Threshold breach | Shed non-critical traffic |
| Link utilization > 80% | Cadence-based trend | Adjust ECMP weights |
| CRC errors increasing | Rate-of-change alert | Disable interface, alert |

### Automation Safety

Closed-loop automation requires guardrails:

1. **Rate limiting**: Maximum N automated changes per hour
2. **Scope limiting**: Automated actions affect at most one device at a time
3. **Human escalation**: If automated fix fails, alert human
4. **Audit trail**: Every automated action logged with telemetry trigger
5. **Kill switch**: Global disable for all automated responses

### Reaction Time Comparison

| Approach | Detection | Response | Total |
|:---|:---:|:---:|:---:|
| Manual (SNMP + human) | 30-300s | 5-60 min | 5-65 min |
| Alert-driven (SNMP + script) | 30-60s | 1-5 min | 1.5-6 min |
| Telemetry-driven (MDT + automation) | 1-10s | 5-30s | 6-40s |

$$\text{Speedup} = \frac{T_{manual}}{T_{telemetry}} = \frac{300\text{s} + 300\text{s}}{10\text{s} + 30\text{s}} = 15\times$$

---

## See Also

- SNMP
- gNMI/gNOI
- Prometheus
- Grafana
- NetFlow/IPFIX

## References

- OpenConfig: https://www.openconfig.net/
- gnmic: https://gnmic.openconfig.net/
- RFC 8040 (RESTCONF): https://datatracker.ietf.org/doc/html/rfc8040
- RFC 7950 (YANG 1.1): https://datatracker.ietf.org/doc/html/rfc7950
- Cisco MDT Config Guide: https://www.cisco.com/c/en/us/td/docs/iosxr/ncs5500/telemetry/
- "Network Programmability with YANG" — Benoît Claise et al.
