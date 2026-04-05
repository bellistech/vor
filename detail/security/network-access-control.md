# The Engineering of Network Access Control — Evolution, Posture Assessment, and Zero Trust Integration

> *NAC answers the fundamental question: "Should this device be on this network?" It evolved from simple port-level control to a continuous trust evaluation framework that considers identity, device health, behavior, and context for every connection.*

---

## 1. NAC Evolution

### Historical Timeline

```
1990s: Port Security (MAC-based)
  - Static MAC allowlists on switch ports
  - No authentication, no posture
  - Manual, unscalable

2001: IEEE 802.1X Ratified
  - Port-based access control with EAP/RADIUS
  - Identity-based (username/certificate)
  - No endpoint health assessment

2004: Cisco NAC (Network Admission Control)
  - First commercial NAC solution
  - Cisco Trust Agent on endpoints
  - Clean Access Server + Clean Access Manager
  - Posture checks: AV, patches, firewall

2004: Microsoft NAP (Network Access Protection)
  - Windows Server 2008 integration
  - System Health Validators (SHV)
  - System Health Agents (SHA) on endpoints
  - DHCP, VPN, IPsec, 802.1X enforcement
  - Deprecated in Windows Server 2012 R2

2005: TCG TNC (Trusted Network Connect)
  - Open standard from Trusted Computing Group
  - IF-MAP, IF-IMC, IF-IMV protocols
  - Vendor-neutral posture assessment
  - Competed with Cisco NAC and Microsoft NAP

2008: Cisco ISE (Identity Services Engine)
  - Replaced Cisco NAC Appliance (Clean Access)
  - Unified RADIUS, profiling, posture, guest
  - TrustSec integration (SGT-based segmentation)
  - Became the de facto enterprise NAC platform

2020+: Zero Trust NAC
  - Continuous verification (never trust, always verify)
  - Context-aware policies (user + device + location + time)
  - Cloud-delivered NAC (Cisco ISE cloud, Portnox, Foxpass)
  - Integration with SASE and SD-WAN
```

### Framework Comparison

| Framework | Vendor | Status | Standards |
|-----------|--------|--------|-----------|
| Cisco NAC (Clean Access) | Cisco | Deprecated (replaced by ISE) | Proprietary |
| Microsoft NAP | Microsoft | Deprecated (Server 2012 R2) | Proprietary (SoH protocol) |
| TCG TNC | Trusted Computing Group | Active (niche adoption) | IF-MAP, IF-IMC, IF-IMV |
| Cisco ISE NAC | Cisco | Active (market leader) | 802.1X, RADIUS, TrustSec |
| ForeScout CounterACT | ForeScout | Active | Agentless, multi-vendor |
| Aruba ClearPass | HPE Aruba | Active | 802.1X, RADIUS, TACACS+ |
| Portnox Cloud | Portnox | Active | Cloud-native, 802.1X |

### Why NAP and Early NAC Failed

Microsoft NAP and Cisco Clean Access were largely replaced because:

1. **Agent dependency:** Required agents on every endpoint (deployment burden)
2. **Single-vendor lock-in:** NAP required Windows; Clean Access required Cisco
3. **Binary posture:** Compliant or non-compliant (no nuance)
4. **No profiling:** Could not identify unmanaged devices
5. **Poor scalability:** Centralized appliance bottlenecks
6. **Disrupted by BYOD:** Could not handle unknown devices at scale

ISE succeeded by combining NAC with profiling, guest access, BYOD, and TrustSec in a single platform.

---

## 2. Pre-Admission vs Post-Admission Control

### Pre-Admission Control

Pre-admission NAC evaluates endpoints before granting network access:

```
Pre-Admission Decision Flow:

Endpoint connects → Authentication? → Identity verified?
                         |                    |
                        No                   Yes
                         |                    |
                    [Guest/MAB]          Posture check?
                         |                    |
                    [Apply guest          Compliant?
                     policy]                  |
                                        Yes      No
                                         |        |
                                    [Full       [Quarantine
                                     access]    VLAN/ACL]
```

Pre-admission controls:

| Control | Mechanism | Enforcement Point |
|---------|-----------|------------------|
| Identity verification | 802.1X, MAB, WebAuth | Switch/WLC port |
| Posture assessment | Agent check (AV, patches) | RADIUS policy |
| Device profiling | DHCP/CDP/HTTP fingerprint | RADIUS policy |
| Certificate validation | EAP-TLS certificate check | RADIUS server |
| Group membership | AD/LDAP group lookup | RADIUS policy |
| Time-of-day | Policy condition | RADIUS policy |
| Location | NAS-IP, AP location | RADIUS policy |

### Post-Admission Control

Post-admission NAC continuously monitors endpoints after access is granted:

```
Post-Admission Monitoring:

Endpoint authenticated → Full access granted
        |
        v
  [Continuous monitoring]
        |
        ├── Posture re-assessment (periodic, e.g., every 4 hours)
        │   └── If non-compliant → CoA → quarantine
        │
        ├── Behavioral anomaly detection
        │   └── If anomalous → ANC quarantine → investigation
        │
        ├── Threat intelligence feed
        │   └── If endpoint IP/MAC in threat feed → CoA → isolate
        │
        ├── MDM compliance change
        │   └── If device jailbroken/non-compliant → CoA → restrict
        │
        └── Session timeout / re-authentication
            └── Re-evaluate all conditions on re-auth
```

### Post-Admission Enforcement via CoA

$$T_{response} = T_{detection} + T_{evaluation} + T_{CoA} + T_{enforcement}$$

| Trigger | Detection Method | Typical Response Time |
|---------|-----------------|---------------------|
| AV definitions expire | Posture re-check (periodic) | Minutes to hours |
| Endpoint compromised | Behavioral anomaly + threat feed | Seconds to minutes |
| User leaves organization | AD group change + re-auth | Minutes (on next re-auth) |
| Compliance policy change | ISE policy update + CoA | Seconds (admin-initiated) |
| Rogue device detected | Profile change + ANC action | Seconds |

### Pre vs Post Comparison

$$Security_{total} = Security_{pre} + Security_{post}$$

Neither is sufficient alone:

| Scenario | Pre-Admission Only | Post-Admission Only |
|----------|-------------------|-------------------|
| Endpoint compromised after auth | Not detected | Detected (behavioral) |
| Unmanaged device connects | Blocked (no auth) | Not applicable (already on network) |
| AV definitions expire | Not detected (passed at auth time) | Detected (periodic re-check) |
| Credential theft | Not prevented (valid creds) | Detected (anomalous behavior) |

---

## 3. Posture Assessment Frameworks

### Posture Assessment Architecture

```
+-----------------+      +------------------+      +------------------+
| Endpoint        |      | NAC Server       |      | Policy Engine    |
|                 |      |                  |      |                  |
| +-------------+ |      |  +-----------+   |      |  +-----------+   |
| | Posture     | | <==> |  | Posture   |   | <==> |  | Posture   |   |
| | Agent       | |      |  | Service   |   |      |  | Policy    |   |
| +-------------+ |      |  +-----------+   |      |  +-----------+   |
|                 |      |                  |      |                  |
| Checks:        |      | Evaluates:       |      | Defines:         |
| - AV version   |      | - Agent results  |      | - Requirements   |
| - Patch level  |      | - Compliance     |      | - Conditions     |
| - Firewall     |      |   status         |      | - Remediation    |
| - Encryption   |      | - Grace periods  |      | - Grace periods  |
+-----------------+      +------------------+      +------------------+
```

### Compliance Evaluation Model

$$Compliance(E) = \bigwedge_{i=1}^{N} Requirement_i(E)$$

An endpoint $E$ is compliant if and only if ALL requirements are satisfied. Each requirement is evaluated independently:

$$Requirement_i(E) = \begin{cases} True & \text{if } Check_i(E) \geq Threshold_i \\ True & \text{if } T_{current} \leq T_{grace\_expiry_i} \\ False & \text{otherwise} \end{cases}$$

Where:
- $Check_i(E)$ = the measured value (AV version, patch date, etc.)
- $Threshold_i$ = the minimum acceptable value
- $T_{grace\_expiry_i}$ = grace period expiration time

### Grace Periods and Remediation Windows

Grace periods allow temporary access while endpoints remediate:

$$Access_{level} = \begin{cases} Full & \text{if } Compliance(E) = True \\ Full & \text{if } T_{current} < T_{first\_noncompliant} + T_{grace} \\ Quarantine & \text{if } T_{current} \geq T_{first\_noncompliant} + T_{grace} \end{cases}$$

Grace period design considerations:

| Factor | Short Grace (1-4 hours) | Long Grace (1-7 days) |
|--------|------------------------|----------------------|
| Security | Better (faster remediation) | Worse (longer exposure) |
| User impact | Higher (disruption if not remediated) | Lower (more time) |
| Helpdesk load | Higher (more calls) | Lower (self-service) |
| Best for | Critical patches, AV updates | OS upgrades, encryption deployment |

### Posture Assessment Timing

$$T_{posture} = T_{agent\_load} + T_{checks} + T_{report} + T_{evaluation}$$

Typical posture assessment timing:

| Component | Duration | Notes |
|-----------|----------|-------|
| Agent initialization | 2-5 seconds | Agent starts, connects to ISE |
| AV check | 1-3 seconds | Query AV product APIs |
| Patch check | 3-10 seconds | Enumerate installed patches |
| Disk encryption check | 1-2 seconds | Query BitLocker/FileVault |
| Firewall check | <1 second | Check firewall service status |
| File/registry checks | 1-5 seconds | Depends on number of checks |
| Report to ISE | 1-2 seconds | Submit results over HTTPS |
| ISE evaluation | <1 second | Policy rule matching |
| CoA (if needed) | 1-3 seconds | RADIUS CoA to switch |
| **Total** | **10-30 seconds** | **Typical end-to-end** |

---

## 4. Remediation Strategies

### Remediation Architecture

```
Non-Compliant              NAC                    Remediation
Endpoint                   Server                 Infrastructure
    |                        |                         |
    |<-- Quarantine ---------|                         |
    |    (limited access)    |                         |
    |                        |                         |
    |--- Access remediation --|------------------------->|
    |    resources            |                         |
    |                        |                         |
    | Auto-remediation:      |                         |
    | - Agent downloads      |                         |
    |   AV updates          |<--- Patch/update --------|
    | - Agent installs       |     server               |
    |   patches              |                         |
    | - Agent enables        |                         |
    |   firewall             |                         |
    |                        |                         |
    | Manual remediation:    |                         |
    | - User follows portal  |                         |
    |   instructions         |                         |
    | - User installs        |                         |
    |   required software    |                         |
    |                        |                         |
    |--- Re-check posture -->|                         |
    |                        | [Evaluate: compliant]    |
    |                        |--- CoA (full access) --->| Switch
    |                        |                         |
    | [Full network access]  |                         |
```

### Remediation Methods

| Method | Automation | User Action | Reliability |
|--------|-----------|-------------|------------|
| Agent auto-remediation | Full | None | High (agent handles everything) |
| URL redirect to portal | None | User follows instructions | Low (depends on user) |
| WSUS/SCCM integration | Partial | User approves updates | Medium |
| MDM push | Full | None (MDM manages device) | High |
| Self-service portal | None | User downloads/installs | Low |

### Quarantine Network Design

The quarantine network is a critical infrastructure component:

$$Access_{quarantine} = DNS + DHCP + Remediation\_Servers + NAC\_Portal$$

$$Isolation_{quarantine} = Block(Internal) + Block(Internet) + Block(Other\_Quarantine)$$

Design principles:

1. **Minimum viable access:** Only permit traffic necessary for remediation
2. **Isolation:** Quarantined endpoints must not reach production networks
3. **Inter-quarantine isolation:** Quarantined endpoints should not communicate with each other (prevent lateral movement)
4. **Automatic re-check:** After remediation, posture agent automatically re-checks without user intervention
5. **Escalation path:** If auto-remediation fails, provide helpdesk contact information

### Remediation Failure Handling

$$P_{remediation\_success} = P_{auto\_fix} + (1 - P_{auto\_fix}) \times P_{manual\_fix}$$

When remediation fails:

```
Attempt 1: Auto-remediation (agent)
  → Success? → Re-check → Full access
  → Fail?
      |
Attempt 2: User-directed remediation (portal instructions)
  → Success? → Re-check → Full access
  → Fail?
      |
Attempt 3: Helpdesk escalation
  → Technician remediates endpoint
  → Manual re-check initiated
  → Full access

Timeout: If no remediation after T_max
  → Move to restricted VLAN (internet-only)
  → Or disconnect entirely (strict environments)
```

---

## 5. NAC in Zero Trust Architecture

### Zero Trust Principles Applied to NAC

```
Traditional NAC:                    Zero Trust NAC:
+------------------+               +------------------+
| Trust boundary:  |               | Trust boundary:  |
| Network perimeter|               | Per-session,     |
|                  |               | per-transaction  |
| Once inside →    |               |                  |
| trusted          |               | Never trusted →  |
|                  |               | always verified  |
| One-time check   |               | Continuous check |
| (pre-admission)  |               | (pre + post)     |
+------------------+               +------------------+
```

### Zero Trust NAC Components

$$Trust(session) = f(Identity, Device, Posture, Behavior, Context, Time)$$

| Factor | Traditional NAC | Zero Trust NAC |
|--------|----------------|---------------|
| Identity | Username/password or certificate | MFA + certificate + behavioral biometrics |
| Device | Known MAC address | Device certificate + posture + MDM compliance |
| Posture | AV + patches (one-time) | Continuous posture + EDR telemetry |
| Behavior | Not evaluated | ML-based anomaly detection |
| Context | Source IP / location | Location + time + risk score + peer comparison |
| Time | Session timeout (hours) | Continuous re-evaluation (seconds) |
| Segmentation | VLAN-based | Micro-segmentation (SGT, software-defined) |

### Continuous Trust Evaluation

$$Trust\_Score(t) = \alpha \times Identity(t) + \beta \times Device(t) + \gamma \times Behavior(t) + \delta \times Context(t)$$

Where $\alpha + \beta + \gamma + \delta = 1$ and each factor is normalized to $[0, 1]$.

$$Access\_Decision(t) = \begin{cases} Full & \text{if } Trust\_Score(t) \geq T_{full} \\ Limited & \text{if } T_{limited} \leq Trust\_Score(t) < T_{full} \\ Denied & \text{if } Trust\_Score(t) < T_{limited} \end{cases}$$

Trust score degrades over time without re-verification:

$$Trust\_Score(t + \Delta t) = Trust\_Score(t) \times e^{-\lambda \Delta t}$$

Where $\lambda$ is the decay rate. This models the principle that trust is not permanent; the longer since the last verification, the lower the trust.

### NIST SP 800-207 Alignment

NIST Zero Trust Architecture (SP 800-207) defines the Policy Decision Point (PDP) and Policy Enforcement Point (PEP):

```
                    +-----------+
                    | Policy    |
                    | Engine    |  ← ISE / NAC Server
                    | (PDP)     |
                    +-----------+
                         |
              Policy decisions
                         |
                    +-----------+
                    | Policy    |
                    | Admin     |  ← ISE PAN
                    | Point     |
                    +-----------+
                         |
              Enforcement signals
                         |
+--------+         +-----------+         +----------+
|Subject | ------> | Policy    | ------> |Enterprise|
|(User/  |         | Enforcement|        |Resource  |
|Device) |         | Point (PEP)|        |          |
+--------+         +-----------+         +----------+
                    ↑
              Switch/WLC/FW
              (802.1X, ACL, SGT)
```

In Cisco's implementation:
- **PDP:** ISE Policy Service Node (evaluates policy)
- **PEP:** Switch, WLC, FTD (enforces VLAN, ACL, SGT)
- **Policy Admin:** ISE Policy Administration Node (defines policy)
- **Continuous monitoring:** pxGrid + Stealthwatch + AMP for Endpoints

---

## 6. Agent vs Agentless Trade-offs

### Comparison Matrix

| Dimension | Agent-Based | Agentless |
|-----------|------------|-----------|
| Deployment effort | High (install on every endpoint) | Low (no endpoint software) |
| Posture depth | Deep (AV, patches, registry, services, encryption) | Shallow (OS, AV, basic patch level) |
| Remediation | Auto-remediation (agent acts on endpoint) | Manual only (user must remediate) |
| Continuous monitoring | Yes (agent runs persistently) | No (point-in-time check) |
| Platform support | Windows, macOS, Linux (varies by agent) | Any IP device (network-based) |
| IoT support | No (cannot install agent on IoT) | Partial (network profiling only) |
| User friction | Medium (agent install required) | Low (no install, browser-based) |
| Maintenance | Agent updates, compatibility testing | Minimal (server-side) |
| Accuracy | High (direct endpoint inspection) | Medium (inferred from network) |
| Privacy | Concerns (agent has deep endpoint access) | Less concern (network metadata only) |

### Agentless Techniques

Agentless NAC gathers endpoint information without installing software:

$$Visibility_{agentless} = Profiling + NMAP + WMI + SNMP + NetFlow$$

| Technique | Data Gathered | Limitations |
|-----------|--------------|-------------|
| DHCP fingerprinting | OS type, hostname, vendor | Not detailed (OS family, not version) |
| HTTP User-Agent | Browser, OS version | Only for HTTP traffic |
| SNMP queries | CDP/LLDP neighbors, device model | Requires SNMP enabled |
| NMAP scan | Open ports, OS fingerprint | Active scan, can be intrusive |
| WMI (Windows) | Installed software, patches, services | Requires credentials, Windows only |
| SSH scan (Linux) | OS version, packages, services | Requires credentials |
| NetFlow | Traffic patterns, protocols used | Behavioral, not compliance-based |

### Decision Framework

$$Score_{agent} = W_1 \times S_{depth} + W_2 \times S_{auto\_remediation} + W_3 \times S_{continuous}$$

$$Score_{agentless} = W_4 \times S_{no\_deploy} + W_5 \times S_{IoT} + W_6 \times S_{BYOD}$$

| Use Case | Recommendation | Rationale |
|----------|---------------|-----------|
| Managed corporate fleet | Agent | Deep posture, auto-remediation |
| BYOD laptops | Temporal agent or agentless | Minimal install, one-time check |
| IoT devices | Agentless (profiling + MAB) | Cannot install agents |
| Contractor devices | Temporal agent | Session-based, no permanent install |
| Server farm | Agent or agentless (WMI/SSH) | Depends on server OS and policy |
| Guest devices | Agentless | No installation possible |

---

## 7. NAC Scalability

### Scaling Dimensions

$$NAC\_Capacity = f(Endpoints, Auth\_Rate, Policy\_Complexity, Profiling\_Load)$$

| Dimension | Small (1K endpoints) | Medium (10K endpoints) | Large (100K+ endpoints) |
|-----------|---------------------|----------------------|------------------------|
| ISE nodes | 1 (standalone) | 3-5 (distributed) | 10-50 (full distributed) |
| PSN count | 1 | 2-4 | 5-20+ |
| RADIUS RPS | 100 | 1,000 | 10,000+ |
| Profiled endpoints | 1,000 | 10,000 | 100,000-1M |
| Switch/WLC count | 10-20 | 50-200 | 500-5,000 |

### RADIUS Performance

$$T_{auth} = T_{network} + T_{ISE\_processing} + T_{identity\_lookup}$$

Where:
- $T_{network}$ = round-trip time between switch and ISE ($\approx 1\text{-}5\text{ms}$)
- $T_{ISE\_processing}$ = policy evaluation ($\approx 5\text{-}20\text{ms}$)
- $T_{identity\_lookup}$ = AD/LDAP query ($\approx 10\text{-}50\text{ms}$)
- Total: $\approx 20\text{-}75\text{ms}$ per authentication

For a large campus with 50,000 users arriving over 1 hour (morning login surge):

$$RPS_{peak} = \frac{50000}{3600} \times Burst\_Factor \approx 14 \times 3 = 42 \text{ RPS}$$

This is well within a single PSN's capacity (5,000+ RPS), but additional PSNs are needed for redundancy and geographic distribution.

### Profiling Scalability

Profiling creates significant data flow:

$$Probes_{per\_endpoint} = N_{DHCP} + N_{HTTP} + N_{SNMP} + N_{RADIUS}$$

For 100,000 endpoints with DHCP + HTTP + RADIUS probes:

$$Events_{per\_day} \approx 100000 \times 10 = 1,000,000 \text{ profiling events}$$

ISE processes these on the PSN nodes. SNMP polling is the most resource-intensive probe:

$$SNMP\_queries_{per\_cycle} = N_{switches} \times Q_{per\_switch}$$

For 500 switches polled every 10 minutes with 20 queries each:

$$SNMP\_load = \frac{500 \times 20}{600} \approx 17 \text{ queries/second}$$

### Database and Replication

ISE stores endpoint data in PostgreSQL:

$$Storage_{endpoints} \approx N_{endpoints} \times 5\text{KB} \text{ per record}$$

For 1 million profiled endpoints:
$$Storage = 1000000 \times 5\text{KB} = 5\text{GB}$$

Replication from PAN to PSN occurs on policy changes. Full sync time:

$$T_{sync} = \frac{S_{policy\_DB}}{BW_{ISE\_backbone}}$$

For a 500MB policy database over a 1Gbps link:
$$T_{sync} = \frac{500\text{MB}}{125\text{MB/s}} = 4\text{s}$$

During sync, PSNs continue to operate with cached policy.

---

## 8. NAC for IoT Challenges

### The IoT NAC Problem

IoT devices present fundamental challenges to traditional NAC:

| Challenge | Description | Impact |
|-----------|-------------|--------|
| No supplicant | Cannot run 802.1X | Must use MAB (weaker auth) |
| No agent | Cannot install posture agent | No posture assessment |
| Diverse protocols | MQTT, CoAP, BACnet, Modbus, Zigbee | Standard web inspection does not apply |
| Long lifecycles | 10-20 year operational life | Cannot patch, cannot upgrade |
| Vendor fragmentation | Thousands of manufacturers | Profiling difficult |
| Headless operation | No user interface | No user authentication |
| Scale | 10x more IoT devices than users | Profiling and policy at massive scale |

### IoT Identification Strategies

$$Confidence_{IoT}(device) = \sum_{i=1}^{N} W_i \times Signal_i$$

Identification signals:

| Signal | Source | Weight | Reliability |
|--------|--------|--------|-------------|
| MAC OUI | Ethernet frame | Low | Low (OUI can indicate manufacturer, not device type) |
| DHCP fingerprint | DHCP options 55, 60 | Medium | Medium (OS fingerprint) |
| CDP/LLDP TLVs | Device sensor | High | High (device self-identification) |
| mDNS/Bonjour | Service discovery | Medium | Medium (service type) |
| Traffic behavior | NetFlow/IPFIX | Medium | Medium (protocol patterns) |
| Banner grabbing | Active scan | High | High (service identification) |
| NMAP fingerprint | Active scan | Very High | Very High (OS detection) |
| API integration | Manufacturer cloud | Very High | Very High (authoritative source) |

### IoT Segmentation Model

```
Network Segmentation for IoT:

Production Network
├── User VLANs (802.1X authenticated)
│   ├── VLAN 10: Engineering
│   ├── VLAN 20: Finance
│   └── VLAN 30: General
│
├── IoT VLANs (MAB + profiling)
│   ├── VLAN 100: Building Management
│   │   └── ACL: BACnet (UDP 47808) to BMS only
│   ├── VLAN 101: Physical Security
│   │   └── ACL: RTSP (TCP 554) to NVR only
│   ├── VLAN 102: Medical Devices
│   │   └── ACL: HL7 (TCP 2575) to clinical systems only
│   └── VLAN 103: Facilities (HVAC, lighting)
│       └── ACL: HTTP/HTTPS to management server only
│
├── Guest VLAN (WebAuth)
│   └── VLAN 200: Internet-only access
│
└── Quarantine VLAN
    └── VLAN 999: Remediation access only
```

### IoT NAC Policy Framework

The principle of least privilege applied to IoT:

$$ACL_{IoT}(device) = \{Permit(required\_flows)\} \cup \{Deny(all\_else)\}$$

Where $required\_flows$ is the minimum set of network connections the device needs to function:

| Device Type | Required Flows | Everything Else |
|-------------|---------------|----------------|
| IP Camera | RTSP/RTP to NVR, NTP, DNS | Denied |
| HVAC Controller | BACnet to BMS, NTP, DNS | Denied |
| Badge Reader | TCP 3001 to access controller, NTP | Denied |
| Medical Pump | HL7 to clinical system, NTP, DNS | Denied |
| Smart Display | HTTPS to signage server, NTP, DNS | Denied |

### IoT Lifecycle Management

$$Risk_{IoT}(t) = Vulnerabilities_{unpatched}(t) \times Exposure_{network}(t) \times Value_{target}$$

IoT risk increases over time because vulnerabilities accumulate while patching is rare:

$$Vulnerabilities(t) = V_0 + \int_{0}^{t} R_{discovery}(\tau) \, d\tau - \int_{0}^{t} R_{patching}(\tau) \, d\tau$$

For a typical IoT device where $R_{patching} \approx 0$:

$$Vulnerabilities(t) \approx V_0 + R_{discovery} \times t$$

NAC mitigates this risk through network isolation:

$$Exposure_{with\_NAC} = \frac{Allowed\_flows}{Total\_possible\_flows} \ll 1$$

Even if an IoT device is compromised, strict ACLs limit the attacker's ability to move laterally or exfiltrate data. Micro-segmentation with SGT/SGACL provides the strongest isolation without requiring per-port ACL management.

---

## See Also

- dot1x, cisco-ise, radius, zero-trust, cisco-ftd

## References

- [NIST SP 800-207 — Zero Trust Architecture](https://csrc.nist.gov/publications/detail/sp/800-207/final)
- [Cisco ISE Administrator Guide — NAC](https://www.cisco.com/c/en/us/td/docs/security/ise/3-2/admin_guide/b_ise_admin_3_2.html)
- [Cisco ISE Prescriptive Deployment Guide](https://community.cisco.com/t5/security-knowledge-base/ise-secure-wired-access-prescriptive-deployment-guide/ta-p/3641515)
- [NIST SP 800-183 — Networks of Things](https://csrc.nist.gov/publications/detail/sp/800-183/final)
- [TCG TNC Architecture](https://trustedcomputinggroup.org/work-groups/trusted-network-communications/)
- [IEEE 802.1X-2020 — Port-Based Network Access Control](https://standards.ieee.org/standard/802_1X-2020.html)
- [RFC 5176 — Dynamic Authorization Extensions to RADIUS](https://www.rfc-editor.org/rfc/rfc5176)
- [RFC 3748 — Extensible Authentication Protocol (EAP)](https://www.rfc-editor.org/rfc/rfc3748)
- [Cisco TrustSec Design Guide](https://www.cisco.com/c/en/us/solutions/enterprise-networks/trustsec/design-guide-series.html)
- [Forrester Zero Trust eXtended (ZTX) Framework](https://www.forrester.com/report/the-zero-trust-extended-ztx-ecosystem/RES137210)
