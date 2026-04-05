# SD-WAN (Software-Defined Wide Area Network)

Architecture that decouples the WAN control plane from the data plane, enabling centralized policy management, transport-independent overlay tunnels, and application-aware path selection across multiple WAN links.

## Concepts

### Architecture

- **Centralized control plane:** An orchestrator/controller manages policies, topology, and routing decisions for all edge devices
- **Distributed data plane:** Edge appliances (CPE) forward traffic locally based on policies pushed from the controller
- **Overlay tunnels:** Encrypted tunnels (IPsec, GRE, VXLAN) built over any transport, abstracting the underlay
- **Transport independence:** Works over MPLS, broadband, LTE/5G, satellite, or any combination simultaneously

### Core Components

| Component      | Role                                                        |
|----------------|-------------------------------------------------------------|
| Orchestrator   | Central management, provisioning, policy authoring, analytics |
| Controller     | Route computation, topology distribution, path signaling    |
| Edge (CPE)     | Packet forwarding, tunnel termination, local breakout       |
| Analytics      | Real-time telemetry, QoE dashboards, anomaly detection      |

### Overlay Tunnel Types

| Tunnel    | Use Case                              | Overhead  |
|-----------|---------------------------------------|-----------|
| IPsec     | Encrypted site-to-site over internet  | 50-73 B   |
| GRE       | Simple encapsulation, no encryption   | 24 B      |
| VXLAN     | Data center interconnect overlays     | 50 B      |
| WireGuard | Lightweight encrypted tunnels         | 32 B      |

### Application-Aware Routing

- **Deep Packet Inspection (DPI):** Classifies traffic by application signatures (L7)
- **FQDN-based identification:** Matches traffic by DNS name for SaaS and cloud apps
- **SaaS detection:** Recognizes Microsoft 365, Salesforce, Zoom, etc., and applies per-app policies
- **Application SLA profiles:** Define latency, jitter, and packet loss thresholds per application class

### Path Selection

```
Application SLA Profile:
  Voice (DSCP EF):
    Max latency:     150 ms
    Max jitter:       30 ms
    Max packet loss:  0.1%

  Video (DSCP AF41):
    Max latency:     200 ms
    Max jitter:       50 ms
    Max packet loss:  0.5%

  Bulk data (DSCP AF11):
    Max latency:     N/A
    Max jitter:      N/A
    Max packet loss:  1.0%
```

- Edge device continuously probes all paths (BFD, ICMP, HTTP probes)
- When a path violates an SLA threshold, traffic is re-routed to the next best path in sub-second time
- Weighted ECMP distributes traffic across multiple qualifying paths

### Transport Options

| Transport   | Typical Latency | Cost    | Reliability | Use Case                  |
|-------------|-----------------|---------|-------------|---------------------------|
| MPLS        | Low, consistent | High    | SLA-backed  | Mission-critical apps     |
| Broadband   | Variable        | Low     | Best-effort | General traffic, backup   |
| LTE/5G      | Variable        | Medium  | Best-effort | Remote/mobile sites       |
| Satellite   | High (>500 ms)  | High    | Weather-dep | Rural/maritime locations  |

## Zero-Touch Provisioning (ZTP)

```
Deployment workflow:
  1. Pre-stage device serial + site config in orchestrator
  2. Ship appliance to remote site (no IT staff needed)
  3. Appliance boots, reaches cloud/on-prem ZTP server via DHCP/DNS
  4. Downloads bootstrap config, certificates, tunnel parameters
  5. Establishes encrypted tunnels to hub/controller
  6. Pulls full policy set from orchestrator
  7. Site is live — no manual CLI configuration
```

### ZTP DNS/DHCP Requirements

```bash
# DHCP option to point to ZTP server
dhcp-option=43,<vendor-specific-ztp-url>

# DNS SRV record for controller discovery
_sdwan-controller._tcp.example.com. IN SRV 0 0 443 controller.example.com.
```

## Centralized Policy Management

### Policy Types

| Policy Type        | Scope                                           |
|--------------------|--------------------------------------------------|
| Application policy | Per-app SLA, path preference, QoS marking       |
| Data policy        | Traffic steering, NAT, service chaining          |
| Control policy     | Route filtering, topology restriction            |
| Security policy    | Firewall, IPS, URL filtering, DNS security       |

### Policy Push Example (Conceptual)

```yaml
# Steer voice traffic to MPLS, fallback to LTE
policy:
  name: voice-steering
  match:
    application: voip
    dscp: ef
  action:
    preferred-path: mpls
    fallback: lte
    sla:
      latency: 150ms
      jitter: 30ms
      loss: 0.1%
```

## WAN Optimization

### Techniques

- **Deduplication:** Caches byte patterns; subsequent transfers send references instead of data (60-90% reduction for repeated content)
- **Compression:** LZ4/zstd compression of payload data (10-40% reduction)
- **TCP optimization:** TCP proxy at each edge eliminates long-RTT effects; uses local ACKs, window scaling, selective ACK
- **Forward Error Correction (FEC):** Adds redundancy packets to recover from loss without retransmission
- **Packet coalescing:** Aggregates small packets to reduce per-packet overhead

### TCP Optimization Comparison

```
Without optimization (100 ms RTT, 1% loss):
  Effective throughput: ~1.2 Mbps on 100 Mbps link

With TCP proxy at SD-WAN edge:
  Local RTT: 5 ms (LAN to edge)
  Effective throughput: ~85 Mbps on 100 Mbps link
```

## Direct Internet Access (DIA) vs Backhauling

### Backhauling (Traditional)

```
Branch --[MPLS]--> Data Center --[Firewall]--> Internet
  Pros: Centralized security inspection
  Cons: Hairpin latency, DC bandwidth bottleneck, poor SaaS performance
```

### Direct Internet Access

```
Branch --[Local Breakout]--> Internet (for SaaS/cloud)
Branch --[Tunnel]--> Data Center (for internal apps)
  Pros: Lower latency to cloud/SaaS, reduced DC load
  Cons: Requires security at the branch (or cloud-delivered)
```

### Split Tunneling

```
Traffic classification at branch edge:
  Microsoft 365   --> Direct internet breakout (trusted SaaS)
  Zoom/WebEx      --> Direct internet breakout (real-time)
  Internal ERP    --> Tunnel to data center
  Unknown traffic --> Tunnel to data center (inspect first)
```

## SASE Integration (SD-WAN + Security)

### SASE = SD-WAN + SSE (Security Service Edge)

| SSE Component     | Function                                      |
|-------------------|-----------------------------------------------|
| SWG               | Secure Web Gateway: URL filtering, malware    |
| CASB              | Cloud Access Security Broker: SaaS visibility |
| ZTNA              | Zero Trust Network Access: identity-based     |
| FWaaS             | Firewall as a Service: cloud-delivered L3/L4  |
| RBI               | Remote Browser Isolation: sandboxed browsing  |

```
SASE traffic flow:
  User/Branch --> SD-WAN Edge --> SSE PoP --> Internet/SaaS
                                         --> Private Apps (via ZTNA)
```

## SD-WAN vs Traditional WAN

| Aspect             | Traditional WAN (MPLS)       | SD-WAN                         |
|--------------------|------------------------------|--------------------------------|
| Transport          | Single (MPLS)                | Any (MPLS + broadband + LTE)   |
| Path selection     | BGP/OSPF (L3 metrics)        | Application-aware (L7 + SLA)   |
| Provisioning       | Weeks (carrier lead time)    | Minutes (ZTP)                  |
| Policy changes     | Per-device CLI               | Centralized, templated         |
| Cost (typical)     | $500-2000/site/month         | $100-500/site/month            |
| Encryption         | Optional (not default)       | Always-on IPsec                |
| Cloud connectivity | Backhaul through DC          | Direct cloud on-ramps          |
| Redundancy         | Expensive (dual MPLS)        | Built-in (multiple transports) |

## SD-WAN vs VPN

| Aspect            | Site-to-Site VPN              | SD-WAN                          |
|-------------------|-------------------------------|----------------------------------|
| Intelligence      | None (static tunnels)         | App-aware, SLA-based routing     |
| Management        | Per-device config             | Centralized orchestrator         |
| Path selection    | Manual failover               | Automatic, sub-second failover   |
| WAN optimization  | None                          | Dedup, compression, TCP opt      |
| Multi-transport   | Typically single link          | Multiple links, active-active    |
| Scalability       | N-squared tunnel mesh          | Hub-spoke or dynamic mesh        |

## Vendor Landscape

| Vendor               | Product          | Differentiator                      |
|----------------------|------------------|-------------------------------------|
| Cisco                | Viptela/Meraki   | Largest market share, deep routing  |
| VMware (Broadcom)    | VeloCloud        | Multi-cloud, gateway mesh           |
| Fortinet             | FortiSASE        | Integrated security (FortiGuard)    |
| Palo Alto Networks   | Prisma SD-WAN    | SASE-native, IoT security           |
| Versa Networks       | Versa SASE       | Single-stack SD-WAN + SSE           |
| HPE Aruba            | EdgeConnect      | WAN optimization heritage           |

### Open-Source / DIY Options

| Project   | Description                                         |
|-----------|-----------------------------------------------------|
| VyOS      | Full-featured router OS, IPsec, BGP, policy routing |
| pfSense   | Firewall/router with multi-WAN, OpenVPN, IPsec      |
| OpenWrt   | Embedded Linux, mwan3 for multi-WAN load balancing   |
| Netmaker  | WireGuard-based mesh networking with central UI      |

## Inspection and Monitoring

```bash
# Check tunnel status (generic — vendor CLI varies)
show sdwan control connections
show sdwan bfd sessions

# View application-aware routing table
show sdwan app-route statistics

# Monitor path quality (latency, jitter, loss per path)
show sdwan app-route sla-class

# Check OMP (Overlay Management Protocol) routes — Cisco Viptela
show sdwan omp routes

# VyOS: Check IPsec tunnel status
sudo ipsec statusall
sudo ip xfrm state
sudo ip xfrm policy

# pfSense: Check gateway status (multi-WAN)
# Status > Gateways > Gateway Groups in web UI

# OpenWrt: Check mwan3 policy status
mwan3 status
mwan3 interfaces
```

## Tips

- Start with a hybrid approach: keep MPLS for critical apps and add broadband for everything else; migrate off MPLS only after SD-WAN proves stable.
- Always test SLA thresholds with real traffic before production; overly aggressive thresholds cause unnecessary path flapping.
- Use BFD with sub-second timers for fast failover, but tune carefully to avoid false positives on congested links.
- Place analytics and logging before DIA breakout to maintain visibility into internet-bound traffic.
- For SASE deployments, choose vendors where SD-WAN and SSE are truly integrated, not bolted together via acquisition.
- Zero-touch provisioning is only zero-touch if you pre-stage correctly; test with a pilot site before mass deployment.
- When comparing costs, include MPLS circuit lead times and change-order fees, not just monthly recurring charges.
- Use forward error correction (FEC) on high-loss links (LTE, satellite) rather than relying solely on retransmission.
- Monitor per-application QoE metrics, not just link-level stats; a link can look healthy while a specific app suffers.
- Keep firmware versions consistent across all edges; mixed versions are the leading cause of hard-to-diagnose tunnel issues.

## See Also

- ipsec, mpls, bgp, vpn

## References

- [MEF 70.1 — SD-WAN Service Attributes and Service Framework](https://www.mef.net/resources/mef-70-1-sd-wan-service-attributes-and-services/)
- [RFC 9000 — QUIC: A UDP-Based Multiplexed and Secure Transport](https://www.rfc-editor.org/rfc/rfc9000) (relevant to SD-WAN tunnel transports)
- [Cisco SD-WAN (Viptela) Configuration Guide](https://www.cisco.com/c/en/us/td/docs/routers/sdwan/configuration/sdwan-xe-gs-book.html)
- [Fortinet SD-WAN Architecture Guide](https://docs.fortinet.com/document/fortigate/latest/sd-wan)
- [VyOS Documentation — IPsec Site-to-Site](https://docs.vyos.io/en/latest/configuration/vpn/site2site-ipsec.html)
- [OpenWrt mwan3 — Multi-WAN Manager](https://openwrt.org/docs/guide-user/network/wan/multiwan/mwan3)
- [Gartner SD-WAN Magic Quadrant](https://www.gartner.com/reviews/market/sd-wan)
- [NIST SP 800-207 — Zero Trust Architecture](https://csrc.nist.gov/publications/detail/sp/800-207/final) (foundational for SASE/ZTNA)
