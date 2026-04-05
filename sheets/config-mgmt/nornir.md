# Nornir (Python Network Automation Framework)

Pluggable, multi-threaded Python framework for network automation — inventory-driven task execution without DSLs.

## Installation

### Install Nornir core and plugins

```bash
pip install nornir nornir_netmiko nornir_napalm nornir_scrapli nornir_utils nornir_jinja2
```

### Verify installation

```bash
python -c "import nornir; print(nornir.__version__)"
```

## Initialization

### Initialize from config file

```python
from nornir import InitNornir

nr = InitNornir(config_file="config.yaml")
```

### Initialize programmatically

```python
from nornir import InitNornir

nr = InitNornir(
    runner={"plugin": "threaded", "options": {"num_workers": 20}},
    inventory={
        "plugin": "SimpleInventory",
        "options": {
            "host_file": "inventory/hosts.yaml",
            "group_file": "inventory/groups.yaml",
            "defaults_file": "inventory/defaults.yaml",
        },
    },
)
```

### config.yaml

```yaml
---
inventory:
  plugin: SimpleInventory
  options:
    host_file: "inventory/hosts.yaml"
    group_file: "inventory/groups.yaml"
    defaults_file: "inventory/defaults.yaml"

runner:
  plugin: threaded
  options:
    num_workers: 20

logging:
  enabled: true
  level: INFO
```

## SimpleInventory

### hosts.yaml

```yaml
---
spine1:
  hostname: 10.0.0.1
  platform: ios
  port: 22
  username: admin
  password: secret
  groups:
    - spine
    - dc1
  data:
    role: spine
    site: dc1
    asn: 65001

leaf1:
  hostname: 10.0.0.11
  platform: nxos_ssh
  groups:
    - leaf
    - dc1
  data:
    role: leaf
    site: dc1
    vlans: [100, 200, 300]

juniper-rtr:
  hostname: 10.0.0.50
  platform: junos
  groups:
    - wan
  connection_options:
    napalm:
      extras:
        optional_args:
          config_format: set
```

### groups.yaml

```yaml
---
spine:
  data:
    role: spine
    ntp_server: 10.0.0.250

leaf:
  data:
    role: leaf
    ntp_server: 10.0.0.250

dc1:
  username: admin
  password: dc1_secret
  data:
    site: dc1
    dns_servers:
      - 8.8.8.8
      - 8.8.4.4

wan:
  platform: junos
  username: netops
  password: wan_secret
```

### defaults.yaml

```yaml
---
username: admin
password: default_pass
platform: ios
data:
  domain: example.com
  syslog_server: 10.0.0.200
```

## Filtering Inventory

### Filter by attribute

```python
# Filter by platform
ios_devices = nr.filter(platform="ios")

# Filter by group membership
spines = nr.filter(groups__contains="spine")

# Filter by data field
dc1_hosts = nr.filter(site="dc1")
```

### Filter with F objects

```python
from nornir.core.filter import F

# AND filter
ios_spines = nr.filter(F(platform="ios") & F(groups__contains="spine"))

# OR filter
ios_or_nxos = nr.filter(F(platform="ios") | F(platform="nxos_ssh"))

# NOT filter
non_junos = nr.filter(~F(platform="junos"))

# Nested data filter
high_asn = nr.filter(F(asn__gt=65000))

# Contains filter
vlan_100 = nr.filter(F(vlans__contains=100))
```

### Advanced filtering with functions

```python
def has_bgp(host):
    return host.data.get("asn") is not None

bgp_hosts = nr.filter(filter_func=has_bgp)
```

### Chain filters

```python
result = nr.filter(platform="ios").filter(site="dc1")
```

## Running Tasks

### Basic task execution

```python
from nornir_netmiko.tasks import netmiko_send_command

result = nr.run(task=netmiko_send_command, command_string="show version")
```

### Custom task

```python
from nornir.core.task import Task, Result

def get_device_info(task: Task) -> Result:
    cmd = task.run(task=netmiko_send_command, command_string="show version")
    version = cmd.result
    cmd2 = task.run(task=netmiko_send_command, command_string="show inventory")
    inventory = cmd2.result
    return Result(
        host=task.host,
        result=f"Version:\n{version}\n\nInventory:\n{inventory}",
    )

result = nr.run(task=get_device_info)
```

### Grouped task (sub-tasks)

```python
def configure_ntp(task: Task) -> Result:
    ntp_server = task.host.get("ntp_server", "10.0.0.250")
    commands = [
        f"ntp server {ntp_server}",
        "ntp authenticate",
    ]
    task.run(
        task=netmiko_send_config,
        config_commands=commands,
    )
    task.run(
        task=netmiko_send_command,
        command_string="show ntp status",
    )
    return Result(host=task.host, result="NTP configured")
```

## nornir_netmiko

### Send show commands

```python
from nornir_netmiko.tasks import netmiko_send_command

# Single command
result = nr.run(
    task=netmiko_send_command,
    command_string="show ip bgp summary",
)

# With TextFSM parsing
result = nr.run(
    task=netmiko_send_command,
    command_string="show ip interface brief",
    use_textfsm=True,
)

# With Genie parsing
result = nr.run(
    task=netmiko_send_command,
    command_string="show interfaces",
    use_genie=True,
)

# With timing parameters
result = nr.run(
    task=netmiko_send_command,
    command_string="show tech-support",
    read_timeout=120,
    delay_factor=4,
)
```

### Send configuration commands

```python
from nornir_netmiko.tasks import netmiko_send_config

# List of commands
result = nr.run(
    task=netmiko_send_config,
    config_commands=[
        "interface loopback0",
        "ip address 10.255.0.1 255.255.255.255",
        "no shutdown",
    ],
)

# From file
result = nr.run(
    task=netmiko_send_config,
    config_file="configs/acl.cfg",
)

# With exit config mode
result = nr.run(
    task=netmiko_send_config,
    config_commands=["hostname SPINE1"],
    exit_config_mode=True,
)
```

### File transfer

```python
from nornir_netmiko.tasks import netmiko_file_transfer

result = nr.run(
    task=netmiko_file_transfer,
    source_file="firmware.bin",
    dest_file="firmware.bin",
    direction="put",
)
```

## nornir_napalm

### NAPALM getters

```python
from nornir_napalm.plugins.tasks import napalm_get

# Get facts
result = nr.run(task=napalm_get, getters=["facts"])

# Multiple getters
result = nr.run(
    task=napalm_get,
    getters=["facts", "interfaces", "bgp_neighbors"],
)

# Getter with options
result = nr.run(
    task=napalm_get,
    getters=["bgp_neighbors_detail"],
    getters_options={"bgp_neighbors_detail": {"neighbor": "10.0.0.2"}},
)
```

### NAPALM configuration

```python
from nornir_napalm.plugins.tasks import napalm_configure

# Merge config
result = nr.run(
    task=napalm_configure,
    configuration="ntp server 10.0.0.250",
)

# Replace config from file
result = nr.run(
    task=napalm_configure,
    filename="configs/full_config.txt",
    replace=True,
)

# Dry run
result = nr.run(
    task=napalm_configure,
    configuration="hostname NEW-NAME",
    dry_run=True,
)
```

### NAPALM validation

```python
from nornir_napalm.plugins.tasks import napalm_validate

result = nr.run(
    task=napalm_validate,
    src="validation/bgp.yaml",
)
```

## nornir_scrapli

### Scrapli commands

```python
from nornir_scrapli.tasks import send_command, send_configs

# Send command
result = nr.run(task=send_command, command="show ip route")

# Send command with TextFSM
result = nr.run(
    task=send_command,
    command="show ip bgp summary",
    strip_prompt=True,
)

# Send configs
result = nr.run(
    task=send_configs,
    configs=[
        "interface loopback0",
        "ip address 10.255.0.1 255.255.255.255",
    ],
)
```

### Scrapli with structured data

```python
from nornir_scrapli.tasks import send_command

result = nr.run(
    task=send_command,
    command="show version",
)
for host, r in result.items():
    parsed = r.result  # raw output
    structured = r.scrapli_response.textfsm_parse_output()  # parsed
```

## nornir_utils

### Print results

```python
from nornir_utils.plugins.functions import print_result

result = nr.run(task=netmiko_send_command, command_string="show version")
print_result(result)

# Print specific severity
print_result(result, severity_level=logging.WARNING)

# Print single host
print_result(result["spine1"])
```

### Write results to file

```python
from nornir_utils.plugins.tasks.files import write_file

def backup_config(task: Task) -> Result:
    r = task.run(task=netmiko_send_command, command_string="show running-config")
    task.run(
        task=write_file,
        filename=f"backups/{task.host.name}.cfg",
        content=r.result,
    )

nr.run(task=backup_config)
```

### Load YAML data

```python
from nornir_utils.plugins.tasks.data import load_yaml

def load_vars(task: Task) -> Result:
    data = task.run(
        task=load_yaml,
        file=f"host_vars/{task.host.name}.yaml",
    )
    task.host["extra_vars"] = data.result

nr.run(task=load_vars)
```

## Results Handling

### Result object structure

```python
# AggregatedResult — dict-like, keyed by hostname
result = nr.run(task=netmiko_send_command, command_string="show version")

# Access per-host MultiResult
for host, multi_result in result.items():
    print(f"Host: {host}")
    print(f"Failed: {multi_result.failed}")
    # Each item is a Result
    for r in multi_result:
        print(f"  Task: {r.name}")
        print(f"  Output: {r.result}")
        print(f"  Changed: {r.changed}")
        print(f"  Failed: {r.failed}")
        print(f"  Diff: {r.diff}")
```

### Check for failures

```python
result = nr.run(task=some_task)

# Check overall failure
if result.failed:
    print("Some hosts failed")
    for host, r in result.failed_hosts.items():
        print(f"  {host}: {r.exception}")

# Raise on failure
from nornir_utils.plugins.functions import print_result
result.raise_on_error()
```

### Reset failed hosts

```python
# After fixing issues, reset failed hosts to retry
nr.data.reset_failed_hosts()

# Or selectively
nr.data.failed_hosts.discard("spine1")
```

## Jinja2 Templates

### Render templates with nornir_jinja2

```python
from nornir_jinja2.plugins.tasks import template_file, template_string

# Render from file
def generate_config(task: Task) -> Result:
    r = task.run(
        task=template_file,
        template="base.j2",
        path="templates/",
        **task.host.items(),
    )
    task.host["rendered_config"] = r.result
    return Result(host=task.host, result=r.result)

# Render from string
def render_acl(task: Task) -> Result:
    acl_template = """
    ip access-list extended MGMT
    {% for net in mgmt_networks %}
     permit ip {{ net }} any
    {% endfor %}
    """
    r = task.run(
        task=template_string,
        template=acl_template,
        mgmt_networks=task.host.get("mgmt_networks", []),
    )
    return Result(host=task.host, result=r.result)
```

### Template file (templates/base.j2)

```bash
# templates/base.j2
hostname {{ host.name }}
!
{% for iface, config in interfaces.items() %}
interface {{ iface }}
 ip address {{ config.ip }} {{ config.mask }}
 {% if config.description %}
 description {{ config.description }}
 {% endif %}
 no shutdown
!
{% endfor %}
```

### Render and deploy

```python
def configure_from_template(task: Task) -> Result:
    r = task.run(
        task=template_file,
        template="base.j2",
        path="templates/",
        **task.host.items(),
    )
    task.run(
        task=netmiko_send_config,
        config_commands=r.result.splitlines(),
    )
    return Result(host=task.host, changed=True, result="Config deployed")
```

## Processors

### Custom processor

```python
from nornir.core.inventory import Host
from nornir.core.task import AggregatedResult, MultiResult, Result

class SaveResultProcessor:
    def __init__(self, output_dir: str = "results"):
        self.output_dir = output_dir

    def task_started(self, task) -> None:
        print(f"Task started: {task.name}")

    def task_completed(self, task, result: AggregatedResult) -> None:
        print(f"Task completed: {task.name}")

    def task_instance_started(self, task, host: Host) -> None:
        pass

    def task_instance_completed(self, task, host: Host, result: MultiResult) -> None:
        with open(f"{self.output_dir}/{host.name}.txt", "w") as f:
            for r in result:
                f.write(f"{r.name}: {r.result}\n")

    def subtask_instance_started(self, task, host: Host) -> None:
        pass

    def subtask_instance_completed(self, task, host: Host, result: MultiResult) -> None:
        pass

# Use processor
nr_with_proc = nr.with_processors([SaveResultProcessor()])
nr_with_proc.run(task=netmiko_send_command, command_string="show version")
```

## Threading and Parallelism

### Configure thread count

```python
# In config.yaml
# runner:
#   plugin: threaded
#   options:
#     num_workers: 50

# Programmatically
nr = InitNornir(
    runner={"plugin": "threaded", "options": {"num_workers": 50}},
    ...
)
```

### Serial execution

```python
nr = InitNornir(
    runner={"plugin": "serial"},
    ...
)
```

### RetryRunner (community)

```python
# pip install nornir_rich
nr = InitNornir(
    runner={
        "plugin": "RetryRunner",
        "options": {
            "num_workers": 20,
            "num_retries": 3,
            "retry_delay": 5,
        },
    },
)
```

## Error Handling

### Try/except in tasks

```python
def safe_command(task: Task) -> Result:
    try:
        r = task.run(
            task=netmiko_send_command,
            command_string="show ip bgp summary",
        )
        return Result(host=task.host, result=r.result)
    except Exception as e:
        return Result(
            host=task.host,
            result=str(e),
            failed=True,
        )
```

### Handle failed hosts

```python
result = nr.run(task=some_task)

# Get successful hosts
successful = {h: r for h, r in result.items() if not r.failed}

# Retry failed hosts only
if result.failed_hosts:
    failed_nr = nr.filter(filter_func=lambda h: h.name in result.failed_hosts)
    retry_result = failed_nr.run(task=some_task)
```

### Connection retry pattern

```python
import time

def resilient_task(task: Task, max_retries: int = 3) -> Result:
    for attempt in range(max_retries):
        try:
            r = task.run(
                task=netmiko_send_command,
                command_string="show version",
            )
            return Result(host=task.host, result=r.result)
        except Exception as e:
            if attempt < max_retries - 1:
                time.sleep(2 ** attempt)
                task.host.close_connections()
            else:
                return Result(host=task.host, result=str(e), failed=True)
```

## Connection Management

### Close connections

```python
# Close all connections after task
nr.close_connections()

# Use context manager (Nornir 3.x)
with InitNornir(config_file="config.yaml") as nr:
    nr.run(task=some_task)
# connections auto-closed
```

### Connection options per host

```python
# In hosts.yaml
# router1:
#   hostname: 10.0.0.1
#   platform: ios
#   connection_options:
#     netmiko:
#       extras:
#         device_type: cisco_ios
#         secret: enable_secret
#         conn_timeout: 10
#     napalm:
#       extras:
#         optional_args:
#           transport: ssh
#           config_format: set
#     scrapli:
#       extras:
#         auth_strict_key: false
#         ssh_config_file: true
```

## Common Patterns

### Config backup workflow

```python
import os
from datetime import datetime

def backup_all(task: Task) -> Result:
    timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
    backup_dir = f"backups/{timestamp}"
    os.makedirs(backup_dir, exist_ok=True)

    r = task.run(
        task=netmiko_send_command,
        command_string="show running-config",
    )
    task.run(
        task=write_file,
        filename=f"{backup_dir}/{task.host.name}.cfg",
        content=r.result,
    )
    return Result(host=task.host, changed=True, result="Backup saved")

nr.run(task=backup_all)
```

### Compliance check

```python
def check_ntp(task: Task) -> Result:
    r = task.run(
        task=netmiko_send_command,
        command_string="show ntp associations",
        use_textfsm=True,
    )
    expected_server = task.host.get("ntp_server", "10.0.0.250")
    compliant = any(
        assoc["peer"] == expected_server
        for assoc in r.result
        if isinstance(r.result, list)
    )
    return Result(
        host=task.host,
        result=f"NTP {'compliant' if compliant else 'NON-COMPLIANT'}",
        failed=not compliant,
    )
```

## See Also

- NAPALM
- Ansible
- Netmiko
- Scrapli
- Jinja2
- Python

## References

- Nornir Docs: https://nornir.readthedocs.io/
- nornir_netmiko: https://github.com/ktbyers/nornir_netmiko
- nornir_napalm: https://github.com/nornir-automation/nornir_napalm
- nornir_scrapli: https://github.com/carlmontanari/nornir_scrapli
- nornir_utils: https://github.com/nornir-automation/nornir_utils
- nornir_jinja2: https://github.com/nornir-automation/nornir_jinja2
