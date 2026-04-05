# Cisco NSO (Network Services Orchestrator)

Model-driven network automation and orchestration platform using YANG models, transactional configuration management, and network element drivers (NEDs) to manage multi-vendor networks as a single system.

## Architecture Overview

```
+------------------------------------------------------------------+
|                        Cisco NSO                                  |
|                                                                   |
|  +-------------------+  +--------------------+  +---------------+ |
|  | Service Manager   |  | Device Manager     |  | RESTCONF /    | |
|  | - Service models  |  | - NED framework    |  | NETCONF API   | |
|  | - FASTMAP engine  |  | - Device configs   |  | - CLI (ncs)   | |
|  | - Nano services   |  | - Sync management  |  | - Web UI      | |
|  +-------------------+  +--------------------+  +---------------+ |
|           |                      |                                |
|  +----------------------------------------------------+          |
|  |              CDB (Configuration Database)           |          |
|  |  - Running config    - Operational data             |          |
|  |  - Service instances - Transaction log              |          |
|  +----------------------------------------------------+          |
|           |                      |                                |
|  +----------------------------------------------------+          |
|  |              Transaction Manager                    |          |
|  |  - ACID transactions across all devices             |          |
|  |  - Rollback on failure                              |          |
|  |  - Commit queue (async)                             |          |
|  +----------------------------------------------------+          |
+------------------------------------------------------------------+
         |              |              |              |
    +---------+    +---------+    +---------+    +---------+
    | CLI NED |    |NETCONF  |    |Generic  |    | SNMP    |
    | (IOS,   |    |NED      |    |NED      |    | NED     |
    |  IOS-XR,|    |(JUNOS,  |    |(REST,   |    |         |
    |  NX-OS) |    | IOS-XR) |    | custom) |    |         |
    +---------+    +---------+    +---------+    +---------+
         |              |              |              |
    +---------+    +---------+    +---------+    +---------+
    | Cisco   |    | Juniper |    | Cloud   |    | Legacy  |
    | IOS/    |    | JUNOS   |    | APIs    |    | SNMP    |
    | IOS-XR  |    |         |    |         |    | Devices |
    +---------+    +---------+    +---------+    +---------+
```

## NED (Network Element Driver) Types

### NED Comparison

| NED Type | Protocol | Use Case | Config Model |
|----------|----------|----------|-------------|
| CLI NED | SSH/Telnet | IOS, IOS-XE, IOS-XR, NX-OS, ASA | Screen-scraped CLI parsed to YANG |
| NETCONF NED | NETCONF/SSH | JUNOS, IOS-XR (native), EOS | Native YANG models |
| Generic NED | Any (REST, gRPC, custom) | Cloud APIs, SD-WAN controllers | Custom Java/Python adapter |
| SNMP NED | SNMP v2c/v3 | Legacy devices, monitoring | MIB to YANG mapping |

### NED Management

```bash
# List installed NEDs
ncs --version
ncs-make-package --list

# In NSO CLI: show installed packages
admin@nso> show packages package oper-status

# Reload packages after adding a NED
admin@nso> packages reload

# Check NED version for a device
admin@nso> show devices device ce0 platform
```

## Device Onboarding

### Add a Device

```
admin@nso# config
admin@nso(config)# devices device ce0
admin@nso(config-device-ce0)# address 10.1.1.1
admin@nso(config-device-ce0)# port 22
admin@nso(config-device-ce0)# ssh host-key-verification none
admin@nso(config-device-ce0)# authgroup default
admin@nso(config-device-ce0)# device-type cli ned-id cisco-ios-cli-6.85
admin@nso(config-device-ce0)# state admin-state unlocked
admin@nso(config-device-ce0)# commit
```

### Configure Auth Group

```
admin@nso# config
admin@nso(config)# devices authgroups group default
admin@nso(config-group-default)# umap admin
admin@nso(config-umap-admin)# remote-name admin
admin@nso(config-umap-admin)# remote-password Cisco123!
admin@nso(config-umap-admin)# commit
```

### Sync Device Configuration

```bash
# Sync FROM device to NSO (learn current config)
admin@nso> request devices device ce0 sync-from

# Sync TO device from NSO (push NSO config to device)
admin@nso> request devices device ce0 sync-to

# Check sync status
admin@nso> request devices device ce0 check-sync

# Sync all devices
admin@nso> request devices sync-from

# Compare configurations
admin@nso> request devices device ce0 compare-config
```

## NSO CLI Operations

### Navigation and Configuration

```bash
# Enter configuration mode
admin@nso> config

# Show running configuration
admin@nso> show running-config

# Show a specific device config
admin@nso> show running-config devices device ce0 config

# Show specific section
admin@nso> show running-config devices device ce0 config interface

# Configure a device through NSO
admin@nso# config
admin@nso(config)# devices device ce0 config
admin@nso(config-config)# interface Loopback 100
admin@nso(config-if)# ip address 10.100.0.1 255.255.255.255
admin@nso(config-if)# no shutdown
admin@nso(config-if)# commit

# Show configuration changes before commit
admin@nso(config)# show configuration

# Validate configuration
admin@nso(config)# validate

# Commit with label
admin@nso(config)# commit label "Add loopback 100"
```

### Dry-Run

```bash
# Dry-run: show what would be sent to devices (native format)
admin@nso(config)# commit dry-run outformat native

# Dry-run: show what would change in XML
admin@nso(config)# commit dry-run outformat xml

# Dry-run: show changes in CLI format
admin@nso(config)# commit dry-run outformat cli
```

### Rollback

```bash
# List available rollbacks
admin@nso> show configuration rollback changes

# Rollback last commit
admin@nso# rollback configuration
admin@nso# commit

# Rollback to a specific point
admin@nso# rollback configuration 10042
admin@nso# commit

# Show what a rollback would do
admin@nso# rollback configuration 10042
admin@nso# show configuration
```

### Commit Queue

```bash
# Commit with async queue (non-blocking)
admin@nso(config)# commit commit-queue async

# Commit with sync queue (wait for devices)
admin@nso(config)# commit commit-queue sync

# Commit queue with timeout
admin@nso(config)# commit commit-queue sync timeout 120

# Show commit queue status
admin@nso> show devices commit-queue

# Show specific queue item
admin@nso> show devices commit-queue queue-item 12345
```

## YANG Service Models

### Basic Service Model

```yang
module l3vpn {
  namespace "http://example.com/l3vpn";
  prefix l3vpn;

  import ietf-inet-types { prefix inet; }
  import tailf-common { prefix tailf; }
  import tailf-ncs { prefix ncs; }

  list l3vpn {
    key name;

    uses ncs:service-data;          // Required for NSO service tracking
    ncs:servicepoint "l3vpn-servicepoint";  // Links to service code

    leaf name {
      type string;
      description "VPN instance name";
    }

    leaf route-distinguisher {
      type string;
      description "VPN route distinguisher (e.g., 65000:100)";
    }

    list endpoint {
      key "device interface";

      leaf device {
        type leafref {
          path "/ncs:devices/ncs:device/ncs:name";
        }
        description "PE device name (must exist in NSO)";
      }

      leaf interface {
        type string;
        description "Interface name (e.g., GigabitEthernet0/0/0)";
      }

      leaf ip-address {
        type inet:ipv4-address;
      }

      leaf mask {
        type inet:ipv4-address;
      }

      leaf vlan-id {
        type uint16 {
          range "1..4094";
        }
      }
    }
  }
}
```

### Service Package Structure

```
packages/l3vpn/
  package-meta-data.xml     # Package metadata (name, version, NED deps)
  src/
    Makefile                 # Compile YANG + Java/Python
    yang/
      l3vpn.yang             # Service YANG model
    java/
      src/.../L3vpnRFS.java  # Service mapping logic (Reactive FASTMAP)
    python/
      l3vpn/
        main.py              # Python service code (alternative to Java)
  templates/
    l3vpn-template.xml       # XML device config template
  test/
    internal/
      lux/                   # Integration tests
```

### XML Config Template

```xml
<config-template xmlns="http://tail-f.com/ns/config/1.0"
                 servicepoint="l3vpn-servicepoint">
  <devices xmlns="http://tail-f.com/ns/ncs">
    <device>
      <name>{/endpoint/device}</name>
      <config>
        <!-- IOS-XR VRF configuration -->
        <vrf xmlns="http://tail-f.com/ned/cisco-ios-xr">
          <vrf-list>
            <name>{/name}</name>
            <address-family>
              <ipv4>
                <unicast>
                  <import>
                    <route-target>
                      <address-list>
                        <name>{/route-distinguisher}</name>
                      </address-list>
                    </route-target>
                  </import>
                  <export>
                    <route-target>
                      <address-list>
                        <name>{/route-distinguisher}</name>
                      </address-list>
                    </route-target>
                  </export>
                </unicast>
              </ipv4>
            </address-family>
          </vrf-list>
        </vrf>

        <!-- Interface configuration -->
        <interface xmlns="http://tail-f.com/ned/cisco-ios-xr">
          <GigabitEthernet>
            <id>{/endpoint/interface}</id>
            <vrf>{/name}</vrf>
            <ipv4>
              <address>
                <ip>{/endpoint/ip-address}</ip>
                <mask>{/endpoint/mask}</mask>
              </address>
            </ipv4>
          </GigabitEthernet>
        </interface>
      </config>
    </device>
  </devices>
</config-template>
```

## RESTCONF / NETCONF API

### RESTCONF Examples

```bash
# Get all devices
curl -s -u admin:admin \
  "http://localhost:8080/restconf/data/tailf-ncs:devices/device" \
  -H "Accept: application/yang-data+json"

# Get specific device config
curl -s -u admin:admin \
  "http://localhost:8080/restconf/data/tailf-ncs:devices/device=ce0/config" \
  -H "Accept: application/yang-data+json"

# Create a service instance
curl -X POST -u admin:admin \
  "http://localhost:8080/restconf/data" \
  -H "Content-Type: application/yang-data+json" \
  -d '{
    "l3vpn:l3vpn": [
      {
        "name": "CUSTOMER-A",
        "route-distinguisher": "65000:100",
        "endpoint": [
          {
            "device": "pe1",
            "interface": "GigabitEthernet0/0/0",
            "ip-address": "10.0.1.1",
            "mask": "255.255.255.252",
            "vlan-id": 100
          }
        ]
      }
    ]
  }'

# Dry-run via RESTCONF
curl -X POST -u admin:admin \
  "http://localhost:8080/restconf/data?dry-run=native" \
  -H "Content-Type: application/yang-data+json" \
  -d '{ ... }'

# Sync-from a device via RESTCONF
curl -X POST -u admin:admin \
  "http://localhost:8080/restconf/data/tailf-ncs:devices/device=ce0/sync-from"

# Check sync status
curl -X POST -u admin:admin \
  "http://localhost:8080/restconf/data/tailf-ncs:devices/device=ce0/check-sync"

# Rollback via RESTCONF
curl -X POST -u admin:admin \
  "http://localhost:8080/restconf/data/tailf-ncs:devices/rollback" \
  -H "Content-Type: application/yang-data+json" \
  -d '{"input": {"id": 10042}}'
```

### NETCONF Examples

```xml
<!-- Get device configuration via NETCONF -->
<rpc xmlns="urn:ietf:params:xml:ns:netconf:base:1.0" message-id="1">
  <get-config>
    <source><running/></source>
    <filter type="subtree">
      <devices xmlns="http://tail-f.com/ns/ncs">
        <device>
          <name>ce0</name>
          <config/>
        </device>
      </devices>
    </filter>
  </get-config>
</rpc>

<!-- Edit configuration via NETCONF -->
<rpc xmlns="urn:ietf:params:xml:ns:netconf:base:1.0" message-id="2">
  <edit-config>
    <target><running/></target>
    <config>
      <l3vpn xmlns="http://example.com/l3vpn">
        <name>CUSTOMER-B</name>
        <route-distinguisher>65000:200</route-distinguisher>
        <endpoint>
          <device>pe2</device>
          <interface>GigabitEthernet0/0/1</interface>
          <ip-address>10.0.2.1</ip-address>
          <mask>255.255.255.252</mask>
          <vlan-id>200</vlan-id>
        </endpoint>
      </l3vpn>
    </config>
  </edit-config>
</rpc>

<!-- Commit -->
<rpc xmlns="urn:ietf:params:xml:ns:netconf:base:1.0" message-id="3">
  <commit/>
</rpc>
```

## Reactive FASTMAP

### FASTMAP Concept

```
Without FASTMAP (imperative):
  Service create() --> Generate device configs --> Store in CDB
  Service modify() --> Figure out diff --> Update device configs
  Service delete() --> Remember what was created --> Remove configs

  Problem: Service code must handle create, modify, AND delete logic

With FASTMAP (declarative):
  Service create() --> Generate FULL device config from current inputs
  Service modify() --> NSO calls create() again with new inputs
                      --> NSO computes diff automatically
  Service delete() --> NSO removes everything create() produced
                      --> NSO tracked the mapping automatically

  Benefit: Service code ONLY implements create(). NSO handles the rest.
```

### Python Service Code (FASTMAP)

```python
import ncs
from ncs.application import Service

class L3vpnService(Service):
    @Service.create
    def cb_create(self, tctx, root, service, proplist):
        self.log.info(f'Service create: {service.name}')

        for endpoint in service.endpoint:
            # Get device configuration root
            device = root.devices.device[endpoint.device]
            dev_config = device.config

            # Apply VRF configuration
            vrf = dev_config.cisco_ios_xr__vrf.vrf_list.create(service.name)
            af = vrf.address_family.ipv4.unicast
            af.import_.route_target.address_list.create(service.route_distinguisher)
            af.export.route_target.address_list.create(service.route_distinguisher)

            # Apply interface configuration
            intf = dev_config.cisco_ios_xr__interface.GigabitEthernet.create(
                endpoint.interface
            )
            intf.vrf = service.name
            intf.ipv4.address.ip = endpoint.ip_address
            intf.ipv4.address.mask = endpoint.mask

            self.log.info(f'Configured {endpoint.device} {endpoint.interface}')

class Main(ncs.application.Application):
    def setup(self):
        self.register_service('l3vpn-servicepoint', L3vpnService)

    def teardown(self):
        pass
```

## Nano Services

### Nano Service Concept

```
Standard service: Single atomic transaction (all or nothing)

Nano service: Multi-step service with state machine
  State 1: Create base config   --> Wait for device response
  State 2: Verify connectivity  --> Wait for ping success
  State 3: Apply overlay config --> Wait for commit
  State 4: Run compliance check --> Done

Each state is a separate transaction.
Failure at state 3 does not roll back states 1-2.
Can retry individual states.
```

### Nano Service Plan

```
admin@nso> show l3vpn CUSTOMER-A plan

                                                    STATUS
COMPONENT  STATE          WHEN                      RESULT
------------------------------------------------------------------
self       init           2026-04-05T10:00:00-00:00  reached
self       ready          2026-04-05T10:00:05-00:00  reached
endpoint-pe1 init         2026-04-05T10:00:01-00:00  reached
endpoint-pe1 configured   2026-04-05T10:00:03-00:00  reached
endpoint-pe1 verified     2026-04-05T10:00:10-00:00  reached
endpoint-pe2 init         2026-04-05T10:00:01-00:00  reached
endpoint-pe2 configured   2026-04-05T10:00:04-00:00  reached
endpoint-pe2 verified     2026-04-05T10:00:12-00:00  reached
```

## LSA (Layered Service Architecture)

### LSA Topology

```
                 +------------------+
                 |   Upper NSO      |  Customer-facing service models
                 |   (CFS Layer)    |  Business logic, multi-domain
                 +------------------+
                    |           |
            +-------+           +-------+
            |                           |
   +------------------+       +------------------+
   |   Lower NSO #1   |       |   Lower NSO #2   |
   |   (RFS Layer)    |       |   (RFS Layer)     |
   |   DC devices     |       |   WAN devices     |
   +------------------+       +------------------+
      |    |    |                 |    |    |
    [DC1][DC2][DC3]            [PE1][PE2][PE3]

CFS = Customer-Facing Service (upper layer)
RFS = Resource-Facing Service (lower layer)
```

### LSA Benefits

| Benefit | Description |
|---------|-------------|
| Scale | Each lower NSO manages a subset of devices |
| Domain separation | Network teams own their domain's RFS |
| Independent upgrades | Lower NSO nodes upgraded independently |
| Performance | Parallel execution across lower nodes |
| Multi-vendor | Different NEDs per lower NSO |

## Compliance Reporting

```bash
# Define a compliance report
admin@nso# config
admin@nso(config)# compliance reports report check-ntp
admin@nso(config-report-check-ntp)# compare-template ntp-template ce0 ce1 ce2
admin@nso(config-report-check-ntp)# commit

# Run a compliance report
admin@nso> request compliance reports report check-ntp run

# Show compliance results
admin@nso> show compliance report-results

# Define a compliance template
admin@nso# config
admin@nso(config)# devices template ntp-template
admin@nso(config-template-ntp-template)# ned-id cisco-ios-cli-6.85
admin@nso(config-template-ntp-template)# config
admin@nso(config-config)# ntp server 10.0.0.1
admin@nso(config-config)# commit
```

## NSO Actions

```bash
# Custom action example (in service YANG model)
# YANG definition:
#   tailf:action check-connectivity {
#     tailf:actionpoint "check-connectivity";
#     input { leaf device { type string; } }
#     output { leaf result { type string; } }
#   }

# Invoke from CLI
admin@nso> request l3vpn CUSTOMER-A check-connectivity device pe1

# Invoke via RESTCONF
curl -X POST -u admin:admin \
  "http://localhost:8080/restconf/data/l3vpn:l3vpn=CUSTOMER-A/check-connectivity" \
  -H "Content-Type: application/yang-data+json" \
  -d '{"input": {"device": "pe1"}}'
```

## Development Workflow

### Create a New Service Package

```bash
# Generate package skeleton
ncs-make-package --service-skeleton python l3vpn \
  --component-class main.L3vpnService

# Or with Java
ncs-make-package --service-skeleton java l3vpn

# Build the package
cd packages/l3vpn/src
make clean all

# Reload packages in NSO
admin@nso> packages reload

# Verify package loaded
admin@nso> show packages package l3vpn oper-status
```

### Testing with ncs-netsim

```bash
# Create simulated network devices
ncs-netsim create-network cisco-ios-cli-6.85 3 ce
ncs-netsim create-network cisco-iosxr-cli-7.40 2 pe

# Start netsim devices
ncs-netsim start

# List netsim devices
ncs-netsim list

# Connect to a netsim device
ncs-netsim cli-c ce0

# Stop netsim
ncs-netsim stop

# Add devices to NSO from netsim
ncs-setup --netsim-dir ./netsim --dest .
```

### Common Development Commands

```bash
# Start NSO
ncs

# Stop NSO
ncs --stop

# Check NSO status
ncs --status

# NSO CLI (C-style)
ncs_cli -C -u admin

# NSO CLI (J-style / Juniper-like)
ncs_cli -J -u admin

# Show NSO logs
tail -f logs/ncs-python-vm-l3vpn.log

# Debug transaction
admin@nso(config)# commit dry-run outformat native
admin@nso(config)# commit | debug service

# Trace device communication
admin@nso# config
admin@nso(config)# devices device ce0 trace pretty
admin@nso(config)# commit
# Then check logs/netconf-ce0.trace
```

## Tips

- Always sync-from devices before making changes through NSO to avoid out-of-sync errors.
- Use dry-run before every commit in production — it shows exactly what CLI/NETCONF will be sent to each device.
- Write service code using FASTMAP (create-only) — never write separate modify/delete logic. NSO computes diffs automatically.
- Use commit queue for large deployments to avoid holding the transaction lock while waiting for slow devices.
- Test service packages with ncs-netsim before deploying to production — netsim creates virtual devices that behave like real ones.
- Use nano services for multi-step provisioning workflows where you need intermediate verification (like waiting for BGP to come up before applying overlay).
- Enable device trace (trace pretty) during development to see exactly what NSO sends to and receives from devices.
- Use compliance templates to detect configuration drift — run compliance reports regularly to ensure devices match desired state.
- For LSA deployments, keep CFS models stable and push change complexity to RFS models to minimize upper-layer disruption.
- Commit with a label for every change — it makes rollback identification much easier.

## See Also

- ansible, terraform, salt, puppet, chef

## References

- [Cisco NSO Documentation](https://developer.cisco.com/docs/nso/)
- [Cisco NSO Administration Guide](https://www.cisco.com/c/en/us/td/docs/net_mgmt/network_services_orchestrator/admin_guide.html)
- [NSO Developer Guide](https://developer.cisco.com/docs/nso/guides/)
- [NSO YANG Model Reference](https://developer.cisco.com/docs/nso/api/)
- [RFC 6241 — NETCONF Protocol](https://www.rfc-editor.org/rfc/rfc6241)
- [RFC 8040 — RESTCONF Protocol](https://www.rfc-editor.org/rfc/rfc8040)
- [RFC 7950 — YANG 1.1](https://www.rfc-editor.org/rfc/rfc7950)
- [Cisco NSO NED Documentation](https://developer.cisco.com/docs/nso/ned/)
