# pyATS and Genie — Network Test Automation Architecture

> *pyATS (Python Automated Test System) is Cisco's modular test framework originally built for internal IOS-XE/IOS-XR/NX-OS validation, later open-sourced for network automation. Genie is the companion library providing 1000+ device parsers, Ops objects for feature-level state abstraction, and Conf objects for declarative configuration. Together they form a testbed-driven, parser-rich, diff-capable automation stack that transforms unstructured CLI output into programmatically verifiable state.*

---

## 1. Architecture Overview

### System Components

pyATS is organized as a set of loosely coupled Python packages, each responsible for a distinct layer:

| Package | Role | Key Classes |
|:---|:---|:---|
| `pyats.topology` | Testbed, Device, Interface, Link modeling | `Testbed`, `Device`, `Interface` |
| `pyats.connections` | Connection management, Unicon integration | `ConnectionManager` |
| `pyats.aetest` | Test harness (setup/test/cleanup lifecycle) | `CommonSetup`, `Testcase`, `CommonCleanup` |
| `pyats.easypy` | Job runner, parallel execution, reporting | `runtime`, `Task` |
| `pyats.results` | Result tracking and aggregation | `Passed`, `Failed`, `Errored`, `Skipped` |
| `pyats.log` | Logging infrastructure | `TaskLog` |
| `pyats.datastructures` | Shared data structures | `AttrDict`, `ListDict` |
| `genie.libs.parser` | 1000+ CLI parsers per platform | `ShowIpRoute`, `ShowBgpSummary`, etc. |
| `genie.libs.ops` | Feature Ops objects (state abstraction) | `Ospf`, `Bgp`, `Interface`, etc. |
| `genie.libs.conf` | Declarative configuration objects | `Ospf`, `Interface`, etc. |
| `genie.libs.sdk` | API library and triggers/verifications | `get_interface_ip_address`, etc. |
| `genie.utils` | Diff engine, timeout, config utilities | `Diff`, `Timeout` |

### Execution Flow

The standard pyATS execution path follows this sequence:

```
pyats run job → easypy runtime → Task(s) → AEtest script
                    │                           │
                    ├─ load testbed.yaml         ├─ CommonSetup
                    ├─ initialize logging        ├─ Testcase(s)
                    ├─ parallel task dispatch     ├─ CommonCleanup
                    └─ aggregate results         └─ return results
```

Each Task maps to a single test script. The easypy runtime supports parallel task execution across multiple scripts, collecting results into a unified report.

### Plugin Architecture

pyATS uses an entry-point-based plugin system. Each package registers its extensions via `setup.py` or `pyproject.toml` entry points:

- **Connection plugins** — custom transport backends beyond SSH (NETCONF, gNMI, REST)
- **Parser plugins** — community-contributed parsers via `pyats.contrib`
- **Reporter plugins** — custom result formatters (HTML, JUnit XML, JSON)
- **Processor plugins** — pre/post test hooks for data collection

---

## 2. Genie Parser Library

### Parser Architecture

Every Genie parser is a Python class inheriting from `MetaParser`. The parser defines a schema (expected output structure) and one or more `cli()` methods that use regular expressions to extract data from raw CLI output.

The parsing pipeline:

```
device.parse('show ip route')
    │
    ├─ 1. Lookup parser class by command + OS
    │     (registry in genie.libs.parser.<os>)
    │
    ├─ 2. Execute command on device (or accept raw output)
    │
    ├─ 3. Parse raw text via regex patterns
    │     (line-by-line state machine)
    │
    ├─ 4. Build nested dictionary matching schema
    │
    └─ 5. Validate output against schema
          (type checking, required keys)
```

### Schema Validation

Each parser declares a schema using nested dictionaries with type annotations:

```python
class ShowIpRouteSummary(MetaParser):
    schema = {
        'vrf': {
            Any(): {                          # VRF name (wildcard)
                'total_routes': int,
                Optional('connected'): int,
                Optional('static'): int,
                Optional('ospf'): int,
                Optional('bgp'): int,
            }
        }
    }
```

The `Any()` marker acts as a wildcard key (matching any string). `Optional()` marks keys that need not be present. Schema validation runs automatically after parsing, raising `SchemaError` on mismatch.

### Parser Coverage

As of the current release, Genie includes parsers for:

| Platform | Parser Count | Show Commands Covered |
|:---|:---:|:---|
| IOS-XE | ~400+ | Routing, switching, platform, security, wireless |
| IOS-XR | ~300+ | BGP, OSPF, ISIS, MPLS, segment routing |
| NX-OS | ~300+ | VPC, VXLAN, FabricPath, routing, platform |
| JunOS | ~100+ | Routing, interfaces, chassis, MPLS |
| ASA | ~50+ | NAT, ACL, failover, VPN |
| Linux | ~30+ | ip route, iptables, ss, df |

### Writing Custom Parsers

Custom parsers follow the same `MetaParser` pattern:

```python
from genie.metaparser import MetaParser
from genie.metaparser.util.schemaengine import Any, Optional
import re

class ShowCustomCommand(MetaParser):
    schema = {
        'entries': {
            Any(): {
                'status': str,
                'count': int,
            }
        }
    }

    cli_command = 'show custom command'

    def cli(self, output=None):
        if output is None:
            output = self.device.execute(self.cli_command)
        parsed = {'entries': {}}
        for line in output.splitlines():
            m = re.match(r'^(\S+)\s+(up|down)\s+(\d+)$', line)
            if m:
                parsed['entries'][m.group(1)] = {
                    'status': m.group(2),
                    'count': int(m.group(3)),
                }
        return parsed
```

---

## 3. Genie Ops Objects

### State Abstraction Layer

Ops objects provide a platform-independent abstraction over device state. Rather than parsing individual show commands, an Ops object aggregates multiple parsed outputs into a unified data model.

For example, `genie.libs.ops.ospf.ospf.Ospf` internally calls:

```
show ip ospf                    → process/router-level info
show ip ospf interface          → interface-level OSPF config
show ip ospf neighbor detail    → neighbor state
show ip ospf database           → LSDB summary
show ip ospf virtual-links      → virtual link state
```

These are merged into a single `Ospf.info` dictionary following a platform-independent schema.

### Ops Object Lifecycle

```
device.learn('ospf')
    │
    ├─ 1. Instantiate Ospf Ops object for device.os
    │
    ├─ 2. Execute all mapped show commands
    │     (defined in mapping datafiles per OS)
    │
    ├─ 3. Parse each command output
    │
    ├─ 4. Map parsed data into unified schema
    │     (using Ops mapping functions)
    │
    ├─ 5. Store in self.info (dict) and self.table (tabular)
    │
    └─ 6. Return Ops object
```

### Mapping Datafiles

Each Ops object uses a YAML mapping file that defines which show commands to execute and how to map parsed fields to the unified schema. These live in `genie.libs.ops.<feature>.<os>/`:

```yaml
# Simplified mapping structure
variables:
  ospf:
    source_class: ShowIpOspf
    mapping:
      info:
        'router_id': ['vrf', '(?P<vrf>.*)', 'router_id']
        'areas':
          source: ShowIpOspfInterface
          key_map: ['vrf', '(?P<vrf>.*)', 'areas', '(?P<area>.*)']
```

---

## 4. Testbed Topology Model

### Object Hierarchy

The testbed topology model represents the physical and logical structure of a test environment:

```
Testbed
├── credentials (default, enable, etc.)
├── servers (TFTP, syslog, AAA)
├── devices
│   ├── Device (router1)
│   │   ├── os, platform, type, custom attributes
│   │   ├── connections (cli, netconf, gnmi, rest)
│   │   ├── interfaces
│   │   │   ├── Interface (Gig0/0/0)
│   │   │   └── Interface (Gig0/0/1)
│   │   └── credentials (device-specific overrides)
│   └── Device (switch1)
│       └── ...
├── topology
│   └── links
│       ├── Link (link1)
│       │   └── interfaces: [router1:Gig0/0/0, switch1:Eth1/1]
│       └── Link (link2)
│           └── ...
└── custom attributes (any user-defined keys)
```

### Device Object Internals

A `Device` object is more than a connection wrapper. It maintains:

- **Connection pool** — multiple simultaneous connections (CLI, NETCONF, gNMI)
- **State machine** — tracks device state (any, enable, config, rommon)
- **Command log** — records all executed commands with timestamps
- **Abstraction tokens** — OS/platform/model hierarchy for API dispatch

### Connection Pool Management

pyATS supports multiple simultaneous connections per device:

```python
# Default connection (usually CLI/SSH)
device.connect()

# Named alternate connections
device.connect(via='netconf')
device.connect(via='gnmi')

# Access specific connections
device.cli.execute('show version')        # CLI connection
device.netconf.get_config()               # NETCONF connection

# Connection aliases
device.connect(alias='backup_cli', via='cli')
device.backup_cli.execute('show version')
```

The connection manager handles:
- Connection state tracking (connected/disconnected)
- Automatic reconnection on session loss
- Connection sharing across test sections
- Credential resolution (device-specific > testbed default)

---

## 5. Test Harness Design (AEtest)

### Execution Model

AEtest implements a three-phase test execution model:

```
CommonSetup          → runs once at script start
  ├── subsection 1   (topology connect, feature enable)
  ├── subsection 2   (data preparation, loop marking)
  └── subsection N

Testcase 1           → independent test block
  ├── setup          (per-testcase preparation)
  ├── test 1         (assertion/verification)
  ├── test 2
  └── cleanup        (per-testcase teardown)

Testcase 2
  └── ...

CommonCleanup        → runs once at script end
  ├── subsection 1   (disconnect, restore config)
  └── subsection N
```

### Result Propagation

Results flow upward through the hierarchy:

| Section | Possible Results | Effect on Parent |
|:---|:---|:---|
| Test step | Passed, Failed, Errored, Skipped, Blocked, Passx, Aborted | Rolls up to Testcase |
| Testcase setup | Failed/Errored | All tests in testcase become Blocked |
| Testcase cleanup | Failed/Errored | Testcase result unaffected (cleanup is best-effort) |
| CommonSetup subsection | Failed/Errored | Can block downstream Testcases |
| CommonCleanup | Failed/Errored | Script result unaffected |

Result priority (highest to lowest): `Aborted > Errored > Failed > Passx > Passed > Skipped > Blocked`

### Dynamic Looping

AEtest supports dynamic test iteration over parameters:

```python
# Static loop (decorator)
@aetest.loop(vrf=['default', 'mgmt', 'customer_a'])
class VrfCheck(aetest.Testcase):
    ...

# Dynamic loop (in CommonSetup)
aetest.loop.mark(VrfCheck, vrf=discovered_vrfs)

# Parametrized test methods
@aetest.test
@aetest.loop(prefix=['10.0.0.0/8', '172.16.0.0/12', '192.168.0.0/16'])
def verify_route(self, prefix):
    ...
```

---

## 6. Blitz YAML DSL

### Design Philosophy

Blitz enables network testing without writing Python. Test logic is expressed as YAML action sequences that the Blitz engine interprets and executes. This lowers the barrier for network engineers who are comfortable with YAML but less so with Python.

### Action Types

| Action | Purpose | Returns |
|:---|:---|:---|
| `execute` | Send raw CLI command | Raw output string |
| `parse` | Parse CLI command to structured data | Nested dictionary |
| `configure` | Send config-mode commands | None |
| `api` | Call Genie SDK API function | Function return value |
| `learn` | Learn feature Ops object | Ops info dictionary |
| `sleep` | Wait N seconds | None |
| `compare` | Assert conditions | Pass/Fail |
| `loop` | Iterate over data | Per-iteration results |
| `parallel` | Execute actions concurrently | Aggregated results |
| `run_condition` | Conditional execution | Conditional results |

### Variable System

Blitz supports variable capture and substitution:

```yaml
actions:
  - parse:
      device: router1
      command: show ip interface brief
      save:
        - variable_name: interfaces           # save parsed output
          filter: contains('interface')        # optional filter

  - compare:
      items:
        - "'{{ interfaces.interface.Loopback0.status }}' == 'up'"
```

Variables persist across actions within a test and can be referenced using Jinja2 `{{ }}` syntax.

---

## 7. Network Test Automation Patterns

### Golden Config Validation

The most common pyATS pattern: capture a known-good state, then periodically verify the network matches:

1. **Baseline** — `pyats learn all --output golden/` during maintenance window
2. **Verify** — `pyats learn all --output current/` on schedule
3. **Diff** — `pyats diff golden/ current/ --exclude counters uptime`
4. **Alert** — non-empty diff triggers notification

### Pre/Post Change Validation

```
Before change:
  pyats learn ospf bgp interface --output pre_change/

Execute change:
  (manual or automated config push)

After change:
  pyats learn ospf bgp interface --output post_change/

Validate:
  pyats diff pre_change/ post_change/ --exclude counters
  → Expected changes present?
  → No unexpected side effects?
```

### Continuous Monitoring

Periodic pyATS runs that verify:
- All BGP neighbors established
- OSPF adjacencies match expected count
- Interface error counters below threshold
- CPU/memory within bounds
- Route count within expected range

These map naturally to CI/CD pipelines triggered on schedule or config change events.

---

## 8. pyATS in CI/CD Pipelines

### Pipeline Architecture

```
Code Change (git push)
    │
    ├─ CI/CD trigger (Jenkins, GitLab CI, GitHub Actions)
    │
    ├─ Provision test environment (optional: CML, EVE-NG, GNS3)
    │
    ├─ Deploy candidate config to lab devices
    │
    ├─ Run pyATS test suite
    │   ├─ Pre-checks (baseline state)
    │   ├─ Config deployment
    │   ├─ Post-checks (verify state)
    │   ├─ Traffic validation (optional)
    │   └─ Rollback on failure
    │
    ├─ Generate reports (HTML, JUnit XML)
    │
    └─ Gate decision: promote or reject change
```

### Report Formats

pyATS generates multiple report formats:

| Format | Tool | Purpose |
|:---|:---|:---|
| HTML | `pyats logs view` | Interactive browser-based report |
| JUnit XML | `--junit-xml` flag | CI/CD integration (Jenkins, GitLab) |
| JSON | `--json` flag | Programmatic consumption |
| Console | Default stdout | Developer feedback |

### Parallel Execution

For large test suites, easypy supports parallel task execution:

```python
# job.py — parallel tasks
from pyats.easypy import run

def main(runtime):
    run(testscript='test_routing.py', runtime=runtime,
        taskid='routing', devices=['router1', 'router2'])
    run(testscript='test_switching.py', runtime=runtime,
        taskid='switching', devices=['switch1', 'switch2'])
```

Tasks run concurrently by default. The runtime manages device locking to prevent connection conflicts when multiple tasks target the same device.

---

## 9. Diff Engine Internals

### Algorithm

The Genie Diff engine performs recursive dictionary comparison:

1. Walk both dictionaries in parallel
2. For each key present in either dict:
   - **Added** — key exists only in the second dict
   - **Removed** — key exists only in the first dict
   - **Changed** — key exists in both but values differ
   - **Nested** — both values are dicts/lists, recurse
3. List comparison uses index-based or key-based matching

### Exclusion System

The exclusion system supports:

- **Exact key match** — `exclude=['counters']` skips any key named `counters` at any depth
- **Path-based exclusion** — `exclude=['interface.*.counters']` skips counters only under interfaces
- **Regex patterns** — `exclude=[re.compile(r'.*_timestamp')]` skips keys matching pattern
- **Custom callables** — `exclude=[lambda k,v: k.startswith('_')]` arbitrary logic

### Diff Output Format

```
+ added_key: new_value                        # present in second, absent in first
- removed_key: old_value                      # present in first, absent in second
  changed_key:
-   old_value
+   new_value
```

---

## 10. Connection Libraries

### Unicon

Unicon is the default connection library for CLI access. It provides:

- **State machine** — models device CLI states (enable, config, rommon, etc.)
- **Dialog handling** — automated response to interactive prompts
- **Service abstraction** — `execute()`, `configure()`, `reload()`, `ping()`, `copy()`
- **Error detection** — pattern matching for error messages in output
- **Platform plugins** — per-OS state machines and prompt patterns

### Connection Lifecycle

```
device.connect()
    │
    ├─ 1. Resolve connection parameters (via, credentials)
    ├─ 2. Spawn SSH/Telnet session (pexpect/paramiko)
    ├─ 3. Handle initial prompts (username, password, enable)
    ├─ 4. Detect device state (enable, config, etc.)
    ├─ 5. Execute init commands (term length 0, etc.)
    └─ 6. Mark connection as active
```

---

## See Also

- gnmi-gnoi
- netconf
- restconf
- yang-models

## References

- pyATS architecture guide: https://developer.cisco.com/docs/pyats/
- Genie SDK documentation: https://developer.cisco.com/docs/genie-docs/
- Genie parser source: https://github.com/CiscoTestAutomation/genieparser
- pyATS Blitz documentation: https://pubhub.devnetcloud.com/media/genie-docs/docs/blitz/index.html
- Unicon documentation: https://developer.cisco.com/docs/unicon/
- RFC 8199 — YANG Module Classification: https://datatracker.ietf.org/doc/html/rfc8199
