# Data Center Automation (POAP, DCNM, NX-API, Day-0 Provisioning)

Automated provisioning, configuration, and lifecycle management of data center network infrastructure using Cisco NX-OS tooling, programmable APIs, and infrastructure-as-code workflows spanning Day-0 through Day-2 operations.

## PowerOn Auto Provisioning (POAP)

### POAP Boot Process

```
1. Switch powers on with no startup-config
2. POAP process starts automatically
3. Switch sends DHCP Discover (Option 60: "Cisco NX-OS")
4. DHCP server responds with:
   - IP address for mgmt0
   - Default gateway
   - Option 150: TFTP server IP  (script server)
   - Option 67: Boot filename    (POAP script path)
5. Switch downloads POAP script via TFTP/HTTP/SCP
6. Script executes (Python or TCL)
7. Script downloads system image + configuration
8. Switch installs image, applies config, reboots
```

### DHCP Configuration for POAP

```
# ISC DHCP Server — /etc/dhcp/dhcpd.conf
subnet 10.1.1.0 netmask 255.255.255.0 {
    range 10.1.1.100 10.1.1.200;
    option routers 10.1.1.1;
    option domain-name-servers 10.1.1.10;

    # POAP options
    option tftp-server-name "10.1.1.50";
    option bootfile-name "poap_script.py";

    # Match NX-OS devices specifically
    class "nxos-devices" {
        match if option vendor-class-identifier = "Cisco NX-OS";
        option bootfile-name "poap_nexus.py";
    }
}

# Per-switch reservations for deterministic assignment
host leaf-01 {
    hardware ethernet 00:50:56:ab:cd:01;
    fixed-address 10.1.1.101;
    option host-name "leaf-01";
    option bootfile-name "poap_leaf.py";
}
```

### POAP Python Script (Minimal)

```python
#!/usr/bin/env python
"""
Minimal POAP script for NX-OS Day-0 provisioning.
Runs on the switch during POAP boot sequence.
"""

import poap  # NX-OS built-in POAP module
import os
import sys

# Configuration server details
CONFIG_SERVER = "10.1.1.50"
CONFIG_PROTOCOL = "scp"
CONFIG_USER = "poap"
CONFIG_PASS = "P0@p$ecure"

IMAGE_SERVER = "10.1.1.50"
IMAGE_PATH = "/images/nxos.10.3.4a.bin"
CONFIG_PATH = "/configs/"

def get_serial():
    """Get switch serial number for config lookup."""
    output = poap.poap_cli("show inventory chassis")
    for line in output.split("\n"):
        if "SN:" in line:
            return line.split("SN:")[1].strip()
    return None

def download_image():
    """Download and install NX-OS system image."""
    poap.poap_log("Downloading system image...")
    poap.poap_cli(
        "copy %s://%s@%s%s bootflash:nxos.bin vrf management" %
        (CONFIG_PROTOCOL, CONFIG_USER, IMAGE_SERVER, IMAGE_PATH)
    )
    poap.poap_cli("install all nxos bootflash:nxos.bin")

def download_config():
    """Download configuration based on serial number."""
    serial = get_serial()
    config_file = "%s/%s.cfg" % (CONFIG_PATH, serial)

    poap.poap_log("Downloading config for serial: %s" % serial)
    poap.poap_cli(
        "copy %s://%s@%s%s scheduled-config vrf management" %
        (CONFIG_PROTOCOL, CONFIG_USER, CONFIG_SERVER, config_file)
    )

def main():
    poap.poap_log("=== POAP Script Starting ===")
    download_image()
    download_config()
    poap.poap_log("=== POAP Complete — Rebooting ===")

if __name__ == "__main__":
    main()
```

### POAP TCL Script (Legacy)

```tcl
#!/usr/bin/env tclsh
# Legacy POAP TCL script for older NX-OS versions

proc poap_init {} {
    set serial [exec "show inventory chassis" | grep "SN:" | cut -d: -f2]
    set config_server "10.1.1.50"
    set config_file "/configs/${serial}.cfg"

    # Download configuration
    exec "copy scp://$config_server$config_file scheduled-config vrf management"

    # Download system image
    exec "copy scp://$config_server/images/nxos.bin bootflash:nxos.bin vrf management"
    exec "install all nxos bootflash:nxos.bin"
}

poap_init
```

### Verify POAP Status

```bash
# Check POAP status on the switch
show boot poap status

# View POAP log
show poap log

# Enable/disable POAP
poap enable
no poap enable

# Check scheduled-config (downloaded by POAP)
show scheduled-config
```

## DCNM / NDFC (Nexus Dashboard Fabric Controller)

### DCNM Architecture

```
Nexus Dashboard (Platform)
├── NDFC Application (formerly DCNM)
│   ├── Fabric Controller
│   │   ├── Fabric Builder     — design + deploy fabrics
│   │   ├── Topology View      — visual switch/link map
│   │   ├── Image Management   — ISSU, EPLD upgrades
│   │   └── Config Compliance  — intended vs running
│   ├── SAN Controller
│   │   ├── MDS Management     — zoning, VSANs
│   │   └── SAN Analytics      — flow telemetry
│   └── LAN Controller
│       ├── VXLAN EVPN Fabric  — underlay + overlay
│       ├── Classic LAN        — non-VXLAN fabrics
│       └── External Fabric    — third-party devices
├── Nexus Dashboard Insights
├── Nexus Dashboard Orchestrator
└── Nexus Dashboard Data Broker
```

### NDFC REST API (Fabric Operations)

```bash
# Authenticate and get token
TOKEN=$(curl -sk -X POST \
  "https://ndfc.example.com/login" \
  -H "Content-Type: application/json" \
  -d '{"userName":"admin","userPasswd":"C1sc0!23"}' \
  | jq -r '.token')

# List all fabrics
curl -sk -X GET \
  "https://ndfc.example.com/appcenter/cisco/ndfc/api/v1/lan-fabric/rest/control/fabrics" \
  -H "Authorization: Bearer $TOKEN" | jq '.[]|{fabricName,fabricType,templateName}'

# Get fabric inventory (switches)
curl -sk -X GET \
  "https://ndfc.example.com/appcenter/cisco/ndfc/api/v1/lan-fabric/rest/control/fabrics/DC1/inventory" \
  -H "Authorization: Bearer $TOKEN" | jq '.[]|{serialNumber,switchName,ipAddress,model,release}'

# Deploy pending configurations
curl -sk -X POST \
  "https://ndfc.example.com/appcenter/cisco/ndfc/api/v1/lan-fabric/rest/control/fabrics/DC1/config-deploy" \
  -H "Authorization: Bearer $TOKEN"

# Get config compliance status
curl -sk -X GET \
  "https://ndfc.example.com/appcenter/cisco/ndfc/api/v1/lan-fabric/rest/control/fabrics/DC1/config-compliance" \
  -H "Authorization: Bearer $TOKEN" | jq '.[]|{switchName,status,diffCount}'

# Recalculate config for a switch
curl -sk -X POST \
  "https://ndfc.example.com/appcenter/cisco/ndfc/api/v1/lan-fabric/rest/control/fabrics/DC1/config-save" \
  -H "Authorization: Bearer $TOKEN"
```

### NDFC Image Management

```bash
# List available images in the repository
curl -sk -X GET \
  "https://ndfc.example.com/appcenter/cisco/ndfc/api/v1/imagemanagement/rest/imagemanagement/image" \
  -H "Authorization: Bearer $TOKEN" | jq '.[]|{imageName,version,platform}'

# Upload a new NX-OS image
curl -sk -X POST \
  "https://ndfc.example.com/appcenter/cisco/ndfc/api/v1/imagemanagement/rest/imagemanagement/image/upload" \
  -H "Authorization: Bearer $TOKEN" \
  -F "file=@nxos64-cs.10.3.4a.M.bin"

# Set image policy for a switch group
curl -sk -X POST \
  "https://ndfc.example.com/appcenter/cisco/ndfc/api/v1/imagemanagement/rest/policymgnt/platform-policy" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "policyName": "N9K-10.3.4a",
    "nxosVersion": "10.3(4a)",
    "platform": "N9K",
    "packageName": "",
    "epldImgName": ""
  }'

# Trigger ISSU upgrade
curl -sk -X POST \
  "https://ndfc.example.com/appcenter/cisco/ndfc/api/v1/imagemanagement/rest/imagemanagement/upgrade" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"devices":[{"serialNumber":"SAL12345678","policyName":"N9K-10.3.4a"}]}'
```

## NX-API (Programmable Interface)

### Enable NX-API on Switch

```
! Enable NX-API (HTTPS by default on port 443)
feature nxapi

! Optional: change ports, enable sandbox
nxapi http port 8080
nxapi https port 8443
nxapi sandbox

! Restrict access with ACL
nxapi use-vrf management
ip access-list NX-API-ACL
  permit ip 10.1.0.0/16 any
nxapi access-class NX-API-ACL
```

### NX-API CLI Method (JSON-RPC)

```bash
# JSON-RPC request — show version
curl -sk -X POST "https://switch.example.com/ins" \
  -u admin:password \
  -H "Content-Type: application/json" \
  -d '{
    "ins_api": {
      "version": "1.0",
      "type": "cli_show",
      "chunk": "0",
      "sid": "1",
      "input": "show version",
      "output_format": "json"
    }
  }' | jq '.ins_api.outputs.output.body'

# Multiple commands in one request
curl -sk -X POST "https://switch.example.com/ins" \
  -u admin:password \
  -H "Content-Type: application/json" \
  -d '{
    "ins_api": {
      "version": "1.0",
      "type": "cli_show",
      "chunk": "0",
      "sid": "1",
      "input": "show vlan brief ;show interface status",
      "output_format": "json"
    }
  }'

# Configuration command
curl -sk -X POST "https://switch.example.com/ins" \
  -u admin:password \
  -H "Content-Type: application/json" \
  -d '{
    "ins_api": {
      "version": "1.0",
      "type": "cli_conf",
      "chunk": "0",
      "sid": "1",
      "input": "interface eth1/1 ;description Uplink-to-Spine ;no shutdown",
      "output_format": "json"
    }
  }'
```

### NX-API REST (DME Model)

```bash
# Get system info via DME
curl -sk -X GET "https://switch.example.com/api/mo/sys.json" \
  -u admin:password | jq '.imdata[0].topSystem.attributes|{name,serial,version}'

# Get all interfaces
curl -sk -X GET "https://switch.example.com/api/mo/sys/intf.json?rsp-subtree=children" \
  -u admin:password

# Get specific interface
curl -sk -X GET "https://switch.example.com/api/mo/sys/intf/phys-[eth1/1].json" \
  -u admin:password

# Create a VLAN via DME
curl -sk -X POST "https://switch.example.com/api/mo/sys/bd.json" \
  -u admin:password \
  -H "Content-Type: application/json" \
  -d '{
    "bdEntity": {
      "children": [{
        "l2BD": {
          "attributes": {
            "fabEncap": "vlan-100",
            "name": "Production",
            "adminSt": "active"
          }
        }
      }]
    }
  }'

# Get BGP neighbors
curl -sk -X GET "https://switch.example.com/api/mo/sys/bgp.json?rsp-subtree=full" \
  -u admin:password
```

## Day-0 / Day-1 / Day-2 Operations

### Day-0: Initial Provisioning

```
Day-0 Scope:
├── Hardware rack and stack
├── Physical cabling verification
├── Management IP assignment (OOB or in-band)
├── Base OS image installation (POAP or USB)
├── Initial bootstrap config
│   ├── Hostname, domain name
│   ├── Management interface (mgmt0)
│   ├── NTP, DNS, syslog servers
│   ├── AAA (RADIUS/TACACS+)
│   ├── SSH keys, local admin user
│   └── SNMP community/v3
├── License activation
└── Feature enable (nxapi, nv overlay, etc.)
```

### Day-1: Configuration Deployment

```
Day-1 Scope:
├── Underlay Configuration
│   ├── OSPF / IS-IS for routing
│   ├── PIM for multicast (if needed)
│   ├── Loopback interfaces (RID, VTEP)
│   └── Point-to-point links (unnumbered or /31)
├── Overlay Configuration
│   ├── BGP EVPN control plane
│   ├── VXLAN NVE interface
│   ├── VNI-to-VLAN mapping
│   └── Anycast gateway (SVI + fabric forwarding)
├── Tenant Configuration
│   ├── VRF creation and RD/RT assignment
│   ├── VLAN/SVI provisioning
│   ├── Access port and trunk configuration
│   └── DHCP relay
├── Policy Configuration
│   ├── ACLs (interface and VACL)
│   ├── QoS policies (queuing, marking)
│   └── Port security, storm control
└── Verification
    ├── NVE peering
    ├── BGP EVPN neighbor status
    ├── VXLAN flood-and-learn or ingress replication
    └── End-to-end reachability
```

### Day-2: Monitoring and Compliance

```
Day-2 Scope:
├── Monitoring
│   ├── SNMP polling (interface stats, CPU, memory)
│   ├── Streaming telemetry (gNMI, NX-API DME subscriptions)
│   ├── Syslog aggregation (ELK, Splunk)
│   └── NetFlow / sFlow for traffic analytics
├── Compliance
│   ├── Running vs intended config diff
│   ├── Software version compliance
│   ├── Security baseline audit (CIS benchmarks)
│   └── Hardware lifecycle (EOL/EOS tracking)
├── Change Management
│   ├── Config backup (scheduled, event-driven)
│   ├── Pre/post change validation
│   ├── Rollback procedures
│   └── Maintenance windows
└── Capacity Planning
    ├── Interface utilization trending
    ├── TCAM usage monitoring
    ├── MAC/ARP/route table growth
    └── Power and cooling metrics
```

## Ansible for NX-OS

### Inventory and Variables

```yaml
# inventory/hosts.yml
all:
  children:
    dc_fabric:
      children:
        spines:
          hosts:
            spine-01:
              ansible_host: 10.1.1.11
            spine-02:
              ansible_host: 10.1.1.12
        leafs:
          hosts:
            leaf-01:
              ansible_host: 10.1.1.21
            leaf-02:
              ansible_host: 10.1.1.22
      vars:
        ansible_network_os: cisco.nxos.nxos
        ansible_connection: ansible.netcommon.httpapi
        ansible_httpapi_use_ssl: true
        ansible_httpapi_validate_certs: false
        ansible_user: admin
        ansible_password: "{{ vault_nxos_password }}"
```

### Playbook: Day-1 Base Configuration

```yaml
# playbooks/day1_base.yml
---
- name: Day-1 Base Configuration for NX-OS Switches
  hosts: dc_fabric
  gather_facts: false

  tasks:
    - name: Set hostname and domain
      cisco.nxos.nxos_system:
        hostname: "{{ inventory_hostname }}"
        domain_name: dc.example.com

    - name: Configure NTP servers
      cisco.nxos.nxos_ntp_global:
        config:
          servers:
            - server: 10.1.1.10
              vrf: management
              prefer: true
            - server: 10.1.1.11
              vrf: management

    - name: Enable required features
      cisco.nxos.nxos_feature:
        feature: "{{ item }}"
        state: enabled
      loop:
        - nxapi
        - ospf
        - bgp
        - pim
        - interface-vlan
        - vn-segment-vlan-based
        - nv overlay

    - name: Configure VLANs
      cisco.nxos.nxos_vlans:
        config:
          - vlan_id: "{{ item.id }}"
            name: "{{ item.name }}"
            state: active
        state: merged
      loop: "{{ vlans }}"

    - name: Configure L3 interfaces (loopbacks)
      cisco.nxos.nxos_l3_interfaces:
        config:
          - name: loopback0
            ipv4:
              - address: "{{ loopback0_ip }}/32"
          - name: loopback1
            ipv4:
              - address: "{{ vtep_ip }}/32"
        state: merged

    - name: Configure OSPF underlay
      cisco.nxos.nxos_ospfv2:
        config:
          processes:
            - process_id: "1"
              router_id: "{{ loopback0_ip }}"
              areas:
                - area_id: "0.0.0.0"
                  ranges:
                    - prefix: 10.0.0.0/8

    - name: Save configuration
      cisco.nxos.nxos_config:
        save_when: modified
```

### Playbook: VXLAN EVPN Overlay

```yaml
# playbooks/vxlan_overlay.yml
---
- name: Configure VXLAN EVPN Overlay
  hosts: leafs
  gather_facts: false

  tasks:
    - name: Configure NVE interface
      cisco.nxos.nxos_config:
        lines:
          - source-interface loopback1
          - host-reachability protocol bgp
          - "member vni {{ item.vni }}"
          - "  mcast-group {{ item.mcast_group }}"
        parents: interface nve1
      loop: "{{ vxlan_vnis }}"

    - name: Configure EVPN
      cisco.nxos.nxos_config:
        lines:
          - "vni {{ item.vni }} l2"
          - "  rd auto"
          - "  route-target import auto"
          - "  route-target export auto"
        parents: evpn
      loop: "{{ vxlan_vnis }}"

    - name: Configure BGP EVPN neighbors
      cisco.nxos.nxos_bgp_neighbor_address_family:
        config:
          as_number: "{{ bgp_asn }}"
          neighbors:
            - neighbor_address: "{{ item }}"
              address_family:
                - afi: l2vpn
                  safi: evpn
                  send_community:
                    both: true
        state: merged
      loop: "{{ spine_loopbacks }}"
```

## Python Automation

### NX-API with Requests

```python
#!/usr/bin/env python3
"""NX-API automation using raw HTTP requests."""

import requests
import json
import urllib3
urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)

class NXAPIClient:
    def __init__(self, host, username, password, port=443):
        self.base_url = f"https://{host}:{port}"
        self.session = requests.Session()
        self.session.auth = (username, password)
        self.session.verify = False
        self.session.headers.update({"Content-Type": "application/json"})

    def cli_show(self, command):
        """Execute a show command and return parsed JSON."""
        payload = {
            "ins_api": {
                "version": "1.0",
                "type": "cli_show",
                "chunk": "0",
                "sid": "1",
                "input": command,
                "output_format": "json"
            }
        }
        resp = self.session.post(f"{self.base_url}/ins", json=payload)
        resp.raise_for_status()
        return resp.json()["ins_api"]["outputs"]["output"]["body"]

    def cli_conf(self, commands):
        """Execute configuration commands."""
        if isinstance(commands, list):
            commands = " ;".join(commands)
        payload = {
            "ins_api": {
                "version": "1.0",
                "type": "cli_conf",
                "chunk": "0",
                "sid": "1",
                "input": commands,
                "output_format": "json"
            }
        }
        resp = self.session.post(f"{self.base_url}/ins", json=payload)
        resp.raise_for_status()
        return resp.json()

# Usage
switch = NXAPIClient("10.1.1.21", "admin", "password")
version = switch.cli_show("show version")
print(f"Hostname: {version['host_name']}, Version: {version['nxos_ver_str']}")

vlans = switch.cli_show("show vlan brief")
for vlan in vlans["TABLE_vlanbriefxbrief"]["ROW_vlanbriefxbrief"]:
    print(f"  VLAN {vlan['vlanshowbr-vlanid']}: {vlan['vlanshowbr-vlanname']}")
```

### Netmiko for SSH-Based Automation

```python
#!/usr/bin/env python3
"""Netmiko-based NX-OS automation over SSH."""

from netmiko import ConnectHandler

device = {
    "device_type": "cisco_nxos",
    "host": "10.1.1.21",
    "username": "admin",
    "password": "password",
    "timeout": 30,
}

with ConnectHandler(**device) as conn:
    # Show commands
    output = conn.send_command("show ip route vrf all", use_textfsm=True)
    for route in output:
        print(f"{route['network']}/{route['mask']} via {route['nexthop']}")

    # Configuration changes
    config_commands = [
        "interface Ethernet1/48",
        "description Uplink-to-Core",
        "switchport mode trunk",
        "switchport trunk allowed vlan 100-200",
        "no shutdown",
    ]
    conn.send_config_set(config_commands)
    conn.save_config()
```

### NAPALM for Multi-Vendor Abstraction

```python
#!/usr/bin/env python3
"""NAPALM driver for NX-OS — vendor-neutral operations."""

from napalm import get_network_driver

driver = get_network_driver("nxos")
device = driver(
    hostname="10.1.1.21",
    username="admin",
    password="password",
    optional_args={"transport": "https"}
)

device.open()

# Get structured facts
facts = device.get_facts()
print(f"Hostname: {facts['hostname']}, Model: {facts['model']}")

# Get interface counters
interfaces = device.get_interfaces()
for name, data in interfaces.items():
    if data["is_up"]:
        print(f"  {name}: speed={data['speed']}Mbps")

# Config diff and commit
device.load_merge_candidate(
    config="interface Ethernet1/48\n  description Managed-by-NAPALM\n"
)
diff = device.compare_config()
if diff:
    print(f"Pending changes:\n{diff}")
    device.commit_config()
else:
    device.discard_config()

device.close()
```

## Terraform for ACI

### ACI Provider Configuration

```hcl
# main.tf — Terraform for Cisco ACI
terraform {
  required_providers {
    aci = {
      source  = "CiscoDevNet/aci"
      version = "~> 2.13"
    }
  }
}

provider "aci" {
  username = var.apic_username
  password = var.apic_password
  url      = var.apic_url
  insecure = true
}

# Tenant
resource "aci_tenant" "production" {
  name        = "Production"
  description = "Production tenant managed by Terraform"
}

# VRF
resource "aci_vrf" "prod_vrf" {
  tenant_dn   = aci_tenant.production.id
  name        = "Prod-VRF"
  description = "Production VRF"
}

# Bridge Domain
resource "aci_bridge_domain" "web_bd" {
  tenant_dn          = aci_tenant.production.id
  name               = "Web-BD"
  relation_fv_rs_ctx = aci_vrf.prod_vrf.id
}

# Subnet
resource "aci_subnet" "web_subnet" {
  parent_dn = aci_bridge_domain.web_bd.id
  ip        = "10.100.1.1/24"
  scope     = ["public"]
}

# Application Profile
resource "aci_application_profile" "web_app" {
  tenant_dn = aci_tenant.production.id
  name      = "Web-App"
}

# EPGs
resource "aci_application_epg" "web_epg" {
  application_profile_dn = aci_application_profile.web_app.id
  name                   = "Web-EPG"
  relation_fv_rs_bd      = aci_bridge_domain.web_bd.id
}

# Contract
resource "aci_contract" "web_to_db" {
  tenant_dn = aci_tenant.production.id
  name      = "Web-to-DB"
  scope     = "tenant"
}

resource "aci_contract_subject" "https" {
  contract_dn = aci_contract.web_to_db.id
  name        = "HTTPS"
}
```

## Configuration Compliance and Drift Detection

### Compliance Check Script

```python
#!/usr/bin/env python3
"""Configuration compliance checker — compares running config against golden templates."""

import difflib
import json
from datetime import datetime
from netmiko import ConnectHandler

GOLDEN_CONFIGS = {
    "leaf": "templates/leaf_golden.cfg",
    "spine": "templates/spine_golden.cfg",
}

DEVICES = [
    {"host": "10.1.1.21", "role": "leaf", "name": "leaf-01"},
    {"host": "10.1.1.22", "role": "leaf", "name": "leaf-02"},
    {"host": "10.1.1.11", "role": "spine", "name": "spine-01"},
]

def get_running_config(device):
    conn_params = {
        "device_type": "cisco_nxos",
        "host": device["host"],
        "username": "admin",
        "password": "password",
    }
    with ConnectHandler(**conn_params) as conn:
        return conn.send_command("show running-config")

def check_compliance(device):
    """Compare running config against golden template."""
    running = get_running_config(device).splitlines()
    golden_path = GOLDEN_CONFIGS[device["role"]]
    with open(golden_path) as f:
        golden = f.read().splitlines()

    diff = list(difflib.unified_diff(
        golden, running,
        fromfile="golden", tofile="running",
        lineterm=""
    ))

    return {
        "device": device["name"],
        "compliant": len(diff) == 0,
        "drift_lines": len([l for l in diff if l.startswith("+") or l.startswith("-")]),
        "diff": "\n".join(diff[:50]),
        "checked_at": datetime.utcnow().isoformat(),
    }

# Run compliance check
results = [check_compliance(d) for d in DEVICES]
for r in results:
    status = "COMPLIANT" if r["compliant"] else "DRIFT DETECTED"
    print(f"{r['device']}: {status} ({r['drift_lines']} lines differ)")
```

### Scheduled Config Backup

```python
#!/usr/bin/env python3
"""Scheduled config backup with Git versioning."""

import os
import subprocess
from datetime import datetime
from netmiko import ConnectHandler

BACKUP_DIR = "/opt/network-backups/configs"
DEVICES = [
    {"host": "10.1.1.21", "name": "leaf-01"},
    {"host": "10.1.1.22", "name": "leaf-02"},
    {"host": "10.1.1.11", "name": "spine-01"},
]

def backup_device(device):
    conn_params = {
        "device_type": "cisco_nxos",
        "host": device["host"],
        "username": "admin",
        "password": "password",
    }
    with ConnectHandler(**conn_params) as conn:
        config = conn.send_command("show running-config")

    filepath = os.path.join(BACKUP_DIR, f"{device['name']}.cfg")
    with open(filepath, "w") as f:
        f.write(config)
    return filepath

def git_commit(message):
    subprocess.run(["git", "-C", BACKUP_DIR, "add", "-A"], check=True)
    result = subprocess.run(
        ["git", "-C", BACKUP_DIR, "diff", "--cached", "--quiet"],
        capture_output=True
    )
    if result.returncode != 0:
        subprocess.run(
            ["git", "-C", BACKUP_DIR, "commit", "-m", message],
            check=True
        )
        return True
    return False

# Backup all devices
timestamp = datetime.utcnow().strftime("%Y-%m-%d %H:%M UTC")
for device in DEVICES:
    backup_device(device)
    print(f"Backed up {device['name']}")

if git_commit(f"Config backup: {timestamp}"):
    print(f"Changes committed at {timestamp}")
else:
    print("No configuration changes detected")
```

## Tips

- Always test POAP scripts on a single switch before deploying fleet-wide; use a lab switch or VIRL/CML
- Set POAP DHCP reservations per-switch (MAC-to-IP binding) for deterministic provisioning
- Store POAP configs in a Git repo alongside the script so every change is auditable
- Use NDFC config compliance checks before and after maintenance windows to catch unintended drift
- NX-API sandbox (accessible at https://switch/sandbox) is invaluable for building API payloads interactively
- Prefer NX-API over SSH automation for speed and structured output; SSH scraping is fragile
- In Ansible, use `ansible.netcommon.httpapi` connection over `network_cli` for NX-OS whenever NX-API is enabled
- Tag Day-0, Day-1, Day-2 configs in version control so rollback scope is clear
- Use NAPALM `compare_config()` before every `commit_config()` to verify changes are what you expect
- Terraform `plan` is mandatory before `apply` in ACI automation; review every diff
- Schedule config backups with Git-based versioning to get free diff, blame, and rollback per device
- Keep POAP scripts idempotent so re-running on a partially provisioned switch converges to the correct state

## See Also

ansible, terraform, cisco-aci, vxlan, network-programmability, python-networking

## References

- [Cisco POAP Configuration Guide](https://www.cisco.com/c/en/us/td/docs/switches/datacenter/nexus9000/sw/93x/fundamentals/cisco-nexus-9000-series-nx-os-fundamentals-configuration-guide-93x/m-using-poap.html)
- [NDFC REST API Reference](https://developer.cisco.com/docs/nexus-dashboard-fabric-controller/)
- [NX-API CLI Reference](https://www.cisco.com/c/en/us/td/docs/switches/datacenter/nexus9000/sw/93x/programmability/guide/b-cisco-nexus-9000-series-nx-os-programmability-guide-93x/b-cisco-nexus-9000-series-nx-os-programmability-guide-93x_chapter_0101.html)
- [Cisco NX-OS Ansible Collection](https://galaxy.ansible.com/ui/repo/published/cisco/nxos/)
- [NAPALM NX-OS Driver](https://napalm.readthedocs.io/en/latest/support/nxos.html)
- [Terraform ACI Provider](https://registry.terraform.io/providers/CiscoDevNet/aci/latest/docs)
- [Cisco DevNet — NX-OS Programmability](https://developer.cisco.com/docs/nx-os/)
