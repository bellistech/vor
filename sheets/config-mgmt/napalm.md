# NAPALM (Network Automation and Programmability Abstraction Layer with Multivendor support)

Unified Python API for interacting with network devices across vendors — consistent getters and config management regardless of platform.

## Installation

### Install NAPALM

```bash
pip install napalm
```

### Install specific drivers

```bash
pip install napalm-ios napalm-junos napalm-nxos napalm-eos
# Community drivers
pip install napalm-ros       # MikroTik RouterOS
pip install napalm-sros      # Nokia SR OS
pip install napalm-panos     # Palo Alto PAN-OS
```

### Verify installation

```bash
python -c "import napalm; print(napalm.__version__)"
napalm --help
```

## Supported Platforms

### Built-in drivers

```bash
# Platform      Driver Name    Transport
# IOS           ios            SSH (Netmiko)
# IOS-XR        iosxr          SSH (Netmiko) / XML-RPC
# NX-OS         nxos           NX-API (HTTP) or SSH
# EOS           eos            eAPI (HTTP)
# JunOS         junos          NETCONF (ncclient)
```

## Connection Handling

### Basic connection

```python
from napalm import get_network_driver

driver = get_network_driver("ios")
device = driver(
    hostname="10.0.0.1",
    username="admin",
    password="secret",
)
device.open()

# ... do work ...

device.close()
```

### Context manager

```python
from napalm import get_network_driver

driver = get_network_driver("eos")
with driver("10.0.0.1", "admin", "secret") as device:
    facts = device.get_facts()
    print(facts)
# connection auto-closed
```

### Optional arguments

```python
device = driver(
    hostname="10.0.0.1",
    username="admin",
    password="secret",
    optional_args={
        "port": 22,
        "secret": "enable_secret",       # enable password (IOS)
        "transport": "ssh",              # ssh or telnet
        "config_lock": True,             # lock config during changes
        "dest_file_system": "bootflash:",
        "ssh_config_file": "~/.ssh/config",
        "allow_agent": False,
        "timeout": 60,
    },
)
```

### Timeout and keepalive

```python
device = driver(
    hostname="10.0.0.1",
    username="admin",
    password="secret",
    timeout=120,
    optional_args={
        "keepalive": 30,
        "global_delay_factor": 2,
    },
)
```

## Getter Methods

### get_facts

```python
facts = device.get_facts()
# Returns:
# {
#     "hostname": "router1",
#     "fqdn": "router1.example.com",
#     "vendor": "Cisco",
#     "model": "CSR1000V",
#     "serial_number": "9ABCDEF1234",
#     "os_version": "17.03.04a",
#     "uptime": 1234567,
#     "interface_list": ["GigabitEthernet1", "GigabitEthernet2", "Loopback0"]
# }
```

### get_interfaces

```python
interfaces = device.get_interfaces()
# Returns:
# {
#     "GigabitEthernet1": {
#         "is_up": True,
#         "is_enabled": True,
#         "description": "Uplink to spine",
#         "mac_address": "00:1A:2B:3C:4D:5E",
#         "speed": 1000,
#         "mtu": 1500,
#         "last_flapped": 1234567890.0
#     },
#     ...
# }
```

### get_interfaces_ip

```python
ips = device.get_interfaces_ip()
# Returns:
# {
#     "GigabitEthernet1": {
#         "ipv4": {
#             "10.0.0.1": {"prefix_length": 24}
#         },
#         "ipv6": {
#             "2001:db8::1": {"prefix_length": 64}
#         }
#     }
# }
```

### get_interfaces_counters

```python
counters = device.get_interfaces_counters()
# Returns per-interface:
# {
#     "GigabitEthernet1": {
#         "tx_octets": 123456789,
#         "rx_octets": 987654321,
#         "tx_unicast_packets": 100000,
#         "rx_unicast_packets": 200000,
#         "tx_errors": 0,
#         "rx_errors": 0,
#         "tx_discards": 0,
#         "rx_discards": 0,
#     }
# }
```

### get_bgp_neighbors

```python
bgp = device.get_bgp_neighbors()
# Returns:
# {
#     "global": {
#         "router_id": "10.0.0.1",
#         "peers": {
#             "10.0.0.2": {
#                 "local_as": 65001,
#                 "remote_as": 65002,
#                 "remote_id": "10.0.0.2",
#                 "is_up": True,
#                 "is_enabled": True,
#                 "uptime": 86400,
#                 "address_family": {
#                     "ipv4 unicast": {
#                         "received_prefixes": 150,
#                         "accepted_prefixes": 148,
#                         "sent_prefixes": 100
#                     }
#                 }
#             }
#         }
#     }
# }
```

### get_bgp_neighbors_detail

```python
bgp_detail = device.get_bgp_neighbors_detail()
# Returns extended BGP neighbor information including:
# keepalive timers, hold time, messages sent/received,
# configured/active address families, route refresh capability
```

### get_bgp_config

```python
bgp_config = device.get_bgp_config()
# Returns BGP configuration structured data:
# local AS, router ID, neighbors, peer groups,
# address families, route policies
```

### get_lldp_neighbors

```python
lldp = device.get_lldp_neighbors()
# Returns:
# {
#     "GigabitEthernet1": [
#         {
#             "hostname": "spine1",
#             "port": "Ethernet1"
#         }
#     ]
# }
```

### get_lldp_neighbors_detail

```python
lldp_detail = device.get_lldp_neighbors_detail()
# Additional fields: remote_system_description,
# remote_system_capab, remote_port_description
```

### get_arp_table

```python
arp = device.get_arp_table()
# Returns:
# [
#     {
#         "interface": "GigabitEthernet1",
#         "mac": "00:1A:2B:3C:4D:5E",
#         "ip": "10.0.0.2",
#         "age": 300.0
#     },
#     ...
# ]
```

### get_mac_address_table

```python
mac_table = device.get_mac_address_table()
# Returns:
# [
#     {
#         "mac": "00:1A:2B:3C:4D:5E",
#         "interface": "Ethernet1",
#         "vlan": 100,
#         "static": False,
#         "active": True,
#         "moves": 0,
#         "last_move": 0.0
#     }
# ]
```

### get_route_to

```python
route = device.get_route_to(destination="10.0.0.0/24")
# Returns routing table entries for the prefix
# Including: protocol, next_hop, preference, metric, age
```

### get_environment

```python
env = device.get_environment()
# Returns:
# {
#     "fans": {"fan1": {"status": True}},
#     "temperature": {"CPU": {"temperature": 45.0, "is_alert": False, "is_critical": False}},
#     "power": {"PSU1": {"status": True, "capacity": 350.0, "output": 120.0}},
#     "cpu": {"%usage": 5.0},
#     "memory": {"available_ram": 8192, "used_ram": 2048}
# }
```

### get_ntp_servers / get_ntp_peers

```python
ntp_servers = device.get_ntp_servers()
# Returns: {"10.0.0.250": {}, "10.0.0.251": {}}

ntp_peers = device.get_ntp_peers()
# Returns: {"10.0.0.252": {}}
```

### get_snmp_information

```python
snmp = device.get_snmp_information()
# Returns: community strings, contact, location, chassis_id
```

### get_users

```python
users = device.get_users()
# Returns:
# {
#     "admin": {"level": 15, "password": "", "sshkeys": []},
#     "readonly": {"level": 1, "password": "", "sshkeys": []}
# }
```

### get_optics

```python
optics = device.get_optics()
# Returns per-interface transceiver data:
# output_power, input_power, laser_bias_current, temperature
# with alert thresholds
```

### get_config

```python
config = device.get_config()
# Returns:
# {
#     "running": "...",
#     "startup": "...",
#     "candidate": "..."   # JunOS/EOS only
# }

# Filtered config
config = device.get_config(retrieve="running")
# or
config = device.get_config(retrieve="running", full=True)
```

### get_network_instances

```python
vrf = device.get_network_instances()
# Returns VRF/routing instance information
```

### All getters at once

```python
# List available getters
print(device.get_facts.__doc__)

# Common pattern: collect all data
getters = [
    "get_facts", "get_interfaces", "get_interfaces_ip",
    "get_bgp_neighbors", "get_lldp_neighbors",
    "get_arp_table", "get_environment",
]
results = {}
for getter in getters:
    try:
        results[getter] = getattr(device, getter)()
    except NotImplementedError:
        results[getter] = "Not supported"
```

## Configuration Management

### Load merge candidate

```python
# From string
device.load_merge_candidate(config="ntp server 10.0.0.250")

# From file
device.load_merge_candidate(filename="configs/ntp.cfg")

# Multi-line config
device.load_merge_candidate(config="""
interface Loopback0
 ip address 10.255.0.1 255.255.255.255
 no shutdown
""")
```

### Load replace candidate

```python
# Full config replacement
device.load_replace_candidate(filename="configs/full_config.cfg")
```

### Compare config (diff)

```python
device.load_merge_candidate(config="hostname NEW-NAME")
diff = device.compare_config()
print(diff)
# Output:
# +hostname NEW-NAME
# -hostname OLD-NAME
```

### Commit or discard

```python
# Load candidate
device.load_merge_candidate(config="ntp server 10.0.0.250")

# Review diff
diff = device.compare_config()
if diff:
    print(f"Changes to apply:\n{diff}")
    # Commit changes
    device.commit_config()
else:
    print("No changes needed")
    device.discard_config()
```

### Rollback

```python
# Rollback to previous config (after commit)
device.rollback()
```

### Commit with confirmation (JunOS)

```python
# JunOS supports confirmed commit
device.load_merge_candidate(config="set system hostname NEW-NAME")
device.commit_config(confirmed=True, timeout=300)
# If not confirmed within 300 seconds, auto-rollback
```

### Full workflow

```python
from napalm import get_network_driver

driver = get_network_driver("ios")
device = driver("10.0.0.1", "admin", "secret")
device.open()

try:
    # Load config
    device.load_merge_candidate(filename="changes.cfg")

    # Review
    diff = device.compare_config()
    if not diff:
        print("No changes")
        device.discard_config()
    else:
        print(f"Diff:\n{diff}")
        # Commit
        device.commit_config()
        print("Committed successfully")
except Exception as e:
    print(f"Error: {e}")
    device.discard_config()
finally:
    device.close()
```

## Validation (Compliance Report)

### Validation file (validation.yaml)

```yaml
---
- get_facts:
    hostname: router1
    vendor: Cisco

- get_interfaces:
    GigabitEthernet1:
      is_up: true
      is_enabled: true
      speed: 1000

- get_bgp_neighbors:
    global:
      peers:
        _mode: strict
        10.0.0.2:
          is_up: true
          is_enabled: true

- get_ntp_servers:
    _mode: strict
    10.0.0.250: {}
    10.0.0.251: {}
```

### Run validation

```python
report = device.compliance_report("validation.yaml")
# Returns:
# {
#     "complies": True/False,
#     "skipped": [],
#     "get_facts": {
#         "complies": True,
#         "present": {"hostname": {"complies": True, "nested": False}},
#         "missing": [],
#         "extra": []
#     },
#     "get_interfaces": { ... },
#     "get_bgp_neighbors": { ... }
# }

if not report["complies"]:
    for getter, result in report.items():
        if isinstance(result, dict) and not result.get("complies", True):
            print(f"FAIL: {getter}")
```

### Validation modes

```yaml
# Default mode: check that specified keys exist and match
- get_ntp_servers:
    10.0.0.250: {}  # must exist, extra servers OK

# Strict mode: only specified keys allowed
- get_ntp_servers:
    _mode: strict
    10.0.0.250: {}  # must exist, no extra servers allowed
```

## NAPALM CLI Tool

### Command-line usage

```bash
# Get facts
napalm --user admin --password secret --vendor ios 10.0.0.1 call get_facts

# Get interfaces
napalm --user admin --password secret --vendor eos 10.0.0.1 call get_interfaces

# Compare config
napalm --user admin --password secret --vendor ios 10.0.0.1 configure changes.cfg --dry-run

# Apply config
napalm --user admin --password secret --vendor ios 10.0.0.1 configure changes.cfg

# Validate
napalm --user admin --password secret --vendor ios 10.0.0.1 validate validation.yaml
```

## NAPALM with Nornir

### nornir_napalm tasks

```python
from nornir import InitNornir
from nornir_napalm.plugins.tasks import napalm_get, napalm_configure, napalm_validate
from nornir_utils.plugins.functions import print_result

nr = InitNornir(config_file="config.yaml")

# Get facts from all devices
result = nr.run(task=napalm_get, getters=["facts", "interfaces"])
print_result(result)

# Configure all devices
result = nr.run(
    task=napalm_configure,
    configuration="ntp server 10.0.0.250",
    dry_run=True,
)
print_result(result)

# Validate all devices
result = nr.run(task=napalm_validate, src="validation.yaml")
print_result(result)
```

## NAPALM with Ansible

### Ansible NAPALM modules

```yaml
# napalm_get_facts
- name: Get device facts
  napalm_get_facts:
    hostname: "{{ inventory_hostname }}"
    username: admin
    password: secret
    dev_os: ios
    filter:
      - facts
      - interfaces
      - bgp_neighbors

# napalm_install_config
- name: Deploy configuration
  napalm_install_config:
    hostname: "{{ inventory_hostname }}"
    username: admin
    password: secret
    dev_os: ios
    config_file: "configs/{{ inventory_hostname }}.cfg"
    commit_changes: true
    diff_file: "diffs/{{ inventory_hostname }}.diff"

# napalm_validate
- name: Validate device state
  napalm_validate:
    hostname: "{{ inventory_hostname }}"
    username: admin
    password: secret
    dev_os: ios
    validation_file: "validations/{{ inventory_hostname }}.yaml"
```

## NAPALM with Salt

### Salt NAPALM proxy minion

```yaml
# /etc/salt/proxy (proxy minion config)
proxy:
  proxytype: napalm
  driver: ios
  host: 10.0.0.1
  username: admin
  password: secret

# Salt commands
# salt 'router1' net.facts
# salt 'router1' net.interfaces
# salt 'router1' net.load_config text="ntp server 10.0.0.250"
# salt 'router1' net.config source=running
```

## Error Handling

### Common exceptions

```python
from napalm.base.exceptions import (
    ConnectionException,
    MergeConfigException,
    ReplaceConfigException,
    CommitError,
    LockError,
    UnlockError,
    CommandErrorException,
)

try:
    device.open()
except ConnectionException as e:
    print(f"Connection failed: {e}")

try:
    device.load_merge_candidate(config="invalid config")
except MergeConfigException as e:
    print(f"Merge failed: {e}")

try:
    device.commit_config()
except CommitError as e:
    print(f"Commit failed: {e}")
    device.discard_config()
```

### Platform feature support check

```python
# Not all getters are supported on all platforms
try:
    optics = device.get_optics()
except NotImplementedError:
    print("get_optics not supported on this platform")
```

## See Also

- Nornir
- Ansible
- Netmiko
- Salt
- Python

## References

- NAPALM Docs: https://napalm.readthedocs.io/
- NAPALM GitHub: https://github.com/napalm-automation/napalm
- Supported Drivers: https://napalm.readthedocs.io/en/latest/support/
- NAPALM Ansible: https://napalm-automation.net/napalm-ansible/
- NAPALM Salt: https://napalm-automation.net/napalm-salt/
