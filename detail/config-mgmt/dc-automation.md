# Data Center Automation --- Programmable DC Infrastructure

> *Data center automation transforms manual, error-prone network provisioning into deterministic, repeatable workflows. This deep dive covers the architectural foundations of PowerOn Auto Provisioning (POAP), the control-plane intelligence of DCNM/NDFC, the programmable surface of NX-API, the operational lifecycle model of Day-0/Day-1/Day-2, and the integration patterns with Ansible, Python, and Terraform that make modern fabric management tractable at scale.*

---

## 1. The Operational Lifecycle Model (Day-0 / Day-1 / Day-2)

### Defining the Boundaries

The Day-N model partitions the lifecycle of a network device into three distinct operational phases, each with different automation requirements, risk profiles, and tooling constraints.

**Day-0 (Initial Provisioning)** covers everything from the moment a switch is physically racked until it has a management IP, a base OS image, and enough configuration to be remotely reachable. The defining characteristic of Day-0 is that the device has no prior state --- it is a blank slate. Automation at this stage must be self-bootstrapping: the device cannot reach out to a controller or management station until the automation itself provides connectivity. This is the domain of POAP, ZTP (Zero Touch Provisioning), and USB-based imaging.

**Day-1 (Configuration Deployment)** begins once the device is reachable over the network and ends when the device is fully participating in the production fabric. This phase deploys the functional configuration: routing protocols (OSPF, IS-IS, BGP), overlay protocols (VXLAN EVPN), tenant provisioning (VRFs, VLANs, SVIs), access policies (ACLs, QoS), and verification. Day-1 automation assumes network reachability to the device and typically uses SSH, NX-API, NETCONF, or a fabric controller like NDFC.

**Day-2 (Steady-State Operations)** is the longest phase, spanning the entire operational life of the device. It encompasses monitoring, compliance checking, change management, capacity planning, software upgrades, and incident response. Day-2 automation must be continuous, non-disruptive, and idempotent. It runs on schedules (backup, compliance audit) and on events (syslog trigger, threshold breach).

### Risk Profile by Phase

| Phase | Risk Level | Blast Radius | Rollback Complexity | Automation Maturity |
|:---:|:---:|:---:|:---:|:---:|
| Day-0 | Low | Single device | Re-image from scratch | High (POAP is deterministic) |
| Day-1 | High | Fabric-wide | Config rollback, checkpoint | Medium (requires testing) |
| Day-2 | Medium | Varies by change | Depends on change type | High (well-understood patterns) |

The risk asymmetry is important: Day-0 failures affect one switch and are trivially recoverable (re-POAP), while Day-1 mistakes can black-hole traffic across an entire fabric. This is why Day-1 changes demand pre/post validation, staged rollouts, and config diff review.

### Automation Tool Mapping

| Phase | Primary Tools | Secondary Tools |
|:---|:---|:---|
| Day-0 | POAP, ZTP, USB imaging | PXE boot, iPXE, console servers |
| Day-1 | NDFC, Ansible, Terraform (ACI) | NX-API scripts, NETCONF/YANG |
| Day-2 | Streaming telemetry, NDFC compliance | Syslog/SNMP, custom scripts, ServiceNow |

---

## 2. PowerOn Auto Provisioning (POAP) --- Architecture and Process

### The POAP State Machine

POAP is a firmware-level feature in NX-OS that activates when a Nexus switch boots without a startup configuration file. The process is a well-defined state machine with the following transitions:

```
POWER_ON
  │
  ▼
CHECK_STARTUP_CONFIG ──[exists]──► NORMAL_BOOT
  │
  [missing]
  │
  ▼
DHCP_DISCOVERY ──[timeout]──► RETRY (exponential backoff)
  │
  [offer received]
  │
  ▼
SCRIPT_DOWNLOAD ──[failure]──► RETRY_DOWNLOAD
  │
  [success]
  │
  ▼
SCRIPT_EXECUTE ──[failure]──► ABORT (log error, retry POAP)
  │
  [success]
  │
  ▼
IMAGE_INSTALL ──[skip if current]──► CONFIG_APPLY
  │                                      │
  [install + reboot]                     ▼
  │                               SAVE_STARTUP
  ▼                                      │
SECOND_BOOT ──► CONFIG_APPLY ──►   POAP_COMPLETE
```

### DHCP Interaction Details

The POAP DHCP exchange is a standard DHCPv4 DORA sequence with Cisco-specific options:

| DHCP Field | Purpose | Example Value |
|:---|:---|:---|
| Option 60 (Vendor Class) | Identifies the device as NX-OS | `Cisco NX-OS` |
| Option 66 (TFTP Server) | Script server address | `10.1.1.50` |
| Option 67 (Boot File) | POAP script filename | `poap_script.py` |
| Option 150 (TFTP Server) | Alternative to Option 66 | `10.1.1.50` |
| Option 12 (Hostname) | Requested hostname | `leaf-01` |
| Option 15 (Domain Name) | Domain suffix | `dc.example.com` |

The switch tries multiple protocols in order to download the script: TFTP first, then HTTP, then SCP. The script itself determines how to download the image and configuration, allowing full flexibility in the provisioning logic.

### Script Server Architecture

A production POAP deployment requires a script server that maps incoming switches to their intended configuration. The mapping can be done by any of these identifiers:

1. **Serial number** --- most reliable, hardware-bound, requires pre-population of the serial-to-config mapping before the switch arrives
2. **MAC address** --- reliable, but requires knowing the mgmt0 MAC in advance
3. **DHCP-assigned hostname** --- requires per-switch DHCP reservations
4. **Interface/CDP neighbor** --- the script can run `show cdp neighbors` to determine its position in the topology and self-assign a role

The serial-number approach is the most common in production because it creates a deterministic, one-to-one mapping between physical hardware and logical identity.

### POAP Script Capabilities

The POAP execution environment provides a Python interpreter (Python 2.7 or 3.x depending on NX-OS version) with the `poap` module, which exposes:

| Function | Purpose |
|:---|:---|
| `poap.poap_cli(cmd)` | Execute NX-OS CLI commands, return output |
| `poap.poap_log(msg)` | Write to POAP log (visible in `show poap log`) |
| `poap.set_poap_script_done()` | Signal successful completion |
| `poap.poap_cli_error(msg)` | Report error and optionally abort |

The script runs in a limited environment: no pip, no external libraries, only stdlib and the poap module. All file transfers must use NX-OS CLI copy commands rather than Python socket libraries.

### POAP Security Considerations

POAP has a significant security surface that must be addressed in production:

1. **DHCP trust**: any device on the POAP VLAN that responds to DHCP can hijack the provisioning process. Mitigate with DHCP snooping on upstream switches and dedicated OOB management networks.
2. **Script integrity**: the downloaded script executes with full privilege on the switch. Use SCP (not TFTP) for encrypted transfer, and validate script checksums in the DHCP response or within a wrapper script.
3. **Credential exposure**: POAP scripts often contain credentials for SCP servers. Store credentials in a separate file downloaded first, or use token-based authentication with short-lived tokens.
4. **Network isolation**: the POAP VLAN should be isolated from production traffic to prevent lateral movement if a rogue device enters the provisioning network.

---

## 3. DCNM / NDFC --- Fabric Controller Architecture

### Evolution: DCNM to NDFC

Cisco Data Center Network Manager (DCNM) was a standalone Java application that managed NX-OS and MDS switches. In 2021, Cisco rebranded and re-architected it as Nexus Dashboard Fabric Controller (NDFC), running as a microservices application on the Nexus Dashboard platform.

| Aspect | DCNM (Legacy) | NDFC (Current) |
|:---|:---|:---|
| Deployment | Standalone VM / bare-metal | Nexus Dashboard app (Kubernetes) |
| Architecture | Monolithic Java | Microservices (containers) |
| Platform | CentOS VM | Nexus Dashboard (RHEL-based) |
| Scale | ~500 switches | ~500+ switches |
| API | REST (v10/v11) | REST (v12+), aligned with ND |
| Multi-site | DCNM federation | ND Orchestrator integration |
| Telemetry | Separate install | Integrated via ND Insights |

### Fabric Templates and Profiles

NDFC uses a template-driven approach to fabric management. Each fabric type has a template that defines the intended configuration for every role in the fabric:

**Easy Fabric (VXLAN EVPN)** --- the most common template for greenfield data centers:
- Spine: BGP route reflector, OSPF/IS-IS underlay, PIM RP (if multicast)
- Leaf: BGP EVPN peer, VXLAN VTEP (NVE interface), anycast gateway
- Border Leaf: external connectivity (VRF-lite, MPLS handoff), route leaking
- Border Gateway: multi-site EVPN (BGW), DCI

**Classic LAN** --- for non-VXLAN deployments (traditional L2/L3):
- Core / Distribution / Access roles
- Spanning tree root placement
- HSRP/VRRP gateway assignment

**External Fabric** --- for brownfield or third-party devices:
- Minimal management (inventory, config backup)
- No template-driven config push
- Useful for integrating legacy equipment into NDFC visibility

### Config Compliance Engine

NDFC maintains two configuration states for every managed switch:

1. **Intended Configuration** --- generated by NDFC from the fabric template, switch role, and user-defined parameters (VLANs, VRFs, interfaces). This is what NDFC believes the switch should be running.

2. **Running Configuration** --- periodically pulled from the switch via NX-API or SSH.

The compliance engine performs a structured diff between these two states:

```
Intended Config          Running Config
      │                       │
      ▼                       ▼
  ┌────────────────────────────────┐
  │       Structured Diff Engine   │
  │  (section-aware, order-aware)  │
  └────────────┬───────────────────┘
               │
        ┌──────┴──────┐
        │             │
   [No Diff]    [Diff Found]
        │             │
        ▼             ▼
   COMPLIANT    NON-COMPLIANT
                      │
              ┌───────┴────────┐
              │                │
        [Auto-remediate]  [Alert Only]
              │                │
              ▼                ▼
        Config Deploy    Operator Review
```

The diff is not a naive line-by-line comparison. NDFC understands NX-OS configuration structure --- it knows that `interface Ethernet1/1` is a section parent, that `switchport mode trunk` and `switchport trunk allowed vlan 100` are children of that section, and that command ordering within a section may not matter. This structural awareness reduces false-positive drift alerts.

### NDFC Deployment Modes

| Mode | Description | Use Case |
|:---|:---|:---|
| Managed | Full lifecycle control (config push, compliance, image) | Greenfield VXLAN fabrics |
| Monitored | Read-only (inventory, topology, config backup) | Brownfield, gradual migration |
| Hybrid | Some switches managed, others monitored | Phased migration |

---

## 4. NX-API --- The Programmable Surface

### NX-API Architecture

NX-API is a feature of NX-OS that exposes the switch's CLI and data model over HTTP/HTTPS. It runs as an embedded web server (nginx) on the switch, translating API requests into internal CLI calls or DME (Data Management Engine) queries.

```
External Client (curl, Python, Ansible)
        │
        ▼
    HTTPS (443)
        │
        ▼
┌─────────────────────────────────────┐
│           NX-API Frontend           │
│         (nginx + auth module)       │
├─────────────────────────────────────┤
│      Message Broker / Dispatcher    │
├─────────┬───────────┬───────────────┤
│ CLI     │ JSON-RPC  │  REST (DME)   │
│ Backend │ Handler   │  Handler      │
├─────────┴───────────┴───────────────┤
│         NX-OS CLI Engine            │
│           or DME Query              │
└─────────────────────────────────────┘
```

### Three API Styles

NX-API exposes three distinct interaction styles, each suited to different use cases:

**1. NX-API CLI (JSON-RPC style)** --- wraps NX-OS CLI commands in a JSON envelope. Input is the exact CLI command string; output is structured JSON. This is the easiest to adopt because it reuses existing CLI knowledge.

- Endpoint: `POST /ins`
- Types: `cli_show` (show commands), `cli_show_ascii` (raw text), `cli_conf` (config mode)
- Best for: rapid automation of existing CLI workflows, one-off scripts

**2. NX-API REST (DME model)** --- a RESTful interface to the NX-OS Data Management Engine, which represents the switch configuration as a hierarchical object tree (similar to ACI's MIT). Objects are addressed by Distinguished Names (DNs).

- Endpoint: `GET/POST/PUT/DELETE /api/mo/{dn}.json`
- Best for: CRUD operations on specific objects, integration with REST-native tools

**3. NX-API JSON-RPC 2.0** --- a standards-compliant JSON-RPC 2.0 interface, useful for batch operations and when strict JSON-RPC tooling is required.

- Endpoint: `POST /jsonrpc`
- Best for: batch operations, JSON-RPC client libraries

### DME Object Model

The NX-OS DME organizes all switch state into a Management Information Tree (MIT), a hierarchical namespace:

```
topRoot
└── topSystem (sys)
    ├── bgpEntity (bgp)
    │   └── bgpInst
    │       └── bgpDom (default)
    │           └── bgpPeer (10.1.1.1)
    │               └── bgpPeerAf (ipv4-ucast)
    ├── interfaceEntity (intf)
    │   ├── l1PhysIf (phys-[eth1/1])
    │   ├── l1PhysIf (phys-[eth1/2])
    │   └── sviIf (svi-[vlan100])
    ├── bdEntity (bd)
    │   └── l2BD (vlan-100)
    ├── ipv4Entity (ipv4)
    │   └── ipv4Dom (default)
    │       └── ipv4Route (10.0.0.0/8)
    └── nveEntity (nve)
        └── nveIf (nve1)
            └── nveVni (5100)
```

Every object in the tree has a Distinguished Name (DN) constructed from the path:

| Object | DN |
|:---|:---|
| System | `sys` |
| Interface eth1/1 | `sys/intf/phys-[eth1/1]` |
| VLAN 100 | `sys/bd/bd-[vlan-100]` |
| BGP neighbor | `sys/bgp/inst/dom-[default]/peer-[10.1.1.1]` |
| NVE VNI 5100 | `sys/nve/nve1/vni-[5100]` |

### NX-API Performance Characteristics

| Operation | Typical Latency | Concurrent Limit |
|:---|:---:|:---:|
| Single show command | 100-500 ms | 8 sessions default |
| Batch show (10 commands) | 500-2000 ms | Shared session pool |
| Config command (single) | 200-1000 ms | Serialized internally |
| DME GET (single object) | 50-200 ms | 8 sessions |
| DME GET (subtree query) | 200-5000 ms | Depends on tree depth |

NX-API has a configurable session limit (default 8 concurrent HTTP sessions). Exceeding this returns HTTP 503. For high-throughput automation, batch commands in single requests or use connection pooling with keep-alive.

---

## 5. Ansible for NX-OS --- Module Architecture and Patterns

### The cisco.nxos Collection

The `cisco.nxos` Ansible collection provides resource modules that follow the Ansible network resource module model. Each module manages a specific NX-OS feature and supports four states:

| State | Behavior |
|:---|:---|
| `merged` | Add/update configuration without removing existing entries |
| `replaced` | Replace configuration for specified resources (leave others untouched) |
| `overridden` | Replace ALL configuration for the resource type (remove unspecified entries) |
| `deleted` | Remove specified configuration |
| `gathered` | Read current configuration into structured data (no changes) |
| `rendered` | Generate CLI commands without applying them |
| `parsed` | Parse offline config text into structured data |

### Connection Methods

| Method | Connection Plugin | Transport | Best For |
|:---|:---|:---|:---|
| NX-API (HTTPS) | `ansible.netcommon.httpapi` | HTTPS | Structured output, speed |
| SSH (CLI) | `ansible.netcommon.network_cli` | SSH | Legacy, no NX-API |
| NETCONF | `ansible.netcommon.netconf` | SSH (830) | YANG model-driven |

The httpapi connection is preferred because it uses NX-API under the hood, providing structured JSON responses that Ansible modules can parse without TextFSM or regex. The SSH/CLI connection falls back to screen-scraping, which is slower and more fragile.

### Idempotency in Network Automation

Ansible's idempotency model for network devices differs from server automation. Network modules must:

1. **Gather current state** (run show commands or API calls)
2. **Compare desired state** (from playbook variables) against current state
3. **Generate minimal diff** (only the commands needed to converge)
4. **Apply diff** (push only changed configuration)
5. **Verify convergence** (re-gather state and confirm match)

This is why network modules have `state: gathered` --- it separates the read operation from the write, allowing operators to inspect current state before making changes.

### Role-Based Playbook Architecture

A production NX-OS automation project should follow a role-based structure:

```
site/
├── inventory/
│   ├── hosts.yml             # Device inventory
│   ├── group_vars/
│   │   ├── all.yml           # Global variables
│   │   ├── spines.yml        # Spine-specific vars
│   │   └── leafs.yml         # Leaf-specific vars
│   └── host_vars/
│       ├── leaf-01.yml       # Per-device overrides
│       └── leaf-02.yml
├── roles/
│   ├── base/                 # Day-0: hostname, NTP, DNS, AAA
│   ├── underlay/             # Day-1: OSPF/IS-IS, PIM, loopbacks
│   ├── overlay/              # Day-1: BGP EVPN, NVE, VNIs
│   ├── tenants/              # Day-1: VRFs, VLANs, SVIs
│   ├── access_ports/         # Day-1: host-facing port config
│   ├── compliance/           # Day-2: config audit
│   └── backup/               # Day-2: config backup
├── playbooks/
│   ├── day0_provision.yml
│   ├── day1_fabric.yml
│   ├── day2_compliance.yml
│   └── day2_backup.yml
└── ansible.cfg
```

### Jinja2 Templates for NX-OS

When resource modules do not cover a feature, `cisco.nxos.nxos_config` with Jinja2 templates provides a fallback:

```jinja2
{# templates/underlay.j2 #}
{% for intf in underlay_interfaces %}
interface {{ intf.name }}
  description {{ intf.description }}
  no switchport
  ip address {{ intf.ip }}/{{ intf.mask }}
  ip ospf network point-to-point
  ip router ospf {{ ospf_process }} area {{ ospf_area }}
  no shutdown
{% endfor %}

router ospf {{ ospf_process }}
  router-id {{ loopback0_ip }}
  log-adjacency-changes detail
  area {{ ospf_area }} range {{ ospf_summary }}
```

---

## 6. Python Automation Ecosystem

### Library Comparison

| Library | Transport | Output | Config Push | Multi-Vendor | Best For |
|:---|:---:|:---:|:---:|:---:|:---|
| requests (NX-API) | HTTPS | Structured JSON | Yes | No (NX-OS only) | Direct NX-API control |
| Netmiko | SSH | Raw text + TextFSM | Yes | Yes (70+ platforms) | Multi-vendor CLI automation |
| NAPALM | HTTPS/SSH | Structured dicts | Yes (merge/replace) | Yes (6 core drivers) | Vendor-neutral operations |
| ncclient | NETCONF | XML/YANG | Yes | Yes (NETCONF devices) | Model-driven automation |
| pyATS/Genie | SSH/API | Parsed models | Limited | Yes (Cisco focus) | Testing, verification |

### When to Use Each Library

**Use raw requests + NX-API when:**
- You need maximum performance (lowest overhead)
- You are building a single-purpose tool for NX-OS only
- You want full control over API payloads and error handling

**Use Netmiko when:**
- You need SSH access (NX-API not available or not enabled)
- You are automating multiple vendor platforms via CLI
- You need TextFSM parsing for structured output from CLI text

**Use NAPALM when:**
- You want a vendor-neutral abstraction layer
- You need config diff/merge/replace/rollback semantics
- You are building a tool that must work across NX-OS, IOS-XE, Junos, EOS, etc.

**Use pyATS/Genie when:**
- You need pre/post change verification with structured assertions
- You want Cisco's official test framework for network state validation
- You need to parse complex show command output into Python objects

### Error Handling Patterns

```python
"""Robust NX-API error handling patterns."""

import requests
import time
from requests.exceptions import ConnectionError, Timeout

class NXAPIError(Exception):
    """Raised when NX-API returns an error response."""
    def __init__(self, code, message):
        self.code = code
        self.message = message
        super().__init__(f"NX-API error {code}: {message}")

def nxapi_call_with_retry(session, url, payload, max_retries=3, backoff=2):
    """Execute NX-API call with exponential backoff retry."""
    for attempt in range(max_retries):
        try:
            resp = session.post(url, json=payload, timeout=30)

            if resp.status_code == 503:
                # Session limit exceeded — back off
                wait = backoff ** attempt
                time.sleep(wait)
                continue

            resp.raise_for_status()
            result = resp.json()

            # Check for NX-API-level errors
            output = result["ins_api"]["outputs"]["output"]
            if isinstance(output, list):
                for item in output:
                    if item.get("code") != "200":
                        raise NXAPIError(item["code"], item["msg"])
            else:
                if output.get("code") != "200":
                    raise NXAPIError(output["code"], output["msg"])

            return result

        except (ConnectionError, Timeout) as e:
            if attempt == max_retries - 1:
                raise
            time.sleep(backoff ** attempt)

    raise RuntimeError(f"Failed after {max_retries} attempts")
```

---

## 7. Terraform for ACI --- Infrastructure as Code

### ACI Object Model (MIT)

Cisco ACI uses a Management Information Tree (MIT) that is conceptually similar to the NX-OS DME but far more extensive. The key abstractions:

```
Uni (Universe)
├── Tenant
│   ├── VRF (fvCtx)
│   ├── Bridge Domain (fvBD)
│   │   └── Subnet (fvSubnet)
│   ├── Application Profile (fvAp)
│   │   └── EPG (fvAEPg)
│   │       ├── Static Binding (fvRsPathAtt)
│   │       ├── Domain Association (fvRsDomAtt)
│   │       ├── Provided Contract (fvRsProv)
│   │       └── Consumed Contract (fvRsCons)
│   ├── Contract (vzBrCP)
│   │   └── Subject (vzSubj)
│   │       └── Filter (vzRsSubjFiltAtt)
│   └── L3Out (l3extOut)
│       └── External EPG (l3extInstP)
├── Fabric
│   ├── Access Policies
│   │   ├── VLAN Pool
│   │   ├── Domain (Physical/VMM)
│   │   ├── AEP (attachable entity profile)
│   │   └── Interface Policies
│   └── Fabric Policies
│       ├── NTP
│       ├── DNS
│       └── SNMP
└── Infrastructure
    └── Pod Policy Group
```

### Terraform State Management for ACI

ACI Terraform automation requires careful state management because the ACI fabric is a shared resource with multiple tenants and operators:

1. **State per tenant**: each tenant should have its own Terraform state file to enable independent lifecycle management
2. **Remote state**: use Terraform Cloud, S3, or Consul for shared state locking
3. **Import existing resources**: use `terraform import` to bring brownfield ACI objects under Terraform management without recreating them
4. **Sensitive values**: store APIC credentials in Vault or environment variables, never in `.tf` files

### Terraform Plan/Apply Workflow for ACI

```
Developer writes .tf files
        │
        ▼
terraform plan ──► Review diff
        │              │
        │         [Approve]
        │              │
        ▼              ▼
terraform apply ──► APIC REST API calls
        │
        ▼
State file updated
        │
        ▼
terraform show ──► Verify applied state
```

The `plan` output for ACI resources shows the APIC REST API calls that will be made, including the DN (Distinguished Name) of each object. This is critical for review because ACI's object model means that creating a contract requires creating child objects (subjects, filters) in a specific order.

---

## 8. Configuration Compliance and Drift Detection

### Sources of Configuration Drift

Configuration drift occurs when the running configuration of a device diverges from the intended configuration. Common causes:

1. **Out-of-band CLI changes**: an engineer logs into a switch and makes changes manually, bypassing the automation pipeline
2. **Emergency break-fix**: a production incident requires immediate configuration changes that are not backported to the source of truth
3. **Automation partial failure**: a playbook runs halfway and fails, leaving some devices updated and others not
4. **Software upgrade side effects**: NX-OS upgrades can add default configuration lines or change command syntax
5. **Feature interaction**: enabling a new feature can inject configuration that was not in the intended template

### Compliance Architecture

A robust compliance system has four components:

```
┌─────────────────────────────────────────────────┐
│              Source of Truth (SoT)               │
│   (Git repo, NDFC templates, Ansible vars)       │
└──────────────────────┬──────────────────────────┘
                       │ Generate intended config
                       ▼
              ┌─────────────────┐
              │ Intended Config │
              │   (per device)  │
              └────────┬────────┘
                       │
                       │ Diff
                       │
              ┌────────┴────────┐
              │ Running Config  │◄── Periodic pull (NX-API, SSH)
              │   (per device)  │
              └────────┬────────┘
                       │
                       ▼
              ┌─────────────────┐
              │ Compliance      │
              │ Report          │
              │ - Compliant     │
              │ - Drift details │
              │ - Severity      │
              │ - Remediation   │
              └─────────────────┘
```

### Structured vs Unstructured Diff

Naive line-by-line diff produces many false positives because NX-OS configuration has semantically equivalent representations:

```
# These are equivalent but produce a textual diff:
interface Ethernet1/1
  switchport trunk allowed vlan 100,200,300

interface Ethernet1/1
  switchport trunk allowed vlan 100,200
  switchport trunk allowed vlan add 300
```

A structured compliance tool must:
- Parse configuration into a tree (sections, subsections, commands)
- Normalize command syntax (expand abbreviations, sort VLAN lists)
- Compare semantically, not textually
- Classify differences by severity (critical: routing change, low: description change)

### NDFC Compliance vs Custom Compliance

| Aspect | NDFC Built-in | Custom (Script-based) |
|:---|:---|:---|
| Source of truth | NDFC templates | Git repo / YAML vars |
| Diff engine | Structured, section-aware | Depends on implementation |
| Remediation | One-click config deploy | Manual or Ansible playbook |
| Scope | NDFC-managed switches only | Any reachable device |
| Customization | Limited to NDFC template params | Fully customizable |
| Alerting | NDFC dashboard / email | Webhook, Slack, PagerDuty |

---

## 9. Streaming Telemetry and Day-2 Monitoring

### Telemetry vs SNMP

| Aspect | SNMP Polling | Streaming Telemetry |
|:---|:---|:---|
| Model | Pull (manager polls agent) | Push (device streams data) |
| Latency | Poll interval (30s-5min typical) | Sub-second (event-driven) |
| Overhead | Each poll is a full request/response | Persistent gRPC session |
| Data model | MIB (ASN.1) | YANG (structured, typed) |
| Scalability | Degrades with device count | Scales linearly |
| Encoding | BER (binary, complex) | Protobuf / JSON (compact, simple) |

### NX-OS Telemetry Configuration

```
! Enable telemetry feature
feature telemetry

! Define sensor group (what to stream)
telemetry
  sensor-group 100
    data-source DME
    path sys/intf depth unbounded
  sensor-group 200
    data-source DME
    path sys/bgp depth unbounded

  ! Define destination group (where to stream)
  destination-group 100
    ip address 10.1.1.100 port 57000 protocol gRPC encoding GPB

  ! Define subscription (bind sensor to destination)
  subscription 100
    snsr-grp 100 sample-interval 10000
    dst-grp 100
  subscription 200
    snsr-grp 200 sample-interval 30000
    dst-grp 100
```

### Key Telemetry Sensor Paths

| Sensor Path | Data | Use Case |
|:---|:---|:---|
| `sys/intf` | Interface counters, state | Utilization monitoring |
| `sys/bgp` | BGP neighbor state, prefix count | Routing health |
| `sys/nve` | NVE peer state, VNI counters | VXLAN overlay health |
| `sys/procsys` | CPU, memory utilization | Resource monitoring |
| `sys/ch` | Fan, PSU, temperature | Hardware health |
| `sys/ipv4/inst/dom-default/rt` | IPv4 routing table | Route table growth |
| `sys/mac/table` | MAC address table | MAC table growth, learning |
| `sys/eps` | EVPN state | Multi-site EVPN health |

---

## 10. Integration Patterns and Best Practices

### GitOps for Network Automation

The GitOps model applies infrastructure-as-code principles to network configuration:

```
Developer               Git Repo              CI/CD Pipeline           Network
   │                       │                       │                     │
   │── Commit change ─────►│                       │                     │
   │                       │── Webhook trigger ───►│                     │
   │                       │                       │── Lint + validate ──│
   │                       │                       │── Plan (dry-run) ──►│
   │                       │                       │     (diff only)     │
   │                       │                       │                     │
   │◄── Review PR ────────│◄── Post results ──────│                     │
   │── Approve + merge ──►│                       │                     │
   │                       │── Webhook trigger ───►│                     │
   │                       │                       │── Apply config ────►│
   │                       │                       │── Verify + test ───►│
   │                       │                       │── Report status ───►│
   │◄── Notification ─────│◄── Update status ─────│                     │
```

### CI/CD Pipeline for Network Changes

A production-grade pipeline for NX-OS automation should include:

1. **Lint** --- validate YAML syntax, Jinja2 templates, variable completeness
2. **Static analysis** --- check for conflicting VLANs, overlapping subnets, missing dependencies
3. **Dry-run** --- execute with `--check --diff` (Ansible) or `plan` (Terraform)
4. **Lab test** --- apply to a VIRL/CML lab environment and run integration tests
5. **Staged rollout** --- apply to canary devices first, verify, then proceed to full fabric
6. **Post-change validation** --- run pyATS/Genie tests to verify BGP, NVE, reachability
7. **Rollback gate** --- automatic rollback if post-change validation fails

### Multi-Tool Orchestration

Most production data center automation uses multiple tools in concert:

| Layer | Tool | Scope |
|:---|:---|:---|
| Source of Truth | NetBox | IPAM, device inventory, topology |
| Version Control | Git | Config templates, variables, playbooks |
| Orchestration | Ansible AWX/Tower | Playbook execution, scheduling, RBAC |
| Fabric Controller | NDFC | VXLAN fabric lifecycle, compliance |
| Policy Controller | ACI + Terraform | Application-centric policy |
| Monitoring | Telegraf + InfluxDB + Grafana | Streaming telemetry visualization |
| Compliance | Custom scripts + NDFC | Drift detection, audit |
| ITSM | ServiceNow | Change management, ticketing |

---

## Prerequisites

networking-fundamentals, vxlan, bgp, ansible, python-networking, terraform, rest-apis

## Complexity

| Operation | Time | Notes |
|:---|:---:|:---|
| POAP boot-to-configured | O(1) per switch | ~10-20 min depending on image size |
| NDFC fabric discovery | O(n) switches | Initial import scans all devices |
| NDFC config compliance check | O(n) switches | Diff per device, parallelized |
| NX-API show command | O(1) | 100-500 ms per command |
| NX-API config push | O(k) commands | Serialized on switch, 200ms per command |
| Ansible playbook (full fabric) | O(n * k) | n devices, k tasks, parallelized by forks |
| Terraform plan (ACI tenant) | O(r) resources | r resources in state, API call per resource |
| Config backup (full DC) | O(n) | Parallelized SSH/API, ~5-30s per device |
| Compliance drift check | O(n * L) | L = config lines per device |
| Streaming telemetry setup | O(1) per subscription | Persistent gRPC, sub-second updates |

## References

- [Cisco POAP Configuration Guide](https://www.cisco.com/c/en/us/td/docs/switches/datacenter/nexus9000/sw/93x/fundamentals/cisco-nexus-9000-series-nx-os-fundamentals-configuration-guide-93x/m-using-poap.html)
- [NDFC REST API Reference](https://developer.cisco.com/docs/nexus-dashboard-fabric-controller/)
- [NX-API Developer Guide](https://www.cisco.com/c/en/us/td/docs/switches/datacenter/nexus9000/sw/93x/programmability/guide/b-cisco-nexus-9000-series-nx-os-programmability-guide-93x.html)
- [Cisco NX-OS YANG Models](https://github.com/YangModels/yang/tree/main/vendor/cisco/nx)
- [Ansible cisco.nxos Collection](https://galaxy.ansible.com/ui/repo/published/cisco/nxos/)
- [NAPALM Documentation](https://napalm.readthedocs.io/en/latest/)
- [Terraform ACI Provider](https://registry.terraform.io/providers/CiscoDevNet/aci/latest/docs)
- [pyATS Documentation](https://developer.cisco.com/docs/pyats/)
- [RFC 8040 --- RESTCONF Protocol](https://datatracker.ietf.org/doc/html/rfc8040)
- [RFC 6241 --- NETCONF Protocol](https://datatracker.ietf.org/doc/html/rfc6241)
