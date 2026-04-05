# SPAN and ERSPAN — Traffic Mirroring Architecture

> *Traffic mirroring is the art of creating perfect copies without disturbing the original. From local ASIC-level port mirroring to GRE-encapsulated remote capture across IP networks, the engineering involves packet replication pipelines, header insertion state machines, and the fundamental tension between visibility and performance. Understanding the forwarding plane mechanics, encapsulation formats, and scaling limits separates effective monitoring from unreliable guesswork.*

---

## 1. The Fundamental Problem — Passive Visibility

### Why Mirroring Exists

Network monitoring and security tools (IDS, IPS, forensic recorders, DLP, APM) need to see traffic without sitting inline. Inline deployment creates single points of failure and adds latency. Passive monitoring requires a copy of the traffic delivered to a dedicated analysis port.

There are two approaches to passive visibility:

**Network TAPs (Test Access Points):** Physical devices that sit in the cable path and optically or electrically split the signal. They provide guaranteed, lossless copies but require physical installation and cannot be reconfigured remotely.

**SPAN (Switched Port Analyzer):** A software/ASIC feature built into switches that copies traffic from source ports or VLANs to a destination port. SPAN is flexible, remotely configurable, and free (included in the switch), but operates on a best-effort basis and can lose packets under oversubscription.

### The Copy Semantics

SPAN creates a *best-effort copy* of traffic. This is a critical distinction:

- The original packet forwarding path is never affected. Source port traffic is forwarded at line rate regardless of SPAN status.
- The copy is generated at the ASIC level (on modern hardware) or in the CPU (on older platforms).
- If the destination port cannot absorb all copied traffic, excess frames are silently dropped. No counters, no logs, no alerts on the source port.
- The switch prioritizes production traffic over mirrored traffic in every contention scenario.

This best-effort model means SPAN is unsuitable for applications that require 100% packet capture (lawful intercept compliance, financial transaction recording) unless the monitoring design guarantees zero oversubscription.

---

## 2. Local SPAN — Single-Switch Mirroring

### ASIC-Level Packet Replication

On modern switches (Memory + ASIC architectures like Memory + Memory, or crossbar fabrics), local SPAN operates as follows:

1. A packet arrives at the ingress port and enters the forwarding pipeline.
2. The ASIC performs normal L2/L3 lookup (MAC table, routing table, ACL evaluation).
3. The forwarding result determines the egress port(s) for the original packet.
4. If the ingress or egress port is configured as a SPAN source, the ASIC additionally queues a copy of the packet to the SPAN destination port.
5. The copy may be queued after ACL evaluation (post-filter) or before (pre-filter), depending on the platform.

The replication happens in the ASIC's packet buffer. On platforms with a shared memory architecture (Memory + ASIC), the SPAN copy is simply an additional pointer to the same packet buffer — no actual data duplication occurs until egress. This is why local SPAN adds negligible latency and CPU load on modern hardware.

### Source and Destination Semantics

**Source ports** can be:
- Physical interfaces (access or trunk)
- Port channels (EtherChannel / LAG)
- VLANs (all active ports in the VLAN become sources)

**Direction filtering:**
- `both` (default): copies ingress and egress traffic
- `rx` (ingress only): copies packets as they arrive at the source port
- `tx` (egress only): copies packets as they leave the source port

When direction filtering is used on trunk ports, the VLAN tag behavior varies:
- `rx`: the copied packet includes the incoming 802.1Q tag
- `tx`: the copied packet includes the outgoing 802.1Q tag (after any tag manipulation)

**Destination port** constraints:
- Cannot be a member of a VLAN (it is removed from all VLANs when designated as SPAN destination)
- Cannot participate in STP (removed from spanning tree)
- Cannot be a source port in the same session
- Operates in a promiscuous receive mode by default (accepts all mirrored frames)
- By default, cannot transmit any traffic other than mirrored frames (ingress can be optionally enabled)

### Duplicate Packet Problem

When both `rx` and `tx` are mirrored (the default `both` direction), a packet transiting the switch appears twice at the destination:

1. Once when it enters the source port (rx copy)
2. Once when it leaves the source port (tx copy)

For traffic that both enters and exits through the same monitored port (e.g., in a router-on-a-stick configuration), this duplication is expected. For traffic that enters one source port and exits another, each port generates its own rx/tx copies.

If multiple source ports are configured and a packet traverses several of them, the destination may receive 2x, 3x, or more copies. Deduplication must be handled by the analysis tool or an intermediate packet broker.

---

## 3. RSPAN — Layer 2 Remote Mirroring

### Architecture

RSPAN extends SPAN across multiple switches within the same Layer 2 domain. Instead of delivering mirrored traffic to a local port, the source switch encapsulates mirrored frames into a special RSPAN VLAN. The RSPAN VLAN carries mirrored traffic across trunk links to a remote destination switch, which extracts the frames and delivers them to a local analysis port.

### The RSPAN VLAN

The RSPAN VLAN is a special VLAN with the following properties:

- MAC address learning is disabled. The switch does not populate its MAC table from frames in the RSPAN VLAN.
- All traffic in the RSPAN VLAN is flooded to all trunk ports that carry the VLAN. This is intentional — the destination switch must receive all mirrored frames regardless of their MAC address.
- STP still operates on the RSPAN VLAN. This means the RSPAN VLAN must not be blocked by STP on any intermediate trunk, or mirrored traffic will be lost.
- The RSPAN VLAN must be created on every switch in the path between source and destination, with the `remote-span` attribute set.

### Reflector Port Concept

On some platforms (notably older Catalyst 6500 supervisors), the RSPAN implementation requires a **reflector port**. The reflector port is a physical port that is internally looped: the switch sends mirrored traffic out the reflector port, and the same traffic re-enters the switch on the same port, now tagged with the RSPAN VLAN. This re-entered traffic is then flooded across trunks like any other RSPAN VLAN frame.

The reflector port is effectively sacrificed — it cannot be used for any other purpose. Modern ASICs (Memory + ASIC architectures in Catalyst 9000, Nexus 9000) perform the RSPAN encapsulation internally and do not require a reflector port.

### RSPAN Limitations

1. **Layer 2 scope only.** RSPAN cannot cross Layer 3 boundaries (routed links). If the source and destination switches are in different VRFs or connected via routed links, RSPAN will not work. Use ERSPAN instead.

2. **Trunk bandwidth consumption.** All mirrored traffic crosses production trunks as RSPAN VLAN frames. On congested trunks, RSPAN traffic competes with production traffic and may be dropped by QoS policies.

3. **STP dependency.** If STP blocks the RSPAN VLAN on an intermediate trunk, mirrored traffic is silently lost. There is no notification mechanism.

4. **Broadcast flooding.** Because MAC learning is disabled on the RSPAN VLAN, all traffic is flooded. This means every switch carrying the RSPAN VLAN receives all mirrored traffic, even switches that are not involved in the monitoring session.

5. **No encapsulation metadata.** Unlike ERSPAN, RSPAN does not add session ID, timestamp, or direction information to the mirrored frames. The destination receives raw copies with only the RSPAN VLAN tag as metadata.

---

## 4. ERSPAN — IP-Based Remote Mirroring

### Architecture

ERSPAN (Encapsulated Remote SPAN) transports mirrored traffic over IP using GRE (Generic Routing Encapsulation) tunnels. This eliminates the Layer 2 adjacency requirement of RSPAN and enables monitoring across routed networks, data centers, WAN links, and even cloud environments.

The source switch:
1. Copies the traffic from the source port(s)
2. Encapsulates each frame in an ERSPAN header
3. Wraps the ERSPAN-encapsulated frame in a GRE header
4. Wraps the GRE packet in an IP header with the destination set to the remote collector
5. Routes the encapsulated packet through the normal IP forwarding path

The destination (collector) receives the GRE packet, strips the IP and GRE headers, parses the ERSPAN header for metadata, and delivers the original mirrored frame to the analysis tool.

### ERSPAN Type I

The original ERSPAN implementation on Catalyst 6500. Type I has no ERSPAN-specific header — it simply wraps the mirrored frame in a GRE header with protocol type 0x88BE. The only metadata available is the GRE sequence number.

Type I is largely obsolete. It does not carry VLAN, CoS, session ID, or any identifying information beyond the GRE key. Modern deployments should use Type II or Type III.

### ERSPAN Type II — The Standard Format

Type II adds an 8-byte ERSPAN header between the GRE header and the mirrored frame. The complete encapsulation stack:

```
┌──────────────────────────────────────────────────────────┐
│  Outer Ethernet Header (14 bytes)                        │
│    Dst MAC | Src MAC | EtherType 0x0800                  │
├──────────────────────────────────────────────────────────┤
│  Outer IP Header (20 bytes)                              │
│    Src IP: origin-ip of source switch                    │
│    Dst IP: collector IP                                  │
│    Protocol: 47 (GRE)                                    │
│    TTL: configurable (default 255)                       │
│    DSCP: configurable (default 0)                        │
├──────────────────────────────────────────────────────────┤
│  GRE Header (8 bytes)                                    │
│    Flags: S=1 (sequence number present)                  │
│    Protocol Type: 0x88BE                                 │
│    Sequence Number: monotonically increasing             │
├──────────────────────────────────────────────────────────┤
│  ERSPAN Type II Header (8 bytes)                         │
│    Version: 1 (4 bits)                                   │
│    VLAN: original VLAN ID (12 bits)                      │
│    COS: original CoS/PCP (3 bits)                        │
│    En: encapsulation type (2 bits)                        │
│    T: trunk bit (1 bit)                                   │
│    Session ID: SPAN session number (10 bits)             │
│    Reserved: (12 bits)                                    │
│    Index: port/module index (20 bits)                     │
├──────────────────────────────────────────────────────────┤
│  Original Mirrored Frame (variable)                      │
│    Original Ethernet header + payload                    │
│    (may include 802.1Q tag if source was trunk)          │
└──────────────────────────────────────────────────────────┘
```

**Total overhead per packet:** 14 (Ethernet) + 20 (IP) + 8 (GRE) + 8 (ERSPAN) = 50 bytes minimum. If the outer path requires 802.1Q tagging, add 4 more bytes (54 total).

### ERSPAN Type III — Timestamped Mirroring

Type III extends Type II with a 12-byte platform-specific sub-header that adds:

- **Hardware timestamp (32 bits):** The exact time the original packet was received or transmitted, at up to 100-nanosecond granularity. This is critical for latency-sensitive analysis (financial trading, real-time systems).
- **SGT (Security Group Tag, 16 bits):** Cisco TrustSec security group tag of the original frame.
- **Direction bit (D, 1 bit):** Indicates whether the mirrored copy is from ingress (0) or egress (1). This eliminates the ambiguity present in Type II where both directions produce identical copies.
- **GRA field (2 bits):** Timestamp granularity indicator:
  - 00: 100 microseconds
  - 01: 100 nanoseconds
  - 10: IEEE 1588 PTP-synchronized
  - 11: reserved
- **Optional marker packets:** Periodic keepalive frames that indicate the ERSPAN session is active even when no traffic is being mirrored. Useful for monitoring session health.

Type III uses GRE protocol type 0x22EB (distinct from Type II's 0x88BE), allowing receivers to immediately distinguish between formats.

### ERSPAN Performance Considerations

ERSPAN encapsulation on most platforms is performed in software (CPU) rather than hardware (ASIC). This has significant implications:

1. **CPU load:** Each mirrored packet requires IP header construction, GRE encapsulation, ERSPAN header insertion, and IP routing lookup. At 10 Gbps line rate with small packets (64 bytes, ~14.88 Mpps), the CPU may not keep up.

2. **Packet loss under load:** When the CPU cannot process all mirrored packets, excess frames are dropped. The source traffic is unaffected, but the mirrored copy is incomplete.

3. **MTU fragmentation:** The 50+ byte overhead means a 1500-byte original frame becomes a 1550-byte ERSPAN packet. If the transit path has a 1500-byte MTU, the ERSPAN packet is fragmented. GRE fragmentation is problematic — many firewalls and routers drop fragmented GRE by default.

4. **Recommendation:** Set the transit path MTU to at least 1554 bytes (1500 + 50 ERSPAN + 4 possible VLAN tag), or configure MTU on the ERSPAN tunnel interface to 1450 to force the source to truncate mirrored frames.

On some modern platforms (Nexus 9000 with Memory + ASIC, Arista 7280R), ERSPAN encapsulation is performed in the forwarding ASIC at line rate, eliminating the CPU bottleneck. Check platform datasheets for "hardware ERSPAN" support.

---

## 5. Linux Kernel ERSPAN Implementation

### Kernel Support Timeline

- **Linux 4.14:** ERSPAN Type I tunnel support (ip link type erspan)
- **Linux 4.18:** ERSPAN Type II support with full header generation
- **Linux 5.0:** ERSPAN Type III support with timestamp
- **Linux 5.2:** Hardware offload API for ERSPAN on capable NICs

### tc mirred Action Architecture

The Linux traffic control (`tc`) subsystem implements mirroring through the `mirred` action. The architecture:

```
Packet arrives at ingress
    │
    ▼
┌─────────────┐     ┌──────────────┐     ┌─────────────────┐
│  qdisc      │ ──► │  classifier  │ ──► │  action chain   │
│  (ingress   │     │  (u32, flower│     │  mirred egress  │
│   or root)  │     │   match)     │     │  mirror dev X   │
└─────────────┘     └──────────────┘     └─────────────────┘
                                                │
                                                ▼
                                         Copy queued to
                                         mirror device X
                                                │
                                                ▼
                                         Original packet
                                         continues normal
                                         forwarding
```

The `mirror` action creates a clone of the skb (socket buffer) and enqueues it to the specified device. The original skb continues through the normal forwarding path. The `redirect` action (alternative to `mirror`) moves the skb to the target device without cloning — this is a redirect, not a copy.

### Hardware Offload

On NICs that support TC hardware offload (Mellanox ConnectX-5/6, Intel E810, Broadcom Memory + ASIC), the mirred action can be pushed to NIC firmware:

```bash
# Check if NIC supports TC offload
ethtool -k eth0 | grep tc-offload
# hw-tc-offload: on

# Enable TC offload
ethtool -K eth0 hw-tc-offload on

# Create offloaded mirror rule (flower classifier supports offload)
tc qdisc add dev eth0 ingress
tc filter add dev eth0 ingress protocol all \
  flower skip_sw \
  action mirred egress mirror dev erspan1

# skip_sw = skip software, execute in hardware only
# skip_hw = skip hardware, execute in software only (default)
```

Hardware-offloaded mirroring runs at line rate with zero CPU load, making it equivalent to a hardware TAP for practical purposes.

---

## 6. SPAN Session Design — Capacity Planning

### Oversubscription Analysis

The fundamental constraint in SPAN design is destination port bandwidth. If the aggregate bandwidth of all source ports exceeds the destination port capacity, packets will be dropped.

**Single source, same speed:**
$$\text{Oversubscription ratio} = \frac{B_{source} \times D_{factor}}{B_{destination}}$$

Where $D_{factor}$ accounts for direction:
- Both directions: $D_{factor} = 2$ (worst case, full duplex at line rate)
- Ingress only: $D_{factor} = 1$
- Egress only: $D_{factor} = 1$

**Multiple sources:**
$$\text{Oversubscription ratio} = \frac{\sum_{i=1}^{n} B_{source_i} \times D_{factor_i}}{B_{destination}}$$

**Example:** Four 1G source ports monitored bidirectionally to a single 10G destination:
$$\frac{4 \times 1G \times 2}{10G} = \frac{8G}{10G} = 0.8$$

This is within capacity (ratio < 1.0). But four 10G source ports to a single 10G destination:
$$\frac{4 \times 10G \times 2}{10G} = \frac{80G}{10G} = 8.0$$

This is 8:1 oversubscribed — up to 87.5% of mirrored traffic may be lost.

### ERSPAN Bandwidth Overhead

ERSPAN adds headers to every packet. The bandwidth overhead depends on packet size:

| Original Packet Size | ERSPAN Overhead | Overhead % | Effective Throughput at 10G |
|:---|:---|:---|:---|
| 64 bytes | 50 bytes | 78.1% | 5.6 Gbps effective |
| 128 bytes | 50 bytes | 39.1% | 7.2 Gbps effective |
| 256 bytes | 50 bytes | 19.5% | 8.4 Gbps effective |
| 512 bytes | 50 bytes | 9.8% | 9.1 Gbps effective |
| 1024 bytes | 50 bytes | 4.9% | 9.5 Gbps effective |
| 1500 bytes | 50 bytes | 3.3% | 9.7 Gbps effective |

For small-packet workloads (VoIP, DNS, SYN floods), the ERSPAN overhead is substantial and must be factored into capacity planning.

### Session Limit Architecture

Switch ASICs allocate dedicated hardware resources for SPAN replication:

- **Replication engine entries:** Each SPAN session consumes one or more entries in the ASIC's multicast/replication engine. The engine has a fixed number of entries shared between SPAN, multicast, and port-channel replication.
- **Copy queues:** Mirrored packets are queued in dedicated copy queues in the packet buffer. These queues have configurable depth (typically 64-256 packets) and are shared across all SPAN sessions.
- **ACL TCAM entries:** ACL-filtered SPAN sessions consume TCAM entries from the same pool used by security ACLs and QoS policies.

When the maximum session count is reached, additional sessions cannot be created. There is no graceful degradation — the configuration is rejected.

---

## 7. Use Cases and Deployment Patterns

### IDS/IPS Deployment

SPAN feeds are the primary input for network-based intrusion detection. The deployment pattern:

1. Identify critical network segments (DMZ ingress, server farm uplinks, WAN edge).
2. Configure SPAN sessions to mirror traffic from these segments to dedicated analysis ports.
3. Connect IDS sensors to the SPAN destination ports.
4. Use ACL filtering to reduce the volume to only relevant traffic (e.g., exclude known-good bulk transfers).

**Critical consideration:** IDS in SPAN mode is purely passive. It can detect but not block. For inline prevention (IPS), use TAPs with fail-open or deploy the IPS inline on the production path.

### Forensic Capture

Full packet capture for incident response and forensic analysis:

1. Configure SPAN or ERSPAN to mirror the segment of interest.
2. Feed mirrored traffic to a dedicated capture appliance (Moloch/Arkime, NetWitness, full PCAP recorder).
3. Size storage based on:

$$\text{Storage (TB)} = \frac{\text{Avg throughput (Gbps)} \times 86400 \times \text{Days}}{8 \times 1000}$$

Example: 1 Gbps average for 30 days:
$$\frac{1 \times 86400 \times 30}{8000} = 324 \text{ TB}$$

4. Use packet slicing (capture only first 128-256 bytes) to reduce storage by 80-90% while preserving all headers.

### Lawful Intercept

Regulatory-mandated traffic capture (CALEA in the US, ETSI LI in Europe):

- Requires guaranteed, lossless capture — SPAN is insufficient alone.
- Production deployments use hardware TAPs feeding dedicated LI mediation devices.
- ERSPAN may be used for transport from TAPs to a central collection point.
- Session isolation is critical — intercepted traffic must not be visible to non-authorized personnel.
- ERSPAN Type III timestamps provide the audit trail required for legal admissibility.

### Application Performance Monitoring (APM)

SPAN feeds provide network-level visibility into application transactions:

1. Mirror traffic between application tiers (web-to-app, app-to-database).
2. Feed to APM tools (Dynatrace, AppDynamics, ExtraHop) that reconstruct TCP sessions and extract application-layer metrics.
3. Filter SPAN to specific ports (80, 443, 3306, 5432) to reduce noise.
4. ERSPAN Type III timestamps enable sub-millisecond transaction latency measurement.

---

## 8. TAP vs SPAN — Engineering Trade-offs

### Signal Integrity

**Optical TAPs** split the optical signal using a passive beam splitter (typically 70/30 or 50/50 split ratio). The 70% path goes to the production device; the 30% path goes to the monitoring tool. This introduces optical insertion loss:

$$\text{Insertion loss (dB)} = -10 \times \log_{10}(\text{split ratio})$$

For a 70/30 split:
- Production path: $-10 \times \log_{10}(0.7) = 1.55$ dB loss
- Monitor path: $-10 \times \log_{10}(0.3) = 5.23$ dB loss

This loss must be within the optical budget of the link. For short-reach optics (SFP+ SR, 300m on OM3), the budget is typically 2-3 dB, leaving margin for the TAP. For long-reach optics, the margin may be insufficient.

**Copper TAPs** use active electronics to regenerate the signal. They require power but introduce no signal loss. Copper TAPs can fail, but most support fail-open mode (bypass relays that directly connect the two production ports when power is lost).

**SPAN** introduces no signal degradation whatsoever on the production path. The copy is generated internally in the ASIC from the already-received and error-checked packet.

### Completeness Guarantees

| Aspect | TAP | SPAN |
|:---|:---|:---|
| Errored frames (CRC errors) | Captured (optical) | Dropped (ASIC discards) |
| Runt frames (< 64 bytes) | Captured (optical) | Dropped |
| Jumbo frames | Passed as-is | May be dropped if dst MTU is smaller |
| PAUSE frames (802.3x) | Captured | Not mirrored (consumed by MAC) |
| Packets during link flaps | Lost during optical realignment | Not applicable |
| Oversubscription drops | Never (1:1 copy) | Yes (silent drops) |

### Cost-Benefit Decision Matrix

For most enterprise deployments, the practical recommendation:

- **Use TAPs** on critical links: data center spine-leaf uplinks, WAN edge, DMZ ingress, financial trading paths, compliance-required capture points.
- **Use SPAN** for ad-hoc troubleshooting, temporary monitoring, development/staging environments, and any scenario where physical access is difficult.
- **Use ERSPAN** when the analysis tools are centralized (SOC, cloud-based SIEM) and the monitored segments are distributed across multiple sites.

---

## 9. Packet Broker Architecture

### The Aggregation Problem

In a large monitoring deployment, the number of SPAN/TAP feeds quickly exceeds the number of analysis tool ports. A packet broker (also called a Network Packet Broker or NPB) sits between the feeds and the tools, providing:

**Aggregation (N:1):** Combine multiple SPAN/TAP feeds into a single output to a monitoring tool. The broker handles deduplication, ensuring that packets seen on multiple feeds are delivered only once.

**Load balancing (1:N):** Distribute traffic from a single high-bandwidth feed across multiple tool ports. The broker performs flow-aware hashing to ensure all packets in a TCP session reach the same tool instance.

**Filtering:** Apply L2-L4 filters to reduce the volume of traffic delivered to each tool. For example, deliver only DNS traffic to the DNS analytics tool, only HTTP traffic to the WAF, and only SMB traffic to the file activity monitor.

**Header manipulation:**
- Strip ERSPAN/GRE/VXLAN headers before delivering to tools that cannot parse encapsulated traffic.
- Add timestamps to every packet for tools that lack their own timestamping.
- Truncate packets to capture only headers (reduce tool and storage load).

### Broker Sizing

$$\text{Required broker throughput} = \sum_{i=1}^{n} F_i \times (1 + O_i)$$

Where:
- $F_i$ = throughput of feed $i$
- $O_i$ = overhead factor for processing (dedup, decryption, etc.)
- Typical $O_i$ values: 0.0 for passthrough, 0.05-0.1 for dedup, 0.2-0.5 for SSL decryption

---

## 10. Security Considerations

### SPAN as an Attack Vector

SPAN sessions create copies of potentially sensitive traffic. Security implications:

1. **Unauthorized monitoring:** An attacker with switch CLI access can create a SPAN session to mirror traffic to a port they control. SPAN session creation should be logged and alerted via SNMP traps or syslog.

2. **ERSPAN interception:** ERSPAN traffic is unencrypted GRE. Any device on the IP path can capture ERSPAN packets and reconstruct the mirrored traffic. In hostile network environments, use IPsec to encrypt the GRE tunnel (GRE-over-IPsec).

3. **RSPAN VLAN exposure:** Any switch port in the RSPAN VLAN receives all mirrored traffic. A compromised switch or misconfigured trunk port can expose monitored traffic.

4. **Destination port security:** The SPAN destination port transmits mirrored copies of production traffic. If an unauthorized device is connected to this port, it receives all mirrored frames. Physically secure SPAN destination ports and use 802.1X or MAC-based port security where possible.

### Hardening Recommendations

- Restrict SPAN configuration to TACACS+/RADIUS-authenticated administrators with privilege level 15.
- Log all `monitor session` configuration changes via syslog.
- Place ERSPAN traffic in a dedicated management VRF with ACLs restricting access to authorized collectors.
- Use IPsec transport mode for ERSPAN tunnels crossing untrusted networks.
- Periodically audit active SPAN sessions (`show monitor session all`) and remove stale sessions.
- On Linux, restrict `tc` filter creation to root and audit via auditd rules on the `tc` binary.

---

## 11. Troubleshooting Decision Tree

### No Traffic on SPAN Destination

```
Is the SPAN session configured?
├── No ──► Configure the session
└── Yes
    Is the session status "Up"?
    ├── No ──► Check "no shut" on session; check source port is up
    └── Yes
        Is the source port passing traffic?
        ├── No ──► Troubleshoot the source port (cable, STP, shutdown)
        └── Yes
            Is the destination port link up?
            ├── No ──► Check cable, SFP, NIC on analyzer
            └── Yes
                Are there output drops on the destination port?
                ├── Yes ──► Oversubscription; reduce sources or increase dst speed
                └── No
                    Is VLAN filtering configured?
                    ├── Yes ──► Verify filtered VLAN matches traffic VLAN
                    └── No
                        Is ACL filtering configured?
                        ├── Yes ──► Verify ACL permits target traffic
                        └── No ──► Check analyzer NIC is in promiscuous mode
```

### ERSPAN-Specific Troubleshooting

```
Can the source switch route to the ERSPAN destination IP?
├── No ──► Fix routing (add route, check VRF, verify next-hop)
└── Yes
    Is GRE (protocol 47) allowed on the path?
    ├── No ──► Open firewall rules for GRE / protocol 47
    └── Yes
        Is the path MTU >= 1554 bytes?
        ├── No ──► Increase MTU or reduce ERSPAN tunnel MTU
        └── Yes
            Is the destination decapsulating correctly?
            ├── No ──► Verify ERSPAN ID matches, check tunnel config
            └── Yes ──► Check analyzer application ERSPAN parsing
```

---

## Prerequisites

- vlan — 802.1Q tagging, trunk ports, VLAN concepts
- tc — Linux traffic control architecture, qdiscs, filters, actions
- tcpdump — Packet capture for verifying mirrored traffic
- ipsec — GRE-over-IPsec for securing ERSPAN tunnels
- cos-qos — QoS impact on mirrored traffic priority
- binary encoding, GRE encapsulation, IP routing

## References

- [IETF — ERSPAN Type II/III Draft (draft-foschiano-erspan-03)](https://datatracker.ietf.org/doc/html/draft-foschiano-erspan)
- [RFC 2784 — Generic Routing Encapsulation (GRE)](https://datatracker.ietf.org/doc/html/rfc2784)
- [RFC 2890 — Key and Sequence Number Extensions to GRE](https://datatracker.ietf.org/doc/html/rfc2890)
- [Cisco — SPAN, RSPAN, and ERSPAN Configuration (Catalyst 9000)](https://www.cisco.com/c/en/us/td/docs/switches/lan/catalyst9300/software/release/17-6/configuration_guide/nmgmt/b_176_nmgmt_9300_cg/configuring_span_and_rspan.html)
- [Cisco — ERSPAN on Nexus 9000 NX-OS](https://www.cisco.com/c/en/us/td/docs/switches/datacenter/nexus9000/sw/93x/interfaces/configuration/guide/b-cisco-nexus-9000-nx-os-interfaces-configuration-guide-93x/b-cisco-nexus-9000-nx-os-interfaces-configuration-guide-93x_chapter_011001.html)
- [Linux Kernel — ERSPAN Tunnel (ip-link-type-erspan)](https://man7.org/linux/man-pages/man8/ip-link.8.html)
- [Linux tc-mirred — Traffic Mirroring Action](https://man7.org/linux/man-pages/man8/tc-mirred.8.html)
- [Open vSwitch — Port Mirroring Documentation](https://docs.openvswitch.org/en/latest/faq/configuration/)
- [Gigamon — GigaSMART Visibility Architecture](https://www.gigamon.com/products/optimize-traffic/gigasmart.html)
- [Keysight (Ixia) — Network Packet Broker Design Guide](https://www.keysight.com/us/en/products/network-packet-brokers.html)
