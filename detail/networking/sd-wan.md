# SD-WAN Deep Dive -- Control Plane Architecture, Path Selection Algorithms, and Cost Analysis

> *SD-WAN replaces static, transport-coupled WAN architectures with a programmable overlay that separates policy from forwarding. The core innovations are centralized control, application-aware path selection using real-time SLA metrics, and transport independence that turns commodity internet into enterprise-grade WAN.*

---

## 1. Control Plane Architecture

### Three-Tier Model

Every SD-WAN implementation follows a variation of this architecture:

```
+--------------------+
|    Orchestrator     |   Policy authoring, provisioning, analytics
|  (Management Plane) |   Single pane of glass for all sites
+--------------------+
         |
         | REST API / NETCONF / gRPC
         v
+--------------------+
|     Controller      |   Route computation, topology distribution
|  (Control Plane)    |   OMP, VRRP, or proprietary protocol
+--------------------+
         |
         | Secure control channel (DTLS/TLS)
         v
+--------------------+
|    Edge / CPE       |   Data forwarding, tunnel termination
|  (Data Plane)       |   IPsec, DPI, local breakout
+--------------------+
```

### Orchestrator

The orchestrator is the single source of truth for the entire SD-WAN fabric:

- **Device inventory:** Serial numbers, site assignments, software versions
- **Policy repository:** Application policies, security rules, SLA profiles
- **Template engine:** Device configuration templates with variables per site
- **Certificate authority:** Issues and manages device identity certificates for mutual TLS/DTLS authentication
- **Analytics engine:** Aggregates telemetry from all edges for dashboards, anomaly detection, and capacity planning

In Cisco Viptela, this is vManage. In VMware VeloCloud, this is the VCO (VeloCloud Orchestrator). In Fortinet, it is FortiManager with SD-WAN extensions.

### Controller

The controller computes the overlay routing table and distributes it to edges:

- **Overlay routing protocol:** Cisco uses OMP (Overlay Management Protocol); others use modified BGP or proprietary protocols
- **Topology awareness:** Maintains a real-time graph of all edges, their transport links, and current path quality
- **Route reflection:** Acts as a route reflector so edges do not need full-mesh peering
- **Policy enforcement point:** Applies control policies (route filtering, topology restrictions) before distributing routes

**Scaling consideration:** Controllers are stateless for forwarding (no data plane traffic passes through them). A pair of controllers can manage thousands of edges. The control channel uses DTLS or TLS, consuming minimal bandwidth (typically under 100 Kbps per edge).

### Edge (CPE)

The edge device is where forwarding decisions happen:

- **Tunnel management:** Establishes and maintains IPsec tunnels to other edges and hub sites
- **Application classification:** Runs DPI engine to identify applications at L7
- **SLA monitoring:** Sends periodic probes (BFD, HTTP, ICMP) across each tunnel to measure latency, jitter, and loss
- **Policy execution:** Applies forwarding decisions based on controller-pushed policies and local SLA measurements
- **Local breakout:** Optionally routes trusted SaaS traffic directly to the internet without backhauling

---

## 2. Overlay Routing Protocols

### OMP (Overlay Management Protocol) -- Cisco Viptela

OMP is a TCP-based protocol (port 12346) running between edges and controllers:

- **OMP routes:** Carry prefix, next-hop (TLOC -- Transport Location), and attributes (preference, origin, community)
- **TLOCs:** A tuple of (system-IP, color, encapsulation) uniquely identifying a transport endpoint
- **Service routes:** Advertise services (firewall, IDS) available at specific sites for service chaining

```
OMP route entry:
  Prefix:      10.1.0.0/24
  Origin:      connected
  TLOC:        1.1.1.1, mpls, ipsec
  Preference:  100
  Site-ID:     100
  VPN:         1
```

### VXLAN/EVPN-Based Overlays

Some SD-WAN solutions (particularly for data center interconnect) use VXLAN with EVPN control plane:

- BGP EVPN distributes MAC/IP reachability across sites
- VXLAN encapsulation carries L2/L3 traffic over the WAN
- Useful for extending L2 domains across geographies (VM mobility, disaster recovery)

### Proprietary Protocols

- **VMware VeloCloud:** VCMP (VeloCloud Multi-Path Protocol) -- UDP-based, handles encryption, path selection, and FEC in a single encapsulation
- **Fortinet:** Uses standard IPsec with proprietary control signaling over HTTPS
- **Versa:** Uses standard BGP for overlay routing with extensions for application awareness

---

## 3. Application Identification

### Deep Packet Inspection (DPI)

DPI engines classify traffic by inspecting packet payloads beyond L3/L4 headers:

- **Signature matching:** Database of known application byte patterns (updated regularly)
- **Behavioral analysis:** Classifies unknown flows by behavior (packet sizes, timing, directionality)
- **TLS/SSL inspection:** Extracts SNI (Server Name Indication) from TLS ClientHello to identify applications without decryption
- **Certificate inspection:** Examines X.509 subject/SAN fields for application identification

### Classification Hierarchy

```
Classification priority (highest to lowest):
  1. Custom application signatures (admin-defined)
  2. FQDN-based rules (DNS snooping / SNI matching)
  3. DPI signature database (vendor-maintained)
  4. IP/port/protocol rules (L3/L4 fallback)
  5. Default policy (catch-all)
```

### FQDN-Based Identification

```
DNS snooping workflow:
  1. Edge intercepts DNS query from client
  2. Records mapping: FQDN -> resolved IP addresses
  3. When data packets arrive from that IP, applies the FQDN-based policy
  4. Cache entry expires when DNS TTL expires

Limitation: Fails when applications use DNS-over-HTTPS (DoH)
           or hardcoded IP addresses
```

### SaaS Detection

Major SD-WAN vendors maintain curated databases of SaaS application endpoints:

| SaaS Provider    | Detection Method                           | Endpoints Tracked |
|------------------|--------------------------------------------|-------------------|
| Microsoft 365    | Published IP/URL lists + SNI              | ~150 FQDNs        |
| Salesforce       | SNI + certificate inspection              | ~20 FQDNs         |
| Zoom             | DSCP marking + port ranges + SNI          | ~10 FQDNs         |
| AWS/Azure/GCP    | Published IP ranges (JSON feeds)          | Thousands of CIDRs|

Microsoft publishes its endpoint list via REST API, enabling automatic policy updates:

```
GET https://endpoints.office.com/endpoints/worldwide?clientrequestid=...
Response includes: IP ranges, FQDNs, ports, categories (Optimize/Allow/Default)
```

---

## 4. SLA-Based Path Selection Algorithm

### Probe Mechanism

Each edge continuously monitors path quality using lightweight probes:

```
Probe parameters:
  Protocol:    BFD (Bidirectional Forwarding Detection) or ICMP or HTTP
  Interval:    100 ms - 10 s (configurable, typical: 1 s)
  Samples:     6 (sliding window)
  Multiplier:  6 (declare path down after 6 missed probes)
```

### SLA Metrics Computation

For a sliding window of $N$ probe samples:

**One-way latency:**

$$L = \frac{1}{N} \sum_{i=1}^{N} \frac{RTT_i}{2}$$

**Jitter (inter-packet delay variation):**

$$J = \frac{1}{N-1} \sum_{i=2}^{N} |L_i - L_{i-1}|$$

**Packet loss:**

$$PL = \frac{\text{probes lost}}{N} \times 100\%$$

### Path Selection Decision

```
For each application flow:
  1. Look up SLA profile (latency_max, jitter_max, loss_max)
  2. For each available path:
     a. Retrieve current metrics (L, J, PL)
     b. Check: L <= latency_max AND J <= jitter_max AND PL <= loss_max
     c. If all thresholds met: path is ELIGIBLE
  3. Among eligible paths:
     a. Apply preference order (e.g., MPLS > broadband > LTE)
     b. If multiple paths share same preference: weighted ECMP
  4. If NO path meets SLA:
     a. Use best-available path (lowest composite score)
     b. Alert orchestrator for operator notification
```

### Composite Score Calculation

When multiple paths are eligible, some implementations use a weighted composite score:

$$S_{path} = w_L \cdot \frac{L}{L_{max}} + w_J \cdot \frac{J}{J_{max}} + w_{PL} \cdot \frac{PL}{PL_{max}}$$

Where $w_L + w_J + w_{PL} = 1$ (configurable weights).

Lower score is better. Typical weights for voice: $w_L = 0.4$, $w_J = 0.3$, $w_{PL} = 0.3$.

### Failover Timing

| Event                     | Detection Time   | Recovery Action     |
|---------------------------|------------------|---------------------|
| Link down (physical)      | < 1 s (BFD)     | Immediate reroute   |
| SLA violation             | 1-10 s (probes)  | Gradual migration   |
| Path quality restoration  | 30-60 s (dampening)| Revert after stable |

Dampening prevents flapping: a path must remain within SLA for a holddown period before traffic returns to it.

---

## 5. WAN Optimization Techniques

### Data Deduplication

```
First transfer of file (100 MB):
  Edge A segments data into variable-length chunks (Rabin fingerprint)
  Each chunk is hashed (SHA-256) and stored in local cache
  Full 100 MB is transmitted over the WAN
  Edge B receives, segments identically, populates its cache

Subsequent transfer of same/similar file:
  Edge A computes chunk hashes
  Matches found in cache: send hash references (32 bytes each) instead of data
  New/changed chunks: send full data
  Typical reduction: 60-95% for repeated content
```

**Cache sizing:** A 50 GB dedup cache at each edge covers approximately 30 days of unique data for a typical branch with 100 Mbps throughput.

### Compression

| Algorithm | Compression Ratio | CPU Cost  | Use Case                |
|-----------|-------------------|-----------|-------------------------|
| LZ4       | 2:1 - 3:1         | Very low  | Real-time, high-throughput |
| zstd      | 3:1 - 5:1         | Low       | Balanced speed/ratio    |
| gzip      | 3:1 - 6:1         | Medium    | Batch transfers         |

Compression is applied after deduplication. Combined effect:

$$\text{Effective reduction} = 1 - (1 - D) \times (1 - C)$$

Where $D$ = dedup ratio, $C$ = compression ratio. For $D = 0.70$ and $C = 0.50$:

$$1 - (0.30 \times 0.50) = 0.85 = 85\% \text{ reduction}$$

### TCP Optimization

The fundamental problem with TCP over WANs:

$$\text{Throughput}_{max} = \frac{W_{max}}{RTT}$$

Where $W_{max}$ is the TCP window size. For a 64 KB window and 100 ms RTT:

$$\frac{65536 \text{ bytes}}{0.1 \text{ s}} = 655 \text{ KB/s} = 5.2 \text{ Mbps}$$

Even on a 1 Gbps link, TCP throughput is capped at 5.2 Mbps with these parameters.

**SD-WAN TCP proxy solution:**

```
Client <--5ms--> Edge A <--100ms--> Edge B <--5ms--> Server

Without proxy:
  Effective RTT: 110 ms
  Max throughput: ~4.8 Mbps (64K window)

With proxy:
  Client sees RTT: 5 ms (to local edge)
  Edge-to-edge: separate optimized TCP (large windows, SACK, timestamps)
  Max throughput: ~85 Mbps (64K window, 5ms RTT to edge)
```

### Forward Error Correction (FEC)

FEC adds redundancy packets so the receiver can reconstruct lost packets without retransmission:

$$\text{FEC ratio} = \frac{k}{n}$$

Where $k$ = data packets, $n$ = total packets (data + parity). A ratio of 4/5 means 1 parity packet per 4 data packets (20% overhead, can recover from any single packet loss in the group).

| Link Loss Rate | FEC Ratio | Bandwidth Overhead | Effective Loss After FEC |
|----------------|-----------|--------------------|--------------------------|
| 1%             | 9/10      | 11%                | ~0.01%                   |
| 5%             | 4/5       | 25%                | ~0.25%                   |
| 10%            | 3/4       | 33%                | ~1.0%                    |

---

## 6. SASE Architecture (SSE + SD-WAN)

### The Convergence

SASE (Secure Access Service Edge), coined by Gartner in 2019, merges networking (SD-WAN) and security (SSE) into a unified cloud-delivered service:

```
Traditional architecture:
  Branch --> MPLS --> DC Firewall --> Internet
  Remote User --> VPN --> DC Firewall --> Internet
  Problem: All traffic hairpins through the data center

SASE architecture:
  Branch --> SD-WAN Edge --> Nearest SSE PoP --> Internet/SaaS
  Remote User --> ZTNA Agent --> Nearest SSE PoP --> Private Apps
  Benefit: Security applied at the edge, closest to user/branch
```

### SSE Components in Detail

**Secure Web Gateway (SWG):**
- Inline proxy for all HTTP/HTTPS traffic
- URL categorization and filtering (80+ categories)
- Anti-malware scanning (signature + sandboxing)
- Data loss prevention (DLP) for outbound content inspection

**Cloud Access Security Broker (CASB):**
- Shadow IT discovery: identifies unsanctioned SaaS usage
- Inline CASB: enforces policies on sanctioned SaaS in real time
- API CASB: connects to SaaS APIs (Microsoft 365, Google Workspace) for at-rest scanning
- Tracks 30,000+ cloud applications with risk scores

**Zero Trust Network Access (ZTNA):**
- Replaces site-to-site VPN for remote access
- Per-application access control (no network-level access)
- Identity + device posture + context = access decision
- Applications are never exposed to the internet (broker-mediated)

**Firewall as a Service (FWaaS):**
- Cloud-delivered L3/L4/L7 firewall
- Replaces branch hardware firewalls
- Centralized policy, distributed enforcement at PoPs

### SASE PoP Architecture

```
SASE Point of Presence (PoP):
  +--------------------------------------------------+
  |  Anycast IP ingress                                |
  |  +-----------+  +-----------+  +---------------+  |
  |  |  SD-WAN   |  |    SWG    |  |     ZTNA      |  |
  |  |  Gateway  |  |  + CASB   |  |    Broker     |  |
  |  +-----------+  +-----------+  +---------------+  |
  |  +-----------+  +-----------+  +---------------+  |
  |  |  FWaaS    |  |    DLP    |  |   Sandboxing  |  |
  |  +-----------+  +-----------+  +---------------+  |
  |  Single-pass inspection engine (no service chain) |
  +--------------------------------------------------+

Traffic flow:
  1. Branch SD-WAN edge sends traffic to nearest PoP (anycast)
  2. Single-pass engine inspects: FW -> SWG -> CASB -> DLP
  3. Clean traffic forwarded to destination
  4. Return traffic follows same path for symmetric inspection
```

---

## 7. Multi-Cloud Connectivity Patterns

### Pattern 1: Cloud On-Ramp (Direct Connect)

```
Branch --> SD-WAN Edge --> IPsec Tunnel --> Cloud Provider VPN Gateway
                                            |
                                            v
                                     VPC/VNet (workloads)

Providers:
  AWS:   Virtual Private Gateway / Transit Gateway
  Azure: Virtual WAN Hub / VPN Gateway
  GCP:   Cloud VPN / Cloud Interconnect
```

SD-WAN orchestrators automate tunnel setup to cloud VPN gateways, including BGP peering for dynamic route exchange.

### Pattern 2: Virtual SD-WAN Edge in Cloud

```
Branch --> SD-WAN Edge --> IPsec --> Cloud-hosted SD-WAN Edge (VM)
                                         |
                                         v
                                    VPC/VNet (workloads)

Benefits:
  - Full SD-WAN policy enforcement inside the cloud
  - Application-aware routing between cloud regions
  - Consistent policy across on-prem and cloud
```

### Pattern 3: Multi-Cloud Transit

```
                    SD-WAN Controller
                    /       |       \
          Branch Edge   AWS Edge   Azure Edge
              |            |           |
          On-prem       AWS VPC    Azure VNet
              \            |           /
               Unified overlay routing
               (any-to-any connectivity)
```

This pattern creates a multi-cloud backbone using SD-WAN overlay routing, avoiding the need for separate cloud-native transit solutions (AWS Transit Gateway, Azure Virtual WAN) for cross-cloud traffic.

---

## 8. Cost Analysis: MPLS vs SD-WAN

### Per-Site Monthly Cost Comparison

| Cost Component            | MPLS Only      | SD-WAN (Hybrid)  | SD-WAN (Internet Only) |
|---------------------------|----------------|------------------|------------------------|
| MPLS circuit (50 Mbps)    | $800           | $800 (retained)  | $0                     |
| Broadband (200 Mbps)      | $0             | $80              | $80                    |
| LTE backup (unlimited)    | $0             | $50              | $50                    |
| SD-WAN edge license       | $0             | $150             | $150                   |
| SD-WAN SaaS (per-site)    | $0             | $50              | $50                    |
| **Monthly total**         | **$800**       | **$1,130**       | **$330**               |
| **Bandwidth**             | **50 Mbps**    | **250+ Mbps**    | **200+ Mbps**          |
| **Cost per Mbps**         | **$16.00**     | **$4.52**        | **$1.65**              |

### TCO Over 3 Years (100 Sites)

| Item                       | MPLS Only       | SD-WAN Hybrid    | SD-WAN Internet Only |
|----------------------------|-----------------|------------------|----------------------|
| Circuits (36 months)       | $2,880,000      | $3,348,000       | $468,000             |
| Hardware (edge appliances) | $0 (carrier CPE)| $300,000         | $300,000             |
| Deployment (professional)  | $50,000         | $100,000         | $100,000             |
| Management platform        | $0 (carrier)    | $180,000         | $180,000             |
| **3-year TCO**             | **$2,930,000**  | **$3,928,000**   | **$1,048,000**       |
| **Total bandwidth**        | **5 Gbps**      | **25 Gbps**      | **20 Gbps**          |
| **TCO per Gbps per year**  | **$195,333**    | **$52,373**      | **$17,467**          |

### Hidden MPLS Costs Often Overlooked

- **Change-order fees:** $200-500 per bandwidth change or routing modification
- **Lead times:** 30-90 days for new circuits; SD-WAN broadband deploys in days
- **Over-provisioning:** MPLS circuits are often sized for peak, wasting 60-70% of capacity
- **Geographic constraints:** MPLS availability is limited in rural areas; broadband/LTE has wider coverage
- **Multi-provider complexity:** International MPLS requires inter-carrier agreements, adding cost and latency

### Break-Even Analysis

The hybrid approach (MPLS + SD-WAN) costs more than MPLS-only in pure dollars, but delivers 5x the bandwidth. The break-even point for migrating entirely off MPLS depends on application tolerance:

```
If all applications can tolerate:
  Latency:     < 50 ms (domestic broadband)
  Jitter:      < 10 ms (with FEC)
  Loss:        < 0.5% (with FEC)
Then:
  Full internet SD-WAN saves 64% vs MPLS over 3 years

If voice/video require:
  Latency:     < 20 ms
  Jitter:      < 5 ms
  Loss:        < 0.1%
Then:
  Keep MPLS for voice/video (~20% of traffic)
  Move remaining 80% to broadband
  Net savings: ~45% vs full MPLS
```

### Cost-per-Mbps Trend

MPLS pricing has been relatively flat over the past decade while broadband bandwidth per dollar doubles roughly every 3 years:

| Year | MPLS ($/Mbps/month) | Broadband ($/Mbps/month) | Ratio   |
|------|---------------------|--------------------------|---------|
| 2015 | $20                 | $2.00                    | 10x     |
| 2018 | $18                 | $1.00                    | 18x     |
| 2021 | $16                 | $0.50                    | 32x     |
| 2024 | $14                 | $0.30                    | 47x     |

This widening gap is the fundamental economic driver behind SD-WAN adoption.
