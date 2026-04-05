# NetFlow and IPFIX — Flow-Based Traffic Measurement Architecture

> *NetFlow and IPFIX implement flow-based traffic measurement by aggregating packets into unidirectional flow records keyed on shared attributes. The architecture spans metering processes on network devices, export protocols over UDP/TCP/SCTP, and collection/analysis systems. Understanding flow semantics, template mechanics, sampling theory, and collector design is essential for network visibility at scale.*

---

## 1. Flow Definition and the Metering Process

### What Constitutes a Flow

A network flow is a set of IP packets passing an observation point during a certain time interval, where all packets share a common set of properties called the **flow key**. The metering process on a router or switch inspects each packet, computes the flow key, and either creates a new cache entry or updates an existing one.

The classic NetFlow v5 flow key is a fixed 7-tuple:

| Field | Description | Bytes |
|:---|:---|:---:|
| Source IP Address | Layer 3 source | 4 |
| Destination IP Address | Layer 3 destination | 4 |
| Source Port | Layer 4 source (TCP/UDP) | 2 |
| Destination Port | Layer 4 destination (TCP/UDP) | 2 |
| IP Protocol | TCP=6, UDP=17, ICMP=1, etc. | 1 |
| Type of Service (ToS) | DSCP + ECN bits | 1 |
| Input Interface | SNMP ifIndex of ingress | 2 |

With template-based formats (NetFlow v9 and IPFIX), the flow key is user-configurable. Any combination of supported Information Elements can serve as match fields, enabling application-aware, VLAN-aware, MPLS-aware, or IPv6-native flow tracking.

### Flow Cache Mechanics

The flow cache is a hash table maintained in the forwarding plane (or in software on low-end platforms). Each entry stores:

- **Flow key fields** — for packet classification
- **Counters** — byte count, packet count
- **Timestamps** — first packet time, last packet time
- **Derived fields** — TCP flags (cumulative OR), next-hop, output interface, AS numbers

$$\text{Cache Memory} = \text{Max Entries} \times \text{Entry Size (bytes)}$$

Typical entry sizes:

| Platform | Entry Size | Default Max Entries | Cache Memory |
|:---|:---:|:---:|:---:|
| IOS (software) | 64 bytes | 4,096 | 256 KiB |
| IOS-XE | 64 bytes | 65,536 | 4 MiB |
| NX-OS | 80 bytes | 131,072 | 10 MiB |
| ASR 9000 (hardware) | 48 bytes | 2,000,000 | 96 MiB |

### Flow Expiration and Export Triggers

Flows are exported from the cache under these conditions:

| Trigger | Default Value | Configurable | Purpose |
|:---|:---:|:---:|:---|
| Active timeout | 1800s (30 min) | Yes | Long-lived flows generate periodic exports |
| Inactive timeout | 15s | Yes | Idle flows are cleaned from cache |
| TCP FIN/RST | Immediate | No | Connection termination detected |
| Cache full | Oldest entry | No | Eviction under memory pressure |
| Forced flush | Manual | N/A | Operator-triggered cache clear |

The active timeout is critical for long-lived flows like SSH sessions or bulk transfers. Without it, a persistent flow would consume a cache entry indefinitely and never be reported to the collector.

$$\text{Exports per flow} = \left\lceil \frac{\text{Flow Duration}}{\text{Active Timeout}} \right\rceil$$

A 2-hour file transfer with a 60-second active timeout generates 120 export records for a single logical flow. The collector must stitch these into a single conversation.

---

## 2. Protocol Evolution: v5, v9, IPFIX

### NetFlow v5 — The Fixed-Format Origin

Cisco introduced NetFlow v5 in the mid-1990s as the first widely deployed flow export protocol. Its fixed 48-byte record format is simple to parse but inflexible:

**Limitations of v5:**
- No IPv6 support (only 4-byte address fields)
- No MPLS, VLAN, or application-layer fields
- Fixed 48-byte record — cannot add or remove fields
- Maximum 30 records per export packet
- UDP-only transport (no delivery guarantees)
- No template mechanism — parser must know the format a priori

Despite these limitations, v5 remains widely deployed due to its simplicity and universal collector support. It is sufficient for basic traffic accounting on IPv4-only networks.

### NetFlow v9 — Template-Based Flexibility

NetFlow v9 (RFC 3954) introduced template-based export, decoupling the record format from the protocol specification:

**Architecture of v9:**

1. **Template FlowSet** — Sent periodically, defines the field layout for subsequent Data FlowSets. Each template has a unique Template ID (256-65535) scoped to the exporter.

2. **Data FlowSet** — Contains flow records formatted according to a previously advertised template. The Set ID references the Template ID.

3. **Options Template FlowSet** — Exports metadata about the metering process itself: interface names, sampling parameters, exporter capabilities.

Template refresh is controlled by two parameters:
- **Packet-based:** Resend template every N export packets (e.g., every 20 packets)
- **Time-based:** Resend template every N minutes (e.g., every 30 minutes)

If a collector receives a Data FlowSet referencing an unknown Template ID, it must buffer the data until the template arrives. This creates a bootstrapping problem at collector startup.

### IPFIX — The IETF Standard (RFC 7011-7015)

IPFIX (IP Flow Information Export) was standardized by the IETF in 2013 based on NetFlow v9. Key improvements:

| Feature | NetFlow v9 | IPFIX |
|:---|:---|:---|
| Transport | UDP only | UDP, TCP, SCTP |
| Version field | 9 | 10 |
| Variable-length fields | No | Yes (e.g., HTTP URL, DNS name) |
| Enterprise IEs | Informal | Standardized (PEN + IE ID) |
| Template withdrawal | No | Yes (explicit removal) |
| Structured data | No | basicList, subTemplateList, subTemplateMultiList |
| Sequence numbering | Per-template | Per observation domain |
| Standard | Cisco proprietary | IETF RFC 7011 |
| Default port | 9996 (convention) | 4739 (IANA assigned) |

**IPFIX Information Element (IE) structure:**

Each IE is identified by a 16-bit Information Element ID and optionally a 32-bit Private Enterprise Number (PEN) for vendor-specific extensions:

$$\text{IE Identifier} = \begin{cases} \text{IE ID (0-32767)} & \text{IANA-registered} \\ \text{PEN (32 bits) + IE ID (15 bits)} & \text{Enterprise-specific} \end{cases}$$

The IANA registry contains 500+ standardized IEs covering Layer 2 through Layer 7, including:
- Network layer: IP addresses, protocol, DSCP, TTL, fragmentation
- Transport layer: ports, TCP flags, TCP window size
- Application layer: HTTP URL, DNS query name (via variable-length)
- Infrastructure: MPLS labels, VLAN IDs, BGP AS path, VRF name

---

## 3. Sampling Theory and Accuracy

### Why Sample

At high packet rates, maintaining per-packet flow state becomes prohibitively expensive:

$$\text{Packets per second (10 GbE)} = \frac{10 \times 10^9}{(64 + 20) \times 8} \approx 14.88 \text{ Mpps (worst case)}$$

Processing 14.88 million packets per second for flow classification requires dedicated hardware. Sampling reduces the processing load by examining only a fraction of packets.

### Sampling Methods

**Deterministic (systematic) sampling:** Every Nth packet is sampled. Simple to implement but susceptible to periodic traffic patterns that align with the sampling interval.

$$P(\text{packet sampled}) = \frac{1}{N}$$

**Random (probabilistic) sampling:** Each packet is independently sampled with probability 1/N. Avoids the periodicity problem but requires a random number generator in the forwarding path.

**Hash-based sampling:** A hash of selected packet fields determines whether to sample. Ensures that all packets of the same flow are either all sampled or all skipped. Useful for maintaining flow coherence but biases toward certain flow keys.

### Statistical Accuracy

For a flow with true packet count $P$ and sampling rate $1/N$, the expected number of sampled packets is:

$$E[\hat{P}] = \frac{P}{N}$$

The estimator for the true count is $\hat{P}_{est} = \hat{P} \times N$, which is unbiased. The variance is:

$$\text{Var}(\hat{P}_{est}) = P \times N \times (N - 1)$$

The coefficient of variation (relative error):

$$CV = \frac{\sqrt{\text{Var}(\hat{P}_{est})}}{E[\hat{P}_{est}]} = \sqrt{\frac{N - 1}{P}} \approx \sqrt{\frac{N}{P}}$$

This means accuracy depends on the ratio of true flow size to sampling rate:

| True Packets (P) | Sampling Rate (1/N) | Sampled Packets | Relative Error (CV) |
|:---:|:---:|:---:|:---:|
| 100 | 1:100 | 1 | 100% |
| 1,000 | 1:100 | 10 | 31.6% |
| 10,000 | 1:100 | 100 | 10% |
| 100,000 | 1:100 | 1,000 | 3.2% |
| 1,000,000 | 1:100 | 10,000 | 1% |
| 100 | 1:1000 | 0.1 | ~316% |
| 10,000 | 1:1000 | 10 | 31.6% |

Key insight: **sampled NetFlow is accurate for elephant flows but unreliable for mice flows.** A flow of 50 packets sampled at 1:100 has only a 39.5% chance of being observed at all:

$$P(\text{flow observed}) = 1 - \left(1 - \frac{1}{N}\right)^P = 1 - \left(\frac{99}{100}\right)^{50} = 0.395$$

### Sampling Rate Selection Guidelines

| Link Speed | Recommended Rate | Use Case |
|:---|:---:|:---|
| < 1 Gbps | 1:1 (unsampled) | Full accuracy, forensics, billing |
| 1-10 Gbps | 1:100 to 1:500 | Capacity planning, top talkers |
| 10-40 Gbps | 1:1000 | DDoS detection, trending |
| 40-100 Gbps | 1:2000 to 1:4000 | Coarse visibility, anomaly detection |
| 100+ Gbps | 1:8000 to 1:10000 | Only elephant flow detection |

---

## 4. sFlow — The Counter-Based Alternative

### Architecture Differences

sFlow (RFC 3176) takes a fundamentally different approach from NetFlow/IPFIX:

| Aspect | NetFlow/IPFIX | sFlow |
|:---|:---|:---|
| Sampling unit | Flow (aggregated) | Packet (raw header) |
| State on device | Stateful (flow cache) | Stateless (no cache) |
| Data exported | Aggregated counters per flow | Raw packet headers + interface counters |
| Counter polling | Not built-in | Built-in (every N seconds) |
| CPU impact | Moderate (cache maintenance) | Low (no aggregation) |
| Memory impact | Cache-dependent | Minimal |
| Accuracy model | Deterministic for tracked flows | Statistical for all traffic |
| Collector role | Receives pre-aggregated records | Must aggregate flows from samples |

### sFlow Sample Types

**Flow samples:** Contains the first 128 bytes (configurable) of a sampled packet header, plus metadata about the sampling context (input/output interface, sampling rate, sample pool).

**Counter samples:** Periodic snapshots of interface counters (ifInOctets, ifOutOctets, ifInErrors, etc.) sent at a configurable polling interval (typically 20-30 seconds).

The collector reconstructs traffic patterns by:
1. Parsing packet headers from flow samples to classify traffic
2. Scaling counts by the sampling rate for volume estimation
3. Correlating with counter samples for interface utilization

### When to Choose sFlow over NetFlow

- **High-port-density switches:** sFlow's stateless design scales better on switches with 48-96 ports
- **Multi-vendor environments:** sFlow is an open standard with broad switch vendor support
- **Real-time visibility:** No flow cache delays — samples arrive immediately
- **Resource-constrained devices:** No CPU or memory overhead for flow cache

### When to Choose NetFlow/IPFIX over sFlow

- **Billing and accounting:** Pre-aggregated flow records are more precise for per-customer metering
- **Forensic analysis:** Complete flow records with start/end times enable session reconstruction
- **Compliance auditing:** Auditors prefer deterministic per-flow records over statistical samples
- **Application visibility:** IPFIX variable-length fields capture Layer 7 data (URLs, DNS names)

---

## 5. Collector Architecture and Scalability

### Collector Pipeline

A production flow collection system consists of multiple stages:

```
[Exporters] → [Receivers] → [Decoders] → [Enrichment] → [Storage] → [Analysis/UI]
```

**Receivers:** Accept UDP/TCP/SCTP connections from exporters. Must handle out-of-order delivery (UDP) and template bootstrapping. Multiple receivers behind a load balancer for high availability.

**Decoders:** Parse raw export packets using template definitions. Must cache templates per exporter (identified by source IP + observation domain ID). Template state is critical — loss of templates means inability to decode data.

**Enrichment:** Augment flow records with external data:
- DNS reverse lookup for IP addresses
- GeoIP mapping for geographic attribution
- BGP AS path lookup for transit analysis
- Application classification via port/protocol mapping or DPI results
- SNMP interface name resolution (ifIndex to ifName)

**Storage:** Time-series optimized storage. Options include:
- Flat files with time-based rotation (nfcapd model)
- Elasticsearch/OpenSearch with time-based indices
- ClickHouse or TimescaleDB for high-throughput analytical queries
- Apache Kafka for streaming pipeline with multiple consumers

### Collector Sizing

$$\text{Storage per day} = \text{Flows/sec} \times 86400 \times \text{Bytes per record}$$

With typical compression (gzip or LZ4), stored record sizes are 50-100 bytes per flow:

| Flows/sec | Records/day | Storage/day (compressed) | Storage/30 days |
|:---:|:---:|:---:|:---:|
| 100 | 8.64M | 430 MB - 860 MB | 13-26 GB |
| 1,000 | 86.4M | 4.3-8.6 GB | 130-260 GB |
| 10,000 | 864M | 43-86 GB | 1.3-2.6 TB |
| 100,000 | 8.64B | 430-860 GB | 13-26 TB |

Collector CPU and memory scale with flows/sec, enrichment complexity, and query load:

| Flows/sec | CPU Cores | RAM | Storage Type |
|:---:|:---:|:---:|:---|
| < 1,000 | 2 | 4 GB | HDD |
| 1,000-10,000 | 4-8 | 16-32 GB | SSD |
| 10,000-100,000 | 8-16 | 32-64 GB | SSD RAID / NVMe |
| > 100,000 | 16+ (distributed) | 64+ GB | Distributed storage |

### High Availability Patterns

**Active-active collectors:** Multiple collectors receive the same export stream (exporter sends to multiple destinations). Each collector stores independently. Query layer merges results. Risk: duplicate flow records require deduplication logic.

**Active-passive with flow replication:** Primary collector receives exports. A flow replicator (e.g., samplicator) mirrors UDP packets to a standby collector. Failover is manual or scripted.

**Kafka-based pipeline:** Exporters send to receivers that publish to Kafka topics. Multiple consumer groups process flows independently for storage, alerting, and real-time dashboards. Kafka handles buffering and replay.

---

## 6. Collector Software Comparison

### Open Source Collectors

| Collector | Input Formats | Storage | Query Interface | Strengths |
|:---|:---|:---|:---|:---|
| nfdump/nfcapd | v5, v9, IPFIX, sFlow | Flat files | CLI (nfdump) | Fast CLI queries, low resource use |
| ntopng + nProbe | v5, v9, IPFIX, sFlow | Redis + time series | Web UI | Real-time dashboards, DPI |
| pmacct | v5, v9, IPFIX, sFlow | SQL, Kafka, files | SQL queries | Flexible output, BGP correlation |
| GoFlow2 | v5, v9, IPFIX, sFlow | Kafka, files | Downstream consumers | High throughput, Go-based |
| Akvorado | IPFIX, sFlow | ClickHouse | Web UI + API | Modern, ClickHouse-native |

### Commercial Collectors

| Collector | Differentiator |
|:---|:---|
| SolarWinds NTA | Tight integration with Orion NPM, SNMP correlation |
| PRTG | NetFlow/sFlow sensor per interface, easy setup |
| ManageEngine NetFlow Analyzer | Affordable, NBAR application mapping |
| Kentik | SaaS-based, BGP-aware, DDoS detection, API-first |
| Arbor / NETSCOUT | Carrier-grade DDoS detection, peering analytics |
| Plixer Scrutinizer | Compliance reporting, forensic analysis |

---

## 7. Analysis Methodologies

### Top-N Analysis

The most fundamental analysis: rank flows, hosts, ports, protocols, or AS numbers by volume (bytes, packets, or flow count).

$$\text{Top-N by bytes} = \text{sort}(\text{aggregate}(\text{flows}, \text{key}), \text{bytes}, \text{desc})[:N]$$

Top-N answers questions like:
- Who are the top bandwidth consumers? (Top source IPs by bytes)
- What services are most used? (Top destination ports by flows)
- Which external networks send us the most traffic? (Top source ASNs by bytes)
- Which internal servers handle the most connections? (Top destination IPs by flows)

### Traffic Matrix Construction

A traffic matrix maps source-destination pairs to traffic volumes, essential for capacity planning and traffic engineering:

$$T_{ij} = \sum_{\text{flows}} \text{bytes}(f) \quad \text{where } \text{src}(f) \in i, \text{dst}(f) \in j$$

Dimensions can be:
- **Router-to-router:** Traffic between PoPs or data centers
- **Subnet-to-subnet:** Inter-department or inter-tenant traffic
- **AS-to-AS:** Peering and transit traffic for ISPs

### Anomaly Detection

Flow data enables several anomaly detection techniques:

**Volume-based:** Alert when traffic to/from a host or subnet exceeds a baseline threshold (e.g., 3 standard deviations above the hourly mean).

**Flow-count-based:** A sudden spike in flow count to a single destination (with low packets-per-flow) indicates a SYN flood or port scan.

**Protocol-ratio anomalies:** A shift in the TCP/UDP/ICMP ratio compared to historical norms may indicate scanning or tunneling activity.

**Behavioral analysis:** New external destinations, unusual port usage, off-hours traffic patterns — compared against per-host behavioral profiles built from historical flow data.

### 95th Percentile Billing

ISPs and transit providers commonly use 95th percentile billing:

1. Collect traffic volume samples every 5 minutes (288 samples/day)
2. At the end of the billing period, sort all samples ascending
3. Discard the top 5% of samples
4. The next highest value is the 95th percentile — the billed rate

$$\text{95th percentile index} = \lceil 0.95 \times N \rceil$$

For a 30-day month: $N = 30 \times 288 = 8640$ samples. The 95th percentile is sample number $\lceil 0.95 \times 8640 \rceil = 8208$.

This model tolerates occasional traffic spikes (bursting) while billing for sustained usage.

---

## 8. Flexible NetFlow and Advanced Configuration

### Flow Record Design Principles

When designing custom flow records (Flexible NetFlow), consider:

**Flow key granularity vs. cache pressure:** More match fields create more unique flows, increasing cache utilization and export volume.

$$\text{Unique flows} \propto \prod_{i=1}^{n} |K_i|$$

where $|K_i|$ is the cardinality of the $i$-th key field. Adding VLAN ID (4094 values) to a 5-tuple key can multiply the number of unique flows significantly.

**Collect fields vs. export bandwidth:** Each non-key collect field adds bytes to every exported record. For a monitor exporting 10,000 flows/sec:

$$\text{Export bandwidth} = \text{Flows/sec} \times \text{Record size (bytes)} \times 8 \text{ bits}$$

| Record Size | Flows/sec | Export Bandwidth |
|:---:|:---:|:---:|
| 48 bytes (v5) | 10,000 | 3.84 Mbps |
| 80 bytes (custom) | 10,000 | 6.4 Mbps |
| 120 bytes (rich) | 10,000 | 9.6 Mbps |
| 200 bytes (app-aware) | 10,000 | 16 Mbps |

### Application Visibility with NBAR

Cisco NBAR (Network-Based Application Recognition) integrates with Flexible NetFlow to add Layer 7 application identification as a flow key:

```
flow record APP-AWARE
  match application name                      # NBAR application (e.g., youtube, webex)
  match ipv4 source address
  match ipv4 destination address
  collect counter bytes long
  collect counter packets long
```

This enables per-application traffic accounting without requiring the collector to perform DPI.

### Multi-Monitor Configurations

A single interface can have multiple flow monitors attached simultaneously, each with different records and exporters:

- **Security monitor:** Keys on src/dst IP, ports, TCP flags. Exports to SIEM.
- **Performance monitor:** Keys on DSCP, application. Exports to capacity planning system.
- **Billing monitor:** Keys on src subnet, interface. Exports to billing platform.

Each monitor maintains its own flow cache, so memory consumption is additive.

---

## 9. Deployment Best Practices

### Exporter Configuration Checklist

1. **Source interface:** Always use a loopback address. Physical interface IPs change during failover; the collector uses source IP to identify the exporter.

2. **Active timeout:** Reduce from the 30-minute default to 60 seconds for near-real-time visibility. Lower values increase export volume but improve time resolution.

3. **Template refresh:** For v9/IPFIX over UDP, set template timeout to 60 seconds and packet-based refresh to every 20 packets. Collectors need templates to decode data — frequent refreshes minimize data loss during collector restart.

4. **Sampling:** Enable on links above 1 Gbps. Start with 1:1000 and decrease the rate (increase accuracy) only if the platform can handle the load.

5. **Transport:** Use IPFIX over TCP or SCTP when reliable delivery matters (billing, compliance). UDP is acceptable for best-effort monitoring when occasional record loss is tolerable.

6. **VRF separation:** Export flow data through a management VRF to prevent flow export traffic from competing with production data on congested links.

### Collector Deployment Checklist

1. **Time synchronization:** All exporters and collectors must use NTP. Flow timestamps are derived from the exporter's system clock — clock drift causes incorrect time correlation.

2. **Template caching:** Persist template state to disk so that collector restarts do not lose the ability to decode in-flight data.

3. **Retention policy:** Define retention windows based on use case — 7 days for operational monitoring, 30 days for capacity planning, 90+ days for compliance and forensics.

4. **Alerting integration:** Connect the collector to alerting systems (PagerDuty, Prometheus Alertmanager) for volumetric and behavioral anomalies.

5. **Backup:** Flow data is a forensic asset. Include collector storage in backup schedules or replicate to a secondary site.

### Common Pitfalls

- **Missing templates at collector startup:** The collector does not receive templates until the next refresh interval. Shorten template timeout on exporters.
- **Asymmetric routing:** Flows captured on ingress only show one direction. Enable both ingress and egress monitoring or deploy exporters at every routing hop.
- **NAT traversal:** Flows before and after NAT have different source/destination IPs. Correlate using timestamps and the NAT translation table.
- **Sampled flow stitching:** When stitching sampled flows across active timeout boundaries, the collector may over-count if it does not properly match flow keys across exports.
- **SNMP ifIndex instability:** Some platforms reassign ifIndex values after reboot. Use ifName-based resolution and re-map after each device restart.

---

## 10. Security and Privacy Considerations

### Flow Data as Metadata

Flow records are metadata — they reveal who communicated with whom, when, for how long, and how much data was exchanged. This metadata can be as sensitive as packet content for privacy purposes:

- Communication patterns between internal hosts can reveal organizational structure
- External destination IPs can reveal business relationships, research interests, or employee behavior
- Flow volumes and timing can identify data exfiltration even without content inspection

### Data Protection Requirements

| Regulation | Requirement for Flow Data |
|:---|:---|
| GDPR (EU) | IP addresses are personal data. Flow data must have legal basis, retention limits, and access controls. |
| HIPAA (US) | Flows involving healthcare systems may be PHI metadata. Encrypt in transit and at rest. |
| PCI-DSS | Flows involving cardholder data environments require 90-day retention and access logging. |
| SOC 2 | Flow collection systems must be in scope for security monitoring and access control audits. |

### Securing the Flow Pipeline

- **Encrypt export traffic:** IPFIX over TLS (TCP) or DTLS (UDP) prevents interception of flow metadata on the management plane.
- **Access control:** Restrict collector access to authorized security and network operations personnel.
- **Anonymization:** For shared datasets or research, anonymize IP addresses using CryptoPAn or prefix-preserving anonymization.
- **Retention limits:** Delete flow data after the defined retention window. Indefinite retention increases breach exposure.

---

## Prerequisites

- IP networking fundamentals (addressing, subnetting, routing)
- TCP/UDP transport layer concepts (ports, flags, connection lifecycle)
- Router/switch CLI basics (IOS, IOS-XE, NX-OS)
- SNMP concepts (ifIndex, interface counters, MIB structure)
- Basic statistics (mean, variance, percentiles, sampling theory)

## Complexity

- **Beginner:** NetFlow v5 configuration, basic nfdump queries, understanding flow keys
- **Intermediate:** Flexible NetFlow record design, IPFIX template mechanics, collector deployment, sFlow comparison
- **Advanced:** Sampling accuracy analysis, traffic matrix construction, high-throughput collector architecture, anomaly detection algorithms, 95th percentile billing

## References

- [RFC 7011 — IPFIX Protocol Specification for the Exchange of Flow Information](https://www.rfc-editor.org/rfc/rfc7011)
- [RFC 7012 — Information Model for IP Flow Information Export (IPFIX)](https://www.rfc-editor.org/rfc/rfc7012)
- [RFC 7013 — Guidelines for Authors and Reviewers of IPFIX Information Elements](https://www.rfc-editor.org/rfc/rfc7013)
- [RFC 7014 — Flow Selection Techniques](https://www.rfc-editor.org/rfc/rfc7014)
- [RFC 7015 — IPFIX File Format](https://www.rfc-editor.org/rfc/rfc7015)
- [RFC 3954 — Cisco Systems NetFlow Services Export Version 9](https://www.rfc-editor.org/rfc/rfc3954)
- [RFC 3176 — InMon Corporation's sFlow: A Method for Monitoring Traffic in Switched and Routed Networks](https://www.rfc-editor.org/rfc/rfc3176)
- [RFC 5101 — IPFIX Protocol Specification (original, obsoleted by 7011)](https://www.rfc-editor.org/rfc/rfc5101)
- [RFC 5102 — Information Model for IPFIX (original, obsoleted by 7012)](https://www.rfc-editor.org/rfc/rfc5102)
- [IANA IPFIX Information Elements Registry](https://www.iana.org/assignments/ipfix/ipfix.xhtml)
- [IANA IPFIX Structured Data Types Registry](https://www.iana.org/assignments/ipfix/ipfix.xhtml#ipfix-structured-data-types-semantics)
- [Cisco Flexible NetFlow Configuration Guide](https://www.cisco.com/c/en/us/td/docs/ios-xml/ios/fnetflow/configuration/xe-17/fnf-xe-17-book.html)
- [Cisco NX-OS NetFlow Configuration Guide](https://www.cisco.com/c/en/us/td/docs/switches/datacenter/nexus9000/sw/93x/netflow/cisco-nexus-9000-nx-os-netflow-configuration-guide-93x.html)
