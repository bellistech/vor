# The Engineering of Cisco ISE — Architecture, Policy Engine, and TrustSec Propagation

> *ISE is the brain of Cisco's zero-trust architecture: it evaluates identity, posture, and context for every network connection, then pushes enforcement to the infrastructure in real time via RADIUS and TrustSec.*

---

## 1. ISE Architecture Internals

### Node Communication

```
+-------------------+         +-------------------+
|   PAN (Primary)   |<------->|  PAN (Secondary)  |
|   Configuration   | Sync    |   Hot standby     |
|   Policy store    |         |   Auto-promote    |
+-------------------+         +-------------------+
        |                              |
        | Replication                  |
        v                              v
+----------+  +----------+  +----------+  +----------+
|  PSN 1   |  |  PSN 2   |  |  MnT 1   |  |  MnT 2   |
| (RADIUS) |  | (RADIUS) |  | (Logging)|  | (Standby)|
| (pxGrid) |  | (Portal) |  | (Reports)|  |          |
+----------+  +----------+  +----------+  +----------+
```

### Internal Services

Each ISE node runs multiple internal services:

| Service | Port | Purpose |
|---------|------|---------|
| Admin Portal | 443 | Web-based management interface |
| RADIUS | 1812/1813 | Authentication/accounting |
| TACACS+ | 49 | Device administration |
| ERS API | 9060 | External REST API |
| Open API | 443 (/api/) | Modern REST API (ISE 3.1+) |
| pxGrid | 8910 | Context sharing |
| Guest Portal | 8443 (configurable) | Guest/sponsor/BYOD portals |
| Profiler (DHCP) | 67 | DHCP probe listener |
| Profiler (SNMP) | 161 | SNMP queries to switches |
| SXP | 64999 | SGT Exchange Protocol |
| Internal DB | 5432 | PostgreSQL (policy store) |
| Inter-node | 1521, 8905 | Replication, messaging |

### Data Replication Model

ISE uses a primary-secondary replication model:

$$T_{sync} = \frac{S_{policy\_data}}{BW_{inter\_node}} + T_{commit}$$

- **PAN to PAN:** Full database replication (active-passive)
- **PAN to PSN:** Policy and configuration push (one-way)
- **PSN to MnT:** Log and event data (Syslog over TCP/UDP)
- **Sync trigger:** Manual (policy push) or automatic (configuration change)

### High Availability

| Component | HA Mechanism | Failover Time |
|-----------|-------------|---------------|
| PAN | Primary/Secondary auto-promote | 5-10 minutes |
| PSN | Multiple PSNs + RADIUS load balancing | Immediate (next request) |
| MnT | Primary/Secondary log target | Automatic redirect |
| pxGrid | Multiple pxGrid controllers | Client reconnect |

PAN failover is **not instantaneous**: the secondary PAN must promote itself, which involves database role change. During PAN failover, no configuration changes can be made, but PSNs continue processing RADIUS/TACACS+ with cached policy.

---

## 2. Policy Evaluation Engine

### Evaluation Pipeline

```
RADIUS Request Received at PSN
         |
         v
+------------------+
| Policy Set Match |  Match on conditions: NAS type, location, device type
+------------------+
         |
         v
+------------------------+
| Authentication Policy  |  Select identity source, EAP method
| Rule Evaluation        |  Top-down, first match
+------------------------+
         |
    Auth Result
    (Pass/Fail/Error)
         |
         v
+------------------------+
| Authorization Policy   |  Match on identity group, posture, profile, time
| Rule Evaluation        |  Top-down, first match
+------------------------+
         |
    Authz Profile
    (VLAN, SGT, dACL, etc.)
         |
         v
+-------------------+
| Build RADIUS      |  Construct Access-Accept with attributes
| Access-Accept     |
+-------------------+
```

### Condition Evaluation

Each policy rule consists of conditions combined with AND/OR logic:

$$Rule_{match} = \bigwedge_{i=1}^{n} C_i \quad \text{or} \quad \bigvee_{i=1}^{n} C_i$$

Condition types:

| Category | Attributes | Example |
|----------|-----------|---------|
| RADIUS | NAS-IP, NAS-Port-Type, Called-Station-Id | NAS-Port-Type == Ethernet |
| Identity | AD-Group, Identity-Group | AD-Group == Domain Admins |
| Device | Device-Type, Location, IETF-RADIUS-Name | Location == Building-A |
| Endpoint | EndpointProfile, EndpointIdentityGroup | Profile == Cisco-IP-Phone |
| Posture | PostureStatus | PostureStatus == Compliant |
| Time | TimeAndDate | Within business hours |
| Custom | Dictionary attributes | Any RADIUS or vendor-specific attribute |

### Rule Evaluation Order

Within a policy set, rules are evaluated **strictly top-down**:

$$Result = \text{first } R_i \text{ where } Conditions(R_i) = True$$

If no rule matches, the **default rule** at the bottom applies (typically DenyAccess for authorization).

### Rule Evaluation Performance

$$T_{evaluation} = \sum_{i=1}^{K} T_{condition\_check_i}$$

Where $K$ is the number of rules checked before a match. For a policy set with $N$ rules and an endpoint that matches rule $K$:

- **Best case:** $K = 1$ (first rule matches)
- **Worst case:** $K = N$ (last rule or default)
- **Optimization:** Place most-matched rules at the top

ISE caches frequently-used condition results (AD group lookups, endpoint profiles) to reduce evaluation time.

---

## 3. Profiling Probe Analysis

### Probe Data Quality

Each probe provides different signals with varying reliability:

| Probe | Data Quality | CPU Impact on ISE | Accuracy |
|-------|-------------|-------------------|----------|
| RADIUS | Medium (limited attributes) | Low | Medium |
| DHCP | High (OS fingerprint, hostname) | Low | High |
| HTTP | High (User-Agent = OS/browser) | Medium | High |
| SNMP | High (CDP/LLDP = device model) | High (polling) | Very High |
| NetFlow | Low (traffic patterns only) | Medium | Low |
| DNS | Low (hostname only) | Low | Low |
| NMAP | Very High (OS fingerprint, ports) | Very High | Very High |
| AD | Medium (machine type, domain) | Medium | Medium |

### Certainty Factor Model

ISE profiling uses a weighted scoring system:

$$CF_{endpoint} = \sum_{p=1}^{P} \sum_{c=1}^{C_p} W_{p,c} \times M_{p,c}$$

Where:
- $P$ = number of probes that returned data
- $C_p$ = number of conditions evaluated from probe $p$
- $W_{p,c}$ = weight of condition $c$ from probe $p$
- $M_{p,c}$ = 1 if condition matches, 0 otherwise

An endpoint is classified as profile $X$ if:
1. $CF_{endpoint} \geq CF_{minimum}(X)$ (default: 10)
2. $CF_X > CF_Y$ for all other profiles $Y$

### Profiling Hierarchy

ISE profiles are organized in a tree:

```
All Endpoints
├── Cisco-Device
│   ├── Cisco-IP-Phone
│   │   ├── Cisco-IP-Phone-7900
│   │   ├── Cisco-IP-Phone-8800
│   │   └── Cisco-IP-Phone-DX
│   ├── Cisco-Switch
│   └── Cisco-AP
├── Microsoft-Workstation
│   ├── Windows10-Workstation
│   └── Windows11-Workstation
├── Apple-Device
│   ├── Apple-MacBook
│   ├── Apple-iPhone
│   └── Apple-iPad
└── Unknown
```

An endpoint first matches the parent profile, then ISE attempts to match more specific child profiles as additional probe data arrives.

### Probe Deployment Recommendations

| Environment | Minimum Probes | Recommended Probes |
|-------------|---------------|-------------------|
| Small office | RADIUS | RADIUS + DHCP + HTTP |
| Enterprise wired | RADIUS + DHCP | RADIUS + DHCP + SNMP + HTTP |
| Enterprise wireless | RADIUS + DHCP + HTTP | RADIUS + DHCP + HTTP + SNMP |
| IoT-heavy | RADIUS + DHCP + SNMP | All probes including NMAP (selective) |

---

## 4. TrustSec SGT Propagation Methods

### Method Comparison

```
Method 1: Inline Tagging (CMD - Cisco Meta Data)
+------+--------+--------+--------+-------+
| DA   | SA     | CMD    | 802.1Q | Data  |  (SGT in Ethernet header)
+------+--------+--------+--------+-------+

Method 2: SXP (SGT Exchange Protocol)
PSN ---[TCP 64999]---> Switch
      IP-SGT binding table

Method 3: Static Assignment
Switch CLI: cts role-based sgt-map 10.1.1.0/24 sgt 5

Method 4: RADIUS (Dynamic Assignment)
ISE ---[RADIUS Accept]---> Switch
      cisco-av-pair: cts:security-group-tag=0005-00
```

### Inline Tagging (CMD)

Inline tagging embeds the SGT in the Ethernet frame using Cisco Meta Data (CMD):

```
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|  Version  | Length    |      Option Type       |    SGT Value |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

- **Requires:** Hardware support (Catalyst 3K+, Nexus 7K/9K, ASR 1K)
- **Requires:** MACsec or SGT-capable interfaces
- **Advantage:** No control-plane overhead; SGT travels with the packet
- **Advantage:** Works across L2 and L3 boundaries
- **Limitation:** All devices in the path must support CMD

### SXP Propagation

SXP is a control-plane protocol that propagates IP-to-SGT bindings:

$$Bindings_{SXP} = \{(IP_1, SGT_1), (IP_2, SGT_2), \ldots, (IP_n, SGT_n)\}$$

SXP roles:
- **Speaker:** Sends IP-SGT bindings (typically access switch or ISE PSN)
- **Listener:** Receives IP-SGT bindings (typically distribution/core switch or firewall)
- **Both:** Can send and receive

SXP scaling:

$$Memory_{SXP} \approx N_{bindings} \times 20 \text{ bytes}$$

For 50,000 endpoints:
$$Memory_{SXP} \approx 50000 \times 20 = 1 \text{ MB}$$

SXP is lightweight but has limitations:
- TCP-based, requires persistent connections
- No built-in redundancy (use multiple peers)
- Propagation delay (seconds, not real-time)

### SGT Propagation Decision Matrix

| Criterion | Inline | SXP | Static | RADIUS |
|-----------|--------|-----|--------|--------|
| Hardware requirement | High (CMD-capable) | Low (TCP stack) | None | RADIUS client |
| Scalability | Excellent | Good (50K+) | Poor (manual) | Excellent |
| Dynamic updates | Instant (per-packet) | Seconds | Manual | Per-authentication |
| L3 boundary crossing | Yes (with CMD) | Yes (TCP tunnel) | Yes (per-subnet) | Yes (per-session) |
| Recommended for | Modern campus | Legacy devices, firewalls | Static infra (servers) | Access layer |

### End-to-End SGT Flow

```
Endpoint --> Access Switch --> Distribution --> Core --> Firewall --> Server
   |              |                 |            |          |            |
   | [802.1X]     | [RADIUS]        |            |          |            |
   |              | [SGT=5]         |            |          |            |
   |              |                 |            |          |            |
   |         [Inline tag]     [Inline tag]  [Inline]  [SXP or     [Static
   |          SGT=5            SGT=5        SGT=5     Inline]      SGT=10]
   |                                                   |
   |                                              [SGACL: SGT 5->10
   |                                               permit tcp 443
   |                                               deny ip]
```

---

## 5. pxGrid Pub/Sub Architecture

### pxGrid 2.0 (WebSocket-Based)

```
+------------------+
|  pxGrid          |
|  Controller      |<-------- WebSocket connections
|  (ISE PSN)       |
+------------------+
    |          |          |
    v          v          v
+--------+ +--------+ +--------+
| FMC    | | SIEM   | | DNA-C  |
| (Sub)  | | (Sub)  | | (Pub/  |
|        | |        | |  Sub)  |
+--------+ +--------+ +--------+
```

### pxGrid Topics and Operations

| Topic | Publisher | Subscribers | Data |
|-------|----------|-------------|------|
| SessionDirectory | ISE | FMC, SIEM, Stealthwatch | User-IP-MAC-SGT mappings |
| TrustSecConfiguration | ISE | Switches, FMC | SGT names, SGACL policies |
| EndpointProfile | ISE | DNA Center, CMDB | Device classification |
| AdaptiveNetworkControl | ISE, Partners | ISE | Quarantine/unquarantine actions |
| MDMCompliance | MDM vendors | ISE | Device compliance status |
| ThreatIntelligence | Threat platforms | ISE | IOC data for rapid containment |

### pxGrid Message Flow

```
1. Client registers with pxGrid controller (mutual TLS)
2. Client subscribes to topic (e.g., SessionDirectory)
3. ISE publishes event (e.g., new session authenticated)
4. pxGrid controller fans out to all subscribers
5. Subscribers receive JSON payload via WebSocket
```

### pxGrid Message Example

```json
{
  "operation": "CREATE",
  "sessions": [
    {
      "timestamp": "2026-04-05T14:30:00Z",
      "userName": "jsmith@corp.example.com",
      "callingStationId": "AA:BB:CC:DD:EE:FF",
      "framedIpAddress": "10.1.20.100",
      "nasIpAddress": "10.1.1.1",
      "nasPortId": "GigabitEthernet1/0/5",
      "securityGroup": "Employees",
      "endpointProfile": "Windows10-Workstation",
      "postureStatus": "Compliant"
    }
  ]
}
```

### pxGrid Scaling

| Parameter | Limit (ISE 3.x) |
|-----------|-----------------|
| pxGrid controllers | 2 (HA pair) |
| pxGrid clients | 50+ per controller |
| Session topic rate | 10,000+ sessions/min |
| WebSocket connections | Limited by ISE node resources |

---

## 6. ISE Scalability

### Maximum Deployment Limits (ISE 3.x)

| Parameter | Small (SNS-3615) | Medium (SNS-3655) | Large (SNS-3695) |
|-----------|-------------------|---------------------|-------------------|
| Active sessions (per PSN) | 10,000 | 25,000 | 50,000 |
| Active sessions (deployment) | 100,000 | 500,000 | 2,000,000 |
| Profiled endpoints | 50,000 | 250,000 | 1,000,000 |
| Internal users | 300,000 | 300,000 | 300,000 |
| Network devices | 5,000 | 10,000 | 50,000 |
| PSN nodes | 5 | 20 | 50 |
| Total nodes | 8 | 30 | 50 |
| RADIUS requests/sec (per PSN) | 5,000 | 10,000 | 20,000 |

### RADIUS Performance

$$Throughput_{deployment} = N_{PSN} \times RPS_{per\_PSN}$$

For a deployment with 10 PSNs (medium) and 10,000 RPS per PSN:
$$Throughput = 10 \times 10000 = 100,000 \text{ RADIUS requests/second}$$

### Load Balancing

PSNs should be deployed behind a load balancer for RADIUS:

$$Load_{per\_PSN} = \frac{Total\_RPS}{N_{PSN} \times LB\_efficiency}$$

Where $LB\_efficiency \approx 0.9$ (accounting for uneven distribution).

Recommended load balancers:
- F5 BIG-IP with RADIUS persistence (Calling-Station-Id)
- Cisco SLB (IOS-based)
- ISE built-in RADIUS server sequence (no true LB, but failover)

---

## 7. RADIUS Change of Authorization (CoA)

### CoA Message Types (RFC 5176)

| CoA Type | Code | Purpose |
|----------|------|---------|
| CoA-Request | 43 | Change session attributes (re-authorize) |
| Disconnect-Request | 40 | Terminate session immediately |
| CoA-ACK | 44 | NAD confirms CoA applied |
| CoA-NAK | 45 | NAD rejects CoA |

### CoA Use Cases

```
ISE ----[CoA-Request]----> Switch
         |
    CoA Actions:
    1. Reauthenticate (re-run 802.1X)
    2. Port bounce (link down/up)
    3. Port shutdown
    4. Change VLAN (via new authorization)
    5. Apply new dACL
    6. Assign new SGT
```

### CoA Flow (Posture Change)

```
Time    Event                         CoA Action
----    -----                         ----------
T0      User authenticates            ISE: Accept + Redirect to posture portal
T1      User installs posture agent   Agent checks endpoint compliance
T2      Agent reports compliant       ISE receives posture status
T3      ISE sends CoA-Request         Switch: reauthenticate the port
T4      Switch re-auths the port      ISE: Access-Accept with full access
T5      User gets full network access dACL/VLAN/SGT updated
```

### CoA Timing

$$T_{CoA} = T_{ISE\_decision} + T_{CoA\_delivery} + T_{NAD\_processing}$$

Typical CoA latency: **1-5 seconds** from trigger to enforcement.

For posture-based CoA:
$$T_{posture\_CoA} = T_{agent\_check} + T_{report} + T_{ISE\_eval} + T_{CoA}$$
$$T_{posture\_CoA} \approx 10s + 1s + 1s + 3s = 15s \text{ (typical)}$$

---

## 8. Posture Remediation Flow

### Remediation Architecture

```
Endpoint             ISE                  Remediation Server
    |                  |                          |
    | [Non-compliant]  |                          |
    |                  |                          |
    |<-- Redirect to --|                          |
    |    remediation   |                          |
    |    portal        |                          |
    |                  |                          |
    |--- Download -----|------------------------->|
    |    updates       |                          |
    |                  |                          |
    | [Install patches,|                          |
    |  update AV, etc.]|                          |
    |                  |                          |
    |--- Re-check ---->|                          |
    |    posture       |                          |
    |                  |                          |
    |<-- Compliant ----|                          |
    |                  |--- CoA (full access) --->| Switch
```

### Remediation Actions

| Condition | Remediation | Mechanism |
|-----------|-------------|-----------|
| AV out of date | Update AV definitions | WSUS/SCCM URL allowed in quarantine ACL |
| Missing patches | Install OS patches | WSUS/SCCM access |
| Firewall disabled | Enable firewall | Agent-initiated remediation |
| Prohibited app | Remove application | Agent notification + user action |
| No encryption | Enable disk encryption | Agent-initiated or manual |

### Quarantine Network Design

The quarantine VLAN/dACL must provide:
1. **DNS access** to resolve remediation server names
2. **DHCP access** for IP assignment
3. **Access to remediation servers** (WSUS, SCCM, AV update servers)
4. **Access to ISE portals** for posture re-check
5. **Block all other access** (especially internet and internal resources)

$$ACL_{quarantine} = Permit(DNS) + Permit(DHCP) + Permit(Remediation) + Permit(ISE) + Deny(all)$$

---

## 9. ISE in SD-Access

### SD-Access Integration

ISE is the policy engine for Cisco SD-Access (SDA):

```
DNA Center              ISE                   Fabric Switches
(Orchestrator)    (Policy Engine)           (Control + Data Plane)
     |                  |                          |
     | Define policy    |                          |
     |----------------->| Store policy             |
     |                  |                          |
     |                  |<-- RADIUS auth ----------|
     |                  |--- SGT assignment ------>|
     |                  |                          |
     |                  |<-- pxGrid session data --|
     |                  |--- SGACL download ------>|
     |                  |                          |
```

### SGT in VXLAN (SD-Access)

In SD-Access, SGT is carried in the VXLAN header's Group Policy ID field:

```
VXLAN Header:
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|R|R|R|R|I|R|R|R|            Group Policy ID                    |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                VXLAN Network Identifier (VNI) |   Reserved    |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+

Group Policy ID = SGT value (16 bits)
```

This eliminates the need for SXP or CMD in the fabric; the SGT is natively part of the VXLAN encapsulation.

### ISE + DNA Center Role Division

| Function | DNA Center | ISE |
|----------|-----------|-----|
| Fabric provisioning | Yes | No |
| Policy definition (abstract) | Yes | Receives from DNAC |
| RADIUS authentication | No | Yes |
| SGT assignment | No | Yes |
| SGACL enforcement | No (defines policy) | Yes (pushes to devices) |
| Endpoint visibility | Via ISE (pxGrid) | Yes (native) |
| Assurance/analytics | Yes | Monitoring (MnT) |

---

## See Also

- radius, cisco-ftd, ipsec, snmp

## References

- [Cisco ISE Administrator Guide](https://www.cisco.com/c/en/us/td/docs/security/ise/3-2/admin_guide/b_ise_admin_3_2.html)
- [Cisco ISE Performance and Scalability Guide](https://www.cisco.com/c/en/us/td/docs/security/ise/performance_and_scalability/b_ise_perf_and_scale.html)
- [Cisco TrustSec System Architecture](https://www.cisco.com/c/en/us/solutions/enterprise-networks/trustsec/trustsec-system-architecture.html)
- [Cisco pxGrid Documentation](https://developer.cisco.com/docs/pxgrid/)
- [Cisco SD-Access Design Guide](https://www.cisco.com/c/en/us/td/docs/solutions/CVD/Campus/cisco-sda-design-guide.html)
- [RFC 2865 — RADIUS](https://www.rfc-editor.org/rfc/rfc2865)
- [RFC 5176 — Dynamic Authorization Extensions to RADIUS](https://www.rfc-editor.org/rfc/rfc5176)
- [RFC 7170 — EAP-TEAP](https://www.rfc-editor.org/rfc/rfc7170)
