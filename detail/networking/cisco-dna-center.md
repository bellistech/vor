# Cisco DNA Center — Intent-Based Networking Platform Architecture

> *Cisco DNA Center (now Catalyst Center) is the realization of intent-based networking: administrators declare what the network should do, and the controller translates that intent into device-level configurations, policy enforcement, and continuous assurance. Understanding its architecture means understanding how abstraction layers convert business intent into CLI commands, TCAM entries, and telemetry pipelines.*

---

## 1. Intent-Based Networking Theory

### The Intent Abstraction Stack

Traditional network management operates at the device configuration level — engineers write CLI commands per device. Intent-based networking (IBN) introduces abstraction layers:

```
Layer 4: Business Intent     "Sales team can access CRM but not engineering servers"
Layer 3: Network Policy       SGT(Sales) → SGT(CRM_Servers) = Permit
Layer 2: Device Policy         SGACL on switch: permit tcp dst eq 443
Layer 1: Data Plane            TCAM entry: tag 10 → tag 40 → permit
```

Each layer translates higher-level abstractions into lower-level primitives:

$$\text{Intent}_{L_n} \xrightarrow{\text{translation}} \text{Config}_{L_{n-1}} \xrightarrow{\text{push}} \text{Device}_{L_0}$$

The value proposition is that changes at Layer 4 automatically propagate through all lower layers. An administrator adds a new user to the "Sales" group in ISE, and the network automatically enforces the correct access policies on every switch, router, and wireless controller.

### Intent Translation Pipeline

The Catalyst Center intent pipeline follows this sequence:

```
1. Design    → Define network hierarchy, IP pools, credentials
2. Policy    → Define who can talk to whom, with what QoS
3. Provision → Map policies to devices, push configurations
4. Assure    → Verify the network is behaving as intended
5. Remediate → Detect deviations, suggest or auto-fix
```

This is a closed-loop system. Assurance data feeds back into the intent engine, which can detect when the actual network state diverges from the intended state:

$$\Delta_{\text{drift}} = |\text{State}_{\text{intended}} - \text{State}_{\text{actual}}|$$

When $\Delta_{\text{drift}} > \text{threshold}$, the system raises an issue, suggests remediation, or (in mature deployments) auto-remediates.

---

## 2. Microservices Architecture

### Platform Foundation

Catalyst Center runs on a Kubernetes-based platform called Maglev. The architecture layers:

```
┌──────────────────────────────────────────────┐
│             Application Services             │
│  (Design, Policy, Provision, Assurance)      │
├──────────────────────────────────────────────┤
│              Platform Services               │
│  (API Gateway, Auth, Task Manager, NDP)      │
├──────────────────────────────────────────────┤
│            Infrastructure Services           │
│  (Service Mesh, Config Store, Message Bus)   │
├──────────────────────────────────────────────┤
│              Kubernetes / Docker             │
├──────────────────────────────────────────────┤
│           Linux OS (CentOS-based)            │
├──────────────────────────────────────────────┤
│          Cisco UCS Hardware (Bare Metal)     │
└──────────────────────────────────────────────┘
```

### Key Infrastructure Services

| Service | Purpose | Technology |
|:---|:---|:---|
| API Gateway | Single entry point for all REST calls | Kong / custom |
| Message Bus | Async inter-service communication | RabbitMQ |
| Configuration Store | Persistent state for all services | PostgreSQL |
| Document Store | Unstructured data (templates, images) | MongoDB |
| Search Engine | Full-text search across inventory | Elasticsearch |
| Cache | High-speed data access | Redis |
| Service Registry | Service discovery and health | Consul |
| Task Manager | Async task orchestration and status | Custom |

### Service Decomposition

The monolithic network management functions are decomposed into microservices:

$$\text{NMS}_{\text{monolithic}} \rightarrow \{S_1, S_2, \ldots, S_n\}$$

Where each service $S_i$ owns a bounded context:

- **Inventory Service:** Device discovery, lifecycle, credentials
- **Site Service:** Network hierarchy (areas, sites, buildings, floors)
- **SWIM Service:** Image repository, compliance, distribution
- **Template Service:** Day-N template storage, rendering, versioning
- **Policy Service:** SGT policy, application policy, virtual networks
- **Assurance Service:** Health calculation, issue detection, AI analytics
- **PnP Service:** Plug-and-Play device onboarding
- **Command Runner:** On-demand CLI execution
- **NDP Service:** Telemetry collection, storage, processing

### Cluster Architecture

A 3-node cluster provides high availability:

```
Node 1 (Leader)         Node 2 (Member)        Node 3 (Member)
┌──────────────┐       ┌──────────────┐       ┌──────────────┐
│  App Services│       │  App Services│       │  App Services│
│  Platform    │       │  Platform    │       │  Platform    │
│  K8s Master  │  ←──→ │  K8s Worker  │  ←──→ │  K8s Worker  │
│  etcd Leader │       │  etcd Member │       │  etcd Member │
│  DB Primary  │       │  DB Replica  │       │  DB Replica  │
└──────────────┘       └──────────────┘       └──────────────┘
        ↕                      ↕                      ↕
   ─────────── Cluster VIP (Virtual IP) ───────────────
```

Cluster properties:
- **Leader election:** etcd-based Raft consensus
- **Database replication:** Synchronous for PostgreSQL, async for MongoDB
- **Service scheduling:** Kubernetes distributes pods across nodes
- **Failure tolerance:** Survives single-node failure (quorum = 2 of 3)
- **Split-brain prevention:** etcd requires majority quorum

The cluster availability model:

$$A_{\text{cluster}} = 1 - (1 - A_{\text{node}})^2$$

For a 3-node cluster with individual node availability of 99.9%:

$$A_{\text{cluster}} = 1 - (0.001)^2 = 1 - 0.000001 = 99.9999\%$$

This assumes independent failures, which is optimistic — correlated failures (power, network) reduce actual availability.

---

## 3. Southbound Communication Model

### Protocol Selection Logic

Catalyst Center selects the southbound protocol based on device capabilities and operation type:

| Operation | Protocol | Fallback |
|:---|:---|:---|
| Configuration push | NETCONF/YANG | SSH/CLI |
| Configuration read | NETCONF/YANG | SSH/CLI (show commands) |
| Monitoring | Streaming telemetry (gRPC) | SNMP polling |
| Discovery | SNMP + CDP/LLDP | Ping sweep |
| Image transfer | SCP/SFTP | TFTP |
| PnP onboarding | HTTPS (PnP protocol) | N/A |

### NETCONF/YANG Model-Driven Configuration

For supported devices, Catalyst Center uses YANG models to generate NETCONF RPCs:

```
Intent: "VLAN 100 on interface Gi1/0/1"
     ↓ (Policy Service)
YANG Model: ietf-interfaces + cisco-ios-xe-native
     ↓ (Template Renderer)
NETCONF RPC:
<edit-config>
  <target><running/></target>
  <config>
    <native xmlns="http://cisco.com/ns/yang/Cisco-IOS-XE-native">
      <interface>
        <GigabitEthernet>
          <name>1/0/1</name>
          <switchport>
            <access>
              <vlan><vlan>100</vlan></vlan>
            </access>
          </switchport>
        </GigabitEthernet>
      </interface>
    </native>
  </config>
</edit-config>
```

### CLI Fallback and Template Rendering

When NETCONF is unavailable, Catalyst Center renders Jinja2 templates into CLI commands and pushes them via SSH:

```
Template (Jinja2) + Variables (from intent) → CLI commands → SSH session → device
```

The rendering pipeline:

$$\text{CLI}_{\text{output}} = \text{render}(\text{template}, \text{variables}_{\text{global}} \cup \text{variables}_{\text{site}} \cup \text{variables}_{\text{device}})$$

Variable resolution order (last wins):
1. Global variables (defined at template level)
2. Site variables (inherited from network hierarchy)
3. Device variables (bound during provisioning)

### Telemetry Collection

Catalyst Center collects telemetry through multiple channels:

```
Device → gRPC Dial-Out (streaming telemetry)  → NDP Collector → Elasticsearch
Device → SNMP Traps                           → NDP Collector → Elasticsearch
Device → Syslog                               → NDP Collector → Elasticsearch
Device → NetFlow/IPFIX                        → NDP Collector → Elasticsearch
Device → SNMP Polling (5-min intervals)       → NDP Collector → Elasticsearch
```

Data volume estimation for a network of $N$ devices, each with $I$ interfaces:

$$\text{Data}_{\text{daily}} = N \times I \times S \times \frac{86400}{P}$$

Where $S$ is the average sample size in bytes and $P$ is the polling/streaming interval in seconds.

For 1,000 devices with 48 interfaces each, 200-byte samples at 30-second intervals:

$$\text{Data}_{\text{daily}} = 1000 \times 48 \times 200 \times \frac{86400}{30} = 27.6 \text{ GB/day}$$

This explains the large storage requirements (multi-TB) on Catalyst Center appliances.

---

## 4. Network Data Platform (NDP)

### Data Pipeline Architecture

NDP is the telemetry backbone of Catalyst Center. It implements a streaming data pipeline:

```
Sources           Collection         Processing          Storage            Consumption
┌─────────┐      ┌──────────┐      ┌──────────┐      ┌──────────┐      ┌──────────┐
│ Devices  │ ──→  │Collectors│ ──→  │ Enrichment│ ──→  │Time-Series│ ──→  │Dashboards│
│ Clients  │      │          │      │ Correlation│     │ DB       │      │  APIs    │
│ Apps     │      │          │      │ Aggregation│     │          │      │  AI/ML   │
└─────────┘      └──────────┘      └──────────┘      └──────────┘      └──────────┘
```

### Data Enrichment

Raw telemetry is enriched with context from the inventory and topology databases:

```
Raw: { "device": "10.0.0.1", "interface": "Gi1/0/1", "in_octets": 123456789 }
     ↓ enrichment
Enriched: {
  "device": "10.0.0.1",
  "device_name": "Floor2-SW1",
  "site": "HQ/Building-A/Floor-2",
  "interface": "Gi1/0/1",
  "interface_name": "User-Port-1",
  "connected_client": "aa:bb:cc:dd:ee:ff",
  "client_user": "jsmith",
  "sgt": "Employees",
  "in_octets": 123456789,
  "in_rate_bps": 850000,
  "utilization_pct": 0.085
}
```

Enrichment transforms device-centric data into business-centric data, enabling queries like "show me all interfaces serving the Sales team with utilization above 80%."

### Retention and Tiering

| Data Type | Hot Storage | Warm Storage | Cold Storage |
|:---|:---:|:---:|:---:|
| Health scores | 7 days (1-min) | 30 days (5-min avg) | 90 days (hourly avg) |
| Client sessions | 14 days | 30 days | N/A |
| Device metrics | 7 days (5-min) | 30 days (15-min avg) | 90 days (hourly avg) |
| Syslog/events | 14 days | 30 days | N/A |
| Application metrics | 7 days | 30 days | N/A |

Hot storage uses full-resolution data for real-time dashboards. Warm and cold tiers use aggregated data (averages, percentiles) to reduce storage while preserving trends.

---

## 5. Health Score Mathematics

### Device Health Score

Each device receives a health score from 0 to 10, computed as a weighted composite:

$$H_{\text{device}} = \sum_{i=1}^{n} w_i \times h_i$$

Where $h_i$ is the individual metric health (0-10) and $w_i$ is the weight ($\sum w_i = 1$).

Metric health mapping (example for CPU utilization):

$$h_{\text{cpu}} = \begin{cases} 10 & \text{if } \text{CPU} < 60\% \\ 10 - \frac{10 \times (\text{CPU} - 60)}{40} & \text{if } 60\% \leq \text{CPU} \leq 100\% \\ 0 & \text{if CPU unreachable} \end{cases}$$

This is a piecewise linear function that maps CPU utilization to a 0-10 health score, with full marks below 60% and linear degradation above.

### Client Health Score

Client health is more complex, incorporating multiple phases:

$$H_{\text{client}} = \min(H_{\text{onboard}}, H_{\text{connect}}, H_{\text{app}})$$

The minimum function means any single poor phase drags down the overall score. This is intentional — a client with excellent RF signal but failed DHCP is not healthy.

Onboarding health decomposes further:

$$H_{\text{onboard}} = H_{\text{assoc}} \times H_{\text{auth}} \times H_{\text{dhcp}} \times \frac{1}{\text{max\_score}^3}$$

Where each sub-component is 0-10 and the result is normalized back to 0-10.

### Network Health Aggregation

Site-level health aggregates individual device scores:

$$H_{\text{site}} = \frac{\sum_{d \in \text{devices}} H_d \times \text{criticality}(d)}{\sum_{d \in \text{devices}} \text{criticality}(d)}$$

Where criticality weights core switches and WLCs higher than access switches.

The overall network health is the weighted average of all site health scores:

$$H_{\text{network}} = \frac{\sum_{s \in \text{sites}} H_s \times |\text{devices}(s)|}{\sum_{s \in \text{sites}} |\text{devices}(s)|}$$

---

## 6. AI/ML Analytics Architecture

### Baseline Learning

The AI engine learns behavioral baselines using time-series decomposition:

$$X(t) = T(t) + S(t) + R(t)$$

Where:
- $T(t)$ = trend component (long-term growth/decline)
- $S(t)$ = seasonal component (time-of-day, day-of-week patterns)
- $R(t)$ = residual (noise and anomalies)

The baseline is $B(t) = T(t) + S(t)$, and an anomaly is detected when:

$$|X(t) - B(t)| > k \times \sigma_R$$

Where $\sigma_R$ is the standard deviation of the residual and $k$ is the sensitivity parameter (typically 2-3).

### Anomaly Detection Pipeline

```
Time Series Data → Seasonal Decomposition → Baseline Extraction → Residual Analysis
                                                                        ↓
                                                              Anomaly Candidates
                                                                        ↓
                                                              Correlation Engine
                                                                        ↓
                                                              Root Cause Ranking
                                                                        ↓
                                                              AI-Driven Issues
```

### Peer Group Comparison

Devices are clustered into peer groups based on:
- Device type and model
- Role (core, distribution, access)
- Site characteristics (size, client density)
- Network topology position

Within a peer group, outlier detection identifies devices performing significantly worse than peers:

$$\text{outlier}(d) = \frac{H_d - \mu_{\text{peer}}}{\sigma_{\text{peer}}} < -z_{\alpha}$$

Where $z_{\alpha}$ is the z-score threshold (typically -2, representing the bottom 2.5% of the peer group).

### Predictive Analytics

The cloud-connected AI engine (Cisco AI Network Analytics) uses regression models to predict future capacity:

$$\hat{X}(t + \Delta) = T(t + \Delta) + S(t + \Delta)$$

Predictions enable alerts like:
- "AP coverage will be insufficient for projected client growth in 90 days"
- "WAN link utilization will exceed 80% within 30 days at current growth rate"
- "Switch CPU will reach critical levels during peak hours within 14 days"

---

## 7. Policy Enforcement Chain

### Group-Based Policy Flow

The full policy enforcement chain from Catalyst Center to data plane:

```
Catalyst Center                    ISE                         Network Device
┌──────────┐                 ┌──────────┐                 ┌──────────────┐
│ Define   │   pxGrid sync   │ SGT      │   RADIUS        │ Classify     │
│ SGTs and │ ──────────────→ │ Database │ ──────────────→ │ (assign SGT) │
│ Policies │                 │ + SGACLs │                 │              │
└──────────┘                 └──────────┘                 │ Enforce      │
                                                          │ (apply SGACL)│
                                                          └──────────────┘
```

Step-by-step:

1. Admin defines SGTs and access contracts in Catalyst Center
2. Catalyst Center syncs SGT definitions to ISE via pxGrid
3. ISE creates corresponding SGACLs and authorization profiles
4. User/device authenticates to ISE (RADIUS/802.1X)
5. ISE assigns SGT in RADIUS response
6. Ingress switch tags traffic with SGT (in Ethernet CMD header or VXLAN)
7. Egress switch looks up SGACL for source-SGT → destination-SGT pair
8. SGACL is applied in hardware (TCAM)

### SGT Propagation Methods

| Method | Mechanism | Scale | Latency |
|:---|:---|:---:|:---:|
| Inline tagging (SGT/CMD) | Ethernet header (6 bytes) | Unlimited | Wire speed |
| SXP (SGT Exchange Protocol) | TCP-based control plane | ~10,000 bindings | Seconds |
| VXLAN-GPO | VXLAN Group Policy Option (16 bits) | Fabric-wide | Wire speed |
| REST API (pxGrid) | ISE → device push | Controller-limited | Seconds |

### SGACL TCAM Programming

Each SGACL entry consumes TCAM space:

$$\text{TCAM}_{\text{SGACL}} = \text{SGT}_{\text{source}} \times \text{SGT}_{\text{dest}} \times \text{ACEs}_{\text{avg}}$$

For 50 source SGTs, 50 destination SGTs, and an average of 5 ACEs per contract:

$$\text{TCAM}_{\text{SGACL}} = 50 \times 50 \times 5 = 12,500 \text{ entries}$$

This can strain TCAM on access switches. Optimization strategies:
- Reduce SGT count (consolidate similar groups)
- Use coarse contracts (fewer ACEs)
- Use the "default" policy for majority of cells in the matrix
- Deploy enforcement only on critical boundaries

---

## 8. SD-Access Integration

### SD-Access as the Data Plane

Catalyst Center is the controller for Cisco SD-Access, which uses VXLAN + LISP + TrustSec as the data plane:

```
Catalyst Center (Controller Plane)
         ↓ provisions
┌────────────────────────────────────┐
│          SD-Access Fabric          │
│                                    │
│  Control Plane:  LISP Map-Server  │
│  Data Plane:     VXLAN tunnels    │
│  Policy Plane:   SGT (TrustSec)  │
│                                    │
│  ┌────────┐  ┌────────┐          │
│  │ Edge   │──│ Border │──→ WAN   │
│  │ Node   │  │ Node   │          │
│  └────────┘  └────────┘          │
│       ↑                           │
│  ┌────────┐                       │
│  │ Control│                       │
│  │ Plane  │                       │
│  │ Node   │                       │
│  └────────┘                       │
└────────────────────────────────────┘
```

### Fabric Roles

| Role | Function | Typical Device |
|:---|:---|:---|
| Control Plane Node | LISP Map-Server/Map-Resolver | Cat 9500, 9800 |
| Edge Node | Endpoint attachment, VXLAN encap/decap | Cat 9300, 9400 |
| Border Node | Fabric-to-external connectivity | Cat 9500, ISR 4000 |
| Intermediate Node | Transit (no fabric role, just IP forwarding) | Any L3 switch |
| Wireless Controller | AP management, fabric wireless | Cat 9800 |

### VXLAN Encapsulation Overhead

SD-Access uses VXLAN-GPO (Group Policy Option) for data plane:

$$\text{Overhead}_{\text{VXLAN}} = \text{Outer Ethernet}(14) + \text{Outer IP}(20) + \text{UDP}(8) + \text{VXLAN}(8) + \text{GPO}(4) = 54 \text{ bytes}$$

This reduces the effective MTU:

$$\text{MTU}_{\text{effective}} = \text{MTU}_{\text{physical}} - \text{Overhead}_{\text{VXLAN}}$$

For a standard 1500-byte MTU: $\text{MTU}_{\text{effective}} = 1500 - 54 = 1446$ bytes.

The fabric underlay should use jumbo frames (MTU 9100+) to avoid fragmentation:

$$\text{MTU}_{\text{underlay}} \geq \text{MTU}_{\text{overlay}} + \text{Overhead}_{\text{VXLAN}} = 1500 + 54 = 1554$$

---

## 9. API Architecture and Automation Patterns

### API Gateway Design

All API requests flow through a single gateway:

```
Client → HTTPS → API Gateway (Kong) → Auth Service → Target Microservice → Response
                      ↓
              Rate Limiting
              Token Validation
              Request Logging
              API Versioning
```

### Authentication Model

Catalyst Center uses token-based authentication:

```
1. POST /dna/system/api/v1/auth/token (Basic Auth)
2. Receive JWT token (60-minute TTL)
3. Use token in X-Auth-Token header for all subsequent requests
4. Token refresh: request new token before expiry
```

Token expiry management for automation scripts:

$$T_{\text{refresh}} = T_{\text{issued}} + T_{\text{TTL}} - T_{\text{buffer}}$$

Where $T_{\text{buffer}}$ is a safety margin (typically 5 minutes) to avoid mid-request expiry.

### Asynchronous Task Pattern

Most write operations are asynchronous:

```
1. POST /dna/intent/api/v1/... (create/update/delete)
2. Response: { "executionId": "uuid", "executionStatusUrl": "/api/v1/task/uuid" }
3. Poll: GET /dna/intent/api/v1/task/{taskId}
4. Response: { "progress": "...", "isError": false, "endTime": ... }
5. On completion: GET /dna/intent/api/v1/file/{fileId} (if output file)
```

Polling strategy with exponential backoff:

$$T_{\text{wait}}(n) = \min(T_{\text{initial}} \times 2^n, T_{\text{max}})$$

Where $n$ is the attempt number, $T_{\text{initial}} = 1$ second, and $T_{\text{max}} = 30$ seconds.

### Rate Limiting

The API gateway enforces rate limits per token:

| API Category | Rate Limit | Burst |
|:---|:---:|:---:|
| Intent APIs | 5 req/sec | 10 |
| Command Runner | 1 req/sec | 5 |
| Event Subscription | 5 req/sec | 10 |
| Auth Token | 1 req/10sec | 1 |

For bulk automation (e.g., provisioning 500 devices), the effective throughput:

$$T_{\text{total}} = \frac{N}{\text{rate\_limit}} + N \times T_{\text{task\_completion}}$$

With 500 devices at 5 req/sec and 30-second average task completion:

$$T_{\text{total}} = \frac{500}{5} + 500 \times 30 = 100 + 15000 = 15100 \text{ seconds} \approx 4.2 \text{ hours}$$

This underscores why bulk provisioning requires careful scheduling and parallelization strategies.

---

## 10. Scaling and Capacity Planning

### Device Scale Limits

| Deployment | Nodes | Managed Devices | Assurance Clients | Concurrent API Sessions |
|:---|:---:|:---:|:---:|:---:|
| Small (single node) | 1 | 1,000 | 5,000 | 25 |
| Medium (single node) | 1 | 2,500 | 12,500 | 50 |
| Large (3-node cluster) | 3 | 10,000 | 50,000 | 100 |

### Storage Capacity

Storage consumption grows with device count, client count, and retention period:

$$\text{Storage}_{\text{total}} = \text{Storage}_{\text{base}} + (D \times S_d + C \times S_c) \times R$$

Where:
- $D$ = device count
- $S_d$ = storage per device per day (~50 MB)
- $C$ = client count
- $S_c$ = storage per client per day (~5 MB)
- $R$ = retention period in days

For 5,000 devices and 25,000 clients with 30-day retention:

$$\text{Storage} = \text{base} + (5000 \times 50 + 25000 \times 5) \times 30 = \text{base} + 11.25 \text{ TB}$$

### Network Bandwidth Requirements

| Traffic Type | Bandwidth Estimate |
|:---|:---|
| SNMP polling (5-min, 1000 devices) | ~5 Mbps sustained |
| Streaming telemetry (30-sec, 1000 devices) | ~50 Mbps sustained |
| Image distribution (10 devices concurrent) | ~1 Gbps burst |
| PnP onboarding (batch of 100) | ~200 Mbps burst |
| API traffic | ~10 Mbps sustained |

Management network should be provisioned with dedicated bandwidth, ideally on a separate VLAN/VRF from production traffic.

### High Availability Recovery

| Failure Scenario | Detection Time | Recovery Time | Data Loss |
|:---|:---:|:---:|:---:|
| Single node (in cluster) | 30 seconds | 2-5 minutes | None (replicated) |
| Database corruption | Immediate | 30-60 minutes (restore) | Last backup interval |
| Full cluster failure | Immediate | 2-4 hours (rebuild) | Since last backup |
| Network partition (split-brain) | 30 seconds | Auto-resolves (quorum) | None |

---

## Prerequisites

- networking-fundamentals, vlan, vxlan, lisp, radius, tacacs, snmp, netconf, restconf, rest-apis, cisco-ise, qos

## Complexity

| Operation | Time | Space |
|-----------|------|-------|
| Device discovery (SNMP + CDP) | O(n) | O(n) |
| Health score computation | O(n * m) per interval | O(n * m) |
| SGACL matrix programming | O(s^2 * a) | O(s^2 * a) TCAM |
| Template rendering | O(t * v) | O(t) |
| Path trace computation | O(h) per path | O(h) |
| AI baseline learning | O(n * d) | O(n * w) |

Where $n$ = devices, $m$ = metrics per device, $s$ = SGT count, $a$ = ACEs per contract, $t$ = template size, $v$ = variables, $h$ = hops, $d$ = training data points, $w$ = time window.

---

*Catalyst Center represents Cisco's bet that network management will shift from device-by-device configuration to declarative intent. The platform succeeds when an engineer can express "isolate IoT devices from corporate servers" and have it translate into VXLAN VNIs, SGT tags, SGACLs, and ISE authorization policies across thousands of switches — without writing a single CLI command.*
