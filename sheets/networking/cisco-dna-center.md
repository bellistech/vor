# Cisco DNA Center (Network Controller and Assurance Platform)

Intent-based networking controller that automates network design, provisioning, policy enforcement, and assurance across campus, branch, and WAN infrastructure using Cisco Catalyst Center (formerly DNA Center).

## Architecture

### Platform Overview

- **Catalyst Center** (rebranded from DNA Center in 2023) — on-prem appliance or cluster
- Runs on Cisco UCS hardware (DN2-HW-APL / DN2-HW-APL-L / DN2-HW-APL-XL)
- Built on a microservices architecture running on Kubernetes
- Single-node for small deployments, 3-node cluster for HA and scale
- Communicates with devices via NETCONF/YANG, SSH/CLI, SNMP, RESTCONF
- Southbound protocols: NETCONF, CLI, SNMP, streaming telemetry (gRPC)
- Northbound: REST APIs (Westbound: ISE/CMX/Spaces integration)

### Appliance Tiers

| Appliance | Model | Managed Devices | Assurance Devices |
|:---|:---|:---:|:---:|
| Standard | DN2-HW-APL | Up to 1,000 | Up to 2,000 |
| Large | DN2-HW-APL-L | Up to 2,500 | Up to 5,000 |
| Extra Large | DN2-HW-APL-XL | Up to 5,000 | Up to 10,000 |
| 3-node Cluster | 3x DN2-HW-APL-XL | Up to 10,000 | Up to 25,000 |

### Core Services

```
┌─────────────────────────────────────────────────────┐
│                  Catalyst Center                     │
│                                                     │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌────────┐│
│  │  Design  │ │  Policy  │ │ Provision│ │Assurance││
│  └──────────┘ └──────────┘ └──────────┘ └────────┘│
│  ┌──────────┐ ┌──────────┐ ┌──────────┐           │
│  │   NDP    │ │   SWIM   │ │  PnP     │           │
│  └──────────┘ └──────────┘ └──────────┘           │
│  ┌──────────────────────────────────────┐          │
│  │        Platform / APIs / SDK         │          │
│  └──────────────────────────────────────┘          │
│  ┌──────────────────────────────────────┐          │
│  │    Kubernetes / Microservices         │          │
│  └──────────────────────────────────────┘          │
└─────────────────────────────────────────────────────┘
```

## Design Workflow

### Network Hierarchy

```
Global
 └── Area (Region/Country)
      └── Site
           └── Building
                └── Floor
```

- **Global:** Top-level settings inherited by all sites (DNS, NTP, AAA, syslog, SNMP)
- **Area:** Organizational grouping (no device assignment)
- **Site:** Logical site with specific settings
- **Building:** Physical location where devices are assigned
- **Floor:** Floor maps for wireless planning (imported from CAD/Ekahau)

### Global Network Settings

```
Settings → Network Settings → Global
 ├── DNS Servers (primary, secondary)
 ├── NTP Servers
 ├── AAA (ISE integration)
 ├── DHCP Servers
 ├── Syslog Servers
 ├── SNMP
 ├── Message of the Day
 └── Telemetry (NetFlow collectors)
```

### Device Credentials

```
Settings → Device Credentials
 ├── CLI Credentials (SSH username/password/enable)
 ├── SNMP v2c (read/write communities)
 ├── SNMP v3 (user/auth/priv)
 └── HTTPS Credentials (for REST API-managed devices)
```

### IP Address Management

```
Settings → IP Address Pools
 ├── Global Pool (supernet)
 │    └── Site-specific sub-pools (carved from global)
 ├── Management Pool (device management IPs)
 ├── AP Pool (access point management)
 └── IoT Pool (IoT device segments)
```

### Network Profiles

```
Design → Network Profiles
 ├── Switching Profile
 │    ├── Onboarding Template (Day-0)
 │    ├── Day-N Templates (ongoing config)
 │    └── VLAN-to-Fabric assignments
 ├── Wireless Profile
 │    ├── SSID assignments
 │    ├── Flex Connect settings
 │    └── Sensor settings
 └── Routing Profile
      └── WAN/Branch templates
```

## Policy

### Group-Based Access Control (TrustSec / SGT)

```
Policy → Group-Based Access Control
 ├── Scalable Groups (SGTs)
 │    ├── Employees (SGT 10)
 │    ├── Guests (SGT 20)
 │    ├── IoT_Devices (SGT 30)
 │    ├── Servers (SGT 40)
 │    └── Quarantine (SGT 50)
 ├── Access Contract (ACL-like rules)
 │    ├── Permit
 │    ├── Deny
 │    └── Permit with logging
 └── Policy Matrix
      ├── Source SGT → Destination SGT → Contract
      └── Default policy (Permit/Deny/None)
```

### Policy Matrix Example

| Source \ Dest | Servers (40) | Internet | Quarantine (50) |
|:---|:---:|:---:|:---:|
| Employees (10) | Permit | Permit | Deny |
| Guests (20) | Deny | Permit_Web | Deny |
| IoT (30) | Permit_IoT | Deny | Deny |
| Quarantine (50) | Deny | Deny | Deny |

### Application Policy (QoS)

```
Policy → Application → QoS
 ├── Application Sets
 │    ├── Business Relevant (voice, video, critical apps)
 │    ├── Default (unclassified)
 │    └── Business Irrelevant (social media, streaming)
 ├── Queuing Profiles
 │    ├── CVD (Cisco Validated Design) defaults
 │    └── Custom profiles
 └── Marking Policy
      ├── DSCP mapping per application set
      └── Per-hop behavior assignments
```

### Virtual Network (VN) and Macro/Micro Segmentation

```
Policy → Virtual Networks
 ├── VN_Corporate
 │    ├── SGT: Employees, Printers
 │    └── IP Pool: 10.10.0.0/16
 ├── VN_Guest
 │    ├── SGT: Guests
 │    └── IP Pool: 10.20.0.0/16
 └── VN_IoT
      ├── SGT: IoT_Devices, Cameras
      └── IP Pool: 10.30.0.0/16
```

- **Macro-segmentation:** VN isolation (like VRFs — traffic between VNs requires a firewall or fusion router)
- **Micro-segmentation:** SGT-based policy within a VN (permits/denies between groups in same VN)

## Provision

### Plug and Play (PnP)

```
Provision → Plug and Play
 ├── Unclaimed Devices (discovered via PnP)
 ├── Planned Devices (pre-staged with serial number)
 └── Onboarded Devices (successfully provisioned)
```

PnP Workflow:

```
1. New device boots with factory default
2. Device sends PnP discovery (DHCP option 43, DNS, cloud redirect)
3. Catalyst Center receives PnP request
4. Admin claims device → assigns site, template, image
5. Device downloads image (SWIM) and configuration
6. Device reboots with production config
7. Device appears in Inventory as Managed
```

### DHCP Option 43 for PnP

```
! DHCP server configuration for PnP discovery
ip dhcp pool PNP_POOL
 network 10.0.0.0 255.255.255.0
 default-router 10.0.0.1
 option 43 ascii "5A1N;B2;K4;I10.1.1.100;J80"
 ! 5A1N = PnP identifier
 ! B2 = address type (2=IPv4)
 ! K4 = transport (4=HTTP)
 ! I = Catalyst Center IP
 ! J = port
```

### Software Image Management (SWIM)

```
Design → Image Repository
 ├── Import Images (from cisco.com or local upload)
 ├── Golden Images (approved per device family)
 ├── Image Compliance
 │    ├── Compliant (running golden image)
 │    ├── Non-compliant (running different version)
 │    └── Unknown (no golden image defined)
 └── Image Update Workflow
      ├── Schedule distribution
      ├── Schedule activation (reboot)
      └── Rolling upgrade support
```

### Day-N Templates

```
Design → Network Profiles → Templates
 ├── Regular Templates
 │    ├── Jinja2 syntax with variables
 │    ├── Bound to network profile → site
 │    └── Pushed via Provision workflow
 ├── Composite Templates
 │    ├── Chain multiple regular templates
 │    └── Ordered execution
 └── Template Variables
      ├── System variables (hostname, mgmt_ip, site)
      ├── Bind variables (user-defined per device)
      └── Global variables (same across all devices)
```

### Template Example (Jinja2)

```
! Day-N template for access switch
interface range {{ access_port_range }}
 switchport mode access
 switchport access vlan {{ data_vlan }}
 switchport voice vlan {{ voice_vlan }}
 spanning-tree portfast
 spanning-tree bpduguard enable

{% if enable_dot1x %}
 dot1x pae authenticator
 authentication port-control auto
 authentication order dot1x mab
 authentication priority dot1x mab
{% endif %}

ntp server {{ ntp_server_1 }}
ntp server {{ ntp_server_2 }}
```

### Provision Workflow

```bash
Provision → Inventory → Select Device
 ├── Assign to Site
 ├── Provision (push network settings, templates, policies)
 ├── Re-provision (update after policy/template change)
 └── Delete (remove from management)
```

## Assurance

### Health Dashboard

```
Assurance → Health
 ├── Overall Network Health (0-100 score)
 ├── Client Health
 │    ├── Wired clients
 │    ├── Wireless clients
 │    ├── Onboarding success rate
 │    └── Per-SSID health
 ├── Network Device Health
 │    ├── Switches
 │    ├── Routers
 │    ├── Wireless Controllers
 │    └── Access Points
 └── Application Health
      ├── Business Relevant apps
      ├── Business Irrelevant apps
      └── Per-application experience scores
```

### Health Score Calculation

| Component | Weight | Factors |
|:---|:---:|:---|
| Network Health | CPU, memory, link errors, reachability | Per-device 0-10 score |
| Client Health | Onboarding, connectivity, RSSI, SNR | Per-client 0-10 score |
| Application Health | Latency, jitter, packet loss | Per-app 0-10 score |

- Score 1-3: Poor (red)
- Score 4-7: Fair (orange)
- Score 8-10: Good (green)
- Score 0: Critical/down (red)

### Path Trace

```
Assurance → Path Trace
 ├── Source (IP or client name)
 ├── Destination (IP or client name)
 ├── Protocol/Port (optional)
 └── Results
      ├── Hop-by-hop path visualization
      ├── Per-hop latency and interface stats
      ├── ACL/policy evaluation at each hop
      └── QoS marking at each hop
```

### AI-Driven Insights (AI Network Analytics)

```
Assurance → AI-Driven → Issues
 ├── AI-detected anomalies
 │    ├── AP performance degradation
 │    ├── Client onboarding failures
 │    ├── Unusual traffic patterns
 │    └── Device health anomalies
 ├── Baseline deviations
 │    ├── Throughput below baseline
 │    ├── Latency above baseline
 │    └── Client count anomaly
 └── Suggested Actions
      ├── Root cause analysis
      ├── Remediation steps
      └── Similar past incidents
```

### Network Data Platform (NDP)

- Collects telemetry from all managed devices
- Data types: syslog, SNMP traps, NetFlow, streaming telemetry (gRPC), SPAN
- Stores data for trend analysis (default 14 days, configurable)
- Powers the AI/ML analytics engine
- Feeds into Assurance dashboards and issues

## Cisco AI Network Analytics

### Machine Learning Pipeline

```
Device Telemetry → NDP Collection → Feature Extraction → ML Models → Insights
                                                                    ↓
                                                            AI-Driven Issues
                                                            Baseline Trends
                                                            Predictive Alerts
```

### Capabilities

- **Baselining:** Automatically learns normal behavior per site, device type, and time-of-day
- **Anomaly detection:** Flags deviations from learned baselines
- **Peer comparison:** Compares device/client metrics against similar devices in same role
- **Predictive analytics:** Forecasts capacity issues, AP coverage gaps
- **Global insights:** Anonymized cross-customer data for known-issue detection (opt-in via Cisco cloud)

### Cloud vs On-Prem Analytics

| Feature | On-Prem (NDP) | Cloud (AI Analytics) |
|:---|:---:|:---:|
| Baselining | Yes | Yes |
| Anomaly detection | Basic | Advanced ML |
| Peer comparison | Local only | Global cross-customer |
| Predictive alerts | No | Yes |
| Connectivity | Air-gapped OK | Requires Cisco cloud |

## Integration with ISE

### ISE Integration Setup

```
System → Settings → Authentication and Policy Servers
 ├── ISE Server IP/FQDN
 ├── Shared Secret
 ├── ISE admin credentials (for pxGrid)
 ├── pxGrid integration (for SGT/TrustSec)
 └── Certificate exchange (mutual trust)
```

### ISE Integration Points

- **AAA:** Catalyst Center provisions RADIUS/TACACS+ settings to devices pointing to ISE
- **pxGrid:** Bidirectional SGT policy sync between Catalyst Center and ISE
- **Endpoint profiling:** ISE profiles endpoints, Catalyst Center uses profiles for assurance
- **Guest services:** ISE guest portal, Catalyst Center provisions guest SSID/VN
- **Compliance:** ISE posture assessment feeds into Catalyst Center client health

### Integration with CMX / Spaces

```
System → Settings → CMX/Spaces Integration
 ├── CMX Server IP (on-prem location analytics)
 ├── Cisco Spaces cloud (SaaS location analytics)
 ├── Floor map sync (Catalyst Center → CMX/Spaces)
 └── Client location overlay on Assurance maps
```

## REST APIs

### API Categories

```
Platform → Developer Toolkit → APIs
 ├── Intent APIs (abstracted, recommended)
 │    ├── Site Management
 │    ├── Network Discovery
 │    ├── Device Management
 │    ├── Path Trace
 │    ├── Template Programmer
 │    ├── Command Runner
 │    ├── Task Management
 │    └── Event Management
 ├── Multivendor SDK
 └── Webhooks / Events
```

### Authentication

```bash
# Get authentication token (valid for 60 minutes)
curl -X POST "https://dnac.example.com/dna/system/api/v1/auth/token" \
  -H "Content-Type: application/json" \
  -u "admin:password" \
  --insecure

# Response:
# { "Token": "eyJ..." }

# Use token in subsequent requests
curl -X GET "https://dnac.example.com/dna/intent/api/v1/network-device" \
  -H "Content-Type: application/json" \
  -H "X-Auth-Token: eyJ..." \
  --insecure
```

### Common API Endpoints

```bash
# List all managed devices
GET /dna/intent/api/v1/network-device

# Get device by ID
GET /dna/intent/api/v1/network-device/{id}

# Get device health
GET /dna/intent/api/v1/device-health

# Get client health
GET /dna/intent/api/v1/client-health

# Get site hierarchy
GET /dna/intent/api/v1/site

# Create site
POST /dna/intent/api/v1/site

# Get all templates
GET /dna/intent/api/v1/template-programmer/template

# Run CLI command on device
POST /dna/intent/api/v1/network-device-poller/cli/read-request
# Body: { "commands": ["show version"], "deviceUuids": ["uuid1"] }

# Path trace
POST /dna/intent/api/v1/flow-analysis
# Body: { "sourceIP": "10.0.0.1", "destIP": "10.0.0.2", "protocol": "TCP", "destPort": "443" }

# Get task result (async operations)
GET /dna/intent/api/v1/task/{taskId}
```

### Command Runner

```bash
# Execute CLI commands on devices via API
POST /dna/intent/api/v1/network-device-poller/cli/read-request

# Request body
{
  "name": "show_commands",
  "commands": [
    "show version",
    "show ip interface brief",
    "show running-config | section interface"
  ],
  "deviceUuids": [
    "3fa85f64-5717-4562-b3fc-2c963f66afa6"
  ]
}

# Response returns a taskId — poll task endpoint for results
GET /dna/intent/api/v1/task/{taskId}

# When task completes, retrieve output
GET /dna/intent/api/v1/file/{fileId}
```

### Template Editor API

```bash
# List all templates
GET /dna/intent/api/v1/template-programmer/template

# Get template by ID
GET /dna/intent/api/v1/template-programmer/template/{templateId}

# Create a new template
POST /dna/intent/api/v1/template-programmer/project/{projectId}/template

# Deploy a template
POST /dna/intent/api/v1/template-programmer/template/deploy
# Body:
{
  "templateId": "template-uuid",
  "forcePushTemplate": false,
  "targetInfo": [
    {
      "id": "device-uuid",
      "type": "MANAGED_DEVICE_UUID",
      "params": {
        "access_port_range": "GigabitEthernet1/0/1-24",
        "data_vlan": "100",
        "voice_vlan": "200"
      }
    }
  ]
}
```

### Webhooks and Events

```bash
# Subscribe to events
POST /dna/intent/api/v1/event/subscription

# Event types:
# - NETWORK-DEVICES-1-1 (device unreachable)
# - NETWORK-CLIENTS-1-1 (client connectivity issue)
# - SWIM-1-1 (image compliance change)
# - POLICY-1-1 (policy deployment status)

# Webhook payload example
{
  "version": "1.0",
  "instanceId": "event-uuid",
  "eventId": "NETWORK-DEVICES-1-1",
  "namespace": "ASSURANCE",
  "name": "Device Unreachable",
  "description": "Switch-Floor2 is not reachable",
  "severity": 1,
  "domain": "Know Your Network",
  "source": "DNAC",
  "timestamp": 1704067200000
}
```

## ITSM Integration

### ServiceNow Integration

```
System → Settings → Integration → ITSM
 ├── ServiceNow instance URL
 ├── Credentials (API user)
 ├── Event forwarding rules
 │    ├── Map Catalyst Center events → ServiceNow incidents
 │    ├── Severity mapping (P1-P4)
 │    └── Assignment group routing
 └── CMDB sync
      ├── Push device inventory → CMDB CIs
      ├── Sync on discovery/provision
      └── Bidirectional status sync
```

### Supported ITSM Platforms

- ServiceNow (native connector)
- BMC Remedy (via REST webhook)
- Custom ITSM (via webhook + event subscription)

## Troubleshooting

### System Health

```bash
# SSH to Catalyst Center appliance
ssh -p 2222 maglev@<catalyst-center-ip>

# Check system health
maglev system status

# Check all services
maglev service status

# Check specific service
maglev service status <service-name>

# View logs
maglev service logs <service-name>

# Check cluster status (3-node)
maglev cluster status

# Check disk usage
maglev system disk-usage

# NTP sync status
maglev system ntp-status

# Package versions
maglev package status
```

### Common Issues

```bash
# Devices not discovered
# Check: network reachability, SNMP credentials, SSH credentials
# Verify: Settings → Device Credentials match device config

# PnP device not claiming
# Check: DHCP option 43 configuration
# Check: DNS _pnp-gateway._tcp SRV record
# Check: device serial number matches planned device

# Template deployment failure
# Check: Provision → Inventory → device status
# Check: template syntax (preview before deploy)
# Check: task status via GET /dna/intent/api/v1/task/{taskId}

# ISE integration broken
# Check: pxGrid certificate validity
# Check: ISE admin credentials
# Check: network connectivity (TCP 8910 for pxGrid)
```

## Tips

- Always define golden images per device family in SWIM before provisioning; non-compliant devices cause deployment failures.
- Use composite templates to modularize Day-N configs; individual templates are easier to test and version.
- Set up ISE integration before creating SGT policies; without ISE, group-based policy cannot be enforced on the network.
- Use the API, not the GUI, for bulk operations (site creation, device provisioning, template deployment); the GUI has no bulk import.
- Path Trace requires IP connectivity and SNMP/CLI access to all intermediate devices; unmanaged hops appear as gaps.
- Back up Catalyst Center configuration regularly; there is no built-in HA for single-node deployments.
- Health score thresholds are global; tune them per site if default thresholds generate excessive alerts.
- PnP works best with DHCP option 43; DNS-based discovery adds latency and requires SRV record configuration.
- Template variables are case-sensitive; use consistent naming conventions across all templates.
- AI Network Analytics requires Cisco cloud connectivity; air-gapped deployments only get on-prem NDP analytics.
- Catalyst Center manages the control plane; always verify data plane behavior independently with client testing.
- Use Command Runner API sparingly; it generates SSH sessions to devices and does not scale for polling.

## See Also

- cisco-sd-access, cisco-ise, cisco-wlc, snmp, netconf, restconf, radius, tacacs, qos

## References

- [Cisco Catalyst Center Administrator Guide](https://www.cisco.com/c/en/us/td/docs/cloud-systems-management/network-automation-and-management/catalyst-center/index.html)
- [Cisco DNA Center API Reference](https://developer.cisco.com/docs/dna-center/)
- [Cisco DNA Center Design Guide (CVD)](https://www.cisco.com/c/en/us/td/docs/solutions/CVD/Campus/cisco-campus-network-design-guide.html)
- [Cisco SD-Access Solution Design Guide](https://www.cisco.com/c/en/us/td/docs/solutions/CVD/Campus/cisco-sda-design-guide.html)
- [Cisco pxGrid Integration Guide](https://developer.cisco.com/docs/pxgrid/)
- [Cisco DNA Center on DevNet](https://developer.cisco.com/dnacenter/)
- [Cisco Catalyst Center Release Notes](https://www.cisco.com/c/en/us/support/cloud-systems-management/dna-center/products-release-notes-list.html)
