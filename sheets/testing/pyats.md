# pyATS / Genie (Network Test Automation)

Cisco's test automation framework for network devices — testbed-driven connections, 1000+ parsers for structured data, state snapshots, diff comparison, and CI/CD-ready test harness.

## Testbed YAML

### Minimal testbed definition

```yaml
# testbed.yaml — defines devices, credentials, and connections
testbed:
  name: lab_testbed
  credentials:
    default:
      username: admin
      password: cisco123
    enable:
      password: enable123

devices:
  router1:
    os: iosxe
    type: router
    connections:
      defaults:
        class: unicon.Unicon
      cli:
        protocol: ssh
        ip: 10.0.0.1
        port: 22
    credentials:
      default:
        username: admin
        password: cisco123

  switch1:
    os: nxos
    type: switch
    connections:
      defaults:
        class: unicon.Unicon
      cli:
        protocol: ssh
        ip: 10.0.0.2
```

### Testbed with topology links

```yaml
topology:
  links:
    link1:
      type: ethernet
      interfaces:
        router1:
          interface: GigabitEthernet0/0/0
        switch1:
          interface: Ethernet1/1
    link2:
      type: ethernet
      interfaces:
        router1:
          interface: GigabitEthernet0/0/1
        router2:
          interface: GigabitEthernet0/0/0
```

### Multi-platform testbed

```yaml
devices:
  iosxe_device:
    os: iosxe
    platform: cat9k
    type: router
    connections:
      defaults:
        class: unicon.Unicon
      cli:
        protocol: ssh
        ip: 10.0.0.1

  iosxr_device:
    os: iosxr
    platform: asr9k
    type: router
    connections:
      defaults:
        class: unicon.Unicon
      cli:
        protocol: ssh
        ip: 10.0.0.2

  nxos_device:
    os: nxos
    platform: n9k
    type: switch
    connections:
      defaults:
        class: unicon.Unicon
      cli:
        protocol: ssh
        ip: 10.0.0.3

  junos_device:
    os: junos
    type: router
    connections:
      defaults:
        class: unicon.Unicon
      cli:
        protocol: ssh
        ip: 10.0.0.4
```

## Device Connections

### Connect and disconnect

```python
from genie.testbed import load

# Load testbed and connect
testbed = load('testbed.yaml')
device = testbed.devices['router1']
device.connect()                              # SSH connect using testbed creds

# Connect with options
device.connect(learn_hostname=True,           # learn device hostname
               log_stdout=False,              # suppress connection output
               init_exec_commands=[],          # skip initial exec commands
               init_config_commands=[])        # skip initial config commands

# Execute commands
output = device.execute('show version')       # raw CLI output (string)
device.configure('hostname ROUTER1')          # enter config mode, send command

# Disconnect
device.disconnect()
```

### Connection via context manager

```python
from genie.testbed import load

testbed = load('testbed.yaml')
device = testbed.devices['router1']

with device.connect(learn_hostname=True):
    output = device.execute('show ip interface brief')
    print(output)
# auto-disconnects when context exits
```

### Connect to all devices

```python
testbed = load('testbed.yaml')
testbed.connect()                             # connect all devices in parallel

for name, device in testbed.devices.items():
    if device.connected:
        print(f"{name}: connected")
        output = device.execute('show version')
```

## Genie Parsers (Structured Data)

### Parse show commands

```python
# Parse output into structured dict
parsed = device.parse('show ip interface brief')
# Returns:
# {'interface': {
#     'GigabitEthernet0/0/0': {
#         'ip_address': '10.0.0.1',
#         'interface_is_ok': 'YES',
#         'method': 'manual',
#         'status': 'up',
#         'protocol': 'up'
#     }, ...
# }}

# Access specific values
for intf, data in parsed['interface'].items():
    print(f"{intf}: {data['ip_address']} - {data['status']}")
```

### Common parsers

```python
# Routing
parsed = device.parse('show ip route')
parsed = device.parse('show ip bgp summary')
parsed = device.parse('show ip ospf neighbor')
parsed = device.parse('show ip eigrp neighbors')

# Interfaces
parsed = device.parse('show interfaces')
parsed = device.parse('show ip interface brief')
parsed = device.parse('show interfaces description')

# Platform
parsed = device.parse('show version')
parsed = device.parse('show inventory')
parsed = device.parse('show processes cpu')
parsed = device.parse('show memory statistics')

# MPLS / Segment Routing
parsed = device.parse('show mpls ldp neighbor')
parsed = device.parse('show mpls forwarding-table')

# Multicast
parsed = device.parse('show ip mroute')
parsed = device.parse('show ip pim neighbor')

# Security
parsed = device.parse('show access-lists')
parsed = device.parse('show crypto isakmp sa')
```

### Parse with raw output

```python
from genie.libs.parser.iosxe.show_interface import ShowIpInterfaceBrief

# Parse from previously captured output
parser = ShowIpInterfaceBrief(device=device)
parsed = parser.parse(output=raw_output_string)
```

### Check available parsers

```bash
# List all parsers for a platform
genie parse --testbed-file testbed.yaml --device router1 --output /tmp/parsed

# Search available parsers
pyats parse --help
```

## Genie Learn (Device Snapshot)

### Learn device features

```python
# Learn a single feature — returns Ops object with structured state
ospf = device.learn('ospf')
print(ospf.info)                              # full OSPF state as nested dict

bgp = device.learn('bgp')
interface = device.learn('interface')
routing = device.learn('routing')
platform = device.learn('platform')
vrf = device.learn('vrf')
acl = device.learn('acl')
arp = device.learn('arp')
dot1x = device.learn('dot1x')
hsrp = device.learn('hsrp')
mcast = device.learn('mcast')
stp = device.learn('stp')
vlan = device.learn('vlan')
```

### Snapshot all features

```python
# Learn all supported features
device.learn('all')

# Learn specific list
learnt = {}
for feature in ['ospf', 'bgp', 'interface', 'routing']:
    learnt[feature] = device.learn(feature)
```

## Genie Diff (State Comparison)

### Compare two snapshots

```python
from genie.utils.diff import Diff

# Take before/after snapshots
ospf_before = device.learn('ospf')
# ... make changes ...
ospf_after = device.learn('ospf')

# Compute diff
diff = Diff(ospf_before.info, ospf_after.info)
diff.findDiff()
print(diff)
# Output shows + (added), - (removed) lines
```

### Diff with exclusions

```python
# Exclude volatile fields from diff
diff = Diff(before.info, after.info, exclude=[
    'statistics',
    'counters',
    'last_clear',
    'uptime',
])
diff.findDiff()
```

### Compare running configs

```python
from genie.utils.diff import Diff

config_before = device.execute('show running-config')
# ... make changes ...
config_after = device.execute('show running-config')

diff = Diff(config_before, config_after)
diff.findDiff()
print(diff)
```

## pyATS Test Scripts (AEtest)

### Basic test structure

```python
import logging
from pyats import aetest
from genie.testbed import load

logger = logging.getLogger(__name__)

class CommonSetup(aetest.CommonSetup):
    """Connect to all devices."""

    @aetest.subsection
    def load_testbed(self, testbed):
        self.parent.parameters['testbed'] = testbed

    @aetest.subsection
    def connect_devices(self, testbed):
        for device in testbed.devices.values():
            device.connect(log_stdout=False)

    @aetest.subsection
    def mark_tests(self, testbed):
        # Dynamically loop testcases over devices
        aetest.loop.mark(
            InterfaceCheck,
            device=[d for d in testbed.devices.values()]
        )

class InterfaceCheck(aetest.Testcase):
    """Verify all interfaces are up."""

    @aetest.setup
    def setup(self, device):
        self.parsed = device.parse('show ip interface brief')

    @aetest.test
    def check_interfaces_up(self, device):
        for intf, data in self.parsed['interface'].items():
            if data['status'] != 'up':
                self.failed(f"{intf} is {data['status']} on {device.name}")

    @aetest.cleanup
    def cleanup(self):
        pass

class CommonCleanup(aetest.CommonCleanup):
    """Disconnect from all devices."""

    @aetest.subsection
    def disconnect(self, testbed):
        for device in testbed.devices.values():
            device.disconnect()
```

### Test with data-driven loops

```python
class BGPNeighborCheck(aetest.Testcase):

    @aetest.setup
    def setup(self, device):
        self.bgp = device.parse('show ip bgp summary')

    @aetest.test
    @aetest.loop(neighbor=['10.0.0.1', '10.0.0.2', '10.0.0.3'])
    def verify_neighbor(self, device, neighbor):
        neighbors = self.bgp.get('vrf', {}).get('default', {}) \
                        .get('neighbor', {})
        if neighbor not in neighbors:
            self.failed(f"BGP neighbor {neighbor} not found")
        state = neighbors[neighbor].get('session_state', '')
        if state.lower() != 'established':
            self.failed(f"Neighbor {neighbor} state: {state}")
```

### Run test with pyATS job file

```python
# job.py
import os
from pyats.easypy import run

def main(runtime):
    run(
        testscript=os.path.join(os.path.dirname(__file__), 'test_script.py'),
        runtime=runtime,
        taskid='interface_check',
    )
```

## pyATS CLI

### Run jobs and scripts

```bash
# Run a job
pyats run job job.py --testbed-file testbed.yaml

# Run a test script directly
pyats run job job.py --testbed-file testbed.yaml --html-logs /tmp/logs

# Generate HTML report
pyats logs view
pyats logs list
```

### Parse commands via CLI

```bash
# Parse a show command directly
pyats parse "show ip interface brief" \
  --testbed-file testbed.yaml \
  --device router1 \
  --output /tmp/parsed_output

# Parse multiple commands
pyats parse "show ip route" "show ip ospf neighbor" \
  --testbed-file testbed.yaml \
  --device router1
```

### Learn features via CLI

```bash
# Learn a feature
pyats learn ospf \
  --testbed-file testbed.yaml \
  --device router1 \
  --output /tmp/ospf_snapshot

# Learn all features
pyats learn all \
  --testbed-file testbed.yaml \
  --output /tmp/full_snapshot
```

### Diff snapshots via CLI

```bash
# Diff two snapshot directories
pyats diff /tmp/snapshot_before /tmp/snapshot_after \
  --output /tmp/diff_results

# Diff with exclusions
pyats diff /tmp/before /tmp/after \
  --exclude uptime \
  --exclude counters
```

## Blitz (YAML-Driven Testing)

### Blitz test definition

```yaml
# blitz_test.yaml
test:
  groups:
    - test_group:
        - verify_interfaces:
            - parse:
                device: router1
                command: show ip interface brief
                save:
                  - variable_name: intf_output
            - loop:
                loop_variable_name: interface
                value: "{{ intf_output['interface'] }}"
                actions:
                  - compare:
                      items:
                        - "'{{ interface.value.status }}' == 'up'"
```

### Blitz actions reference

```yaml
# Common Blitz actions:
actions:
  - execute:                                  # raw CLI command
      device: router1
      command: show version

  - parse:                                    # parsed CLI command
      device: router1
      command: show ip route

  - configure:                                # config mode command
      device: router1
      command: |
        interface Loopback99
         ip address 99.99.99.99 255.255.255.255

  - api:                                      # call Genie API
      device: router1
      function: get_interface_ip_address
      arguments:
        interface: GigabitEthernet0/0/0

  - learn:                                    # learn feature
      device: router1
      feature: ospf

  - compare:                                  # assert condition
      items:
        - "'up' == 'up'"

  - sleep:                                    # wait
      sleep_time: 10
```

### Run Blitz tests

```bash
pyats run job blitz_job.py --testbed-file testbed.yaml
```

## pyATS with Robot Framework

### Robot test file

```robot
*** Settings ***
Library    pyats.robot.pyATSRobot

*** Test Cases ***
Connect To Devices
    use testbed "testbed.yaml"
    connect to all devices

Verify Interface Status
    ${output}=    parse "show ip interface brief" on device "router1"
    Log    ${output}

Learn OSPF State
    ${ospf}=    learn "ospf" on device "router1"
    Log    ${ospf}
```

### Run Robot tests

```bash
pyats run robot robot_test.robot --testbed-file testbed.yaml
robot --listener pyats.robot.pyATSRobotListener robot_test.robot
```

## Network Health Checks

### Reachability verification

```python
class ReachabilityCheck(aetest.Testcase):

    @aetest.test
    def ping_test(self, device, destinations):
        for dest in destinations:
            result = device.execute(f'ping {dest} repeat 5')
            if 'Success rate is 100' not in result:
                self.failed(f"Ping to {dest} failed from {device.name}")
```

### CPU and memory thresholds

```python
class ResourceCheck(aetest.Testcase):

    @aetest.test
    def check_cpu(self, device, threshold=80):
        parsed = device.parse('show processes cpu')
        cpu_5sec = parsed.get('five_sec_cpu_total', 0)
        if cpu_5sec > threshold:
            self.failed(f"CPU at {cpu_5sec}% on {device.name}")

    @aetest.test
    def check_memory(self, device, threshold=80):
        parsed = device.parse('show memory statistics')
        used = parsed.get('statistics', {}).get('system', {}).get('used', 0)
        total = parsed.get('statistics', {}).get('system', {}).get('total', 1)
        pct = (used / total) * 100
        if pct > threshold:
            self.failed(f"Memory at {pct:.1f}% on {device.name}")
```

## CI/CD Integration

### GitLab CI example

```yaml
# .gitlab-ci.yml
stages:
  - test

network_tests:
  stage: test
  image: ciscotestautomation/pyats:latest
  script:
    - pyats run job job.py --testbed-file testbed.yaml
  artifacts:
    paths:
      - logs/
    when: always
```

### Jenkins pipeline

```groovy
pipeline {
    agent { docker { image 'ciscotestautomation/pyats:latest' } }
    stages {
        stage('Network Tests') {
            steps {
                sh 'pyats run job job.py --testbed-file testbed.yaml'
            }
        }
    }
    post {
        always {
            archiveArtifacts artifacts: 'logs/**', fingerprint: true
        }
    }
}
```

### Install pyATS

```bash
pip install "pyats[full]"                     # full install with all libs
pip install pyats genie                       # minimal install
pip install pyats.contrib                     # community parsers
```

## See Also

- gnmi-gnoi
- netconf
- restconf
- yang-models

## References

- pyATS documentation: https://developer.cisco.com/docs/pyats/
- Genie parser library: https://pubhub.devnetcloud.com/media/genie-feature-browser/docs/#/parsers
- pyATS Blitz guide: https://pubhub.devnetcloud.com/media/genie-docs/docs/blitz/index.html
- AEtest documentation: https://pubhub.devnetcloud.com/media/pyats/docs/aetest/index.html
